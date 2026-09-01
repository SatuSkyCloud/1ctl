package chat

import (
	openai "github.com/sashabaranov/go-openai"
)

// defaultMaxTokens is the conversation budget: estimated tokens across all
// messages (system + history) before oldest messages are trimmed.
const defaultMaxTokens = 8000

// Session holds the conversation state: the system prompt and the message
// history. History is trimmed to the token budget (see Trim) so long chats
// stay within the model's context window.
type Session struct {
	System    string
	Messages  []openai.ChatCompletionMessage
	MaxTokens int
}

// NewSession creates an empty session with the given system prompt and the
// default token budget.
func NewSession(system string) *Session {
	return &Session{System: system, MaxTokens: defaultMaxTokens}
}

// Add appends a message with the given role ("user", "assistant", ...).
func (s *Session) Add(role, content string) {
	s.Messages = append(s.Messages, openai.ChatCompletionMessage{Role: role, Content: content})
}

// SystemMessage returns the system prompt as a chat message.
func (s *Session) SystemMessage() openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: s.System}
}

// Raw returns the conversation messages (without the system prompt). The
// caller composes the final request as system + Raw().
func (s *Session) Raw() []openai.ChatCompletionMessage {
	return s.Messages
}

// EstimateTokens is a documented heuristic: roughly one token per 4
// characters, with a floor of 1. Real tokenizers vary, but this keeps the
// budget conservative enough for trimming.
func EstimateTokens(s string) int {
	return len(s)/4 + 1
}

// estimatedTokens sums the estimated tokens for the system prompt plus all
// conversation messages.
func (s *Session) estimatedTokens() int {
	total := EstimateTokens(s.System)
	for _, m := range s.Messages {
		total += EstimateTokens(m.Content)
	}
	return total
}

// Trim drops the oldest non-system messages while the estimated token total
// (system included) exceeds the budget. The system prompt is never dropped.
func (s *Session) Trim() {
	if s.MaxTokens <= 0 {
		s.MaxTokens = defaultMaxTokens
	}
	for s.estimatedTokens() > s.MaxTokens {
		drop := -1
		for i, m := range s.Messages {
			if m.Role != openai.ChatMessageRoleSystem {
				drop = i
				break
			}
		}
		if drop == -1 {
			break // only system messages left; nothing to drop
		}
		s.Messages = append(s.Messages[:drop], s.Messages[drop+1:]...)
	}
}
