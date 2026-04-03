package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"skate/internal/config"
)

var localInitCmd = &cobra.Command{
	Use:   "local-init",
	Short: "Initialize or update local project configuration (.skate.yaml)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		// Current board ID from existing local config
		currentBoardID := cfg.BoardID

		boardList, err := svc.ListBoards()
		if err != nil {
			return fmt.Errorf("listing boards: %w", err)
		}

		if len(boardList) == 0 {
			return fmt.Errorf("no boards found")
		}

		fmt.Println("Available boards:")
		defaultIdx := -1
		for i, b := range boardList {
			marker := "  "
			if b.ID == currentBoardID {
				marker = "* "
				defaultIdx = i
			}
			fmt.Printf("  %s%d. %s %s (ID: %s)\n", marker, i+1, b.Icon, b.Title, b.ID)
		}
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		defaultHint := ""
		if defaultIdx >= 0 {
			defaultHint = fmt.Sprintf("%d", defaultIdx+1)
		}
		input := prompt(reader, "Enter board number or ID", defaultHint)

		var boardID string
		// Try as number
		for i, b := range boardList {
			if fmt.Sprintf("%d", i+1) == input {
				boardID = b.ID
				break
			}
		}
		// Try as ID
		if boardID == "" {
			for _, b := range boardList {
				if b.ID == input {
					boardID = b.ID
					break
				}
			}
		}
		// Default selection
		if boardID == "" && input == defaultHint && defaultIdx >= 0 {
			boardID = boardList[defaultIdx].ID
		}
		if boardID == "" {
			return fmt.Errorf("invalid selection: %s", input)
		}

		// Find board name for confirmation
		var boardName string
		for _, b := range boardList {
			if b.ID == boardID {
				boardName = strings.TrimSpace(b.Icon + " " + b.Title)
				break
			}
		}

		if err := config.SaveLocal(".skate.yaml", boardID); err != nil {
			return fmt.Errorf("saving local config: %w", err)
		}
		fmt.Printf("Local config saved to .skate.yaml (board: %s)\n", boardName)
		return nil
	},
}
