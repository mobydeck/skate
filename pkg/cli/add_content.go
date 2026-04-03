package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"skate/internal/boards"
	"skate/internal/config"
)

var validBlockTypes = map[string]bool{
	"text":     true,
	"h1":       true,
	"h2":       true,
	"h3":       true,
	"divider":  true,
	"checkbox": true,
	"image":    true,
}

var addContentCmd = &cobra.Command{
	Use:   "add-content <TASK_ID> [TEXT]",
	Short: "Add a content block to a task's description",
	Long:  "Supported types: text (default), h1, h2, h3, divider, checkbox, image.\nFor image type, TEXT is the file path to upload.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		cardID := args[0]
		blockType, _ := cmd.Flags().GetString("type")
		if !validBlockTypes[blockType] {
			return fmt.Errorf("invalid type %q. Supported: text, h1, h2, h3, divider, checkbox, image", blockType)
		}

		text := ""
		if len(args) > 1 {
			text = args[1]
		}

		if blockType == "image" {
			if text == "" {
				return fmt.Errorf("file path is required for image blocks")
			}
			return addImageBlock(cfg, svc, cardID, text)
		}

		if blockType != "divider" && text == "" {
			return fmt.Errorf("text is required for %s blocks", blockType)
		}

		// Convert heading types to text blocks with markdown prefix
		actualType := blockType
		switch blockType {
		case "h1":
			actualType = "text"
			text = "# " + text
		case "h2":
			actualType = "text"
			text = "## " + text
		case "h3":
			actualType = "text"
			text = "### " + text
		}

		_ = cfg
		card, err := svc.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("getting card: %w", err)
		}

		now := time.Now().UnixMilli()
		block := &boards.Block{
			ParentID: cardID,
			BoardID:  card.BoardID,
			Type:     actualType,
			Title:    text,
			CreateAt: now,
			UpdateAt: now,
		}

		_, err = svc.CreateContentBlock(card.BoardID, cardID, block)
		if err != nil {
			return fmt.Errorf("adding content block: %w", err)
		}

		fmt.Printf("Content block added (%s).\n", blockType)
		return nil
	},
}

func addImageBlock(cfg *config.Config, svc *boards.Service, cardID, filePath string) error {
	card, err := svc.GetCard(cardID)
	if err != nil {
		return fmt.Errorf("getting card: %w", err)
	}

	fileID, err := svc.UploadFile(cfg.TeamID, card.BoardID, filePath)
	if err != nil {
		return fmt.Errorf("uploading image: %w", err)
	}

	now := time.Now().UnixMilli()
	block := &boards.Block{
		ParentID: cardID,
		BoardID:  card.BoardID,
		Type:     "image",
		Title:    filepath.Base(filePath),
		Fields:   map[string]interface{}{"fileId": fileID},
		CreateAt: now,
		UpdateAt: now,
	}

	_, err = svc.CreateContentBlock(card.BoardID, cardID, block)
	if err != nil {
		return fmt.Errorf("adding image block: %w", err)
	}

	fmt.Printf("Image content block added (fileId: %s).\n", fileID)
	return nil
}

func init() {
	addContentCmd.Flags().StringP("type", "t", "text", "Block type: text, h1, h2, h3, divider, checkbox, image")
}
