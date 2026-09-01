package chat

import (
	"context"
	"errors"
	"io"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// StreamResult carries the outcome of a completion: the assembled text,
// the total token usage when the provider reports it, and the model.
type StreamResult struct {
	Text   string
	Tokens int
	Model  string
}

// StreamCompletion streams a chat completion. Each content delta is written
// to out verbatim (no added newlines), so the caller controls formatting.
// It returns the assembled text plus usage/model metadata. Errors are
// classified through classifyError, so the caller gets friendly text.
func StreamCompletion(ctx context.Context, client *openai.Client, model string, messages []openai.ChatCompletionMessage, out io.Writer) (StreamResult, error) {
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	}
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return StreamResult{}, classifyError(err)
	}
	defer stream.Close() //nolint:errcheck // stream is drained to EOF before Close; nothing to recover

	var result StreamResult
	result.Model = model
	var sb strings.Builder
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
			if delta := chunk.Choices[0].Delta.Content; delta != "" {
				if _, err := io.WriteString(out, delta); err != nil {
					result.Text = sb.String()
					return result, err
				}
				sb.WriteString(delta)
			}
		}
		if chunk.Usage != nil {
			result.Tokens = chunk.Usage.TotalTokens
		}
		if chunk.Model != "" {
			result.Model = chunk.Model
		}
	}
	result.Text = sb.String()
	return result, nil
}

// RunCompletion performs a single non-streaming chat completion and writes
// the full response to out (when out is non-nil). Used by scripting paths
// and connect tests that want a simple round trip.
func RunCompletion(ctx context.Context, client *openai.Client, model string, messages []openai.ChatCompletionMessage, out io.Writer) (StreamResult, error) {
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return StreamResult{}, classifyError(err)
	}
	result := StreamResult{Model: model}
	if len(resp.Choices) > 0 {
		result.Text = resp.Choices[0].Message.Content
	}
	if resp.Usage.TotalTokens != 0 {
		result.Tokens = resp.Usage.TotalTokens
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
