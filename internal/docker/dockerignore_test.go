package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeIgnorePattern(t *testing.T) {
	tests := map[string]string{
		"/node_modules":   "node_modules",
		"/.git":           ".git",
		"./dist":          "dist",
		"dist/":           "dist",
		"/build/":         "build",
		"**/node_modules": "**/node_modules",
		"!/keep.txt":      "!keep.txt",
		"! important.md":  "!important.md",
		"*.log":           "*.log",
		"a//b":            "a/b",
		"/":               "/",
		"!":               "",
	}
	for in, want := range tests {
		if got := normalizeIgnorePattern(in); got != want {
			t.Errorf("normalizeIgnorePattern(%q) = %q, want %q", in, got, want)
		}
	}
}

// Issue #132: the .dockerignore written by fly launch's Node generator uses
// root-anchored patterns. They matched nothing, so node_modules and .git were
// uploaded and the build upload was refused with HTTP 413.
//
// The test goes through readDockerignore and shouldIgnore, the two pieces the
// packer uses for every path, so it does not depend on PackageContext's signature.
func TestRootAnchoredPatternsExcludePaths(t *testing.T) {
	patterns := patternsFrom(t, "/.git\n/node_modules\n.dockerignore\n.env\nDockerfile\nfly.toml\n")

	for _, path := range []string{"node_modules", "node_modules/next/package.json", ".git", ".git/HEAD", ".env", "fly.toml"} {
		if !shouldIgnore(path, patterns) {
			t.Errorf("shouldIgnore(%q) = false, want true with patterns %v", path, patterns)
		}
	}
	for _, path := range []string{"package.json", "app/page.tsx", "src/node_modules_helper.ts"} {
		if shouldIgnore(path, patterns) {
			t.Errorf("shouldIgnore(%q) = true, want false with patterns %v", path, patterns)
		}
	}
}

// A root-anchored re-include must still re-include.
func TestRootAnchoredNegation(t *testing.T) {
	patterns := patternsFrom(t, "*.md\n!/README.md\n")
	if shouldIgnore("README.md", patterns) {
		t.Error("README.md is ignored, want it re-included by !/README.md")
	}
	if !shouldIgnore("NOTES.md", patterns) {
		t.Error("NOTES.md is kept, want it ignored by *.md")
	}
}

func patternsFrom(t *testing.T, dockerignore string) []string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte(dockerignore), 0o600); err != nil {
		t.Fatal(err)
	}
	patterns, err := readDockerignore(dir)
	if err != nil {
		t.Fatalf("readDockerignore() error = %v", err)
	}
	return patterns
}
