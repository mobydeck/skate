package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"skate/internal/boards"
	"skate/internal/translate"
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show the highest-priority Not Started task (the one to start next)",
	Long: `Pick the top-of-queue task: status 'Not Started', sorted by priority.
Renders the same markdown as 'skate task <ID>' so you can start work immediately.

Use --mine to limit to tasks assigned to you.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		flagBoard, _ := cmd.Flags().GetString("board")
		flagMine, _ := cmd.Flags().GetBool("mine")

		boardID, err := requireBoardID(cfg, flagBoard)
		if err != nil {
			return err
		}

		board, err := svc.GetBoard(boardID)
		if err != nil {
			return fmt.Errorf("getting board: %w", err)
		}
		cards, err := svc.ListCards(boardID)
		if err != nil {
			return fmt.Errorf("listing cards: %w", err)
		}

		uc := boards.NewUserCache(svc)
		defer uc.Flush()
		defs := boards.ParsePropertyDefs(board)
		resolved := boards.ResolveCards(cards, defs, uc)
		boards.SortByPriority(resolved)

		var queue []boards.ResolvedCard
		for _, rc := range resolved {
			if strings.EqualFold(rc.Status, "Not Started") {
				queue = append(queue, rc)
			}
		}

		if flagMine {
			if me, err := svc.GetMe(); err == nil {
				queue = boards.FilterMine(queue, me)
			}
		}

		if len(queue) == 0 {
			fmt.Println("No tasks ready to start.")
			return nil
		}

		top := queue[0]
		card, err := svc.GetCard(top.ID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}
		blocks, err := svc.GetBlocks(boardID, top.ID)
		if err != nil {
			return fmt.Errorf("getting blocks: %w", err)
		}
		summaries, _ := svc.GetTimeSummary(boardID, top.ID)

		printStructured(cmd, map[string]any{
			"card":      card,
			"board":     board,
			"blocks":    blocks,
			"summaries": summaries,
		}, func() {
			var tr *translate.Translator
			if noTr, _ := cmd.Flags().GetBool("no-translate"); !noTr {
				tr = translate.New(cfg.Translate)
			}
			md := boards.RenderCardMarkdown(card, board, blocks, summaries, uc, tr, defaultMaxComments)
			printMarkdown(cmd, md)
		})
		return nil
	},
}

func init() {
	nextCmd.Flags().StringP("board", "b", "", "Board ID (default from .skate.yaml)")
	nextCmd.Flags().Bool("mine", false, "Only consider tasks assigned to you")
	nextCmd.Flags().BoolP("no-translate", "T", false, "Skip translation even if enabled in config")
}
