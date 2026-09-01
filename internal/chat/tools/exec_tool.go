package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// maxReadChars caps read_file output so a single tool result never floods
// the model context; larger files are truncated with a note.
const maxReadChars = 100_000

// maxExecOutput caps each captured stream (stdout, stderr) from run_shell.
const maxExecOutput = 32 * 1024

// blockedPatterns are shell fragments refused even with user confirmation:
// they are destructive beyond any reasonable chat use.
var blockedPatterns = []string{
	"rm -rf ~",
	":(){",
	"mkfs",
	"dd if=/dev/zero of=/dev/",
	"> /dev/sd",
	"shutdown",
	"reboot",
	"sudo rm -rf",
}

// isBlocked reports whether a shell command matches a destructive pattern.
// "rm -rf /" is checked precisely (root, another slash, or whitespace
// after it) so that e.g. "rm -rf /tmp/build" is not falsely caught by the
// root rule — though it still needs confirmation like any run_shell.
func isBlocked(cmd string) bool {
	norm := strings.ToLower(strings.Join(strings.Fields(cmd), " "))
	if i := strings.Index(norm, "rm -rf /"); i >= 0 {
		rest := norm[i+len("rm -rf /"):]
		if rest == "" || rest[0] == ' ' || rest[0] == '/' {
			return true
		}
	}
	for _, pat := range blockedPatterns {
		if strings.Contains(norm, pat) {
			return true
		}
	}
	return false
}

// readFile returns the requested line range of a file (1-based offset,
// optional line limit; zero means "unset"), capped at maxReadChars.
func (e *Executor) readFile(path string, offset, limit int) string {
	full, err := resolveUnder(e.Cwd, path)
	if err != nil {
		return "error: " + err.Error()
	}
	data, err := os.ReadFile(full) // #nosec G304 -- path resolved via resolveUnder (traversal/symlink guard)
	if err != nil {
		return "error: read " + path + ": " + err.Error()
	}
	lines := strings.Split(string(data), "\n")
	start := 0
	if offset > 1 {
		start = offset - 1
	}
	if start > len(lines) {
		start = len(lines)
	}
	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	content := strings.Join(lines[start:end], "\n")
	if len(content) > maxReadChars {
		content = content[:maxReadChars] + fmt.Sprintf("\n... [truncated at %d characters]", maxReadChars)
	}
	return content
}

// writeFile writes content to path, creating parent directories. The write
// is atomic (temp file + rename). An existing file is only overwritten
// after user confirmation.
func (e *Executor) writeFile(path, content string) string {
	full, err := resolveUnder(e.Cwd, path)
	if err != nil {
		return "error: " + err.Error()
	}
	if _, err := os.Lstat(full); err == nil {
		if !e.confirm("overwrite " + path + "?") {
			return "cancelled by user: overwrite of " + path + " not confirmed"
		}
	}
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil { // #nosec G301 -- project directories, standard perms
		return "error: create parent directories for " + path + ": " + err.Error()
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), ".1ctl-chat-write-*")
	if err != nil {
		return "error: create temp file: " + err.Error()
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck // best-effort cleanup of the temp file
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close() //nolint:errcheck // the write failed; nothing to recover
		return "error: write " + path + ": " + err.Error()
	}
	if err := tmp.Close(); err != nil {
		return "error: write " + path + ": " + err.Error()
	}
	if err := os.Chmod(tmpName, 0644); err != nil { // #nosec G302 -- scaffolded project files, not secrets
		return "error: write " + path + ": " + err.Error()
	}
	if err := os.Rename(tmpName, full); err != nil {
		return "error: write " + path + ": " + err.Error()
	}
	return fmt.Sprintf("wrote %s (%d bytes)", path, len(content))
}

// listDir lists a directory: one line per entry "name kind size", sorted
// by name. Names are relative to the listed directory so the model can
// feed them straight back into read_file/write_file.
func (e *Executor) listDir(path string) string {
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	full, err := resolveUnder(e.Cwd, path)
	if err != nil {
		return "error: " + err.Error()
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return "error: list " + path + ": " + err.Error()
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var b strings.Builder
	for _, entry := range entries {
		kind := "file"
		size := "-"
		if entry.IsDir() {
			kind = "dir"
		} else if info, err := entry.Info(); err == nil {
			size = humanSize(info.Size())
		}
		name := entry.Name()
		if path != "." {
			name = filepath.Join(path, name)
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\n", name, kind, size)
	}
	return strings.TrimRight(b.String(), "\n")
}

// humanSize renders a byte count with binary units (1536 → 1.5 KiB).
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// runShell executes a shell command (sh -c) under Cwd (or the resolved
// cwd argument) with a timeout, capturing stdout and stderr (each capped
// at maxExecOutput). run_shell always requires confirmation and refuses
// blocked patterns outright.
func (e *Executor) runShell(command, cwd string) string {
	if strings.TrimSpace(command) == "" {
		return "error: run_shell requires a command"
	}
	if isBlocked(command) {
		return "refused: command matches a destructive pattern — not executed"
	}
	if !e.confirm("run shell command: " + strings.TrimSpace(command)) {
		return "cancelled by user: shell command not run"
	}
	dir := e.Cwd
	if strings.TrimSpace(cwd) != "" {
		full, err := resolveUnder(e.Cwd, cwd)
		if err != nil {
			return "error: " + err.Error()
		}
		dir = full
	}
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = defaultShellTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command) // #nosec G204 -- run_shell is the feature; user-confirmed, timeout-bounded, destructive patterns blocked
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return "error: command timed out after " + timeout.String()
	}
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if ok := errors.As(err, &exitErr); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "error: run command: " + err.Error()
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "exit code %d", exitCode)
	if out := capOutput(stdout.String()); out != "" {
		b.WriteString("\n--- stdout ---\n" + out)
	}
	if errOut := capOutput(stderr.String()); errOut != "" {
		b.WriteString("\n--- stderr ---\n" + errOut)
	}
	return b.String()
}

// capOutput trims whitespace and truncates a captured stream to
// maxExecOutput characters.
func capOutput(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxExecOutput {
		s = s[:maxExecOutput] + "\n... [output truncated]"
	}
	return s
}
