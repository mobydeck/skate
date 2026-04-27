package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var updateStatusCmd = &cobra.Command{
	Use:          "update-status <TASK_ID> [TASK_ID ...] <STATUS>",
	Short:        "Update one or more task statuses",
	SilenceUsage: true,
	Long: `Set the same status on one or more tasks. The last argument is the status;
all preceding arguments are task IDs.

Examples:
  skate update-status <ID> "In Progress"
  skate update-status <ID> "In Progress" -t          # also start timer (single-task only)
  skate update-status <ID1> <ID2> <ID3> "Completed"  # batch close

On batch errors the command continues and reports per-task failures, exiting
non-zero if any failed.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardIDs := args[:len(args)-1]
		statusValue := args[len(args)-1]
		startTimer, _ := cmd.Flags().GetBool("timer")

		if len(cardIDs) > 1 && startTimer {
			return fmt.Errorf("--timer is only supported for single-task updates (got %d task IDs)", len(cardIDs))
		}

		// Resolve status once on the first card's board (all tasks must share a board today).
		first, err := svc.GetCard(cardIDs[0])
		if err != nil {
			return fmt.Errorf("getting card %s: %w", cardIDs[0], err)
		}
		board, err := svc.GetBoard(first.BoardID)
		if err != nil {
			return fmt.Errorf("getting board: %w", err)
		}
		defs := boards.ParsePropertyDefs(board)
		statusProp := boards.FindPropertyByName(defs, "Status")
		if statusProp == nil {
			return fmt.Errorf("board has no Status property")
		}
		option := boards.FindOptionByValue(statusProp, statusValue)
		if option == nil {
			var options []string
			for _, o := range statusProp.Options {
				options = append(options, o.Value)
			}
			return fmt.Errorf("invalid status %q. Available: %v", statusValue, options)
		}
		patch := &boards.CardPatch{
			UpdatedProperties: map[string]any{statusProp.ID: option.ID},
		}

		var failed []string
		for _, id := range cardIDs {
			if _, err := svc.PatchCard(id, patch); err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", id, err))
				continue
			}
			fmt.Printf("%s → %q\n", id, option.Value)
		}

		if startTimer {
			cardID := cardIDs[0]
			resp, err := svc.StartTimer(first.BoardID, cardID)
			if err != nil {
				fmt.Println("Time tracking is not available on this Mattermost instance.")
			} else {
				fmt.Printf("Timer started on: %s\n", first.Title)
				if resp.StoppedEntry != nil {
					fmt.Printf("Auto-stopped previous timer on: %s (%s)\n", resp.StoppedEntry.CardName, resp.StoppedEntry.DurationDisplay)
				}
			}
		}

		if len(failed) > 0 {
			return fmt.Errorf("%d of %d failed:\n  %s", len(failed), len(cardIDs), strings.Join(failed, "\n  "))
		}
		return nil
	},
}

func init() {
	updateStatusCmd.Flags().BoolP("timer", "t", false, "Start timer after updating status")
}
