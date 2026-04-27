package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteBlockCmd = &cobra.Command{
	Use:   "delete-block <TASK_ID> <BLOCK_ID>",
	Short: "Delete a content block, comment, or attachment from a task",
	Long: `Delete a single block (content, comment, or attachment) by ID.

Find block IDs with 'skate task <ID> --json' (look at the blocks array) or
'skate task-files <ID>' for attachments. The TASK_ID is needed so the card's
content order can be cleaned up.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}
		cardID := args[0]
		blockID := args[1]

		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}
		if err := svc.DeleteBlock(card.BoardID, cardID, blockID); err != nil {
			return fmt.Errorf("deleting block: %w", err)
		}
		fmt.Printf("Deleted block %s\n", blockID)
		return nil
	},
}
