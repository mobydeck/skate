package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"skate/internal/boards"
	"skate/internal/client"
	"skate/internal/config"
	"skate/internal/translate"
	"skate/internal/version"
)

func RunServer() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	c := client.New(config.BaseURL(cfg), cfg.Token)
	svc := boards.NewService(c, cfg.TeamID)

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "skate",
		Version: version.Version,
	}, nil)

	if err := registerTools(server, svc, cfg); err != nil {
		return fmt.Errorf("registering tools: %w", err)
	}

	return server.Run(context.Background(), &mcpsdk.StdioTransport{})
}

func registerTools(s *mcpsdk.Server, svc *boards.Service, cfg *config.Config) error {
	// skate_boards
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_boards",
		Description: "List available Mattermost boards for the current user",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		boardList, err := svc.ListBoards()
		if err != nil {
			return errResult(err), nil, nil
		}
		var lines []string
		for _, b := range boardList {
			lines = append(lines, fmt.Sprintf("- %s %s (ID: %s)", b.Icon, b.Title, b.ID))
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_tasks
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_tasks",
		Description: "List board tasks sorted by priority. By default shows only 'Not Started' and 'In Progress' tasks. Set show_all=true to show all statuses.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"board_id": map[string]any{"type": "string", "description": "Board ID (optional, uses default from config)"},
				"status":   map[string]any{"type": "string", "description": "Filter by specific status (optional)"},
				"show_all": map[string]any{"type": "boolean", "description": "Show all tasks regardless of status (default: false)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		boardID := getStr(input, "board_id")
		if boardID == "" {
			boardID = cfg.BoardID
		}
		if boardID == "" {
			return errResult(fmt.Errorf("board_id required")), nil, nil
		}

		board, err := svc.GetBoard(boardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		cards, err := svc.ListCards(boardID)
		if err != nil {
			return errResult(err), nil, nil
		}

		defs := boards.ParsePropertyDefs(board)
		resolved := boards.ResolveCards(cards, defs)
		boards.SortByPriority(resolved)

		statusFilter := getStr(input, "status")
		showAll, _ := input["show_all"].(bool)

		if statusFilter != "" {
			lower := strings.ToLower(statusFilter)
			var filtered []boards.ResolvedCard
			for _, rc := range resolved {
				if strings.ToLower(rc.Status) == lower {
					filtered = append(filtered, rc)
				}
			}
			resolved = filtered
		} else if !showAll {
			activeStatuses := map[string]bool{"not started": true, "in progress": true}
			var filtered []boards.ResolvedCard
			for _, rc := range resolved {
				if activeStatuses[strings.ToLower(rc.Status)] {
					filtered = append(filtered, rc)
				}
			}
			resolved = filtered
		}

		var lines []string
		for _, rc := range resolved {
			lines = append(lines, fmt.Sprintf("- [%s] %s | Status: %s | Priority: %s | Assignee: %s",
				rc.ID, rc.Title, rc.Status, rc.Priority, rc.Assignee))
		}
		if len(lines) == 0 {
			return textResult("No tasks found."), nil, nil
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_task
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_task",
		Description: "Get full task details rendered as markdown, including properties, description, comments, attachments, and time tracking.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id"},
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "Task/card ID"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		board, err := svc.GetBoard(card.BoardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		blocks, err := svc.GetBlocks(card.BoardID, cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		summaries, _ := svc.GetTimeSummary(card.BoardID, cardID)
		uc := boards.NewUserCache(svc)
		defer uc.Flush()
		tr := translate.New(cfg.Translate)
		md := boards.RenderCardMarkdown(card, board, blocks, summaries, uc, tr)
		return textResult(md), nil, nil
	})

	// skate_update_status
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_update_status",
		Description: "Change a task's status (e.g., 'Not Started', 'In Progress', 'Done')",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id", "status"},
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "Task/card ID"},
				"status":  map[string]any{"type": "string", "description": "New status value"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		status := getStr(input, "status")

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		board, err := svc.GetBoard(card.BoardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		defs := boards.ParsePropertyDefs(board)
		statusProp := boards.FindPropertyByName(defs, "Status")
		if statusProp == nil {
			return errResult(fmt.Errorf("board has no Status property")), nil, nil
		}
		option := boards.FindOptionByValue(statusProp, status)
		if option == nil {
			return errResult(fmt.Errorf("invalid status %q", status)), nil, nil
		}
		patch := &boards.CardPatch{UpdatedProperties: map[string]interface{}{statusProp.ID: option.ID}}
		if _, err := svc.PatchCard(cardID, patch); err != nil {
			return errResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Status updated to %q", option.Value)), nil, nil
	})

	// skate_create_task
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_create_task",
		Description: "Create a new task on the board",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"title"},
			"properties": map[string]any{
				"title":       map[string]any{"type": "string", "description": "Task title"},
				"board_id":    map[string]any{"type": "string", "description": "Board ID (optional)"},
				"status":      map[string]any{"type": "string", "description": "Initial status"},
				"priority":    map[string]any{"type": "string", "description": "Priority level"},
				"description": map[string]any{"type": "string", "description": "Task description"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		boardID := getStr(input, "board_id")
		if boardID == "" {
			boardID = cfg.BoardID
		}
		if boardID == "" {
			return errResult(fmt.Errorf("board_id required")), nil, nil
		}

		board, err := svc.GetBoard(boardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		defs := boards.ParsePropertyDefs(board)
		props := make(map[string]interface{})

		if s := getStr(input, "status"); s != "" {
			if p := boards.FindPropertyByName(defs, "Status"); p != nil {
				if o := boards.FindOptionByValue(p, s); o != nil {
					props[p.ID] = o.ID
				}
			}
		}
		if s := getStr(input, "priority"); s != "" {
			if p := boards.FindPropertyByName(defs, "Priority"); p != nil {
				if o := boards.FindOptionByValue(p, s); o != nil {
					props[p.ID] = o.ID
				}
			}
		}

		now := time.Now().UnixMilli()
		card := &boards.Card{
			BoardID:    boardID,
			Title:      getStr(input, "title"),
			Properties: props,
			CreateAt:   now,
			UpdateAt:   now,
		}
		created, err := svc.CreateCard(boardID, card)
		if err != nil {
			return errResult(err), nil, nil
		}

		if desc := getStr(input, "description"); desc != "" {
			block := &boards.Block{ParentID: created.ID, BoardID: boardID, Type: "text", Title: desc, CreateAt: now, UpdateAt: now}
			svc.CreateContentBlock(boardID, created.ID, block)
		}

		return textResult(fmt.Sprintf("Created: %s (ID: %s)", created.Title, created.ID)), nil, nil
	})

	// skate_comment
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_comment",
		Description: "Add a comment to a task",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id", "text"},
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "Task/card ID"},
				"text":    map[string]any{"type": "string", "description": "Comment text"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		text := getStr(input, "text")

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		now := time.Now().UnixMilli()
		block := &boards.Block{ParentID: cardID, BoardID: card.BoardID, Type: "comment", Title: text, CreateAt: now, UpdateAt: now}
		if _, err := svc.CreateBlock(card.BoardID, []*boards.Block{block}); err != nil {
			return errResult(err), nil, nil
		}
		return textResult("Comment added."), nil, nil
	})

	// skate_add_content
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_add_content",
		Description: "Add a content block to a task's description. Use for long-term reference notes, discoveries, headings, dividers, checklists, and inline images.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id"},
			"properties": map[string]any{
				"task_id":    map[string]any{"type": "string", "description": "Task/card ID"},
				"text":       map[string]any{"type": "string", "description": "Block text (required for all types except divider). For image type, this is the local file path."},
				"block_type": map[string]any{"type": "string", "description": "Block type: text (default), h1, h2, h3, divider, checkbox, image", "default": "text"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		text := getStr(input, "text")
		blockType := getStr(input, "block_type")
		if blockType == "" {
			blockType = "text"
		}

		validTypes := map[string]bool{"text": true, "h1": true, "h2": true, "h3": true, "divider": true, "checkbox": true, "image": true}
		if !validTypes[blockType] {
			return errResult(fmt.Errorf("invalid block_type %q. Supported: text, h1, h2, h3, divider, checkbox, image", blockType)), nil, nil
		}

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}

		if blockType == "image" {
			if text == "" {
				return errResult(fmt.Errorf("file path is required for image blocks")), nil, nil
			}
			fileID, err := svc.UploadFile(cfg.TeamID, card.BoardID, text)
			if err != nil {
				return errResult(fmt.Errorf("uploading image: %w", err)), nil, nil
			}
			now := time.Now().UnixMilli()
			block := &boards.Block{
				ParentID: cardID,
				BoardID:  card.BoardID,
				Type:     "image",
				Title:    text,
				Fields:   map[string]interface{}{"fileId": fileID},
				CreateAt: now,
				UpdateAt: now,
			}
			if _, err := svc.CreateContentBlock(card.BoardID, cardID, block); err != nil {
				return errResult(err), nil, nil
			}
			return textResult(fmt.Sprintf("Image content block added (fileId: %s).", fileID)), nil, nil
		}

		if blockType != "divider" && text == "" {
			return errResult(fmt.Errorf("text is required for %s blocks", blockType)), nil, nil
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

		now := time.Now().UnixMilli()
		block := &boards.Block{ParentID: cardID, BoardID: card.BoardID, Type: actualType, Title: text, CreateAt: now, UpdateAt: now}
		if _, err := svc.CreateContentBlock(card.BoardID, cardID, block); err != nil {
			return errResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Content block added (%s).", blockType)), nil, nil
	})

	// skate_find
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_find",
		Description: "Search tasks by title and content. Returns title matches first, then content matches with snippets.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"query"},
			"properties": map[string]any{
				"query":    map[string]any{"type": "string", "description": "Search query (case-insensitive, partial match)"},
				"board_id": map[string]any{"type": "string", "description": "Board ID (optional, uses default from config)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		query := strings.ToLower(getStr(input, "query"))
		boardID := getStr(input, "board_id")
		if boardID == "" {
			boardID = cfg.BoardID
		}
		if boardID == "" {
			return errResult(fmt.Errorf("board_id required")), nil, nil
		}

		board, err := svc.GetBoard(boardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		cards, err := svc.ListCards(boardID)
		if err != nil {
			return errResult(err), nil, nil
		}

		defs := boards.ParsePropertyDefs(board)
		resolved := boards.ResolveCards(cards, defs)

		var lines []string
		for _, rc := range resolved {
			if strings.Contains(strings.ToLower(rc.Title), query) {
				lines = append(lines, fmt.Sprintf("- [%s] %s | Status: %s | Match: title", rc.ID, rc.Title, rc.Status))
			}
		}
		for _, rc := range resolved {
			if strings.Contains(strings.ToLower(rc.Title), query) {
				continue
			}
			blocks, err := svc.GetBlocks(boardID, rc.ID)
			if err != nil {
				continue
			}
			for _, b := range blocks {
				if strings.Contains(strings.ToLower(b.Title), query) {
					snippet := b.Title
					runes := []rune(snippet)
					if len(runes) > 80 {
						snippet = string(runes[:77]) + "..."
					}
					lines = append(lines, fmt.Sprintf("- [%s] %s | Status: %s | Match in %s: %s", rc.ID, rc.Title, rc.Status, b.Type, snippet))
					break
				}
			}
		}

		if len(lines) == 0 {
			return textResult("No tasks found."), nil, nil
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_comments
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_comments",
		Description: "Get all comments for a task",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id"},
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "Task/card ID"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		blocks, err := svc.GetBlocks(card.BoardID, cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		uc := boards.NewUserCache(svc)
		defer uc.Flush()
		tr := translate.New(cfg.Translate)
		md := boards.RenderComments(blocks, uc, tr)
		return textResult(md), nil, nil
	})

	// skate_task_files
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_task_files",
		Description: "List files attached to a task",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id"},
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "Task/card ID"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		blocks, err := svc.GetBlocks(card.BoardID, cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		var lines []string
		for _, b := range blocks {
			if b.Type == "image" || b.Type == "attachment" {
				fileID := ""
				if fid, ok := b.Fields["fileId"]; ok {
					fileID = fmt.Sprintf("%v", fid)
				}
				name := b.Title
				if name == "" {
					name = fileID
				}
				lines = append(lines, fmt.Sprintf("- %s (type: %s, fileId: %s)", name, b.Type, fileID))
			}
		}
		if len(lines) == 0 {
			return textResult("No files attached."), nil, nil
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_config
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_config",
		Description: "Show effective skate configuration (mentions, translate, board settings)",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		lines := []string{
			fmt.Sprintf("mattermost_url: %s", cfg.MattermostURL),
			fmt.Sprintf("team_id: %s", cfg.TeamID),
		}
		if cfg.BoardID != "" {
			lines = append(lines, fmt.Sprintf("board_id: %s", cfg.BoardID))
		}
		lines = append(lines,
			fmt.Sprintf("only_mine: %v", cfg.OnlyMine),
			fmt.Sprintf("mentions: %v", cfg.MentionsEnabled()),
			fmt.Sprintf("translate: %s", translate.FormatProviderInfo(cfg.Translate)),
		)
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_timer_start
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_timer_start",
		Description: "Start a timer on a task. Auto-stops any running timer.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id"},
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "Task/card ID"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := svc.StartTimer(card.BoardID, cardID)
		if err != nil {
			return textResult("Time tracking is not available on this Mattermost instance."), nil, nil
		}
		msg := fmt.Sprintf("Timer started on: %s", card.Title)
		if resp.StoppedEntry != nil {
			msg += fmt.Sprintf("\nAuto-stopped previous timer on: %s (%s)", resp.StoppedEntry.CardName, resp.StoppedEntry.DurationDisplay)
		}
		return textResult(msg), nil, nil
	})

	// skate_timer_stop
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_timer_stop",
		Description: "Stop the currently running timer with optional notes",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"notes": map[string]any{"type": "string", "description": "Notes about the work done"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		notes := getStr(input, "notes")
		timer, err := svc.GetRunningTimer()
		if err != nil {
			return textResult("Time tracking is not available on this Mattermost instance."), nil, nil
		}
		if timer == nil {
			return textResult("No timer is running."), nil, nil
		}
		stopped, err := svc.StopTimer(timer.ID, notes)
		if err != nil {
			return textResult("Time tracking is not available on this Mattermost instance."), nil, nil
		}
		return textResult(fmt.Sprintf("Timer stopped: %s — %s", stopped.CardName, stopped.DurationDisplay)), nil, nil
	})

	// skate_time_add
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_time_add",
		Description: "Add manual time to a task",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id", "duration"},
			"properties": map[string]any{
				"task_id":  map[string]any{"type": "string", "description": "Task/card ID"},
				"duration": map[string]any{"type": "string", "description": "Duration in HH:MM format"},
				"notes":    map[string]any{"type": "string", "description": "Notes"},
				"date":     map[string]any{"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, map[string]any, error) {
		cardID := getStr(input, "task_id")
		duration := getStr(input, "duration")
		notes := getStr(input, "notes")

		parts := strings.Split(duration, ":")
		if len(parts) != 2 {
			return errResult(fmt.Errorf("invalid duration, use HH:MM")), nil, nil
		}
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		durationSeconds := int64(hours*3600 + minutes*60)

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}

		dateStr := getStr(input, "date")
		var dateMs int64
		if dateStr != "" {
			t, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return errResult(fmt.Errorf("invalid date: %w", err)), nil, nil
			}
			dateMs = t.Add(12 * time.Hour).UnixMilli()
		} else {
			now := time.Now()
			dateMs = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location()).UnixMilli()
		}

		entry, err := svc.AddManualTime(card.BoardID, cardID, durationSeconds, dateMs, notes)
		if err != nil {
			return textResult("Time tracking is not available on this Mattermost instance."), nil, nil
		}
		return textResult(fmt.Sprintf("Added %s to %s", entry.DurationDisplay, card.Title)), nil, nil
	})

	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}

func errResult(err error) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
		IsError: true,
	}
}
