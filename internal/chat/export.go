package chat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultTranscriptName builds the default /export filename:
// chat-transcript-YYYYMMDD-HHMMSS.md.
func defaultTranscriptName(now time.Time) string {
	return "chat-transcript-" + now.Format("20060102-150405") + ".md"
}

// renderTranscript renders the conversation as markdown: an H1 title with
// the export timestamp, a provider/model line, then each message under a
// `## user` / `## assistant` heading with the raw content (nothing is
// escaped or fenced — transcripts stay readable in any markdown viewer).
func renderTranscript(session *Session, provider Provider, model string, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 1ctl chat — %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "provider: %s · model: %s\n\n", provider, model)
	for _, m := range session.Messages {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", m.Role, m.Content)
	}
	return b.String()
}

// resolveWithinCwd resolves arg relative to cwd, rejecting absolute paths
// and paths that escape cwd (.., ../.., …). label names the operation in
// error messages. Mirrors the traversal guard style used elsewhere in the
// repo.
func resolveWithinCwd(cwd, arg, label string) (string, error) {
	if filepath.IsAbs(arg) {
		return "", fmt.Errorf("%s path must be relative to the current directory (got absolute path %q)", label, arg)
	}
	clean := filepath.Clean(arg)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s path %q escapes the current directory", label, arg)
	}
	return filepath.Join(cwd, clean), nil
}

// resolveExportPath resolves a /export target path relative to cwd. An
// empty arg yields a timestamped default filename in cwd.
func resolveExportPath(cwd, arg string) (string, error) {
	if strings.TrimSpace(arg) == "" {
		return filepath.Join(cwd, defaultTranscriptName(time.Now())), nil
	}
	return resolveWithinCwd(cwd, arg, "export")
}

// exportTranscript writes the session transcript to the resolved path
// (arg, or a timestamped default in cwd). An empty conversation is a
// no-op that returns "" without creating a file. It returns the path
// written on success.
func exportTranscript(session *Session, provider Provider, model, cwd, arg string) (string, error) {
	if len(session.Messages) == 0 {
		return "", nil
	}
	path, err := resolveExportPath(cwd, arg)
	if err != nil {
		return "", err
	}
	content := renderTranscript(session, provider, model, time.Now())
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", fmt.Errorf("create export directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("write transcript: %w", err)
	}
	return path, nil
}
