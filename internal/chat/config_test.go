package chat

import (
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(t.TempDir())
}

func TestStoreLoadMissingFile(t *testing.T) {
	st := testStore(t)
	cfg, err := st.Load()
	if err != nil {
		t.Fatalf("Load() on missing file: %v", err)
	}
	if cfg.ActiveProvider != "" {
		t.Errorf("ActiveProvider = %q, want empty", cfg.ActiveProvider)
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("Providers = %v, want empty", cfg.Providers)
	}
}

func TestStoreSaveLoadRoundtrip(t *testing.T) {
	st := testStore(t)
	want := &Config{
		ActiveProvider: ProviderClaude,
		Providers: map[Provider]ProviderConfig{
			ProviderOpenAI: {APIKey: "sk-abc", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini", Connected: true, LastVerifiedAt: "2025-01-01T00:00:00Z"},
			ProviderClaude: {BaseURL: "https://api.anthropic.com/v1/", Model: "claude-sonnet-4-6"},
		},
	}
	if err := st.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ActiveProvider != want.ActiveProvider {
		t.Errorf("ActiveProvider = %q, want %q", got.ActiveProvider, want.ActiveProvider)
	}
	if len(got.Providers) != len(want.Providers) {
		t.Fatalf("Providers = %v, want %v", got.Providers, want.Providers)
	}
	for name, wantPC := range want.Providers {
		gotPC, ok := got.Providers[name]
		if !ok {
			t.Errorf("provider %s missing after roundtrip", name)
			continue
		}
		if gotPC != wantPC {
			t.Errorf("provider %s = %+v, want %+v", name, gotPC, wantPC)
		}
	}
}

func TestStoreSaveAtomicFileAndPerms(t *testing.T) {
	st := testStore(t)
	cfg := &Config{ActiveProvider: ProviderOpenAI}
	if err := st.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(st.path())
	if err != nil {
		t.Fatalf("chat.json not found after Save: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("chat.json perms = %o, want 0600", info.Mode().Perm())
	}
	// No temp files should be left behind after the rename.
	matches, err := filepath.Glob(filepath.Join(st.Dir(), chatFile+".tmp*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("leftover temp files: %v", matches)
	}
}

func TestStoreSetProviderPreservesOthers(t *testing.T) {
	st := testStore(t)
	if err := st.SetProvider(ProviderOpenAI, ProviderConfig{APIKey: "sk-a", Model: "gpt-4o-mini"}); err != nil {
		t.Fatalf("SetProvider openai: %v", err)
	}
	if err := st.SetProvider(ProviderDeepSeek, ProviderConfig{APIKey: "sk-d", Model: "deepseek-chat"}); err != nil {
		t.Fatalf("SetProvider deepseek: %v", err)
	}
	openaiPC, ok := st.GetProvider(ProviderOpenAI)
	if !ok || openaiPC.APIKey != "sk-a" {
		t.Errorf("openai config lost after second SetProvider: %+v ok=%v", openaiPC, ok)
	}
	deepseekPC, ok := st.GetProvider(ProviderDeepSeek)
	if !ok || deepseekPC.APIKey != "sk-d" {
		t.Errorf("deepseek config = %+v ok=%v, want sk-d", deepseekPC, ok)
	}
}

func TestStoreActiveFallbackToOpenAI(t *testing.T) {
	st := testStore(t)
	info, pc, err := st.Active()
	if err != nil {
		t.Fatalf("Active(): %v", err)
	}
	if info.Name != ProviderOpenAI {
		t.Errorf("Active() fallback = %s, want openai", info.Name)
	}
	if pc.Model != "" {
		t.Errorf("Active() pc = %+v, want empty config", pc)
	}
}

func TestStoreActiveUnknownProvider(t *testing.T) {
	st := testStore(t)
	if err := st.Save(&Config{ActiveProvider: Provider("gemini")}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, _, err := st.Active(); err == nil {
		t.Fatal("Active() with unknown provider: want error, got nil")
	}
}

func TestStoreResolvedKeyEnvPriority(t *testing.T) {
	st := testStore(t)
	info, _ := ParseProvider("openai")
	if err := st.SetProvider(ProviderOpenAI, ProviderConfig{APIKey: "stored-key"}); err != nil {
		t.Fatalf("SetProvider: %v", err)
	}
	pc, _ := st.GetProvider(ProviderOpenAI)

	// No env var set → stored key wins.
	key, fromEnv := st.ResolvedKey(info, pc)
	if key != "stored-key" || fromEnv {
		t.Errorf("without env: key=%q fromEnv=%v, want stored-key false", key, fromEnv)
	}

	// Env var set → env wins, and it is not persisted.
	t.Setenv(info.EnvKey, "env-key")
	key, fromEnv = st.ResolvedKey(info, pc)
	if key != "env-key" || !fromEnv {
		t.Errorf("with env: key=%q fromEnv=%v, want env-key true", key, fromEnv)
	}
	reloaded, _ := st.GetProvider(ProviderOpenAI)
	if reloaded.APIKey != "stored-key" {
		t.Errorf("env key leaked into storage: %q", reloaded.APIKey)
	}
}

func TestStoreDisconnectKeepsModel(t *testing.T) {
	st := testStore(t)
	if err := st.SetProvider(ProviderClaude, ProviderConfig{
		APIKey: "sk-ant-xyz", BaseURL: "https://api.anthropic.com/v1/",
		Model: "claude-sonnet-4-6", Connected: true, LastVerifiedAt: "2025-01-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("SetProvider: %v", err)
	}
	if err := st.Disconnect(ProviderClaude); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	pc, ok := st.GetProvider(ProviderClaude)
	if !ok {
		t.Fatal("provider missing after disconnect")
	}
	if pc.APIKey != "" || pc.Connected || pc.LastVerifiedAt != "" {
		t.Errorf("key/connected not cleared: %+v", pc)
	}
	if pc.Model != "claude-sonnet-4-6" || pc.BaseURL != "https://api.anthropic.com/v1/" {
		t.Errorf("model/base_url not preserved: %+v", pc)
	}
}

func TestStoreDisconnectMissingProvider(t *testing.T) {
	st := testStore(t)
	if err := st.Disconnect(ProviderDeepSeek); err != nil {
		t.Fatalf("Disconnect on missing provider: %v", err)
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "short key fully masked", in: "sk-abc", want: "••••"},
		{name: "exactly 8 chars masked", in: "12345678", want: "••••"},
		{name: "typical openai key", in: "sk-abcdefghijklmnop", want: "sk-a…mnop"},
		{name: "claude key", in: "sk-ant-api03-abcdefghijklmnopqrstuvwxyz", want: "sk-a…wxyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskKey(tt.in); got != tt.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
