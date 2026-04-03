package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <BOARD_ID> <FILE_ID>",
	Short: "Download a file from a board",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		boardID := args[0]
		fileID := args[1]
		output, _ := cmd.Flags().GetString("output")

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
}
