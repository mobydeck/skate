package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"skate/internal/config"
	"skate/internal/translate"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		printStructured(cmd, cfg, func() {
			fmt.Printf("mattermost_url: %s\n", cfg.MattermostURL)
			fmt.Printf("team_id:        %s\n", cfg.TeamID)
			if cfg.BoardID != "" {
				fmt.Printf("board_id:       %s\n", cfg.BoardID)
			}
			fmt.Printf("only_mine:      %v\n", cfg.OnlyMine)
			fmt.Printf("mentions:       %v\n", cfg.MentionsEnabled())
			fmt.Printf("translate:      %s\n", translate.FormatProviderInfo(cfg.Translate))
		})
		return nil
	},
}
