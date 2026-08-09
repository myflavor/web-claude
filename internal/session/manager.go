package session

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

const UploadDirName = ".web-claude/uploads"

type Info struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Cwd          string    `json:"cwd"`
	CwdRel       string    `json:"cwdRel"`
	CreatedAt    time.Time `json:"createdAt"`
	Clients      int       `json:"clients"`
	Alive        bool      `json:"alive"`
	LastActiveAt time.Time `json:"lastActiveAt"`
}

type Client struct {
	ID   string
	Send chan []byte
	Cols uint16
	Rows uint16
}

type Session struct {
	ID        string
	Title     string
	Cwd       string
	CwdRel    string
	CreatedAt time.Time

	cmd  *exec.Cmd
	ptmx *os.File

	mu           sync.Mutex
	clients      map[string]*Client
	ring         *Ring
	alive        bool
	lastActiveAt time.Time
	closed       bool
	ptyCols      uint16
	ptyRows      uint16
}

type Manager struct {
	mu sync.RWMutex

	sessions map[string]*Session

	claudeBin      string
	claudeArgs     []string
	claudeEnv      map[string]string
	claudeHome     string // if set, override HOME for child
	configDir      string // optional CLAUDE_CONFIG_DIR
	inheritUserEnv bool
	ringSize       int
	idleTTL        time.Duration

	logger *log.Logger
}

type ManagerConfig struct {
	ClaudeBin      string
	ClaudeArgs     []string
	ClaudeEnv      map[string]string
	ClaudeHome     string
	ConfigDir      string
	InheritUserEnv bool
	RingSize       int
	IdleTTL        time.Duration
	Logger         *log.Logger
}

func NewManager(cfg ManagerConfig) *Manager {
	lg := cfg.Logger
	if lg == nil {
		lg = log.Default()
	}
	m := &Manager{
		sessions:       make(map[string]*Session),
		claudeBin:      cfg.ClaudeBin,
		claudeArgs:     append([]string{}, cfg.ClaudeArgs...),
		claudeEnv:      cfg.ClaudeEnv,
		claudeHome:     cfg.ClaudeHome,
		configDir:      cfg.ConfigDir,
		inheritUserEnv: cfg.InheritUserEnv,
		ringSize:       cfg.RingSize,
		idleTTL:        cfg.IdleTTL,
		logger:         lg,
	}
	if m.idleTTL > 0 {
		go m.reaperLoop()
	}
	return m
}

// CreateOptions controls how the process is started.
type CreateOptions struct {
	Cwd          string // logical project path (UI / uploads)
	CwdRel       string
	WorkDir      string // process working directory; empty = Cwd
	Title        string
	ResumeID     string // if set, run `claude -r <id>`
	ContinueLast bool   // if set (and no ResumeID), run `claude -c`
	// Shell, if non-empty, runs `bash -lc Shell` instead of claude.
	// Use for multi-step setup (e.g. git clone then exec claude).
	Shell     string
	ExtraArgs []string
}

func (m *Manager) Create(cwd, cwdRel, title string) (*Session, error) {
	return m.CreateWith(CreateOptions{Cwd: cwd, CwdRel: cwdRel, Title: title})
}

func (m *Manager) CreateWith(opts CreateOptions) (*Session, error) {
	cwd := opts.Cwd
	cwdRel := opts.CwdRel
	title := opts.Title
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = cwd
	}

	st, err := os.Stat(workDir)
	if err != nil {
		return nil, fmt.Errorf("workdir: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("workdir is not a directory")
	}
	// Logical cwd may not exist yet (e.g. about-to-be-cloned path).
	if cwd != workDir {
		if st, err := os.Stat(cwd); err == nil && !st.IsDir() {
			return nil, fmt.Errorf("cwd is not a directory")
		}
	}
	if title == "" {
		title = pathBaseTitle(cwd, cwdRel)
	}

	if st, err := os.Stat(cwd); err == nil && st.IsDir() {
		_ = os.MkdirAll(filepath.Join(cwd, UploadDirName), 0o755)
	}

	var cmd *exec.Cmd
	if opts.Shell != "" {
		cmd = exec.Command("bash", "-lc", opts.Shell)
	} else {
		args := append([]string{}, m.claudeArgs...)
		if opts.ResumeID != "" {
			args = append(args, "-r", opts.ResumeID)
		} else if opts.ContinueLast {
			args = append(args, "-c")
		}
		args = append(args, opts.ExtraArgs...)
		cmd = exec.Command(m.claudeBin, args...)
	}
	cmd.Dir = workDir
	cmd.Env = m.buildEnv()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	s := &Session{
		ID:           uuid.NewString(),
		Title:        title,
		Cwd:          cwd,
		CwdRel:       cwdRel,
		CreatedAt:    time.Now(),
		cmd:          cmd,
		ptmx:         ptmx,
		clients:      make(map[string]*Client),
		ring:         NewRing(m.ringSize),
		alive:        true,
		lastActiveAt: time.Now(),
		ptyCols:      80,
		ptyRows:      24,
	}

	m.mu.Lock()
	m.sessions[s.ID] = s
	m.mu.Unlock()

	go m.readLoop(s)
	go m.waitLoop(s)

	m.logger.Printf("session created id=%s cwd=%s workdir=%s resume=%s shell=%v pid=%d",
		s.ID, cwd, workDir, opts.ResumeID, opts.Shell != "", cmd.Process.Pid)
	return s, nil
}

