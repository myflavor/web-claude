package main

import (
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/xin/claude-mobile/internal/api"
	"github.com/xin/claude-mobile/internal/auth"
	"github.com/xin/claude-mobile/internal/config"
	"github.com/xin/claude-mobile/internal/session"
	"github.com/xin/claude-mobile/web"
)

func main() {
	logger := log.New(os.Stdout, "[claude-mobile] ", log.LstdFlags|log.Lmsgprefix)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("config: %v", err)
	}

	if err := os.MkdirAll(cfg.ProjectsRoot, 0o755); err != nil {
		logger.Fatalf("projects root: %v", err)
	}
	if cfg.ClaudeHome != "" {
		if err := os.MkdirAll(cfg.ClaudeHome, 0o700); err != nil {
			logger.Fatalf("claude home: %v", err)
		}
		_ = os.MkdirAll(filepath.Join(cfg.ClaudeHome, ".claude"), 0o700)
	}

	mode := "native"
	if !cfg.InheritUserEnv {
		mode = "isolated-home"
	}
	logger.Printf("mode=%s projects=%s claudeHome=%q inheritUserEnv=%v",
		mode, cfg.ProjectsRoot, cfg.ClaudeHome, cfg.InheritUserEnv)

	guard := auth.New(cfg.AuthToken, cfg.CookieSecure)
	mgr := session.NewManager(session.ManagerConfig{
		ClaudeBin:      cfg.ClaudeBin,
		ClaudeArgs:     cfg.ClaudeArgs,
		ClaudeEnv:      cfg.ClaudeEnv,
		ClaudeHome:     cfg.ClaudeHome,
		ConfigDir:      cfg.ClaudeConfigDir,
		InheritUserEnv: cfg.InheritUserEnv,
		RingSize:       cfg.RingBufferBytes,
		IdleTTL:        cfg.IdleTTL,
		Logger:         logger,
	})

	staticRoot, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		logger.Fatalf("static fs: %v", err)
	}
	static := spaFileServer(staticRoot)

	srv := api.New(cfg, guard, mgr, logger)
	handler := srv.Handler(static)

	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: handler,
	}

	go func() {
		logger.Printf("listening on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen: %v", err)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	logger.Printf("signal %v, shutting down", sig)
	_ = httpServer.Close()
}

// spaFileServer serves embedded SPA assets; unknown paths fall back to index.html
// so Vue Router history mode works.
func spaFileServer(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}
		// strip leading slash for fs.FS
		name := strings.TrimPrefix(path, "/")
		if f, err := root.Open(name); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// fallback to index.html for client routes like /login
		r2 := r.Clone(r.Context())
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
