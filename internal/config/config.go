package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"go.yaml.in/yaml/v3"
)

type TranslateConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url"`
	APIKey   string `yaml:"api_key"`
}

func DefaultTranslateConfig() TranslateConfig {
	return TranslateConfig{
		Enabled:  false,
		Provider: "openai",
		Model:    "gpt-5-mini",
	}
}

type Config struct {
	MattermostURL string          `yaml:"mattermost_url,omitempty"`
	Token         string          `yaml:"token,omitempty"`
	TeamID        string          `yaml:"team_id,omitempty"`
	BoardID       string          `yaml:"board_id,omitempty"`
	OnlyMine      bool            `yaml:"only_mine,omitempty"`
	Translate     TranslateConfig `yaml:"translate"`
}

func GlobalConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err == nil && runtime.GOOS == "windows" {
		return filepath.Join(configDir, "skate", "skate.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "skate.yaml")
}

func CacheDir() string {
	cacheDir, err := os.UserCacheDir()
	if err == nil && runtime.GOOS == "windows" {
		return filepath.Join(cacheDir, "skate")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "skate")
}

func LoadGlobal() (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(GlobalConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("reading global config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing global config: %w", err)
	}

	// Backfill translate defaults for existing configs
	if cfg.MattermostURL != "" && cfg.Translate.Provider == "" && cfg.Translate.Model == "" {
		cfg.Translate = DefaultTranslateConfig()
		_ = cfg.Save(GlobalConfigPath())
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func FindLocalConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, ".skate.yaml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func Load() (*Config, error) {
	cfg, err := LoadGlobal()
	if err != nil {
		return nil, err
	}

	localPath := FindLocalConfig()
	if localPath != "" {
		data, err := os.ReadFile(localPath)
		if err == nil {
			var local Config
			if err := yaml.Unmarshal(data, &local); err == nil {
				mergeLocal(cfg, &local)
			}
		}
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func mergeLocal(global, local *Config) {
	if local.MattermostURL != "" {
		global.MattermostURL = local.MattermostURL
	}
	if local.Token != "" {
		global.Token = local.Token
	}
	if local.TeamID != "" {
		global.TeamID = local.TeamID
	}
	if local.BoardID != "" {
		global.BoardID = local.BoardID
	}
	if local.OnlyMine {
		global.OnlyMine = true
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SKATE_URL"); v != "" {
		cfg.MattermostURL = v
	}
	if v := os.Getenv("SKATE_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("SKATE_TEAM_ID"); v != "" {
		cfg.TeamID = v
	}
	if v := os.Getenv("SKATE_BOARD_ID"); v != "" {
		cfg.BoardID = v
	}
	if v := os.Getenv("SKATE_TRANSLATE_ENABLED"); v == "true" || v == "1" {
		cfg.Translate.Enabled = true
	}
	if v := os.Getenv("SKATE_TRANSLATE_PROVIDER"); v != "" {
		cfg.Translate.Provider = v
	}
	if v := os.Getenv("SKATE_TRANSLATE_MODEL"); v != "" {
		cfg.Translate.Model = v
	}
	if v := os.Getenv("SKATE_TRANSLATE_BASE_URL"); v != "" {
		cfg.Translate.BaseURL = v
	}
	if v := os.Getenv("SKATE_TRANSLATE_API_KEY"); v != "" {
		cfg.Translate.APIKey = v
	}
}

// LocalConfig represents a per-project .skate.yaml with only the board ID.
type LocalConfig struct {
	BoardID string `yaml:"board_id"`
}

func SaveLocal(path string, boardID string) error {
	data, err := yaml.Marshal(&LocalConfig{BoardID: boardID})
	if err != nil {
		return fmt.Errorf("marshaling local config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) Validate() error {
	if c.MattermostURL == "" {
		return fmt.Errorf("mattermost_url is required (run 'skate init' or set SKATE_URL)")
	}
	if c.Token == "" {
		return fmt.Errorf("token is required (run 'skate init' or set SKATE_TOKEN)")
	}
	if c.TeamID == "" {
		return fmt.Errorf("team_id is required (run 'skate init' or set SKATE_TEAM_ID)")
	}
	return nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func BaseURL(cfg *Config) string {
	return cfg.MattermostURL + "/plugins/focalboard/api/v2"
}
