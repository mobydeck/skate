package translate

import (
	"testing"

	"skate/internal/config"
)

func TestNew_Disabled(t *testing.T) {
	cfg := config.TranslateConfig{Enabled: false}
	tr := New(cfg)
	if tr != nil {
		t.Error("disabled config should return nil translator")
	}
}

func TestNew_Enabled(t *testing.T) {
	cfg := config.TranslateConfig{
		Enabled: true,
		Model:   "test-model",
		APIKey:  "test-key",
	}
	tr := New(cfg)
	if tr == nil {
		t.Fatal("enabled config should return translator")
	}
	if tr.model != "test-model" {
		t.Errorf("model: got %q", tr.model)
	}
}

func TestNew_DefaultModel(t *testing.T) {
	cfg := config.TranslateConfig{Enabled: true, APIKey: "k"}
	tr := New(cfg)
	if tr.model != "gpt-4o-mini" {
		t.Errorf("default model: got %q", tr.model)
	}
}

func TestTranslate_NilTranslator(t *testing.T) {
	var tr *Translator
	got := tr.Translate("hello")
	if got != "hello" {
		t.Errorf("nil translator should return input, got %q", got)
	}
}

func TestTranslate_EmptyText(t *testing.T) {
	tr := &Translator{enabled: true}
	got := tr.Translate("")
	if got != "" {
		t.Errorf("empty text should return empty, got %q", got)
	}
}

func TestIsLikelyEnglish(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"Hello world", true},
		{"This is a test with some code: func main() {}", true},
		{"", true},
		{"123 456", true},
		{"Привет мир", false},        // Russian
		{"こんにちは世界", false},       // Japanese
		{"Mixed hello мир text", true}, // mostly ASCII
		{"```\ncode block\n```", true}, // code only
		{"this is a long english sentence with just one word мир in another language", true}, // >80% ASCII
	}

	for _, tt := range tests {
		got := isLikelyEnglish(tt.text)
		if got != tt.want {
			t.Errorf("isLikelyEnglish(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestStripCodeBlocks(t *testing.T) {
	input := "before\n```\ncode here\n```\nafter"
	got := stripCodeBlocks(input)
	if contains(got, "code here") {
		t.Errorf("should strip code blocks, got %q", got)
	}
	if !contains(got, "before") || !contains(got, "after") {
		t.Errorf("should keep non-code text, got %q", got)
	}
}

func TestFormatProviderInfo(t *testing.T) {
	cfg := config.TranslateConfig{Enabled: false}
	if FormatProviderInfo(cfg) != "disabled" {
		t.Error("disabled should return 'disabled'")
	}

	cfg = config.TranslateConfig{Enabled: true}
	got := FormatProviderInfo(cfg)
	if got != "openai/gpt-4o-mini" {
		t.Errorf("defaults: got %q", got)
	}

	cfg = config.TranslateConfig{
		Enabled: true,
		Provider: "ollama",
		Model:   "llama3",
		BaseURL: "http://localhost:11434/v1",
	}
	got = FormatProviderInfo(cfg)
	if got != "ollama/llama3 (http://localhost:11434/v1)" {
		t.Errorf("ollama: got %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
