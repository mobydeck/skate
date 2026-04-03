package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <FILE_ID>",
	Short: "Download a file from a board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		fileID := args[0]
		output, _ := cmd.Flags().GetString("output")
		flagBoard, _ := cmd.Flags().GetString("board")

		boardID, err := requireBoardID(cfg, flagBoard)
		if err != nil {
			return err
		}

		data, err := svc.DownloadFile(cfg.TeamID, boardID, fileID)
		if err != nil {
			return fmt.Errorf("downloading file: %w", err)
		}

		if output == "" || output == "-" {
			os.Stdout.Write(data)
		} else {
			if err := os.WriteFile(output, data, 0o644); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}
			fmt.Printf("Saved to %s (%d bytes)\n", output, len(data))
		}
		return nil
	},
}

func init() {
	downloadCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	downloadCmd.Flags().StringP("board", "b", "", "Board ID (default from .skate.yaml)")
}
