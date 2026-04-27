package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var defaultActiveStatuses = []string{"not started", "in progress"}

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List board tasks sorted by priority",
	Long:  "By default shows only 'Not Started' and 'In Progress' tasks. Use --all to show all, or --status to filter by specific status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		flagBoard, _ := cmd.Flags().GetString("board")
		flagStatus, _ := cmd.Flags().GetString("status")
		flagAll, _ := cmd.Flags().GetBool("all")
		flagMine, _ := cmd.Flags().GetBool("mine")
		flagAllUsers, _ := cmd.Flags().GetBool("all-users")

		boardID, err := requireBoardID(cfg, flagBoard)
		if err != nil {
			return err
		}

		board, err := svc.GetBoard(boardID)
		if err != nil {
			return fmt.Errorf("getting board: %w", err)
		}

		cardList, err := svc.ListCards(boardID)
		if err != nil {
			return fmt.Errorf("listing cards: %w", err)
		}

		defs := boards.ParsePropertyDefs(board)
		resolved := boards.ResolveCards(cardList, defs)
		boards.SortByPriority(resolved)

		// Status filtering
		if flagStatus != "" {
			// Explicit status filter
			lower := strings.ToLower(flagStatus)
			var filtered []boards.ResolvedCard
			for _, rc := range resolved {
				if strings.ToLower(rc.Status) == lower {
					filtered = append(filtered, rc)
				}
			}
			resolved = filtered
		} else if !flagAll {
			// Default: only active statuses
			var filtered []boards.ResolvedCard
			for _, rc := range resolved {
				lower := strings.ToLower(rc.Status)
				for _, active := range defaultActiveStatuses {
					if lower == active {
						filtered = append(filtered, rc)
						break
					}
				}
			}
			resolved = filtered
		}

		// Assignee filtering
		onlyMine := cfg.OnlyMine
		if flagMine {
			onlyMine = true
		}
		if flagAllUsers {
			onlyMine = false
		}

		if onlyMine {
			if me, err := svc.GetMe(); err == nil {
				resolved = boards.FilterMine(resolved, me)
			}
		}

		if len(resolved) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		printStructured(cmd, resolved, func() {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPRIORITY\tASSIGNEE")
			for _, rc := range resolved {
				title := rc.Title
				runes := []rune(title)
				if len(runes) > 50 {
					title = string(runes[:47]) + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					rc.ID, title, rc.Status, rc.Priority, rc.Assignee)
			}
			w.Flush()
		})
		return nil
	},
}

func init() {
	tasksCmd.Flags().StringP("board", "b", "", "Board ID (default from .skate.yaml)")
	tasksCmd.Flags().StringP("status", "s", "", "Filter by specific status")
	tasksCmd.Flags().BoolP("all", "a", false, "Show all tasks regardless of status")
	tasksCmd.Flags().Bool("mine", false, "Show only my tasks")
	tasksCmd.Flags().Bool("all-users", false, "Show tasks for all users (overrides only_mine config)")
}
