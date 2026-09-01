package chat

import "testing"

func TestParseProvider(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   Provider
		wantOK bool
	}{
		{name: "openai lowercase", input: "openai", want: ProviderOpenAI, wantOK: true},
		{name: "openai uppercase", input: "OPENAI", want: ProviderOpenAI, wantOK: true},
		{name: "openai mixed case", input: "OpenAI", want: ProviderOpenAI, wantOK: true},
		{name: "claude with whitespace", input: "  claude  ", want: ProviderClaude, wantOK: true},
		{name: "deepseek tab and newline", input: "\tDeepSeek\n", want: ProviderDeepSeek, wantOK: true},
		{name: "unknown provider", input: "gemini", wantOK: false},
		{name: "empty string", input: "", wantOK: false},
		{name: "whitespace only", input: "   ", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseProvider(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ParseProvider(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.Name != tt.want {
				t.Errorf("ParseProvider(%q) = %q, want %q", tt.input, got.Name, tt.want)
			}
		})
	}
}

func TestAllProvidersOrder(t *testing.T) {
	want := []Provider{ProviderOpenAI, ProviderClaude, ProviderDeepSeek}
	got := AllProviders()
	if len(got) != len(want) {
		t.Fatalf("AllProviders() returned %d providers, want %d", len(got), len(want))
	}
	for i, wantName := range want {
		if got[i].Name != wantName {
			t.Errorf("AllProviders()[%d].Name = %q, want %q", i, got[i].Name, wantName)
		}
	}
}

func TestAllProvidersDefaults(t *testing.T) {
	providers := AllProviders()
	if len(providers) == 0 {
		t.Fatal("AllProviders() returned no providers")
	}
	for _, p := range providers {
		if p.Name == "" {
			t.Error("provider with empty Name")
		}
		if p.DisplayName == "" {
			t.Errorf("%s: empty DisplayName", p.Name)
		}
		if p.BaseURL == "" {
			t.Errorf("%s: empty BaseURL", p.Name)
		}
		if p.DefaultModel == "" {
			t.Errorf("%s: empty DefaultModel", p.Name)
		}
		if p.EnvKey == "" {
			t.Errorf("%s: empty EnvKey", p.Name)
		}
		if p.KeyPrefixHint == "" {
			t.Errorf("%s: empty KeyPrefixHint", p.Name)
		}
		if !p.IsValid() {
			t.Errorf("%s: IsValid() = false for a provider from AllProviders()", p.Name)
		}
	}
}

func TestEnvKeysUnique(t *testing.T) {
	seen := make(map[string]Provider, len(AllProviders()))
	for _, p := range AllProviders() {
		if prev, exists := seen[p.EnvKey]; exists {
			t.Errorf("env key %q shared by providers %s and %s", p.EnvKey, prev, p.Name)
		}
		seen[p.EnvKey] = p.Name
	}
}