// pathBaseTitle is the last path segment of the project dir.
func pathBaseTitle(cwd, cwdRel string) string {
	title := ""
	if cwdRel != "" {
		title = filepath.Base(cwdRel)
	} else if cwd != "" {
		title = filepath.Base(cwd)
	}
	if title == "" || title == "." || title == string(os.PathSeparator) {
		return "projects"
	}
	return title
}

func (m *Manager) buildEnv() []string {
	// Start from host environment so PATH, locale, and existing tools work (WSL/native).
	env := os.Environ()
	env = append(env, "TERM=xterm-256color")

	if !m.inheritUserEnv && m.claudeHome != "" {
		// Docker / isolated mode: point HOME at mounted volume.
		env = setEnv(env, "HOME", m.claudeHome)
	}
	if m.configDir != "" {
		env = setEnv(env, "CLAUDE_CONFIG_DIR", m.configDir)
	}

	// Explicit API/model overrides always win.
	for k, v := range m.claudeEnv {
		env = setEnv(env, k, v)
	}
	return env
}

func setEnv(env []string, key, val string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + val
			return env
		}
	}
	return append(env, prefix+val)
}

func (m *Manager) readLoop(s *Session) {
	buf := make([]byte, 32*1024)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			s.ring.Write(chunk)
			s.broadcast(chunk)
			s.touch()
		}
		if err != nil {
			if err != io.EOF {
				m.logger.Printf("session %s pty read: %v", s.ID, err)
			}
			return
		}
	}
}

func (m *Manager) waitLoop(s *Session) {
	err := s.cmd.Wait()
	s.mu.Lock()
	s.alive = false
	s.mu.Unlock()
	if err != nil {
		m.logger.Printf("session %s process exited: %v", s.ID, err)
	} else {
		m.logger.Printf("session %s process exited ok", s.ID)
	}
	// notify clients with a small banner (optional plain text)
	msg := []byte("\r\n\r\n[claude-mobile] process exited\r\n")
	s.ring.Write(msg)
	s.broadcast(msg)
}

func (s *Session) touch() {
	s.mu.Lock()
	s.lastActiveAt = time.Now()
	s.mu.Unlock()
}

func (s *Session) broadcast(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clients {
		select {
		case c.Send <- data:
		default:
			// slow client: drop chunk rather than block PTY reader
		}
	}
}

func (s *Session) Info() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Info{
		ID:           s.ID,
		Title:        s.Title,
		Cwd:          s.Cwd,
		CwdRel:       s.CwdRel,
		CreatedAt:    s.CreatedAt,
		Clients:      len(s.clients),
		Alive:        s.alive,
		LastActiveAt: s.lastActiveAt,
	}
}

func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *Manager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Info, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.Info())
	}
	return out
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found")
	}
	s.close()
	m.logger.Printf("session deleted id=%s", id)
	return nil
}

