package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var editBlockCmd = &cobra.Command{
	Use:   "edit-block <TASK_ID> <BLOCK_ID> <TEXT>",
	Short: "Replace the text of a content block, comment, or heading",
	Long: `Edit a single block's text in place. Works for text, h1-h3, comment, and
checkbox blocks. For h1/h2/h3, include the markdown prefix in TEXT.

Find block IDs with 'skate task <ID> --json' (look at blocks[]).`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}
		cardID := args[0]
		blockID := args[1]
		text := args[2]

		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}
		if err := svc.UpdateBlockTitle(card.BoardID, blockID, text); err != nil {
			return fmt.Errorf("editing block: %w", err)
		}
		fmt.Printf("Edited block %s\n", blockID)
		return nil
	},
}
