package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestExecutor builds an executor sandboxed to a fresh temp dir with
// the given confirmation gate and a tiny shell timeout (tests never wait
// on the real 60s).
func newTestExecutor(t *testing.T, confirm func(string) bool) *Executor {
	t.Helper()
	e := NewExecutor(t.TempDir(), confirm)
	e.Timeout = 2 * time.Second
	return e
}

func TestDefinitionsExposeAllTools(t *testing.T) {
	defs := Definitions()
	if len(defs) != 6 {
		t.Fatalf("Definitions() = %d tools, want 6 (workspace + satusky)", len(defs))
	}
	names := map[string]bool{}
	for _, d := range defs {
		if d.Type != "function" {
			t.Errorf("tool %q type = %q, want function", d.Function.Name, d.Type)
		}
		if d.Function == nil || d.Function.Name == "" {
			t.Fatalf("tool with nil/empty function definition: %+v", d)
		}
		names[d.Function.Name] = true
	}
	for _, want := range []string{"read_file", "write_file", "list_dir", "run_shell", "satusky_status", "satusky_run"} {
		if !names[want] {
			t.Errorf("Definitions() missing tool %q", want)
		}
	}
}

// writeArgs builds the JSON arguments for a write_file call so content
// (with newlines/quotes) is properly JSON-encoded.
func writeArgs(path, content string) string {
	b, _ := json.Marshal(map[string]string{"path": path, "content": content})
	return string(b)
}

func TestWriteReadRoundtrip(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	res := e.Execute("write_file", []byte(writeArgs("sub/hello.txt", "hi there")))
	if !strings.Contains(res, "wrote sub/hello.txt") {
		t.Fatalf("write_file result = %q", res)
	}
	data, err := os.ReadFile(filepath.Join(e.Cwd, "sub", "hello.txt"))
	if err != nil {
		t.Fatalf("read back file: %v", err)
	}
	if string(data) != "hi there" {
		t.Errorf("file content = %q, want %q", data, "hi there")
	}

	res = e.Execute("read_file", []byte(`{"path":"sub/hello.txt"}`))
	if res != "hi there" {
		t.Errorf("read_file result = %q, want %q", res, "hi there")
	}
}

func TestReadFileOffsetLimit(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	content := "line1\nline2\nline3\nline4\nline5\n"
	if res := e.Execute("write_file", []byte(writeArgs("f.txt", content))); strings.HasPrefix(res, "error") {
		t.Fatalf("write_file: %s", res)
	}
	res := e.Execute("read_file", []byte(`{"path":"f.txt","offset":2,"limit":2}`))
	if res != "line2\nline3" {
		t.Errorf("read_file offset/limit = %q, want %q", res, "line2\nline3")
	}
}

func TestReadFileTruncationNote(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	big := strings.Repeat("a", maxReadChars+5000)
	if res := e.Execute("write_file", []byte(`{"path":"big.txt","content":"`+big+`"}`)); strings.HasPrefix(res, "error") {
		t.Fatalf("write_file: %s", res)
	}
	res := e.Execute("read_file", []byte(`{"path":"big.txt"}`))
	if len(res) > maxReadChars+200 {
		t.Errorf("read_file result too large: %d chars", len(res))
	}
	if !strings.Contains(res, "truncated at") {
		t.Error("read_file result missing truncation note")
	}
}

func TestListDirEntries(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	if res := e.Execute("write_file", []byte(`{"path":"a.txt","content":"hello"}`)); strings.HasPrefix(res, "error") {
		t.Fatalf("write_file a.txt: %s", res)
	}
	if res := e.Execute("write_file", []byte(`{"path":"sub/b.txt","content":"x"}`)); strings.HasPrefix(res, "error") {
		t.Fatalf("write_file sub/b.txt: %s", res)
	}
	res := e.Execute("list_dir", []byte(`{"path":"."}`))
	if !strings.Contains(res, "a.txt\tfile\t5 B") {
		t.Errorf("list_dir missing a.txt entry: %q", res)
	}
	if !strings.Contains(res, "sub\tdir\t-") {
		t.Errorf("list_dir missing sub dir entry: %q", res)
	}
}

