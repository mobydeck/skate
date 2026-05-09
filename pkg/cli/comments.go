package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"skate/internal/boards"
	"skate/internal/translate"
)

var commentsCmd = &cobra.Command{
	Use:   "comments <TASK_ID>",
	Short: "View all comments for a task",
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

		blocks, err := svc.GetBlocks(card.BoardID, cardID)
		if err != nil {
			return fmt.Errorf("getting blocks: %w", err)
		}

		uc := boards.NewUserCache(svc)
		defer uc.Flush()
		var tr *translate.Translator
		if noTr, _ := cmd.Flags().GetBool("no-translate"); !noTr {
			tr = translate.New(cfg.Translate)
		}
		printMarkdown(cmd, boards.RenderComments(blocks, uc, tr))
		return nil
	},
}

func init() {
	commentsCmd.Flags().BoolP("no-translate", "T", false, "Skip translation even if enabled in config")
}
