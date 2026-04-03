package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"skate/internal/boards"
	"skate/internal/translate"
)

var taskCmd = &cobra.Command{
	Use:   "task <TASK_ID>",
	Short: "View task details in markdown",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardID := args[0]
		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}

		board, err := svc.GetBoard(card.BoardID)
		if err != nil {
			return fmt.Errorf("getting board: %w", err)
		}

		blocks, err := svc.GetBlocks(card.BoardID, cardID)
		if err != nil {
			return fmt.Errorf("getting blocks: %w", err)
		}

		summaries, err := svc.GetTimeSummary(card.BoardID, cardID)
		if err != nil {
			summaries = nil
		}

		printStructured(cmd, map[string]any{
			"card":      card,
			"board":     board,
			"blocks":    blocks,
			"summaries": summaries,
		}, func() {
			uc := boards.NewUserCache(svc)
			defer uc.Flush()
			tr := translate.New(cfg.Translate)
			md := boards.RenderCardMarkdown(card, board, blocks, summaries, uc, tr)
			fmt.Print(md)
		})
		return nil
	},
}