func TestWriteFileOverwriteRequiresConfirm(t *testing.T) {
	confirmed := false
	e := newTestExecutor(t, func(string) bool { return confirmed })
	if res := e.Execute("write_file", []byte(`{"path":"f.txt","content":"first"}`)); strings.HasPrefix(res, "error") {
		t.Fatalf("initial write: %s", res)
	}

	// Declined: the file must stay untouched.
	res := e.Execute("write_file", []byte(`{"path":"f.txt","content":"second"}`))
	if !strings.Contains(res, "cancelled by user") {
		t.Errorf("declined overwrite result = %q, want cancellation", res)
	}
	data, _ := os.ReadFile(filepath.Join(e.Cwd, "f.txt"))
	if string(data) != "first" {
		t.Errorf("file after declined overwrite = %q, want %q", data, "first")
	}

	// Confirmed: the file is overwritten.
	confirmed = true
	res = e.Execute("write_file", []byte(`{"path":"f.txt","content":"second"}`))
	if strings.HasPrefix(res, "error") || strings.Contains(res, "cancelled") {
		t.Fatalf("confirmed overwrite result = %q", res)
	}
	data, _ = os.ReadFile(filepath.Join(e.Cwd, "f.txt"))
	if string(data) != "second" {
		t.Errorf("file after confirmed overwrite = %q, want %q", data, "second")
	}
}

