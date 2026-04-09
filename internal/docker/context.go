package docker

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PackageContext creates a gzipped tar of the build context directory,
// respecting .dockerignore patterns. The caller is responsible for removing
// the returned temp file when done.
func PackageContext(contextDir string) (string, error) {
	absContext, err := filepath.Abs(contextDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve context dir: %w", err)
	}

	patterns, err := readDockerignore(absContext)
	if err != nil {
		return "", fmt.Errorf("failed to read .dockerignore: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "satusky-context-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	gzw := gzip.NewWriter(tmpFile)
	tw := tar.NewWriter(gzw)

	walkErr := filepath.WalkDir(absContext, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absContext, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		relSlash := filepath.ToSlash(relPath)

		if shouldIgnore(relSlash, patterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Skip directories and symlinks; only tar regular files.
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		hdr := &tar.Header{
			Name:    relSlash,
			Size:    info.Size(),
			Mode:    int64(info.Mode().Perm()),
			ModTime: info.ModTime(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(path) // #nosec G304 G122 -- path derived from filepath.WalkDir on user-supplied context dir
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }() //nolint:errcheck

		_, err = io.Copy(tw, f)
		return err
	})

	closeErr := func() error {
		if e := tw.Close(); e != nil {
			return e
		}
		if e := gzw.Close(); e != nil {
			return e
		}
		return tmpFile.Close()
	}()

	if walkErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck
		if walkErr != nil {
			return "", fmt.Errorf("failed to package context: %w", walkErr)
		}
		return "", fmt.Errorf("failed to finalize context archive: %w", closeErr)
	}

	return tmpPath, nil
}

// readDockerignore reads .dockerignore from contextDir.
// Returns nil (no patterns) when the file doesn't exist.
func readDockerignore(contextDir string) ([]string, error) {
	path := filepath.Join(contextDir, ".dockerignore")
	f, err := os.Open(path) // #nosec G304
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
}

// shouldIgnore returns true when the path should be excluded from the context.
// Patterns are evaluated in order; later patterns override earlier ones.
// A pattern starting with '!' negates the exclusion.
func shouldIgnore(relPath string, patterns []string) bool {
	ignored := false
	for _, pattern := range patterns {
		negate := strings.HasPrefix(pattern, "!")
		p := pattern
		if negate {
			p = pattern[1:]
		}
		if matchIgnorePattern(relPath, p) {
			ignored = !negate
		}
	}
	return ignored
}

// matchIgnorePattern checks whether relPath matches a single .dockerignore pattern.
func matchIgnorePattern(relPath, pattern string) bool {
	// Handle ** by recursively matching path components.
	if strings.Contains(pattern, "**") {
		return matchDoubleGlob(strings.Split(relPath, "/"), strings.Split(pattern, "/"))
	}

	// Direct match (covers "dir/file" and "*.go" style patterns).
	if m, _ := filepath.Match(pattern, relPath); m { //nolint:errcheck
		return true
	}

	// Basename-only match for patterns without a slash (e.g. "*.go", ".DS_Store").
	if !strings.Contains(pattern, "/") {
		if m, _ := filepath.Match(pattern, filepath.Base(relPath)); m { //nolint:errcheck
			return true
		}
	}

	// Directory prefix match: pattern "vendor" should exclude "vendor/pkg/file".
	if strings.HasPrefix(relPath, strings.TrimRight(pattern, "/")+"/") {
		return true
	}

	return false
}

// matchDoubleGlob handles patterns containing ** by recursing over path/pattern components.
func matchDoubleGlob(pathParts, patternParts []string) bool {
	if len(patternParts) == 0 {
		return len(pathParts) == 0
	}
	if patternParts[0] == "**" {
		// ** matches zero or more path segments.
		for i := 0; i <= len(pathParts); i++ {
			if matchDoubleGlob(pathParts[i:], patternParts[1:]) {
				return true
			}
		}
		return false
	}
	if len(pathParts) == 0 {
		return false
	}
	matched, _ := filepath.Match(patternParts[0], pathParts[0]) //nolint:errcheck
	if !matched {
		return false
	}
	return matchDoubleGlob(pathParts[1:], patternParts[1:])
}
