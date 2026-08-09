package claudehist

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xin/claude-mobile/internal/fsutil"
)

// Entry is a resumable Claude conversation that has actual transcript data.
type Entry struct {
	SessionID string    `json:"sessionId"`
	Project   string    `json:"project"` // absolute cwd Claude used
	CwdRel    string    `json:"cwdRel"`  // relative to projects root if inside
	Display   string    `json:"display"` // last user message snippet
	UpdatedAt time.Time `json:"updatedAt"`
}

type histLine struct {
	Display   string `json:"display"`
	SessionID string `json:"sessionId"`
	Project   string `json:"project"`
	Timestamp int64  `json:"timestamp"` // ms
}

// ConfigDir resolves ~/.claude (or CLAUDE_CONFIG_DIR).
func ConfigDir(claudeHome, configDir string) string {
	if configDir != "" {
		return configDir
	}
	if claudeHome != "" {
		return filepath.Join(claudeHome, ".claude")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// List recent conversations that can actually be resumed with `claude -r`.
// Only sessions with a project transcript jsonl are included.
// If filterRel is non-nil, only conversations whose CwdRel equals *filterRel
// (exact directory; empty string = projects root) are returned.
func List(claudeHome, configDir, projectsRoot string, limit int, filterRel *string) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	cfg := ConfigDir(claudeHome, configDir)
	if cfg == "" {
		return nil, nil
	}

	rootAbs, err := filepath.Abs(projectsRoot)
	if err != nil {
		return nil, err
	}
	rootAbs = filepath.Clean(rootAbs)

	// sessionId -> last user prompt from history.jsonl
	type prompt struct {
		display string
		project string
		ts      time.Time
	}
	prompts := map[string]prompt{}
	histPath := filepath.Join(cfg, "history.jsonl")
	if f, err := os.Open(histPath); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			var h histLine
			if json.Unmarshal([]byte(line), &h) != nil || h.SessionID == "" {
				continue
			}
			p := prompts[h.SessionID]
			if h.Display != "" {
				p.display = h.Display
			}
			if h.Project != "" {
				p.project = h.Project
			}
			if h.Timestamp > 0 {
				ts := time.UnixMilli(h.Timestamp)
				if ts.After(p.ts) {
					p.ts = ts
				}
			}
			prompts[h.SessionID] = p
		}
		_ = f.Close()
	}

	// Only include sessions that have a transcript jsonl under projects/.
	// When filtering by directory, only scan that project's encoded dir.
	// Claude encodes absolute paths by replacing '/' with '-'
	// e.g. /data/projects/demo → -data-projects-demo
	wantProjectDir := ""
	if filterRel != nil {
		if filterAbs, err := fsutil.Resolve(rootAbs, *filterRel); err == nil {
			wantProjectDir = strings.ReplaceAll(filterAbs, "/", "-")
		}
	}

	byID := map[string]*Entry{}
	projRoot := filepath.Join(cfg, "projects")
	if ents, err := os.ReadDir(projRoot); err == nil {
		for _, ent := range ents {
			if !ent.IsDir() {
				continue
			}
			if wantProjectDir != "" && ent.Name() != wantProjectDir {
				continue
			}
			dirPath := filepath.Join(projRoot, ent.Name())
			files, _ := os.ReadDir(dirPath)
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
					continue
				}
				st, err := f.Info()
				if err != nil || st.Size() < 32 {
					continue
				}
				sid := strings.TrimSuffix(f.Name(), ".jsonl")
				e := &Entry{
					SessionID: sid,
					UpdatedAt: st.ModTime(),
				}
				if pr, ok := prompts[sid]; ok {
					e.Display = pr.display
					if pr.project != "" {
						e.Project = pr.project
					}
					if pr.ts.After(e.UpdatedAt) {
						e.UpdatedAt = pr.ts
					}
				}
				if e.Project == "" {
					e.Project = decodeProjectDir(ent.Name())
				}
				byID[sid] = e
			}
		}
	}

	out := make([]Entry, 0, len(byID))
	for _, e := range byID {
		if e.Project == "" {
			continue
		}
		proj := filepath.Clean(e.Project)
		rel, err := fsutil.Rel(rootAbs, proj)
		if err != nil {
			continue
		}
		e.CwdRel = rel
		if filterRel != nil {
			want := strings.Trim(*filterRel, "/")
			got := strings.Trim(rel, "/")
			if want != got {
				continue
			}
		}
		// Leave Display empty when unknown; UI owns the placeholder string.
		out = append(out, *e)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Lookup returns the conversation entry for a session id, or nil.
func Lookup(claudeHome, configDir, projectsRoot, sessionID string) (*Entry, error) {
	if sessionID == "" {
		return nil, nil
	}
	// Prefer direct transcript file hit under projects/*/id.jsonl
	cfg := ConfigDir(claudeHome, configDir)
	if cfg == "" {
		return nil, nil
	}
	rootAbs, err := filepath.Abs(projectsRoot)
	if err != nil {
		return nil, err
	}
	rootAbs = filepath.Clean(rootAbs)

	projRoot := filepath.Join(cfg, "projects")
	if ents, err := os.ReadDir(projRoot); err == nil {
		for _, ent := range ents {
			if !ent.IsDir() {
				continue
			}
			p := filepath.Join(projRoot, ent.Name(), sessionID+".jsonl")
			st, err := os.Stat(p)
			if err != nil || st.Size() < 32 {
				continue
			}
			e := &Entry{
				SessionID: sessionID,
				UpdatedAt: st.ModTime(),
				Project:   decodeProjectDir(ent.Name()),
			}
			// enrich display from history if present
			if display, project, ts := lastPrompt(cfg, sessionID); display != "" || project != "" {
				if display != "" {
					e.Display = display
				}
				if project != "" {
					e.Project = project
				}
				if !ts.IsZero() && ts.After(e.UpdatedAt) {
					e.UpdatedAt = ts
				}
			}
			if rel, err := fsutil.Rel(rootAbs, filepath.Clean(e.Project)); err == nil {
				e.CwdRel = rel
			} else {
				continue
			}
			return e, nil
		}
	}
	return nil, nil
}

func lastPrompt(cfg, sessionID string) (display, project string, ts time.Time) {
	f, err := os.Open(filepath.Join(cfg, "history.jsonl"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.Contains(line, sessionID) {
			continue
		}
		var h histLine
		if json.Unmarshal([]byte(line), &h) != nil || h.SessionID != sessionID {
			continue
		}
		if h.Display != "" {
			display = h.Display
		}
		if h.Project != "" {
			project = h.Project
		}
		if h.Timestamp > 0 {
			ts = time.UnixMilli(h.Timestamp)
		}
	}
	return
}

// decodeProjectDir turns Claude's "-data-projects-demo" into "/data/projects/demo".
// Lossy when path segments contain dashes; only used as fallback.
func decodeProjectDir(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "-") {
		return "/" + strings.ReplaceAll(strings.TrimPrefix(name, "-"), "-", "/")
	}
	return name
}
