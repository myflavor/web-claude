package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr string

	// AuthToken is the shared secret for login / API / WebSocket.
	AuthToken string

	// ProjectsRoot is the only tree clients may browse and start sessions in.
	ProjectsRoot string

	// ClaudeHome is the directory Claude uses for config/sessions
	// (maps to ~/.claude via HOME or CLAUDE_CONFIG_DIR).
	// Empty means: use the real user home (native/WSL mode).
	ClaudeHome string

	// ClaudeConfigDir if set, forces CLAUDE_CONFIG_DIR for child processes.
	// When empty and ClaudeHome is set, CLAUDE_CONFIG_DIR = ClaudeHome/.claude is used
	// only if we also override HOME; see session manager.
	//
	// Preferred native mode: ClaudeHome empty → inherit process HOME / existing ~/.claude.
	ClaudeConfigDir string

	// ClaudeBin is the claude executable (PATH lookup if empty/"claude").
	ClaudeBin string

	// ClaudeArgs extra args appended when starting claude.
	ClaudeArgs []string

	// Env passed into every claude process (ANTHROPIC_* etc).
	// Only keys present in the process environment (or .env) are forwarded.
	ClaudeEnv map[string]string

	// InheritUserEnv: if true, do not rewrite HOME; use the host user environment
	// so existing ~/.claude and shell PATH work (WSL / bare metal).
	InheritUserEnv bool

	// RingBufferBytes of recent PTY output kept for reconnect replay.
	RingBufferBytes int

	// IdleTTL: session with zero clients longer than this may be reaped.
	// 0 disables reaping.
	IdleTTL time.Duration

	// UploadMaxBytes per file.
	UploadMaxBytes int64

	// CookieSecure sets Secure flag on auth cookie (enable behind HTTPS).
	CookieSecure bool
}

func Load() (*Config, error) {
	// Optional dotenv for bare-metal convenience (does not override existing env).
	loadDotEnv(".env")
	if v := os.Getenv("CLAUDE_MOBILE_ENV_FILE"); v != "" {
		loadDotEnv(v)
	}
	if v := os.Getenv("WEB_CLAUDE_ENV_FILE"); v != "" {
		loadDotEnv(v)
	}

	mode := strings.ToLower(firstEnv("RUN_MODE", "auto"))
	// auto: if project/home overrides look container-like, keep custom; else detect.
	if mode == "auto" {
		if firstEnv("WEB_CLAUDE_ROOT", "") != "" ||
			firstEnv("HOME_DIR", "") != "" ||
			firstEnv("CLAUDE_HOME", "") != "" {
			mode = "custom"
		} else if _, err := os.Stat("/.dockerenv"); err == nil {
			mode = "docker"
		} else {
			mode = "native"
		}
	}

	cfg := &Config{
		ListenAddr:      resolveListenAddr(),
		AuthToken:       firstEnv("WEB_CLAUDE_TOKEN", ""),
		ClaudeBin:       firstEnv("CLAUDE_BIN", "claude"),
		RingBufferBytes: envInt("RING_BUFFER_BYTES", 512*1024),
		UploadMaxBytes:  int64(envInt("UPLOAD_MAX_BYTES", 50*1024*1024)),
		CookieSecure:    envBool("COOKIE_SECURE", false),
	}

	switch mode {
	case "docker":
		// Integrated image: isolated home + projects under /data.
		cfg.ProjectsRoot = firstEnv("WEB_CLAUDE_ROOT", "/data/projects")
		cfg.ClaudeHome = firstEnv("CLAUDE_HOME", "HOME_DIR", "/data/home")
		cfg.InheritUserEnv = false
	default: // native / wsl / host / custom / explicit env
		home, _ := os.UserHomeDir()
		// Default project browser root: user home (~/).
		defRoot := home
		if defRoot == "" {
			defRoot = "."
		}
		cfg.ProjectsRoot = firstEnv("WEB_CLAUDE_ROOT", defRoot)
		// Empty ClaudeHome → do not override HOME (use ~/.claude as-is).
		cfg.ClaudeHome = firstEnv("CLAUDE_HOME", "HOME_DIR", "")
		cfg.InheritUserEnv = cfg.ClaudeHome == ""
	}

	// Optional explicit config dir (advanced).
	cfg.ClaudeConfigDir = firstEnv("CLAUDE_CONFIG_DIR", "")

	if v := strings.TrimSpace(os.Getenv("CLAUDE_ARGS")); v != "" {
		cfg.ClaudeArgs = strings.Fields(v)
	}

	idleHours := envInt("IDLE_TTL_HOURS", 24)
	if idleHours > 0 {
		cfg.IdleTTL = time.Duration(idleHours) * time.Hour
	}

	cfg.ClaudeEnv = map[string]string{}
	for _, key := range []string{
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"CLAUDE_CODE_USE_BEDROCK",
		"AWS_REGION",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
	} {
		if v := os.Getenv(key); v != "" {
			cfg.ClaudeEnv[key] = v
		}
	}

	if cfg.AuthToken == "" {
		return nil, fmt.Errorf("WEB_CLAUDE_TOKEN is required (web login password)")
	}
	if cfg.ProjectsRoot == "" {
		return nil, fmt.Errorf("WEB_CLAUDE_ROOT is required")
	}

	// Resolve relative paths against cwd for native use.
	if p, err := filepath.Abs(cfg.ProjectsRoot); err == nil {
		cfg.ProjectsRoot = p
	}
	if cfg.ClaudeHome != "" {
		if p, err := filepath.Abs(cfg.ClaudeHome); err == nil {
			cfg.ClaudeHome = p
		}
	}

	return cfg, nil
}

// resolveListenAddr uses WEB_CLAUDE_PORT (default 3080).
// Accepts "3080", ":3080", or a full host:port.
func resolveListenAddr() string {
	p := strings.TrimSpace(os.Getenv("WEB_CLAUDE_PORT"))
	if p == "" {
		return ":3080"
	}
	if strings.Contains(p, ":") {
		return p
	}
	return ":" + p
}

// firstEnv returns the first non-empty env among keys, or def if all empty.
// When keys is empty, returns def. Last argument is always the default string
// if the previous keys look like env names... actually we pass keys then def.
func firstEnv(keysAndDef ...string) string {
	if len(keysAndDef) == 0 {
		return ""
	}
	def := keysAndDef[len(keysAndDef)-1]
	for _, k := range keysAndDef[:len(keysAndDef)-1] {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// loadDotEnv loads KEY=VAL lines if file exists. Existing env wins.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(line[len("export "):])
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
