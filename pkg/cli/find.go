package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var findCmd = &cobra.Command{
	Use:   "find <QUERY>",
	Short: "Search tasks by title and content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		query := strings.ToLower(args[0])
		flagBoard, _ := cmd.Flags().GetString("board")

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

		type matchResult struct {
			card    boards.ResolvedCard
			matchIn string // "title" or "content"
			snippet string
		}

		var titleMatches, contentMatches []matchResult

		for _, rc := range resolved {
			if strings.Contains(strings.ToLower(rc.Title), query) {
				titleMatches = append(titleMatches, matchResult{card: rc, matchIn: "title"})
				continue
			}

			// Search content blocks and comments
			blocks, err := svc.GetBlocks(boardID, rc.ID)
			if err != nil {
				continue
			}
			for _, b := range blocks {
				lower := strings.ToLower(b.Title)
				if strings.Contains(lower, query) {
					snippet := b.Title
					runes := []rune(snippet)
					if len(runes) > 80 {
						snippet = string(runes[:77]) + "..."
					}
					contentMatches = append(contentMatches, matchResult{card: rc, matchIn: b.Type, snippet: snippet})
					break
				}
			}
		}

		all := append(titleMatches, contentMatches...)
		if len(all) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		printStructured(cmd, all, func() {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, m := range all {
				icon := "  "
				if m.card.Status != "" {
					icon = statusIcon(m.card.Status)
				}
				fmt.Fprintf(w, "%s %s\t%s\n", icon, m.card.ID, m.card.Title)
				fmt.Fprintf(w, "  Status: %s", m.card.Status)
				if m.card.Priority != "" {
					fmt.Fprintf(w, "  |  Priority: %s", m.card.Priority)
				}
				if m.card.Assignee != "" {
					fmt.Fprintf(w, "  |  Assignee: %s", boards.AtPrefix(m.card.Assignee))
				}
				fmt.Fprintln(w)
				if m.snippet != "" {
					fmt.Fprintf(w, "  Match in %s: %s\n", m.matchIn, m.snippet)
				}
				fmt.Fprintln(w)
			}
			w.Flush()
		})
		return nil
	},
}

func statusIcon(status string) string {
	lower := strings.ToLower(status)
	switch {
	case strings.Contains(lower, "not started"):
		return "○"
	case strings.Contains(lower, "in progress"):
		return "◐"
	case strings.Contains(lower, "completed"), strings.Contains(lower, "done"):
		return "●"
	case strings.Contains(lower, "blocked"):
		return "✕"
	default:
		return "○"
	}
}

func init() {
	findCmd.Flags().StringP("board", "b", "", "Board ID (default from .skate.yaml)")
}