func (s *Session) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	clients := make([]*Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.clients = map[string]*Client{}
	s.alive = false
	ptmx := s.ptmx
	cmd := s.cmd
	s.mu.Unlock()

	for _, c := range clients {
		close(c.Send)
	}
	if ptmx != nil {
		_ = ptmx.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// Attach adds a client. Caller must drain Send and call Detach.
// On attach, ring buffer is written to Send once (best-effort).
func (s *Session) Attach() (*Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("session closed")
	}
	c := &Client{
		ID:   uuid.NewString(),
		Send: make(chan []byte, 64),
	}
	// replay
	replay := s.ring.Bytes()
	if len(replay) > 0 {
		// send synchronously into buffer if possible
		select {
		case c.Send <- replay:
		default:
			go func() { c.Send <- replay }()
		}
	}
	s.clients[c.ID] = c
	s.lastActiveAt = time.Now()
	return c, nil
}

func (s *Session) Detach(clientID string) {
	s.mu.Lock()
	c, ok := s.clients[clientID]
	if !ok {
		s.mu.Unlock()
		return
	}
	delete(s.clients, clientID)
	// Safe to close: client is removed so only one Detach/close owns the channel.
	// session.close() closes remaining clients under the same lock after clearing map.
	close(c.Send)
	s.lastActiveAt = time.Now()
	// Recompute PTY size from remaining clients (min cols/rows).
	s.recomputeSizeLocked()
	s.mu.Unlock()
}

func (s *Session) Write(p []byte) error {
	s.mu.Lock()
	ptmx := s.ptmx
	closed := s.closed
	s.mu.Unlock()
	if closed || ptmx == nil {
		return fmt.Errorf("session not writable")
	}
	_, err := ptmx.Write(p)
	if err == nil {
		s.touch()
	}
	return err
}

// ResizeFromClient records this client's viewport and sets the shared PTY to the
// minimum cols/rows among all connected clients (mobile-friendly when PC also attached).
func (s *Session) ResizeFromClient(clientID string, cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return nil
	}
	// hard clamp to sane bounds
	if cols < 20 {
		cols = 20
	}
	if rows < 5 {
		rows = 5
	}
	if cols > 500 {
		cols = 500
	}
	if rows > 200 {
		rows = 200
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil {
		return fmt.Errorf("no pty")
	}
	if c, ok := s.clients[clientID]; ok {
		c.Cols = cols
		c.Rows = rows
	}
	return s.recomputeSizeLocked()
}

// Resize applies an explicit size without client tracking (single-client fallback).
func (s *Session) Resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil {
		return fmt.Errorf("no pty")
	}
	// If any client has reported size, prefer min across clients and ignore bare resize.
	for _, c := range s.clients {
		if c.Cols > 0 && c.Rows > 0 {
			return s.recomputeSizeLocked()
		}
	}
	return s.applySizeLocked(cols, rows)
}

// recomputeSizeLocked picks min(cols), min(rows) across clients that reported size.
// Must hold s.mu.
func (s *Session) recomputeSizeLocked() error {
	var minCols, minRows uint16
	found := false
	for _, c := range s.clients {
		if c.Cols == 0 || c.Rows == 0 {
			continue
		}
		if !found {
			minCols, minRows = c.Cols, c.Rows
			found = true
			continue
		}
		if c.Cols < minCols {
			minCols = c.Cols
		}
		if c.Rows < minRows {
			minRows = c.Rows
		}
	}
	if !found {
		// keep current pty size
		return nil
	}
	return s.applySizeLocked(minCols, minRows)
}

func (s *Session) applySizeLocked(cols, rows uint16) error {
	if cols == s.ptyCols && rows == s.ptyRows {
		return nil
	}
	if err := pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols}); err != nil {
		return err
	}
	s.ptyCols = cols
	s.ptyRows = rows
	s.lastActiveAt = time.Now()
	return nil
}

func (s *Session) WritePath(path string) error {
	// insert path as if typed, with a trailing space for convenience
	return s.Write([]byte(path + " "))
}

func (m *Manager) reaperLoop() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for range t.C {
		m.reapIdle()
	}
}

func (m *Manager) reapIdle() {
	if m.idleTTL <= 0 {
		return
	}
	now := time.Now()
	var toDelete []string

	m.mu.RLock()
	for id, s := range m.sessions {
		s.mu.Lock()
		clients := len(s.clients)
		last := s.lastActiveAt
		alive := s.alive
		s.mu.Unlock()
		if clients == 0 && now.Sub(last) > m.idleTTL {
			toDelete = append(toDelete, id)
			continue
		}
		// also reap dead processes with no clients after short grace
		if !alive && clients == 0 && now.Sub(last) > time.Hour {
			toDelete = append(toDelete, id)
		}
	}
	m.mu.RUnlock()

	for _, id := range toDelete {
		m.logger.Printf("reaping idle session %s", id)
		_ = m.Delete(id)
	}
}
