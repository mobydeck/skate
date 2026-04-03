package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var commentCmd = &cobra.Command{
	Use:   "comment <TASK_ID> <TEXT>",
	Short: "Add a comment to a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardID := args[0]
		text := args[1]

		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}

		now := time.Now().UnixMilli()
		block := &boards.Block{
			ParentID: cardID,
			BoardID:  card.BoardID,
			Type:     "comment",
			Title:    text,
			CreateAt: now,
			UpdateAt: now,
		}

		_, err = svc.CreateBlock(card.BoardID, []*boards.Block{block})
		if err != nil {
			return fmt.Errorf("creating comment: %w", err)
		}

		fmt.Println("Comment added.")
		return nil
	},
}
