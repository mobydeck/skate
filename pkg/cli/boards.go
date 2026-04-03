package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var boardsCmd = &cobra.Command{
	Use:   "boards",
	Short: "List available boards",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		boardList, err := svc.ListBoards()
		if err != nil {
			return fmt.Errorf("listing boards: %w", err)
		}

		printStructured(cmd, boardList, func() {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tTYPE")
			for _, b := range boardList {
				boardType := "Private"
				if b.Type == "O" {
					boardType = "Open"
				}
				fmt.Fprintf(w, "%s\t%s %s\t%s\n", b.ID, b.Icon, b.Title, boardType)
			}
			w.Flush()
		})
		return nil
	},
}
