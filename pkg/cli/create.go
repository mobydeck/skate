package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var createCmd = &cobra.Command{
	Use:   "create <TITLE>",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(1),
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

		board, err := svc.GetBoard(boardID)
		if err != nil {
			return fmt.Errorf("getting board: %w", err)
		}

		defs := boards.ParsePropertyDefs(board)
		props := make(map[string]any)

		// Set status if provided
		if status, _ := cmd.Flags().GetString("status"); status != "" {
			p := boards.FindPropertyByName(defs, "Status")
			if p != nil {
				if o := boards.FindOptionByValue(p, status); o != nil {
					props[p.ID] = o.ID
				}
			}
		}

		// Set priority if provided
		if priority, _ := cmd.Flags().GetString("priority"); priority != "" {
			p := boards.FindPropertyByName(defs, "Priority")
			if p != nil {
				if o := boards.FindOptionByValue(p, priority); o != nil {
					props[p.ID] = o.ID
				}
			}
		}

		// Set assignee if provided. Accepts either a user ID or a username
		// (resolved against the team listing).
		if assignee, _ := cmd.Flags().GetString("assignee"); assignee != "" {
			p := boards.FindPropertyByName(defs, "Assignee")
			if p == nil {
				p = boards.FindPropertyByName(defs, "Assignees")
			}
			if p != nil {
				resolved, err := svc.ResolveUserRef(cfg.TeamID, assignee)
				if err != nil {
					return fmt.Errorf("resolving assignee: %w", err)
				}
				props[p.ID] = resolved
			}
		}

		now := time.Now().UnixMilli()
		card := &boards.Card{
			BoardID:    boardID,
			Title:      args[0],
			Properties: props,
			CreateAt:   now,
			UpdateAt:   now,
		}

		created, err := svc.CreateCard(boardID, card)
		if err != nil {
			return fmt.Errorf("creating card: %w", err)
		}

		// Add description as text block if provided
		if desc, _ := cmd.Flags().GetString("description"); desc != "" {
			block := &boards.Block{
				ParentID: created.ID,
				BoardID:  boardID,
				Type:     "text",
				Title:    desc,
				CreateAt: now,
				UpdateAt: now,
			}
			if _, err := svc.CreateContentBlock(boardID, created.ID, block); err != nil {
				fmt.Printf("Warning: card created but failed to add description: %v\n", err)
			}
		}

		fmt.Printf("Created task: %s (ID: %s)\n", created.Title, created.ID)
		return nil
	},
}

func init() {
	createCmd.Flags().StringP("board", "b", "", "Board ID")
	createCmd.Flags().StringP("status", "s", "", "Initial status")
	createCmd.Flags().StringP("priority", "p", "", "Priority")
	createCmd.Flags().StringP("assignee", "a", "", "Assignee user ID")
	createCmd.Flags().StringP("description", "d", "", "Task description")
}
