package chat

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

var transcriptNameRe = regexp.MustCompile(`^chat-transcript-\d{8}-\d{6}\.md$`)

func transcriptSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession("sys prompt")
	s.Add(openai.ChatMessageRoleUser, "hello there")
	s.Add(openai.ChatMessageRoleAssistant, "hi! how can I help?")
	s.Add(openai.ChatMessageRoleUser, "explain env vs secret")
	return s
}

func TestRenderTranscriptFormat(t *testing.T) {
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	got := renderTranscript(transcriptSession(t), ProviderOpenAI, "gpt-4o-mini", now)

	for _, want := range []string{
		"# 1ctl chat — 2026-03-04 05:06:07",
		"provider: openai · model: gpt-4o-mini",
		"## user",
		"## assistant",
		"hello there",
		"hi! how can I help?",
		"explain env vs secret",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("transcript missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "## user") != 2 || strings.Count(got, "## assistant") != 1 {
		t.Errorf("section counts wrong:\n%s", got)
	}
	// The system prompt must never leak into the transcript.
	if strings.Contains(got, "sys prompt") {
		t.Errorf("system prompt leaked into transcript:\n%s", got)
	}
}

func TestDefaultTranscriptName(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 59, 58, 0, time.UTC)
	name := defaultTranscriptName(now)
	if !transcriptNameRe.MatchString(name) {
		t.Errorf("defaultTranscriptName(%v) = %q, want pattern %s", now, name, transcriptNameRe)
	}
	if name != "chat-transcript-20261231-235958.md" {
		t.Errorf("defaultTranscriptName = %q, want chat-transcript-20261231-235958.md", name)
	}
}

func TestResolveExportPath(t *testing.T) {
	cwd := "/home/user/project"
	tests := []struct {
		name    string
		arg     string
		want    string
		wantErr bool
	}{
		{name: "default name", arg: "", wantErr: false},
		{name: "plain relative", arg: "notes.md", want: "/home/user/project/notes.md"},
		{name: "nested relative", arg: "docs/chat.md", want: "/home/user/project/docs/chat.md"},
		{name: "dot slash", arg: "./out.md", want: "/home/user/project/out.md"},
		{name: "absolute rejected", arg: "/etc/passwd", wantErr: true},
		{name: "parent traversal rejected", arg: "../escape.md", wantErr: true},
		{name: "deep traversal rejected", arg: "a/../../escape.md", wantErr: true},
		{name: "bare dotdot rejected", arg: "..", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveExportPath(cwd, tt.arg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveExportPath(%q) = %q, want error", tt.arg, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveExportPath(%q): %v", tt.arg, err)
			}
			if tt.arg == "" {
				// The default embeds time.Now(); verify shape + location.
				if !transcriptNameRe.MatchString(filepath.Base(got)) {
					t.Errorf("default path base = %q, want %s pattern", filepath.Base(got), transcriptNameRe)
				}
				if filepath.Dir(got) != cwd {
					t.Errorf("default path dir = %q, want cwd %q", filepath.Dir(got), cwd)
				}
				return
			}
			if got != tt.want {
				t.Errorf("resolveExportPath(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestExportTranscriptWritesFile(t *testing.T) {
	dir := t.TempDir()
	path, err := exportTranscript(transcriptSession(t), ProviderOpenAI, "gpt-4o-mini", dir, "out.md")
	if err != nil {
		t.Fatalf("exportTranscript: %v", err)
	}
	if path != filepath.Join(dir, "out.md") {
		t.Errorf("path = %q, want %q", path, filepath.Join(dir, "out.md"))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "## user") || !strings.Contains(content, "hello there") {
		t.Errorf("exported content incomplete:\n%s", content)
	}
}

func TestExportTranscriptDefaultPath(t *testing.T) {
	dir := t.TempDir()
	path, err := exportTranscript(transcriptSession(t), ProviderOpenAI, "gpt-4o-mini", dir, "")
	if err != nil {
		t.Fatalf("exportTranscript: %v", err)
	}
	if !transcriptNameRe.MatchString(filepath.Base(path)) {
		t.Errorf("default filename = %q, want %s pattern", filepath.Base(path), transcriptNameRe)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("exported file not created: %v", err)
	}
}

func TestExportTranscriptEmptyConversationNoOp(t *testing.T) {
	dir := t.TempDir()
	path, err := exportTranscript(NewSession("sys"), ProviderOpenAI, "gpt-4o-mini", dir, "out.md")
	if err != nil {
		t.Fatalf("exportTranscript: %v", err)
	}
	if path != "" {
		t.Errorf("path = %q, want empty for empty conversation", path)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "out.md")); !os.IsNotExist(statErr) {
		t.Errorf("file created for empty conversation (stat err = %v)", statErr)
	}
}

func TestExportTranscriptTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	_, err := exportTranscript(transcriptSession(t), ProviderOpenAI, "gpt-4o-mini", dir, "../escape.md")
	if err == nil {
		t.Fatal("expected traversal rejection, got nil")
	}
	if !strings.Contains(err.Error(), "escapes the current directory") {
		t.Errorf("error = %q, want traversal message", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "escape.md")); !os.IsNotExist(statErr) {
		t.Errorf("file written outside cwd (stat err = %v)", statErr)
	}
}
