package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	content, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if !strings.Contains(content, "1ctl Chat Agent Skill") {
		t.Error("default skill missing identity heading")
	}
	if Name() != embeddedName {
		t.Errorf("Name() = %q, want %q", Name(), embeddedName)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.md")
	if err := os.WriteFile(path, []byte("# Custom skill\ncustom rules\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_CHAT_SKILL", path)

	content, err := Load()
	if err != nil {
		t.Fatalf("Load() with env override: %v", err)
	}
	if content != "# Custom skill\ncustom rules\n" {
		t.Errorf("Load() = %q, want custom content", content)
	}
	if Name() != path {
		t.Errorf("Name() = %q, want %q", Name(), path)
	}
}

func TestLoadEnvMissingFile(t *testing.T) {
	t.Setenv("SATUSKY_CHAT_SKILL", filepath.Join(t.TempDir(), "does-not-exist.md"))
	if _, err := Load(); err == nil {
		t.Fatal("Load() with missing override file: expected error, got nil")
	}
}

func TestLoadPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alt.md")
	if err := os.WriteFile(path, []byte("alt skill body\n"), 0644); err != nil {
		t.Fatal(err)
	}
	content, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	if content != "alt skill body\n" {
		t.Errorf("LoadPath content = %q", content)
	}
	if Name() != path {
		t.Errorf("Name() = %q, want %q", Name(), path)
	}
}

func TestLoadPathMissing(t *testing.T) {
	if _, err := LoadPath(filepath.Join(t.TempDir(), "nope.md")); err == nil {
		t.Fatal("LoadPath of a missing file: expected error, got nil")
	}
}
