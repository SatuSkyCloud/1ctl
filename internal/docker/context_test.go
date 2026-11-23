package docker

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// packedNames packages contextDir and returns the sorted file names in the archive.
func packedNames(t *testing.T, contextDir, dockerfile string) []string {
	t.Helper()
	archivePath, err := PackageContext(contextDir, dockerfile)
	if err != nil {
		t.Fatalf("PackageContext() error = %v", err)
	}
	defer func() { _ = os.Remove(archivePath) }() //nolint:errcheck

	f, err := os.Open(archivePath) // #nosec G304 -- temp file created by PackageContext
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	sort.Strings(names)
	return names
}

// writeProject creates files (relative path -> content) under dir.
func writeProject(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A Dockerfile listed in .dockerignore must still be sent to the cloud builder,
// while the other ignored files stay out of the archive.
func TestPackageContextKeepsDockerfileListedInDockerignore(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"Dockerfile":    "FROM node:20\n",
		".dockerignore": "Dockerfile\n.dockerignore\n.env\n",
		"server.js":     "console.log('hi')\n",
		".env":          "SECRET=1\n",
		"package.json":  "{}\n",
	})

	got := packedNames(t, dir, "Dockerfile")
	want := []string{"Dockerfile", "package.json", "server.js"}
	if !equalNames(got, want) {
		t.Errorf("packed files = %v, want %v", got, want)
	}
}

// The same rule applies when the Dockerfile sits in an ignored folder.
func TestPackageContextKeepsDockerfileInsideIgnoredFolder(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		".dockerignore":     "docker\n",
		"docker/Dockerfile": "FROM node:20\n",
		"docker/notes.txt":  "ignored\n",
		"server.js":         "console.log('hi')\n",
	})

	got := packedNames(t, dir, "./docker/Dockerfile")
	want := []string{".dockerignore", "docker/Dockerfile", "server.js"}
	if !equalNames(got, want) {
		t.Errorf("packed files = %v, want %v", got, want)
	}
}
