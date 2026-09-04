// Package chat contains the core engine for "1ctl chat": provider
// definitions, configuration storage, and (in later phases) the client,
// connect flow, session management, streaming and REPL. It intentionally
// imports no urfave/cli; the CLI wiring lives in internal/commands/chat.
package chat

import "strings"

// Provider identifies a chat model provider by its stable, lowercase name.
type Provider string

const (
	// ProviderOpenAI is OpenAI (api.openai.com).
	ProviderOpenAI Provider = "openai"
	// ProviderClaude is Anthropic's Claude, accessed through Anthropic's
	// OpenAI-compatible endpoint (api.anthropic.com/v1).
	ProviderClaude Provider = "claude"
	// ProviderDeepSeek is DeepSeek (api.deepseek.com).
	ProviderDeepSeek Provider = "deepseek"
)

// ProviderInfo describes a chat provider: how to reach it, which model to
// use by default, and how the user's API key is supplied.
type ProviderInfo struct {
	// Name is the stable, lowercase provider identifier ("openai", ...).
	Name Provider
	// DisplayName is the human-readable provider name.
	DisplayName string
	// BaseURL is the OpenAI-compatible chat completions base URL.
	BaseURL string
	// DefaultModel is the model used when the user does not pick one.
	// Model IDs drift fast; defaults are re-verified against provider docs.
	DefaultModel string
	// EnvKey is the environment variable holding the API key.
	EnvKey string
	// KeyPrefixHint is the expected API key prefix (e.g. "sk-", "sk-ant-"),
	// used for early validation and friendly diagnostics.
	KeyPrefixHint string
}

// IsValid reports whether the ProviderInfo names a known provider.
func (p ProviderInfo) IsValid() bool {
	_, ok := ParseProvider(string(p.Name))
	return ok
}

// AllProviders returns every supported provider in deterministic order:
// openai, claude, deepseek.
func AllProviders() []ProviderInfo {
	return []ProviderInfo{
		{
			Name:          ProviderOpenAI,
			DisplayName:   "OpenAI",
			BaseURL:       "https://api.openai.com/v1",
			DefaultModel:  "gpt-4o-mini",
			EnvKey:        "OPENAI_API_KEY",
			KeyPrefixHint: "sk-",
		},
		{
			Name:          ProviderClaude,
			DisplayName:   "Anthropic Claude",
			BaseURL:       "https://api.anthropic.com/v1/",
			DefaultModel:  "claude-sonnet-4-6",
			EnvKey:        "ANTHROPIC_API_KEY",
			KeyPrefixHint: "sk-ant-",
		},
		{
			Name:          ProviderDeepSeek,
			DisplayName:   "DeepSeek",
			BaseURL:       "https://api.deepseek.com",
			DefaultModel:  "deepseek-v4-flash",
			EnvKey:        "DEEPSEEK_API_KEY",
			KeyPrefixHint: "sk-",
		},
	}
}

// ParseProvider resolves a user-supplied provider name to its ProviderInfo.
// Matching is case-insensitive and surrounding whitespace is ignored; the
// second return value reports whether the name matched a known provider.
func ParseProvider(name string) (ProviderInfo, bool) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, p := range AllProviders() {
		if string(p.Name) == normalized {
			return p, true
		}
	}
	return ProviderInfo{}, false
}
