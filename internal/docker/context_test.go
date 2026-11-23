package docker

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	// .dockerignore is always sent too, as the Docker CLI does, so the builder
	// can keep the Dockerfile out of COPY . . on its side.
	want := []string{".dockerignore", "Dockerfile", "package.json", "server.js"}
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

// The path sent to the backend must be the clean, slash-separated path inside
// the archive, whatever form the user typed.
func TestDockerfileInContext(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name       string
		dockerfile string
		want       string
		wantErr    bool
	}{
		{name: "plain name", dockerfile: "Dockerfile", want: "Dockerfile"},
		{name: "dot slash", dockerfile: "./Dockerfile", want: "Dockerfile"},
		{name: "subfolder", dockerfile: "docker/Dockerfile.prod", want: "docker/Dockerfile.prod"},
		{name: "absolute inside the context", dockerfile: filepath.Join(dir, "build", "Dockerfile"), want: "build/Dockerfile"},
		{name: "outside the context", dockerfile: "../Dockerfile", wantErr: true},
		{name: "absolute outside the context", dockerfile: filepath.Join(filepath.Dir(dir), "Dockerfile"), wantErr: true},
		{name: "the context itself", dockerfile: ".", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DockerfileInContext(dir, tt.dockerfile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DockerfileInContext(%q) error = %v, wantErr %v", tt.dockerfile, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("DockerfileInContext(%q) = %q, want %q", tt.dockerfile, got, tt.want)
			}
			if err != nil && !strings.Contains(err.Error(), "outside the build context") {
				t.Errorf("error = %q, want it to say the Dockerfile is outside the build context", err)
			}
		})
	}
}

// A Dockerfile outside the context fails before anything is uploaded, instead
// of failing on the server with "lstat ... no such file or directory".
func TestPackageContextRejectsDockerfileOutsideContext(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, map[string]string{
		"Dockerfile":    "FROM node:20\n",
		"app/server.js": "console.log('hi')\n",
	})
	if _, err := PackageContext(filepath.Join(root, "app"), "../Dockerfile"); err == nil {
		t.Fatal("PackageContext() error = nil, want an outside-the-context error")
	}
}

// A symlinked Dockerfile is skipped by the walk and rejected by the builder, so
// say so up front.
func TestPackageContextRejectsSymlinkedDockerfile(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"docker/Dockerfile.real": "FROM node:20\n",
	})
	if err := os.Symlink("docker/Dockerfile.real", filepath.Join(dir, "Dockerfile")); err != nil {
		t.Skipf("symlinks not supported here: %v", err)
	}
	_, err := PackageContext(dir, "Dockerfile")
	if err == nil || !strings.Contains(err.Error(), "not a symlink") {
		t.Fatalf("PackageContext() error = %v, want a symlink error", err)
	}
}

// Without a .dockerignore nothing changes: every file is packed once.
func TestPackageContextWithoutDockerignore(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"Dockerfile": "FROM node:20\n",
		"server.js":  "console.log('hi')\n",
	})
	got := packedNames(t, dir, "Dockerfile")
	want := []string{"Dockerfile", "server.js"}
	if !equalNames(got, want) {
		t.Errorf("packed files = %v, want %v", got, want)
	}
}
