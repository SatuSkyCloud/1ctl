package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRuntime(t *testing.T) {
	tests := []struct {
		name   string
		marker string
		want   string // expected Runtime.Name; "" means unknown
	}{
		{"go module", "go.mod", "Go"},
		{"node package", "package.json", "Node.js / Bun"},
		{"python requirements", "requirements.txt", "Python"},
		{"poetry pyproject", "pyproject.toml", "Python (Poetry)"},
		{"rust cargo", "Cargo.toml", "Rust"},
		{"ruby gemfile", "Gemfile", "Ruby"},
		{"java maven", "pom.xml", "Java (Maven)"},
		{"java gradle", "build.gradle", "Java (Gradle)"},
		{"php composer", "composer.json", "PHP"},
		{"unknown stack", "README.md", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.marker), []byte("x"), 0600); err != nil {
				t.Fatalf("seed marker: %v", err)
			}
			got := detectRuntime(dir)
			if got.Name != tt.want {
				t.Errorf("detectRuntime: name = %q, want %q", got.Name, tt.want)
			}
		})
	}
}

func TestDetectRuntime_FirstMarkerWins(t *testing.T) {
	// A monorepo with both go.mod and package.json should report Go because
	// it comes first in the probe order. (Stability matters — if the order
	// changes, the wizard's defaults change for thousands of users.)
	dir := t.TempDir()
	for _, name := range []string{"go.mod", "package.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0600); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if got := detectRuntime(dir); got.Name != "Go" {
		t.Errorf("first-marker-wins: got %q, want Go", got.Name)
	}
}

func TestHasDockerfile(t *testing.T) {
	dir := t.TempDir()
	if hasDockerfile(dir) {
		t.Error("empty dir reports Dockerfile present")
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if !hasDockerfile(dir) {
		t.Error("Dockerfile present but not detected")
	}
}
