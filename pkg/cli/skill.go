package cli

import (
	"github.com/spf13/cobra"

	"skate/internal/skill"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Print the Skate workflow guide (same content the skate_help MCP tool returns)",
	Long: `Print SKILL.md to stdout. Useful for iterating on the guide:

  skate skill | less
  diff <(skate skill) ~/.claude/skills/skate/SKILL.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printMarkdown(cmd, skill.Markdown())
		return nil
	},
}
