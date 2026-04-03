package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var taskFilesCmd = &cobra.Command{
	Use:   "task-files <TASK_ID>",
	Short: "List files attached to a task",
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

		blocks, err := svc.GetBlocks(card.BoardID, cardID)
		if err != nil {
			return fmt.Errorf("getting blocks: %w", err)
		}

		var fileBlocks []*boards.Block
		for _, b := range blocks {
			if b.Type == "image" || b.Type == "attachment" {
				fileBlocks = append(fileBlocks, b)
			}
		}

		if len(fileBlocks) == 0 {
			fmt.Println("No files attached.")
			return nil
		}

		printStructured(cmd, fileBlocks, func() {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TYPE\tFILE_ID\tTITLE")
			for _, b := range fileBlocks {
				fileID := ""
				if fid, ok := b.Fields["fileId"]; ok {
					fileID = fmt.Sprintf("%v", fid)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", b.Type, fileID, b.Title)
			}
			w.Flush()
		})
		return nil
	},
}
