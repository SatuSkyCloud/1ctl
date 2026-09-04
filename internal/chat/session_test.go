package chat

import (
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestNewSessionDefaults(t *testing.T) {
	s := NewSession("you are helpful")
	if s.System != "you are helpful" {
		t.Errorf("System = %q", s.System)
	}
	if s.MaxTokens != defaultMaxTokens {
		t.Errorf("MaxTokens = %d, want %d", s.MaxTokens, defaultMaxTokens)
	}
	if len(s.Raw()) != 0 {
		t.Errorf("new session should be empty, got %v", s.Raw())
	}
}

func TestSessionAddAndRaw(t *testing.T) {
	s := NewSession("sys")
	s.Add(openai.ChatMessageRoleUser, "hi")
	s.Add(openai.ChatMessageRoleAssistant, "hello!")
	raw := s.Raw()
	if len(raw) != 2 {
		t.Fatalf("len(Raw()) = %d, want 2", len(raw))
	}
	if raw[0].Role != openai.ChatMessageRoleUser || raw[0].Content != "hi" {
		t.Errorf("raw[0] = %+v", raw[0])
	}
	if raw[1].Role != openai.ChatMessageRoleAssistant || raw[1].Content != "hello!" {
		t.Errorf("raw[1] = %+v", raw[1])
	}
}

func TestSessionSystemMessage(t *testing.T) {
	s := NewSession("sys prompt")
	msg := s.SystemMessage()
	if msg.Role != openai.ChatMessageRoleSystem || msg.Content != "sys prompt" {
		t.Errorf("SystemMessage() = %+v", msg)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{in: "", want: 1},
		{in: "hello", want: 2},                      // 5/4+1 = 2
		{in: strings.Repeat("a", 4000), want: 1001}, // 4000/4+1
	}
	for _, tt := range tests {
		if got := EstimateTokens(tt.in); got != tt.want {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// longMessage returns a message long enough to dominate the token budget.
func longMessage(n int) string { return strings.Repeat("x", n*4) }

func TestTrimDropsOldestFirstKeepsSystem(t *testing.T) {
	s := NewSession("system")
	s.MaxTokens = 40
	// 5 messages of ~20 estimated tokens each (80 chars -> 21 tokens).
	for i := 0; i < 5; i++ {
		s.Add(openai.ChatMessageRoleUser, longMessage(20))
	}
	s.Trim()
	if len(s.Raw()) == 5 {
		t.Fatal("Trim() did not drop any messages")
	}
	// The system prompt must survive trimming.
	if got := s.SystemMessage().Content; got != "system" {
		t.Errorf("system prompt lost: %q", got)
	}
	if est := s.estimatedTokens(); est > s.MaxTokens {
		t.Errorf("estimated tokens %d still exceeds budget %d", est, s.MaxTokens)
	}
	// The oldest (first) messages must be the ones dropped: the last
	// message in the history must be the final user turn.
	raw := s.Raw()
	if len(raw) == 0 || raw[len(raw)-1].Content != longMessage(20) {
		t.Errorf("most recent message not preserved; tail = %+v", raw)
	}
}

func TestTrimPreservesOrder(t *testing.T) {
	s := NewSession("sys")
	s.MaxTokens = 30
	msgs := []string{longMessage(10), longMessage(11), longMessage(12), longMessage(13)}
	for _, m := range msgs {
		s.Add(openai.ChatMessageRoleUser, m)
	}
	s.Trim()
	raw := s.Raw()
	if len(raw) == 0 {
		t.Fatal("everything trimmed")
	}
	if len(raw) == len(msgs) {
		t.Fatal("expected trimming to drop at least one message")
	}
	// Remaining messages must be a suffix of the original order.
	idx := 0
	for _, want := range msgs {
		if idx < len(raw) && raw[idx].Content == want {
			idx++
		}
	}
	if idx != len(raw) {
		t.Errorf("message order corrupted: %v", raw)
	}
}

func TestTrimEmptyAndUnderBudget(t *testing.T) {
	s := NewSession("sys")
	s.Add(openai.ChatMessageRoleUser, "short")
	s.Trim()
	if len(s.Raw()) != 1 {
		t.Errorf("under-budget session was trimmed: %v", s.Raw())
	}
	empty := NewSession("sys")
	empty.Trim()
	if len(empty.Raw()) != 0 {
		t.Errorf("empty session trimmed to %v", empty.Raw())
	}
}

func TestTrimResetsZeroBudget(t *testing.T) {
	s := NewSession("sys")
	s.MaxTokens = 0
	for i := 0; i < 10; i++ {
		s.Add(openai.ChatMessageRoleUser, longMessage(100))
	}
	s.Trim()
	if s.MaxTokens != defaultMaxTokens {
		t.Errorf("MaxTokens = %d, want default %d", s.MaxTokens, defaultMaxTokens)
	}
	if est := s.estimatedTokens(); est > s.MaxTokens {
		t.Errorf("estimated tokens %d exceed budget %d", est, s.MaxTokens)
	}
}
