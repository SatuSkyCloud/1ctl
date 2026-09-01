package chat

import (
	openai "github.com/sashabaranov/go-openai"
)

// NewClient builds an OpenAI-compatible chat client for the given provider,
// using the provider's base URL and bearer-token auth with the API key.
func NewClient(p ProviderInfo, apiKey string) *openai.Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = p.BaseURL
	return openai.NewClientWithConfig(cfg)
}

// NewClientWithBaseURL builds a chat client with an explicit base URL
// override (custom or OpenAI-compatible endpoints).
func NewClientWithBaseURL(apiKey, baseURL string) *openai.Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return openai.NewClientWithConfig(cfg)
}
