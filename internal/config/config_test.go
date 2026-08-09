package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWebClaudeEnv(t *testing.T) {
	t.Setenv("WEB_CLAUDE_TOKEN", "tok")
	t.Setenv("WEB_CLAUDE_PORT", "7099")
	t.Setenv("WEB_CLAUDE_ROOT", "/tmp/wc-root")
	t.Setenv("RUN_MODE", "native")
	wd, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != ":7099" {
		t.Fatalf("listen %q", cfg.ListenAddr)
	}
	if cfg.AuthToken != "tok" {
		t.Fatalf("token %q", cfg.AuthToken)
	}
	if filepath.Base(cfg.ProjectsRoot) != "wc-root" {
		t.Fatalf("root %q", cfg.ProjectsRoot)
	}
}

func TestDefaults(t *testing.T) {
	t.Setenv("WEB_CLAUDE_TOKEN", "tok")
	t.Setenv("WEB_CLAUDE_PORT", "")
	t.Setenv("WEB_CLAUDE_ROOT", "")
	t.Setenv("RUN_MODE", "native")
	wd, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	home, _ := os.UserHomeDir()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != ":3080" {
		t.Fatalf("default port want :3080 got %q", cfg.ListenAddr)
	}
	if filepath.Clean(cfg.ProjectsRoot) != filepath.Clean(home) {
		t.Fatalf("want home %q got %q", home, cfg.ProjectsRoot)
	}
}

func TestTokenRequired(t *testing.T) {
	t.Setenv("WEB_CLAUDE_TOKEN", "")
	t.Setenv("RUN_MODE", "native")
	wd, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if _, err := Load(); err == nil {
		t.Fatal("expected error without token")
	}
}
