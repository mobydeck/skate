package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var updateCmd = &cobra.Command{
	Use:   "update <TASK_ID>",
	Short: "Update task fields (title, status, priority, assignee, icon)",
	Long: `Update one or more fields on an existing task. Only the flags you pass are changed.

Examples:
  skate update <ID> --title "New title"
  skate update <ID> --priority "High"
  skate update <ID> --assignee arthur            # username or user ID
  skate update <ID> --status "In Progress" -t    # also start timer

This is the general form of 'skate update-status', which remains as a shortcut.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}
		cardID := args[0]

		title, _ := cmd.Flags().GetString("title")
		icon, _ := cmd.Flags().GetString("icon")
		status, _ := cmd.Flags().GetString("status")
		priority, _ := cmd.Flags().GetString("priority")
		assignee, _ := cmd.Flags().GetString("assignee")
		startTimer, _ := cmd.Flags().GetBool("timer")

		if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("icon") &&
			!cmd.Flags().Changed("status") && !cmd.Flags().Changed("priority") &&
			!cmd.Flags().Changed("assignee") && !startTimer {
			return fmt.Errorf("nothing to update — pass at least one of --title --icon --status --priority --assignee --timer")
		}

		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}
		board, err := svc.GetBoard(card.BoardID)
		if err != nil {
			return fmt.Errorf("getting board: %w", err)
		}
		defs := boards.ParsePropertyDefs(board)

		patch := &boards.CardPatch{}
		if cmd.Flags().Changed("title") {
			patch.Title = &title
		}
		if cmd.Flags().Changed("icon") {
			patch.Icon = &icon
		}

		props, err := buildPropertyUpdates(svc, cfg.TeamID, defs, status, priority, assignee)
		if err != nil {
			return err
		}
		if len(props) > 0 {
			patch.UpdatedProperties = props
		}

		var changes []string
		if patch.Title != nil || patch.Icon != nil || len(patch.UpdatedProperties) > 0 {
			if _, err := svc.PatchCard(cardID, patch); err != nil {
				return fmt.Errorf("updating task: %w", err)
			}
			if patch.Title != nil {
				changes = append(changes, fmt.Sprintf("title=%q", *patch.Title))
			}
			if patch.Icon != nil {
				changes = append(changes, fmt.Sprintf("icon=%q", *patch.Icon))
			}
			if status != "" {
				changes = append(changes, fmt.Sprintf("status=%q", status))
			}
			if priority != "" {
				changes = append(changes, fmt.Sprintf("priority=%q", priority))
			}
			if assignee != "" {
				changes = append(changes, fmt.Sprintf("assignee=%q", assignee))
			}
		}
		if len(changes) > 0 {
			fmt.Printf("Updated %s: %s\n", cardID, joinChanges(changes))
		}

		if startTimer {
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

// buildPropertyUpdates resolves status/priority/assignee values to their property
// IDs. Each input is optional — empty values are skipped.
func buildPropertyUpdates(svc *boards.Service, teamID string, defs []boards.PropertyDef, status, priority, assignee string) (map[string]any, error) {
	props := map[string]any{}

	if status != "" {
		p := boards.FindPropertyByName(defs, "Status")
		if p == nil {
			return nil, fmt.Errorf("board has no Status property")
		}
		o := boards.FindOptionByValue(p, status)
		if o == nil {
			return nil, fmt.Errorf("invalid status %q", status)
		}
		props[p.ID] = o.ID
	}

	if priority != "" {
		p := boards.FindPropertyByName(defs, "Priority")
		if p == nil {
			return nil, fmt.Errorf("board has no Priority property")
		}
		o := boards.FindOptionByValue(p, priority)
		if o == nil {
			return nil, fmt.Errorf("invalid priority %q", priority)
		}
		props[p.ID] = o.ID
	}

	if assignee != "" {
		p := boards.FindPropertyByName(defs, "Assignee")
		if p == nil {
			p = boards.FindPropertyByName(defs, "Assignees")
		}
		if p == nil {
			return nil, fmt.Errorf("board has no Assignee property")
		}
		resolved, err := svc.ResolveUserRef(teamID, assignee)
		if err != nil {
			return nil, fmt.Errorf("resolving assignee: %w", err)
		}
		props[p.ID] = resolved
	}

	return props, nil
}

func joinChanges(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

func init() {
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().String("icon", "", "New icon (emoji)")
	updateCmd.Flags().StringP("status", "s", "", "New status (use 'skate statuses' to see options)")
	updateCmd.Flags().StringP("priority", "p", "", "New priority")
	updateCmd.Flags().StringP("assignee", "a", "", "Assignee — username or user ID")
	updateCmd.Flags().BoolP("timer", "t", false, "Start timer after updating")
}
