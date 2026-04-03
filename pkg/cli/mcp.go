package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"skate/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Start MCP server (stdio transport)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := mcp.RunServer(); err != nil {
			return fmt.Errorf("MCP server error: %w", err)
		}
		return nil
	},
}
