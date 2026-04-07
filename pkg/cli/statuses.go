package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var statusesCmd = &cobra.Command{
	Use:   "statuses",
	Short: "List available statuses for the board",
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
		statusProp := boards.FindPropertyByName(defs, "Status")
		if statusProp == nil {
			return fmt.Errorf("board has no Status property")
		}

		var values []string
		for _, o := range statusProp.Options {
			values = append(values, o.Value)
		}

		printStructured(cmd, values, func() {
			fmt.Println(strings.Join(values, ", "))
		})
		return nil
	},
}

func init() {
	statusesCmd.Flags().StringP("board", "b", "", "Board ID (default from .skate.yaml)")
}
