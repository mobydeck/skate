package translate

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"skate/internal/config"
)

const systemPrompt = `You are a translator. Translate the following text to English.
If the text is already in English, return it unchanged.
Return ONLY the translated text, no explanations or extra formatting.
Preserve all markdown formatting, code blocks, links, and special characters exactly as they are.`

// Translator provides LLM-based translation for non-English content.
type Translator struct {
	client  *openai.Client
	model   string
	enabled bool
}

// New creates a Translator from config. Returns nil if translation is disabled.
func New(cfg config.TranslateConfig) *Translator {
	if !cfg.Enabled {
		return nil
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	var opts []option.RequestOption
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)

	return &Translator{
		client:  &client,
		model:   model,
		enabled: true,
	}
}

// Translate translates text to English if it appears non-English.
// Returns the original text if translation is disabled or text is already English.
func (t *Translator) Translate(text string) string {
	if t == nil || !t.enabled || text == "" {
		return text
	}

	// Quick heuristic: skip if text looks like it's already English/ASCII
	if isLikelyEnglish(text) {
		return text
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := t.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: t.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(text),
		},
		MaxCompletionTokens: openai.Int(4096),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "translate: %v\n", err)
		return text
	}

	if len(resp.Choices) > 0 {
		translated := strings.TrimSpace(resp.Choices[0].Message.Content)
		if translated != "" {
			return translated
		}
	}
	return text
}

// TranslateBlock translates a text block's title.
func (t *Translator) TranslateBlock(title string) string {
	return t.Translate(title)
}

// isLikelyEnglish checks if text is predominantly ASCII/Latin characters.
// This is a fast heuristic to skip API calls for English text.
func isLikelyEnglish(text string) bool {
	if len(text) == 0 {
		return true
	}

	// Strip markdown/code blocks for analysis
	clean := stripCodeBlocks(text)
	if len(clean) == 0 {
		return true
	}

	var total, ascii int
	for _, r := range clean {
		if unicode.IsLetter(r) {
			total++
			if r < 128 {
				ascii++
			}
		}
	}

	if total == 0 {
		return true
	}

	// If more than 80% of letters are ASCII, likely English
	return float64(ascii)/float64(total) > 0.8
}

func stripCodeBlocks(text string) string {
	var result strings.Builder
	inCodeBlock := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if !inCodeBlock {
			result.WriteString(line)
			result.WriteRune(' ')
		}
	}
	return result.String()
}

// FormatProviderInfo returns a human-readable string about the translation provider.
func FormatProviderInfo(cfg config.TranslateConfig) string {
	if !cfg.Enabled {
		return "disabled"
	}
	provider := cfg.Provider
	if provider == "" {
		provider = "openai"
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	base := ""
	if cfg.BaseURL != "" {
		base = fmt.Sprintf(" (%s)", cfg.BaseURL)
	}
	return fmt.Sprintf("%s/%s%s", provider, model, base)
}
