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

	"1ctl/internal/utils"
)

// PackageContext creates a gzipped tar of the build context directory,
// respecting .dockerignore patterns. The Dockerfile named by dockerfile is
// always included, even when .dockerignore lists it, because the cloud builder
// needs it to start the build (Docker itself also always sends it). The caller
// is responsible for removing the returned temp file when done.
func PackageContext(contextDir, dockerfile string) (string, error) {
	absContext, err := filepath.Abs(contextDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve context dir: %w", err)
	}

	patterns, err := readDockerignore(absContext)
	if err != nil {
		return "", fmt.Errorf("failed to read .dockerignore: %w", err)
	}

	// The Dockerfile's path inside the archive, in the same slash form as the
	// walk below. The cloud builder looks for it at exactly this path.
	dockerfileRel := ""
	if dockerfile != "" {
		dockerfileRel, err = DockerfileInContext(absContext, dockerfile)
		if err != nil {
			return "", err
		}
	}
	dockerfilePacked := false

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

		// Keep the Dockerfile and .dockerignore even if .dockerignore matches
		// them, as the Docker CLI does. The builder re-applies .dockerignore, so
		// neither file ends up in the image through COPY . . unless asked for.
		// Keep walking into an ignored folder only when it holds the Dockerfile.
		alwaysSent := relSlash == dockerfileRel || relSlash == ".dockerignore"
		if !alwaysSent && shouldIgnore(relSlash, patterns) {
			if d.IsDir() && !strings.HasPrefix(dockerfileRel, relSlash+"/") {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Skip directories silently.
		if info.IsDir() {
			return nil
		}
		// Symlinks aren't followed in the build context tar today. Warn
		// loudly: monorepos using pnpm/yarn/Go workspaces will produce
		// confusing "module not found" errors at build time otherwise.
		if info.Mode()&os.ModeSymlink != 0 {
			utils.PrintWarning("Symlink %s skipped — symlinks are not supported in the build context.", relSlash)
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

		if _, err = io.Copy(tw, f); err != nil {
			return err
		}
		if relSlash == dockerfileRel {
			dockerfilePacked = true
		}
		return nil
	})
	if walkErr == nil && dockerfileRel != "" && !dockerfilePacked {
		// Without this the build fails later on the server with a bare
		// "lstat .../Dockerfile: no such file or directory".
		walkErr = fmt.Errorf("dockerfile %s was not added to the build context; it must be a regular file, not a symlink", dockerfileRel)
	}

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

// DockerfileInContext returns the Dockerfile's path relative to contextDir, in
// slash form, which is the path the cloud builder reads inside the uploaded
// archive. dockerfile may be absolute or relative to contextDir. It is an error
// for the Dockerfile to sit outside the context: the builder only receives the
// archive, so it could never read it.
func DockerfileInContext(contextDir, dockerfile string) (string, error) {
	absContext, err := filepath.Abs(contextDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve context dir: %w", err)
	}
	absDockerfile := dockerfile
	if !filepath.IsAbs(absDockerfile) {
		absDockerfile = filepath.Join(absContext, dockerfile)
	}
	rel, err := filepath.Rel(absContext, absDockerfile)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("dockerfile %s is outside the build context %s; move it into the project folder or run 1ctl deploy from a folder that contains it", dockerfile, absContext)
	}
	return filepath.ToSlash(rel), nil
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
		if pattern := normalizeIgnorePattern(line); pattern != "" {
			patterns = append(patterns, pattern)
		}
	}
	return patterns, scanner.Err()
}

// normalizeIgnorePattern cleans a .dockerignore line the way the Docker CLI
// does before matching: the path is cleaned (so "./dist" and "dist/" become
// "dist") and a leading "/" is dropped, because patterns are always relative
// to the context root. Without this, "/node_modules" and "/.git" — the form
// that generators such as fly launch write — matched nothing, and the whole
// node_modules tree was uploaded. A leading "!" (re-include) is kept.
func normalizeIgnorePattern(line string) string {
	negate := strings.HasPrefix(line, "!")
	pattern := line
	if negate {
		pattern = strings.TrimSpace(pattern[1:])
	}
	if pattern == "" {
		return ""
	}
	pattern = filepath.ToSlash(filepath.Clean(filepath.FromSlash(pattern)))
	if len(pattern) > 1 && pattern[0] == '/' {
		pattern = pattern[1:]
	}
	if negate {
		return "!" + pattern
	}
	return pattern
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
