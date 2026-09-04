package chat

import (
	"context"
	"errors"
	"io"
	"sort"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// StreamResult carries the outcome of a completion: the assembled text,
// the token usage when the provider reports it (via stream include_usage
// or the non-streaming response), the model, and — when the model asked
// for tools — the assembled tool calls plus the finish reason.
type StreamResult struct {
	Text         string
	PromptTokens int
	TotalTokens  int
	Model        string
	ToolCalls    []openai.ToolCall
	FinishReason string
}

// StreamCompletion streams a chat completion. Each content delta is
// written to out verbatim (no added newlines), so the caller controls
// formatting. Tool-call deltas are assembled (see assembleToolCalls) and
// returned on the result; they are never written to out. Passing tools
// enables function calling for this request. Errors are classified
// through classifyError, so the caller gets friendly text.
func StreamCompletion(ctx context.Context, client *openai.Client, model string, messages []openai.ChatCompletionMessage, tools []openai.Tool, out io.Writer) (StreamResult, error) {
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	}
	// Transient provider errors (429/5xx) are retried with backoff; the
	// stream is only created once the request succeeds.
	var stream *openai.ChatCompletionStream
	err := withRetry(ctx, func() error {
		s, e := client.CreateChatCompletionStream(ctx, req)
		if e != nil {
			return e
		}
		stream = s
		return nil
	})
	if err != nil {
		return StreamResult{}, classifyError(err)
	}
	defer stream.Close() //nolint:errcheck // stream is drained to EOF before Close; nothing to recover

	var result StreamResult
	result.Model = model
	var sb strings.Builder
	var toolDeltas []openai.ToolCall
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			result.Text = sb.String()
			return result, classifyError(recvErr)
		}
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			if delta := choice.Delta.Content; delta != "" {
				if _, err := io.WriteString(out, delta); err != nil {
					result.Text = sb.String()
					return result, err
				}
				sb.WriteString(delta)
			}
			if len(choice.Delta.ToolCalls) > 0 {
				toolDeltas = append(toolDeltas, choice.Delta.ToolCalls...)
			}
			if choice.FinishReason != "" {
				result.FinishReason = string(choice.FinishReason)
			}
		}
		if chunk.Usage != nil {
			result.PromptTokens = chunk.Usage.PromptTokens
			result.TotalTokens = chunk.Usage.TotalTokens
		}
		if chunk.Model != "" {
			result.Model = chunk.Model
		}
	}
	result.Text = sb.String()
	result.ToolCalls = assembleToolCalls(toolDeltas)
	return result, nil
}

// assembleToolCalls merges per-index tool-call deltas into complete tool
// calls. Streaming deltas carry an Index on each fragment: the ID, type
// and function name arrive on the first fragment, and function arguments
// arrive as a concatenated string across fragments. The assembled calls
// drop the Index so they serialize as ordinary request messages.
func assembleToolCalls(deltas []openai.ToolCall) []openai.ToolCall {
	type acc struct {
		id   string
		typ  openai.ToolType
		name strings.Builder
		args strings.Builder
	}
	accs := map[int]*acc{}
	indexes := []int{}
	for _, d := range deltas {
		idx := 0
		if d.Index != nil {
			idx = *d.Index
		}
		a, ok := accs[idx]
		if !ok {
			a = &acc{}
			accs[idx] = a
			indexes = append(indexes, idx)
		}
		if d.ID != "" {
			a.id = d.ID
		}
		if d.Type != "" {
			a.typ = d.Type
		}
		if d.Function.Name != "" {
			a.name.WriteString(d.Function.Name)
		}
		a.args.WriteString(d.Function.Arguments)
	}
	sort.Ints(indexes)
	calls := make([]openai.ToolCall, 0, len(indexes))
	for _, idx := range indexes {
		a := accs[idx]
		calls = append(calls, openai.ToolCall{
			ID:       a.id,
			Type:     a.typ,
			Function: openai.FunctionCall{Name: a.name.String(), Arguments: a.args.String()},
		})
	}
	return calls
}

// RunCompletion performs a single non-streaming chat completion and writes
// the full response to out (when out is non-nil). Used by scripting paths
// and connect tests that want a simple round trip.
func RunCompletion(ctx context.Context, client *openai.Client, model string, messages []openai.ChatCompletionMessage, out io.Writer) (StreamResult, error) {
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}
	var resp openai.ChatCompletionResponse
	if err := withRetry(ctx, func() error {
		r, e := client.CreateChatCompletion(ctx, req)
		if e != nil {
			return e
		}
		resp = r
		return nil
	}); err != nil {
		return StreamResult{}, classifyError(err)
	}
	result := StreamResult{Model: model}
	if len(resp.Choices) > 0 {
		result.Text = resp.Choices[0].Message.Content
		result.FinishReason = string(resp.Choices[0].FinishReason)
		result.ToolCalls = resp.Choices[0].Message.ToolCalls
	}
	if resp.Usage.TotalTokens != 0 {
		result.PromptTokens = resp.Usage.PromptTokens
		result.TotalTokens = resp.Usage.TotalTokens
	}
	if resp.Model != "" {
		result.Model = resp.Model
	}
	if result.Text != "" && out != nil {
		if _, err := io.WriteString(out, result.Text); err != nil {
			return result, err
		}
	}
	return result, nil
}