func TestTraversalRejected(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	tests := []struct {
		name string
		args string
	}{
		{name: "parent traversal", args: `{"path":"../escape.txt","content":"x"}`},
		{name: "deep traversal", args: `{"path":"a/../../escape.txt","content":"x"}`},
		{name: "bare dotdot", args: `{"path":"..","content":"x"}`},
		{name: "absolute path", args: `{"path":"/etc/escape.txt","content":"x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := e.Execute("write_file", []byte(tt.args))
			if !strings.Contains(res, "error") {
				t.Errorf("write_file %s = %q, want error", tt.name, res)
			}
		})
	}
	if res := e.Execute("read_file", []byte(`{"path":"../escape.txt"}`)); !strings.Contains(res, "error") {
		t.Errorf("read_file traversal = %q, want error", res)
	}
	if res := e.Execute("list_dir", []byte(`{"path":"../"}`)); !strings.Contains(res, "error") {
		t.Errorf("list_dir traversal = %q, want error", res)
	}
}

func TestSymlinkEscapeRefused(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	outside := t.TempDir()
	link := filepath.Join(e.Cwd, "outside")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res := e.Execute("write_file", []byte(`{"path":"outside/evil.txt","content":"x"}`))
	if !strings.Contains(res, "symlink") {
		t.Errorf("write_file through symlink = %q, want symlink escape error", res)
	}
	if _, err := os.Stat(filepath.Join(outside, "evil.txt")); err == nil {
		t.Error("file written outside the sandbox through a symlink")
	}
	if res := e.Execute("read_file", []byte(`{"path":"outside/evil.txt"}`)); !strings.Contains(res, "symlink") {
		t.Errorf("read_file through symlink = %q, want symlink escape error", res)
	}
}

func TestSymlinkInsideSandboxAllowed(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	target := filepath.Join(e.Cwd, "real")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(e.Cwd, "alias")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res := e.Execute("write_file", []byte(`{"path":"alias/ok.txt","content":"fine"}`))
	if strings.HasPrefix(res, "error") || strings.Contains(res, "cancelled") {
		t.Fatalf("write_file via internal symlink = %q", res)
	}
	data, err := os.ReadFile(filepath.Join(target, "ok.txt"))
	if err != nil || string(data) != "fine" {
		t.Errorf("read via symlink = %q, %v; want fine", data, err)
	}
}

func TestRunShellSuccess(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	res := e.Execute("run_shell", []byte(`{"command":"echo hello"}`))
	if !strings.Contains(res, "exit code 0") || !strings.Contains(res, "hello") {
		t.Errorf("run_shell echo = %q, want exit code 0 + hello", res)
	}
}

func TestRunShellExitCode(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	res := e.Execute("run_shell", []byte(`{"command":"exit 3"}`))
	if !strings.Contains(res, "exit code 3") {
		t.Errorf("run_shell exit 3 = %q, want exit code 3", res)
	}
}

func TestRunShellRunsInCwd(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	if res := e.Execute("write_file", []byte(`{"path":"sub/keep.txt","content":"x"}`)); strings.HasPrefix(res, "error") {
		t.Fatalf("setup: %s", res)
	}
	res := e.Execute("run_shell", []byte(`{"command":"pwd","cwd":"sub"}`))
	if !strings.Contains(res, filepath.Join(e.Cwd, "sub")) {
		t.Errorf("run_shell cwd = %q, want %q", res, filepath.Join(e.Cwd, "sub"))
	}
}

func TestRunShellTimeout(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	e.Timeout = 100 * time.Millisecond
	res := e.Execute("run_shell", []byte(`{"command":"sleep 10"}`))
	if !strings.Contains(res, "timed out") {
		t.Errorf("run_shell timeout = %q, want timeout message", res)
	}
}

func TestRunShellDeclined(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return false })
	res := e.Execute("run_shell", []byte(`{"command":"echo hi"}`))
	if !strings.Contains(res, "cancelled by user") {
		t.Errorf("declined run_shell = %q, want cancellation", res)
	}
}

func TestRunShellBlocklist(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true }) // even confirmed, blocked
	blocked := []string{
		"rm -rf /",
		"rm -rf / ",
		"rm -rf //",
		"rm -rf ~",
		"rm -rf ~/x",
		"sudo rm -rf /",
		":(){ :|:& };:",
		"mkfs.ext4 /dev/sda1",
		"dd if=/dev/zero of=/dev/sda bs=1M",
		"echo x > /dev/sda",
		"shutdown now",
		"reboot",
	}
	for _, cmd := range blocked {
		t.Run(cmd, func(t *testing.T) {
			if !isBlocked(cmd) {
				t.Errorf("isBlocked(%q) = false, want true", cmd)
			}
			res := e.Execute("run_shell", []byte(`{"command":"`+strings.ReplaceAll(cmd, `"`, `\"`)+`"}`))
			if !strings.Contains(res, "refused") {
				t.Errorf("run_shell %q = %q, want refusal", cmd, res)
			}
		})
	}
}

func TestRunShellBlocklistSafeCommands(t *testing.T) {
	safe := []string{
		"rm -rf node_modules",
		"ls -la",
		"npm install",
		"echo rm -rf",
		"grep -i power README.md",
	}
	for _, cmd := range safe {
		if isBlocked(cmd) {
			t.Errorf("isBlocked(%q) = true, want false", cmd)
		}
	}
}

func TestMalformedArgs(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	res := e.Execute("write_file", []byte(`{"path":123}`)) // path must be a string
	if !strings.Contains(res, "error: invalid arguments for write_file") {
		t.Errorf("malformed write_file = %q", res)
	}
	res = e.Execute("read_file", []byte(`not json`))
	if !strings.Contains(res, "error: invalid arguments for read_file") {
		t.Errorf("malformed read_file = %q", res)
	}
}

func TestUnknownTool(t *testing.T) {
	e := newTestExecutor(t, nil)
	res := e.Execute("frobnicate", []byte(`{}`))
	if !strings.Contains(res, `unknown tool "frobnicate"`) {
		t.Errorf("unknown tool result = %q", res)
	}
}

func TestExecuteCustomDispatchTakesPriority(t *testing.T) {
	e := newTestExecutor(t, func(string) bool { return true })
	called := false
	e.Custom = map[string]func([]byte) string{
		"satusky_run": func(argsJSON []byte) string {
			called = true
			return "custom result: " + string(argsJSON)
		},
	}
	res := e.Execute("satusky_run", []byte(`{"args":["app","list"]}`))
	if !called || res != "custom result: {\"args\":[\"app\",\"list\"]}" {
		t.Errorf("Execute = %q, called=%v", res, called)
	}
	// Built-ins still work when not overridden.
	if err := os.WriteFile(filepath.Join(e.Cwd, "x.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if res := e.Execute("list_dir", []byte(`{}`)); !strings.Contains(res, "x.txt") {
		t.Errorf("built-in after Custom = %q", res)
	}
	// An unknown tool still errors even with Custom set.
	if res := e.Execute("frobnicate", nil); !strings.Contains(res, `unknown tool "frobnicate"`) {
		t.Errorf("unknown tool with Custom = %q", res)
	}
}

func TestNilConfirmDeclines(t *testing.T) {
	e := NewExecutor(t.TempDir(), nil)
	e.Timeout = time.Second
	if res := e.Execute("run_shell", []byte(`{"command":"echo hi"}`)); !strings.Contains(res, "cancelled by user") {
		t.Errorf("nil confirm run_shell = %q, want cancellation", res)
	}
}

func TestExecuteNeverPanics(t *testing.T) {
	e := newTestExecutor(t, nil)
	// A tool call whose execution panics must surface as a string, not a
	// panic. We simulate it through an empty-path write to a weird base.
	e.Cwd = string([]byte{0}) // invalid directory: operations must error, not panic
	res := e.Execute("write_file", []byte(`{"path":"x.txt","content":"y"}`))
	if res == "" {
		t.Error("Execute returned empty string, want an error message")
	}
}
