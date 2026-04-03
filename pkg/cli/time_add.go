package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var timeAddCmd = &cobra.Command{
	Use:   "time-add <TASK_ID> <HH:MM>",
	Short: "Add manual time to a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardID := args[0]
		duration := args[1]

		// Parse HH:MM
		parts := strings.Split(duration, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid duration format, use HH:MM")
		}
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid hours: %w", err)
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid minutes: %w", err)
		}
		durationSeconds := int64(hours*3600 + minutes*60)

		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}

		// Parse date or default to today noon
		dateStr, _ := cmd.Flags().GetString("date")
		var dateMs int64
		if dateStr != "" {
			t, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
			dateMs = t.Add(12 * time.Hour).UnixMilli() // noon to avoid timezone shift
		} else {
			now := time.Now()
			dateMs = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location()).UnixMilli()
		}

		notes, _ := cmd.Flags().GetString("notes")

		entry, err := svc.AddManualTime(card.BoardID, cardID, durationSeconds, dateMs, notes)
		if err != nil {
			fmt.Println("Time tracking is not available on this Mattermost instance.")
			return nil
		}

		fmt.Printf("Added %s to %s\n", entry.DurationDisplay, card.Title)
		return nil
	},
}

func init() {
	timeAddCmd.Flags().StringP("date", "d", "", "Date (YYYY-MM-DD, default: today)")
	timeAddCmd.Flags().StringP("notes", "n", "", "Notes for the time entry")
}
