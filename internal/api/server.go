package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/xin/claude-mobile/internal/auth"
	"github.com/xin/claude-mobile/internal/claudehist"
	"github.com/xin/claude-mobile/internal/config"
	"github.com/xin/claude-mobile/internal/fsutil"
	"github.com/xin/claude-mobile/internal/session"
)

type Server struct {
	cfg    *config.Config
	guard  *auth.Guard
	mgr    *session.Manager
	logger *log.Logger
	upg    websocket.Upgrader
}

func New(cfg *config.Config, guard *auth.Guard, mgr *session.Manager, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{
		cfg:    cfg,
		guard:  guard,
		mgr:    mgr,
		logger: logger,
		upg: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// Single-user NAS tool; tighten if exposed more broadly.
				return true
			},
		},
	}
}

func (s *Server) Handler(static http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("POST /api/logout", s.handleLogout)
	mux.HandleFunc("GET /api/me", s.withAuth(s.handleMe))

	mux.HandleFunc("GET /api/sessions", s.withAuth(s.handleListSessions))
	mux.HandleFunc("POST /api/sessions", s.withAuth(s.handleCreateSession))
	mux.HandleFunc("DELETE /api/sessions/{id}", s.withAuth(s.handleDeleteSession))
	mux.HandleFunc("GET /api/sessions/{id}", s.withAuth(s.handleGetSession))
	mux.HandleFunc("GET /api/sessions/{id}/ws", s.handleSessionWS) // auth inside (query/cookie)

	mux.HandleFunc("GET /api/conversations", s.withAuth(s.handleListConversations))
	mux.HandleFunc("GET /api/fs", s.withAuth(s.handleFS))
	mux.HandleFunc("POST /api/fs/mkdir", s.withAuth(s.handleMkdir))
	mux.HandleFunc("POST /api/fs/clone", s.withAuth(s.handleClone))
	mux.HandleFunc("POST /api/sessions/{id}/upload", s.withAuth(s.handleUpload))

	// static last
	mux.Handle("/", static)

	return mux
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.guard.Authorized(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"time": time.Now().UTC(),
	})
}


