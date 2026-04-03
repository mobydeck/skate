package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const timeTrackingUnavailable = "Time tracking is not available on this Mattermost instance."

var timerStartCmd = &cobra.Command{
	Use:   "timer-start <TASK_ID>",
	Short: "Start timer on a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardID := args[0]
		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}

		resp, err := svc.StartTimer(card.BoardID, cardID)
		if err != nil {
			fmt.Println(timeTrackingUnavailable)
			return nil
		}

		fmt.Printf("Timer started on: %s\n", card.Title)
		if resp.StoppedEntry != nil {
			fmt.Printf("Auto-stopped previous timer on: %s (%s)\n",
				resp.StoppedEntry.CardName, resp.StoppedEntry.DurationDisplay)
		}
		return nil
	},
}

var timerStopCmd = &cobra.Command{
	Use:   "timer-stop",
	Short: "Stop the running timer",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		notes, _ := cmd.Flags().GetString("notes")

		timer, err := svc.GetRunningTimer()
		if err != nil {
			fmt.Println(timeTrackingUnavailable)
			return nil
		}
		if timer == nil {
			fmt.Println("No timer is running.")
			return nil
		}

		stopped, err := svc.StopTimer(timer.ID, notes)
		if err != nil {
			fmt.Println(timeTrackingUnavailable)
			return nil
		}

		fmt.Printf("Timer stopped: %s — %s\n", stopped.CardName, stopped.DurationDisplay)
		return nil
	},
}

func init() {
	timerStopCmd.Flags().StringP("notes", "n", "", "Notes for the time entry")
}
