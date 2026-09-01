package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	satuskyctx "1ctl/internal/context"
)

// chatFile is the name of the chat configuration file inside the config
// directory (~/.satusky/chat.json).
const chatFile = "chat.json"

// ProviderConfig holds the per-provider chat settings persisted in
// chat.json. API keys are stored here only when the user explicitly
// connects; environment variables always take priority (see ResolvedKey).
type ProviderConfig struct {
	APIKey         string `json:"api_key,omitempty"`
	BaseURL        string `json:"base_url"`
	Model          string `json:"model"`
	Connected      bool   `json:"connected"`
	LastVerifiedAt string `json:"last_verified_at"`
}

// Config is the on-disk shape of chat.json (see plan §4.5).
type Config struct {
	ActiveProvider Provider                    `json:"active_provider"`
	Providers      map[Provider]ProviderConfig `json:"providers"`
}

// Store reads and writes the chat configuration. The on-disk file lives in
// the SatuSky config directory (~/.satusky/chat.json), is written atomically
// and is mode 0600 because it may contain API keys.
type Store struct {
	dir string
}

// NewStore constructs a chat config store rooted at dir. An empty dir uses
// the default SatuSky config directory. Tests inject a t.TempDir() here.
func NewStore(dir string) *Store {
	if dir == "" {
		dir = satuskyctx.Default().ConfigDir()
	}
	return &Store{dir: dir}
}

// Dir returns the store's root directory.
func (s *Store) Dir() string { return s.dir }

// path returns the absolute path of the chat.json file.
func (s *Store) path() string { return filepath.Join(s.dir, chatFile) }

// Load reads the chat configuration. A missing file yields an empty config,
// not an error.
func (s *Store) Load() (*Config, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read chat config %s: %w", s.path(), err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse chat config %s: %w", s.path(), err)
	}
	if cfg.Providers == nil {
		cfg.Providers = map[Provider]ProviderConfig{}
	}
	return &cfg, nil
}

// Save persists the config atomically: it writes a temp file in the same
// directory (chmod 0600) and renames it over chat.json. The directory is
// created if needed.
func (s *Store) Save(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Providers == nil {
		cfg.Providers = map[Provider]ProviderConfig{}
	}
	if err := os.MkdirAll(s.dir, 0750); err != nil {
		return fmt.Errorf("create chat config directory %s: %w", s.dir, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode chat config: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, chatFile+".tmp*")
	if err != nil {
		return fmt.Errorf("create temp chat config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck // no-op after a successful rename
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close() //nolint:errcheck // failing to close a temp file is moot when we return the chmod error
		return fmt.Errorf("chmod temp chat config: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() //nolint:errcheck // failing to close a temp file is moot when we return the write error
		return fmt.Errorf("write temp chat config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp chat config: %w", err)
	}
	if err := os.Rename(tmpName, s.path()); err != nil {
		return fmt.Errorf("replace chat config %s: %w", s.path(), err)
	}
	return nil
}

// GetProvider returns the stored config for a provider.
func (s *Store) GetProvider(p Provider) (ProviderConfig, bool) {
	cfg, err := s.Load()
	if err != nil {
		return ProviderConfig{}, false
	}
	pc, ok := cfg.Providers[p]
	return pc, ok
}

// SetProvider upserts the config for a single provider, preserving all
// other providers' settings.
func (s *Store) SetProvider(p Provider, pc ProviderConfig) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	if cfg.Providers == nil {
		cfg.Providers = map[Provider]ProviderConfig{}
	}
	cfg.Providers[p] = pc
	return s.Save(cfg)
}

// SetActiveProvider switches the active provider, preserving everything
// else in the config.
func (s *Store) SetActiveProvider(p Provider) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	cfg.ActiveProvider = p
	return s.Save(cfg)
}

// Active resolves the active provider. With no active provider set it
// falls back to openai. It errors when the stored active provider name is
// unknown.
func (s *Store) Active() (ProviderInfo, ProviderConfig, error) {
	cfg, err := s.Load()
	if err != nil {
		return ProviderInfo{}, ProviderConfig{}, err
	}
	name := cfg.ActiveProvider
	if name == "" {
		name = ProviderOpenAI
	}
	info, ok := ParseProvider(string(name))
	if !ok {
		return ProviderInfo{}, ProviderConfig{},
			fmt.Errorf("chat config %s has unknown active provider %q — fix it with /connect", s.path(), name)
	}
	return info, cfg.Providers[name], nil
}

// ResolvedKey returns the API key to use for a provider: the environment
// variable (p.EnvKey) takes priority over the stored key. The bool reports
// whether the key came from the environment.
func (s *Store) ResolvedKey(p ProviderInfo, cfg ProviderConfig) (key string, fromEnv bool) {
	if envKey := os.Getenv(p.EnvKey); envKey != "" {
		return envKey, true
	}
	return cfg.APIKey, false
}

// Disconnect clears a provider's API key and connected state while keeping
// its base URL and model.
func (s *Store) Disconnect(p Provider) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	pc, ok := cfg.Providers[p]
	if !ok {
		return nil
	}
	pc.APIKey = ""
	pc.Connected = false
	pc.LastVerifiedAt = ""
	cfg.Providers[p] = pc
	return s.Save(cfg)
}

// MaskKey renders a key for display: first 4 chars, an ellipsis, last 4
// chars. Short keys are fully masked. Used by --show-key and /providers.
func MaskKey(key string) string {
	switch {
	case key == "":
		return ""
	case len(key) <= 8:
		return "••••"
	default:
		return key[:4] + "…" + key[len(key)-4:]
	}
}
