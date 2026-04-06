package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var updateStatusCmd = &cobra.Command{
	Use:   "update-status <TASK_ID> <STATUS>",
	Short: "Update task status",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardID := args[0]
		statusValue := args[1]

		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}

		board, err := svc.GetBoard(card.BoardID)
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
			UpdatedProperties: map[string]interface{}{
				statusProp.ID: option.ID,
			},
		}

		_, err = svc.PatchCard(cardID, patch)
		if err != nil {
			return fmt.Errorf("updating status: %w", err)
		}

		fmt.Printf("Status updated to %q\n", option.Value)

		if startTimer, _ := cmd.Flags().GetBool("timer"); startTimer {
			resp, err := svc.StartTimer(card.BoardID, cardID)
			if err != nil {
				fmt.Println("Time tracking is not available on this Mattermost instance.")
			} else {
				fmt.Printf("Timer started on: %s\n", card.Title)
				if resp.StoppedEntry != nil {
					fmt.Printf("Auto-stopped previous timer on: %s (%s)\n", resp.StoppedEntry.CardName, resp.StoppedEntry.DurationDisplay)
				}
			}
		}

		return nil
	},
}

func init() {
	updateStatusCmd.Flags().BoolP("timer", "t", false, "Start timer after updating status")
}
