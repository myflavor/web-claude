package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebClaudeEnvPreferred(t *testing.T) {
	t.Setenv("WEB_CLAUDE_TOKEN", "tok")
	t.Setenv("WEB_CLAUDE_PORT", "7099")
	t.Setenv("WEB_CLAUDE_ROOT", "/tmp/wc-root")
	// clear legacy
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("PROJECTS_ROOT", "")
	t.Setenv("RUN_MODE", "native")
	// ensure empty legacy not set via dotenv interference: chdir to temp without .env
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
	if !strings.HasSuffix(filepath.Clean(cfg.ProjectsRoot), "wc-root") && cfg.ProjectsRoot != "/tmp/wc-root" {
		// abs path
		if filepath.Base(cfg.ProjectsRoot) != "wc-root" {
			t.Fatalf("root %q", cfg.ProjectsRoot)
		}
	}
}

func TestDefaultRootIsHome(t *testing.T) {
	t.Setenv("WEB_CLAUDE_TOKEN", "tok")
	t.Setenv("WEB_CLAUDE_PORT", "8080")
	t.Setenv("WEB_CLAUDE_ROOT", "")
	t.Setenv("PROJECTS_ROOT", "")
	t.Setenv("AUTH_TOKEN", "")
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
	if cfg.ProjectsRoot != home && filepath.Clean(cfg.ProjectsRoot) != filepath.Clean(home) {
		t.Fatalf("want home %q got %q", home, cfg.ProjectsRoot)
	}
}

func TestLegacyAliases(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "legacy")
	t.Setenv("LISTEN_ADDR", ":6000")
	t.Setenv("PROJECTS_ROOT", "/tmp/legacy-root")
	t.Setenv("WEB_CLAUDE_TOKEN", "")
	t.Setenv("WEB_CLAUDE_PORT", "")
	t.Setenv("WEB_CLAUDE_ROOT", "")
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
	if cfg.AuthToken != "legacy" || cfg.ListenAddr != ":6000" {
		t.Fatalf("legacy fail: %+v", cfg)
	}
}
