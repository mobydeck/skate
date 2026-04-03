package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobal_Defaults(t *testing.T) {
	// When no config file exists, should return empty config
	os.Setenv("HOME", t.TempDir())
	defer os.Unsetenv("HOME")

	cfg, err := LoadGlobal()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MattermostURL != "" {
		t.Errorf("expected empty URL, got %q", cfg.MattermostURL)
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("SKATE_URL", "https://test.example.com")
	os.Setenv("SKATE_TOKEN", "test-token")
	os.Setenv("SKATE_TEAM_ID", "team-123")
	os.Setenv("SKATE_BOARD_ID", "board-456")
	os.Setenv("SKATE_TRANSLATE_ENABLED", "true")
	os.Setenv("SKATE_TRANSLATE_MODEL", "gpt-4")
	defer func() {
		os.Unsetenv("SKATE_URL")
		os.Unsetenv("SKATE_TOKEN")
		os.Unsetenv("SKATE_TEAM_ID")
		os.Unsetenv("SKATE_BOARD_ID")
		os.Unsetenv("SKATE_TRANSLATE_ENABLED")
		os.Unsetenv("SKATE_TRANSLATE_MODEL")
	}()

	cfg := &Config{}
	applyEnvOverrides(cfg)

	if cfg.MattermostURL != "https://test.example.com" {
		t.Errorf("URL: got %q", cfg.MattermostURL)
	}
	if cfg.Token != "test-token" {
		t.Errorf("Token: got %q", cfg.Token)
	}
	if cfg.TeamID != "team-123" {
		t.Errorf("TeamID: got %q", cfg.TeamID)
	}
	if cfg.BoardID != "board-456" {
		t.Errorf("BoardID: got %q", cfg.BoardID)
	}
	if !cfg.Translate.Enabled {
		t.Error("Translate.Enabled should be true")
	}
	if cfg.Translate.Model != "gpt-4" {
		t.Errorf("Translate.Model: got %q", cfg.Translate.Model)
	}
}

func TestMergeLocal(t *testing.T) {
	global := &Config{
		MattermostURL: "https://global.example.com",
		Token:         "global-token",
		TeamID:        "global-team",
	}
	local := &Config{
		BoardID: "local-board",
	}

	mergeLocal(global, local)

	if global.MattermostURL != "https://global.example.com" {
		t.Error("global URL should not change when local is empty")
	}
	if global.BoardID != "local-board" {
		t.Errorf("BoardID should be overridden, got %q", global.BoardID)
	}
}

func TestMergeLocal_EmptyDoesNotOverride(t *testing.T) {
	global := &Config{
		MattermostURL: "https://global.example.com",
		Token:         "global-token",
	}
	local := &Config{
		MattermostURL: "",
		Token:         "",
		BoardID:       "board-123",
	}

	mergeLocal(global, local)

	if global.MattermostURL != "https://global.example.com" {
		t.Error("empty local URL should not override global")
	}
	if global.Token != "global-token" {
		t.Error("empty local token should not override global")
	}
	if global.BoardID != "board-123" {
		t.Error("local BoardID should override")
	}
}

func TestValidate(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Error("empty config should fail validation")
	}

	cfg.MattermostURL = "https://example.com"
	if err := cfg.Validate(); err == nil {
		t.Error("missing token should fail")
	}

	cfg.Token = "token"
	if err := cfg.Validate(); err == nil {
		t.Error("missing team_id should fail")
	}

	cfg.TeamID = "team"
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should pass: %v", err)
	}
}

func TestSave_OmitsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	cfg := &Config{BoardID: "board-123"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if contains(content, "mattermost_url") {
		t.Error("should not contain empty mattermost_url")
	}
	if contains(content, "token") {
		t.Error("should not contain empty token")
	}
	if !contains(content, "board_id: board-123") {
		t.Error("should contain board_id")
	}
}

func TestBaseURL(t *testing.T) {
	cfg := &Config{MattermostURL: "https://mm.example.com"}
	got := BaseURL(cfg)
	want := "https://mm.example.com/plugins/focalboard/api/v2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindLocalConfig(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(sub, 0o755)

	// No config anywhere
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(sub)

	if got := FindLocalConfig(); got != "" {
		t.Errorf("should find nothing, got %q", got)
	}

	// Create config at parent
	os.WriteFile(filepath.Join(dir, "a", ".skate.yaml"), []byte("board_id: test"), 0o644)

	if got := FindLocalConfig(); got == "" {
		t.Error("should find .skate.yaml in parent")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && filepath.Base(s) != "" && // just use strings
		stringContains(s, substr)
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
