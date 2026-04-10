package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"skate/internal/config"
)

var promptCmd = &cobra.Command{
	Use:   "prompt <agent>",
	Short: "Print an initial prompt that points the AI agent to the skate skill file and board",
	Long:  "Supported agents: claude-code (claude), cursor, codex",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agent := strings.ToLower(args[0])

		skillPath, err := skillPathForAgent(agent)
		if err != nil {
			return err
		}

		if _, err := os.Stat(skillPath); err != nil {
			return fmt.Errorf("skill file not found at %s — run 'skate setup %s' first", skillPath, agent)
		}

		fmt.Printf("Before starting work, read and follow rules in %s\n", skillPath)

		// Include board_id if available
		cfg, err := config.Load()
		if err == nil && cfg.BoardID != "" {
			fmt.Printf("Use board_id: %s for all skate MCP tools that require it.\n", cfg.BoardID)
		}

		return nil
	},
}

func skillPathForAgent(agent string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	switch agent {
	case "claude", "claude-code":
		return filepath.Join(home, ".claude", "skills", "skate", "SKILL.md"), nil
	case "cursor":
		return filepath.Join(home, ".cursor", "skills", "skate", "SKILL.md"), nil
	case "codex":
		return filepath.Join(home, ".codex", "skills", "skate", "SKILL.md"), nil
	default:
		return "", fmt.Errorf("unknown agent %q. Supported: claude-code, cursor, codex", agent)
	}
}
