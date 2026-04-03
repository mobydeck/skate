package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"skate/internal/boards"
)

var attachCmd = &cobra.Command{
	Use:   "attach <TASK_ID> <FILE_PATH>",
	Short: "Upload and attach a file to a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardID := args[0]
		filePath := args[1]

		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}

		fileID, err := svc.UploadFile(cfg.TeamID, card.BoardID, filePath)
		if err != nil {
			return fmt.Errorf("uploading file: %w", err)
		}

		now := time.Now().UnixMilli()
		block := &boards.Block{
			ParentID: cardID,
			BoardID:  card.BoardID,
			Type:     "attachment",
			Title:    filePath,
			Fields:   map[string]interface{}{"fileId": fileID},
			CreateAt: now,
			UpdateAt: now,
		}

		_, err = svc.CreateBlock(card.BoardID, []*boards.Block{block})
		if err != nil {
			return fmt.Errorf("creating attachment block: %w", err)
		}

		fmt.Printf("File attached (fileId: %s)\n", fileID)
		return nil
	},
}