func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	tok := body.Token
	if tok == "" {
		tok = body.Password
	}
	if !s.guard.ValidToken(tok) {
		time.Sleep(200 * time.Millisecond)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}
	s.guard.SetCookie(w, tok)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.guard.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"projectsRoot": s.cfg.ProjectsRoot,
	})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	list := s.mgr.List()
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, sess.Info())
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path         string   `json:"path"`
		Title        string   `json:"title"`
		ResumeID     string   `json:"resumeId"`
		ContinueLast bool     `json:"continueLast"`
		CloneURL     string   `json:"cloneUrl"`
		CloneName    string   `json:"cloneName"`
		ClaudeArgs   []string `json:"claudeArgs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	path := body.Path
	title := ""
	if body.ResumeID != "" && path == "" {
		if e, err := claudehist.Lookup(s.cfg.ClaudeHome, s.cfg.ClaudeConfigDir, s.cfg.ProjectsRoot, body.ResumeID); err == nil && e != nil {
			path = e.CwdRel
		}
	}
	if body.ResumeID != "" {
		if len(body.ResumeID) < 8 || strings.ContainsAny(body.ResumeID, " \t\n") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resumeId"})
			return
		}
	}

	extraArgs, err := sanitizeClaudeArgs(body.ClaudeArgs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Git clone → full history, stream in PTY, then exec claude.
	if body.CloneURL != "" {
		url, err := validateGitURL(body.CloneURL)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		parentAbs, err := fsutil.Resolve(s.cfg.ProjectsRoot, path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		name := strings.TrimSpace(body.CloneName)
		if name == "" {
			name = fsutil.NameFromGitURL(url)
		}
		safe, err := fsutil.SafeName(name)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if strings.ContainsAny(safe, "'\\\"`$") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid folder name"})
			return
		}
		targetAbs := filepath.Join(parentAbs, safe)
		if _, err := os.Stat(targetAbs); err == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "directory already exists"})
			return
		}
		targetRel, err := fsutil.Rel(s.cfg.ProjectsRoot, targetAbs)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		claudeBin := s.cfg.ClaudeBin
		if claudeBin == "" {
			claudeBin = "claude"
		}
		var b strings.Builder
		b.WriteString("set -euo pipefail\n")
		b.WriteString("echo \"[web-claude] git clone \"")
		b.WriteString(shellQuote(url + " " + safe))
		b.WriteString("\n")
		b.WriteString("git clone ")
		b.WriteString(shellQuote(url))
		b.WriteByte(' ')
		b.WriteString(shellQuote(safe))
		b.WriteByte('\n')
		b.WriteString("cd ")
		b.WriteString(shellQuote(safe))
		b.WriteByte('\n')
		b.WriteString("echo \"[web-claude] starting claude…\"\n")
		b.WriteString("exec ")
		b.WriteString(shellQuote(claudeBin))
		for _, a := range s.cfg.ClaudeArgs {
			b.WriteByte(' ')
			b.WriteString(shellQuote(a))
		}
		for _, a := range extraArgs {
			b.WriteByte(' ')
			b.WriteString(shellQuote(a))
		}
		shell := b.String()

		sess, err := s.mgr.CreateWith(session.CreateOptions{
			Cwd:     targetAbs,
			CwdRel:  targetRel,
			WorkDir: parentAbs,
			Title:   "",
			Shell:   shell,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, sess.Info())
		return
	}

	abs, err := fsutil.Resolve(s.cfg.ProjectsRoot, path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	rel, err := fsutil.Rel(s.cfg.ProjectsRoot, abs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	sess, err := s.mgr.CreateWith(session.CreateOptions{
		Cwd:          abs,
		CwdRel:       rel,
		Title:        title,
		ResumeID:     body.ResumeID,
		ContinueLast: body.ContinueLast && body.ResumeID == "",
		ExtraArgs:    extraArgs,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, sess.Info())
}


func validateGitURL(raw string) (string, error) {
	url := strings.TrimSpace(raw)
	if url == "" {
		return "", fmt.Errorf("url required")
	}
	if strings.ContainsAny(url, " \t\n\r;&|<>$`\\\"'") {
		return "", fmt.Errorf("invalid url")
	}
	if !(strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://")) {
		return "", fmt.Errorf("url must be http(s)/ssh git remote")
	}
	return url, nil
}

// shellQuote wraps s in single quotes for bash -lc.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// sanitizeClaudeArgs allows a small whitelist of claude CLI flags for per-session use.
func sanitizeClaudeArgs(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	allowExact := map[string]bool{
		"--dangerously-skip-permissions":       true,
		"--allow-dangerously-skip-permissions": true,
		"--bare":    true,
		"--chrome":  true,
		"--verbose": true,
	}
	allowValue := map[string]bool{
		"--model":                true,
		"--permission-mode":      true,
		"--effort":               true,
		"--agent":                true,
		"--name":                 true,
		"-n":                     true,
		"--append-system-prompt": true,
		"--system-prompt":        true,
		"--settings":             true,
		"--add-dir":              true,
	}
	out := make([]string, 0, len(in))
	for i := 0; i < len(in); i++ {
		a := strings.TrimSpace(in[i])
		if a == "" {
			continue
		}
		if strings.ContainsAny(a, "\t\n\r;&|<>$`\\\"'") || strings.Contains(a, "..") {
			return nil, fmt.Errorf("invalid claude arg")
		}
		if allowExact[a] {
			out = append(out, a)
			continue
		}
		if eq := strings.IndexByte(a, '='); eq > 0 {
			key, val := a[:eq], a[eq+1:]
			if !allowValue[key] {
				return nil, fmt.Errorf("claude arg not allowed: %s", key)
			}
			if val == "" || strings.ContainsAny(val, " \t\n\r;&|<>$`\\\"'") {
				return nil, fmt.Errorf("invalid claude arg value")
			}
			out = append(out, a)
			continue
		}
		if allowValue[a] {
			if i+1 >= len(in) {
				return nil, fmt.Errorf("claude arg %s needs a value", a)
			}
			val := strings.TrimSpace(in[i+1])
			if val == "" || strings.HasPrefix(val, "-") || strings.ContainsAny(val, " \t\n\r;&|<>$`\\\"'") {
				return nil, fmt.Errorf("invalid value for %s", a)
			}
			out = append(out, a, val)
			i++
			continue
		}
		return nil, fmt.Errorf("claude arg not allowed: %s", a)
	}
	if len(out) > 16 {
		return nil, fmt.Errorf("too many claude args")
	}
	return out, nil
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	var filter *string
	if r.URL.Query().Has("path") {
		p := strings.Trim(strings.TrimPrefix(r.URL.Query().Get("path"), "/"), "/")
		filter = &p
	}
	list, err := claudehist.List(s.cfg.ClaudeHome, s.cfg.ClaudeConfigDir, s.cfg.ProjectsRoot, 80, filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": list})
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	rel, err := fsutil.Mkdir(s.cfg.ProjectsRoot, body.Path, body.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "path": rel})
}

// handleClone is a silent/non-interactive fallback; prefer create with cloneUrl.
func (s *Server) handleClone(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		URL  string `json:"url"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	url, err := validateGitURL(body.URL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	parentAbs, err := fsutil.Resolve(s.cfg.ProjectsRoot, body.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = fsutil.NameFromGitURL(url)
	}
	safe, err := fsutil.SafeName(name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	target := filepath.Join(parentAbs, safe)
	if _, err := os.Stat(target); err == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "directory already exists"})
		return
	}
	cmd := exec.CommandContext(r.Context(), "git", "clone", url, target)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(target)
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	rel, err := fsutil.Rel(s.cfg.ProjectsRoot, target)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "path": rel})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mgr.Delete(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleFS(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	entries, cur, err := fsutil.ListDir(s.cfg.ProjectsRoot, path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	parent := ""
	if cur != "" {
		parent = filepath.ToSlash(filepath.Dir(cur))
		if parent == "." {
			parent = ""
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    cur,
		"parent":  parent,
		"entries": entries,
	})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.UploadMaxBytes+1024*1024)
	if err := r.ParseMultipartForm(s.cfg.UploadMaxBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large or invalid form"})
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	if hdr.Size > s.cfg.UploadMaxBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large"})
		return
	}

	orig := filepath.Base(hdr.Filename)
	orig = sanitizeFilename(orig)
	if orig == "" {
		orig = "upload.bin"
	}

	dir := filepath.Join(sess.Cwd, session.UploadDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mkdir failed"})
		return
	}
	name := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), uuid.NewString()[:8])
	if ext := filepath.Ext(orig); ext != "" {
		name += ext
	}
	dstPath := filepath.Join(dir, name)

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(dstPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "write failed"})
		return
	}

	_ = sess.WritePath(dstPath)

	writeJSON(w, http.StatusOK, map[string]any{
		"name":       orig,
		"storedName": name,
		"path":       dstPath,
		"size":       hdr.Size,
	})
}

func (s *Server) handleSessionWS(w http.ResponseWriter, r *http.Request) {
	if !s.guard.Authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	conn, err := s.upg.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	client, err := sess.Attach()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("session closed\r\n"))
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for data := range client.Send {
			_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		}
	}()

	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		if mt == websocket.TextMessage {
			if len(msg) > 0 && msg[0] == '{' {
				var ctrl struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
					Data string `json:"data"`
				}
				if err := json.Unmarshal(msg, &ctrl); err == nil {
					switch ctrl.Type {
					case "resize":
						_ = sess.ResizeFromClient(client.ID, ctrl.Cols, ctrl.Rows)
					case "input":
						_ = sess.Write([]byte(ctrl.Data))
					}
					continue
				}
			}
			_ = sess.Write(msg)
			continue
		}
		if mt == websocket.BinaryMessage {
			_ = sess.Write(msg)
		}
	}

	sess.Detach(client.ID)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, name)
	return strings.TrimSpace(name)
}
