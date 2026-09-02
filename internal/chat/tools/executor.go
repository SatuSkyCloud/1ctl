package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultShellTimeout bounds every run_shell invocation unless the
// executor is configured otherwise (tests shrink it).
const defaultShellTimeout = 60 * time.Second

// Executor runs workspace tools inside the chat working directory. Cwd is
// the sandbox root: every tool path resolves relative to it, and absolute
// paths, ".." traversal, and symlink escapes are rejected. Confirm gates
// mutating actions (run_shell always, write_file only when overwriting an
// existing file); a nil Confirm declines. Timeout bounds run_shell.
// Custom dispatches tool calls by name before the built-in tools (the chat
// REPL registers the SatuSky tools there, keeping this package free of
// SatuSky dependencies).
type Executor struct {
	Cwd     string
	Confirm func(action string) bool
	Timeout time.Duration
	Custom  map[string]func(argsJSON []byte) string
}

// NewExecutor builds an executor sandboxed to cwd with the default 60s
// shell timeout and the given confirmation gate.
func NewExecutor(cwd string, confirm func(action string) bool) *Executor {
	return &Executor{Cwd: cwd, Confirm: confirm, Timeout: defaultShellTimeout}
}

// Execute dispatches one tool call by name and returns a string result —
// the tool's output on success, or an "error: ..." / "refused: ..." /
// "cancelled by user" message on failure. It always returns a string and
// never panics: an unexpected internal failure is captured and returned
// as an error message.
func (e *Executor) Execute(name string, argsJSON []byte) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprintf("error: internal failure executing %s: %v", name, r)
		}
	}()
	if e.Cwd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "error: no working directory configured"
		}
		e.Cwd = cwd
	}
	if fn, ok := e.Custom[name]; ok && fn != nil {
		return fn(argsJSON)
	}
	switch name {
	case "read_file":
		var args struct {
			Path   string `json:"path"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return fmt.Sprintf("error: invalid arguments for read_file: %v", err)
		}
		return e.readFile(args.Path, args.Offset, args.Limit)
	case "write_file":
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return fmt.Sprintf("error: invalid arguments for write_file: %v", err)
		}
		return e.writeFile(args.Path, args.Content)
	case "list_dir":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return fmt.Sprintf("error: invalid arguments for list_dir: %v", err)
		}
		return e.listDir(args.Path)
	case "run_shell":
		var args struct {
			Command string `json:"command"`
			Cwd     string `json:"cwd"`
		}
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return fmt.Sprintf("error: invalid arguments for run_shell: %v", err)
		}
		return e.runShell(args.Command, args.Cwd)
	case "analyze_project":
		var args struct {
			Path string `json:"path"`
		}
		if len(argsJSON) > 0 {
			if err := json.Unmarshal(argsJSON, &args); err != nil {
				return fmt.Sprintf("error: invalid arguments for analyze_project: %v", err)
			}
		}
		return e.analyzeProject(args.Path)
	default:
		return fmt.Sprintf("error: unknown tool %q", name)
	}
}

// confirm runs the injected confirmation gate; a nil gate declines.
func (e *Executor) confirm(action string) bool {
	if e.Confirm == nil {
		return false
	}
	return e.Confirm(action)
}

// resolvePath resolves p against base, rejecting absolute paths and any
// ".." traversal that would escape base. The result is a clean path under
// base (mirroring the traversal guard used for /export in the chat
// package).
func resolvePath(base, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("path %q must be relative to the working directory", p)
	}
	clean := filepath.Clean(p)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the working directory", p)
	}
	return filepath.Join(base, clean), nil
}

// resolveUnder resolves p under base and refuses symlink escapes: the
// deepest existing ancestor of the target is resolved with EvalSymlinks
// and must stay inside base. Used by every file tool.
func resolveUnder(base, p string) (string, error) {
	full, err := resolvePath(base, p)
	if err != nil {
		return "", err
	}
	dir := full
	for {
		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			return checkWithin(base, resolved, full)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if _, err := os.Stat(base); err != nil {
		return "", fmt.Errorf("working directory unavailable: %w", err)
	}
	return full, nil
}

// checkWithin verifies that resolved stays inside base (resolving base
// itself first so both sides compare on real paths), returning the
// original lexical path on success.
func checkWithin(base, resolved, full string) (string, error) {
	baseResolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		baseResolved = base
	}
	rel, err := filepath.Rel(baseResolved, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the working directory via a symlink", filepath.Base(full))
	}
	return full, nil
}
