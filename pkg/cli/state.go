package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Show your current working state (running timer + your in-progress tasks)",
	Long: `One-shot snapshot for resuming a session: who you are, the timer that's
running (if any), and the In Progress tasks assigned to you on the current board.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		flagBoard, _ := cmd.Flags().GetString("board")
		boardID, err := requireBoardID(cfg, flagBoard)
		if err != nil {
			return err
		}

		me, _ := svc.GetMe()
		timer, _ := svc.GetRunningTimer()

		board, err := svc.GetBoard(boardID)
		if err != nil {
			return fmt.Errorf("getting board: %w", err)
		}
		cards, err := svc.ListCards(boardID)
		if err != nil {
			return fmt.Errorf("listing cards: %w", err)
		}
		defs := boards.ParsePropertyDefs(board)
		resolved := boards.ResolveCards(cards, defs)
		boards.SortByPriority(resolved)

		var inProgress []boards.ResolvedCard
		for _, rc := range resolved {
			if strings.EqualFold(rc.Status, "In Progress") {
				inProgress = append(inProgress, rc)
			}
		}
		if me != nil {
			inProgress = boards.FilterMine(inProgress, me)
		}

		printStructured(cmd, map[string]any{
			"me":          me,
			"timer":       timer,
			"in_progress": inProgress,
			"board_id":    boardID,
		}, func() {
			fmt.Print(renderStateText(me, timer, inProgress))
		})
		return nil
	},
}

func renderStateText(me *boards.User, timer *boards.TimeEntry, inProgress []boards.ResolvedCard) string {
	var sb strings.Builder

	if me != nil {
		fmt.Fprintf(&sb, "User: @%s (%s)\n", me.Username, me.ID)
	}

	if timer != nil {
		elapsed := boards.FormatDuration((time.Now().UnixMilli() - timer.StartTime) / 1000)
		fmt.Fprintf(&sb, "Running timer: %s (%s elapsed) [card: %s]\n", timer.CardName, elapsed, timer.CardID)
	} else {
		sb.WriteString("Running timer: none\n")
	}

	sb.WriteString("\nIn Progress (yours):\n")
	if len(inProgress) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, rc := range inProgress {
			fmt.Fprintf(&sb, "  - [%s] %s | Priority: %s\n", rc.ID, rc.Title, rc.Priority)
		}
	}
	return sb.String()
}

func init() {
	stateCmd.Flags().StringP("board", "b", "", "Board ID (default from .skate.yaml)")
}
