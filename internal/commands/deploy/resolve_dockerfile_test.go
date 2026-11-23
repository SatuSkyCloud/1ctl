package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir switches into dir for the test; FindDockerfile searches the working directory.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) }) //nolint:errcheck
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// An existing but invalid Dockerfile reports its own problems. It must not be
// swapped for another Dockerfile, and must not claim that no Dockerfile exists.
func TestResolveDockerfileReportsProblemsInRequestedFile(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeTestFile(t, filepath.Join(dir, "Dockerfile.prod"), "FROM alpine:3.20\nBOGUS x\n")
	writeTestFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine:3.20\n")

	_, err := resolveDockerfile("Dockerfile.prod")
	if err == nil {
		t.Fatal("resolveDockerfile() error = nil, want the validation error for Dockerfile.prod")
	}
	msg := err.Error()
	if !strings.Contains(msg, "line 2: invalid instruction 'BOGUS'") {
		t.Errorf("error = %q, want the line-numbered problem", msg)
	}
	if strings.Contains(msg, "please ensure a Dockerfile exists") || strings.Contains(msg, "no Dockerfile found") {
		t.Errorf("error = %q, must not say the Dockerfile is missing", msg)
	}
}

// A missing Dockerfile still falls back to the common locations.
func TestResolveDockerfileFallsBackWhenRequestedFileIsMissing(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeTestFile(t, filepath.Join(dir, "docker", "Dockerfile"), "FROM alpine:3.20\n")

	got, err := resolveDockerfile("Dockerfile")
	if err != nil {
		t.Fatalf("resolveDockerfile() error = %v", err)
	}
	if filepath.ToSlash(got) != "docker/Dockerfile" {
		t.Errorf("resolveDockerfile() = %q, want docker/Dockerfile", got)
	}
}
