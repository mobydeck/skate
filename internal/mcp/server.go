package mcp

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"skate/internal/boards"
	"skate/internal/client"
	"skate/internal/config"
	"skate/internal/skill"
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
	}, &mcpsdk.ServerOptions{
		Instructions: skill.Pointer(),
	})

	if err := registerTools(server, svc, cfg); err != nil {
		return fmt.Errorf("registering tools: %w", err)
	}

	return server.Run(context.Background(), &mcpsdk.StdioTransport{})
}

func registerTools(s *mcpsdk.Server, svc *boards.Service, cfg *config.Config) error {
	// skate_help
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_help",
		Description: "Get the canonical Skate workflow guide (full SKILL.md). Call once per session if the guide isn't already loaded.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		// Prepend a per-session board_id hint that the static SKILL.md can't carry.
		var header string
		if cfg.BoardID != "" {
			header = fmt.Sprintf("## Board ID (this server)\nConfigured board_id: %s\nPass this to skate_tasks, skate_statuses, skate_find, and skate_create_task when the tool requires it.\n\n---\n\n", cfg.BoardID)
		} else {
			header = "## Board ID (this server)\nNo default board_id is configured. Call skate_boards to list boards, then pass the correct ID to skate_tasks, skate_statuses, skate_find, and skate_create_task.\n\n---\n\n"
		}
		return textResult(header + skill.Markdown()), nil, nil
	})

	// skate_statuses
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_statuses",
		Description: "List available statuses for the board. Call this before skate_update_status to avoid invalid status errors.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"board_id": map[string]any{"type": "string", "description": "Board ID (optional, uses default from config)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
		statusProp := boards.FindPropertyByName(defs, "Status")
		if statusProp == nil {
			return errResult(fmt.Errorf("board has no Status property")), nil, nil
		}

		var values []string
		for _, o := range statusProp.Options {
			values = append(values, o.Value)
		}
		return textResult("Available statuses: " + strings.Join(values, ", ")), nil, nil
	})

	// skate_boards
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_boards",
		Description: "List available Mattermost boards for the current user",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
				"board_id":  map[string]any{"type": "string", "description": "Board ID (optional, uses default from config)"},
				"status":    map[string]any{"type": "string", "description": "Filter by specific status (optional)"},
				"show_all":  map[string]any{"type": "boolean", "description": "Show all tasks regardless of status (default: false)"},
				"mine":      map[string]any{"type": "boolean", "description": "Show only tasks assigned to the authenticated user (overrides config)"},
				"all_users": map[string]any{"type": "boolean", "description": "Show tasks for all users (overrides only_mine config)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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

		// Assignee filtering — mirrors CLI flags. Explicit params win over config.
		onlyMine := cfg.OnlyMine
		if mine, ok := input["mine"].(bool); ok && mine {
			onlyMine = true
		}
		if allUsers, ok := input["all_users"].(bool); ok && allUsers {
			onlyMine = false
		}
		if onlyMine {
			if me, err := svc.GetMe(); err == nil {
				resolved = boards.FilterMine(resolved, me)
			}
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

	// skate_next
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_next",
		Description: "Pick the highest-priority Not Started task and return its full details. Use when the user says 'pick next task' or 'work on the top task' — saves the list+sort+pick round-trip.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"board_id":     map[string]any{"type": "string", "description": "Board ID (optional, uses default from config)"},
				"mine":         map[string]any{"type": "boolean", "description": "Only consider tasks assigned to the authenticated user"},
				"no_translate": map[string]any{"type": "boolean", "description": "Skip translation even if enabled in config (default: false)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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

		var queue []boards.ResolvedCard
		for _, rc := range resolved {
			if strings.EqualFold(rc.Status, "Not Started") {
				queue = append(queue, rc)
			}
		}
		if mine, _ := input["mine"].(bool); mine {
			if me, err := svc.GetMe(); err == nil {
				queue = boards.FilterMine(queue, me)
			}
		}

		if len(queue) == 0 {
			return textResult("No tasks ready to start."), nil, nil
		}

		top := queue[0]
		card, err := svc.GetCard(top.ID)
		if err != nil {
			return errResult(err), nil, nil
		}
		blocks, err := svc.GetBlocks(boardID, top.ID)
		if err != nil {
			return errResult(err), nil, nil
		}
		summaries, _ := svc.GetTimeSummary(boardID, top.ID)
		uc := boards.NewUserCache(svc)
		defer uc.Flush()
		var tr *translate.Translator
		if noTr, _ := input["no_translate"].(bool); !noTr {
			tr = translate.New(cfg.Translate)
		}
		md := boards.RenderCardMarkdown(card, board, blocks, summaries, uc, tr)
		return textResult(md), nil, nil
	})

	// skate_task
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_task",
		Description: "Get full task details rendered as markdown, including properties, description, comments, attachments, and time tracking.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id"},
			"properties": map[string]any{
				"task_id":      map[string]any{"type": "string", "description": "Task/card ID"},
				"no_translate": map[string]any{"type": "boolean", "description": "Skip translation even if enabled in config (default: false)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
		var tr *translate.Translator
		if noTr, _ := input["no_translate"].(bool); !noTr {
			tr = translate.New(cfg.Translate)
		}
		md := boards.RenderCardMarkdown(card, board, blocks, summaries, uc, tr)
		return textResult(md), nil, nil
	})

	// skate_update_status
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_update_status",
		Description: "Change one or more tasks' status. Pass task_id for a single task, or task_ids for a batch. start_timer is single-task only. Batch failures continue and are reported per task.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"status"},
			"properties": map[string]any{
				"task_id":     map[string]any{"type": "string", "description": "Single task/card ID (use task_ids for batch)"},
				"task_ids":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Multiple task/card IDs to update at once"},
				"status":      map[string]any{"type": "string", "description": "New status value"},
				"start_timer": map[string]any{"type": "boolean", "description": "Start timer after updating (single-task only; default: false)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		status := getStr(input, "status")
		startTimer, _ := input["start_timer"].(bool)

		var cardIDs []string
		if raw, ok := input["task_ids"].([]any); ok {
			for _, v := range raw {
				if s, ok := v.(string); ok && s != "" {
					cardIDs = append(cardIDs, s)
				}
			}
		}
		if id := getStr(input, "task_id"); id != "" {
			cardIDs = append(cardIDs, id)
		}
		if len(cardIDs) == 0 {
			return errResult(fmt.Errorf("task_id or task_ids required")), nil, nil
		}
		if len(cardIDs) > 1 && startTimer {
			return errResult(fmt.Errorf("start_timer is only supported for single-task updates")), nil, nil
		}

		first, err := svc.GetCard(cardIDs[0])
		if err != nil {
			return errResult(err), nil, nil
		}
		board, err := svc.GetBoard(first.BoardID)
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
		patch := &boards.CardPatch{UpdatedProperties: map[string]any{statusProp.ID: option.ID}}

		var lines, failed []string
		for _, id := range cardIDs {
			if _, err := svc.PatchCard(id, patch); err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", id, err))
				continue
			}
			lines = append(lines, fmt.Sprintf("%s → %q", id, option.Value))
		}

		if startTimer {
			resp, err := svc.StartTimer(first.BoardID, cardIDs[0])
			if err != nil {
				lines = append(lines, "Time tracking is not available on this Mattermost instance.")
			} else {
				lines = append(lines, fmt.Sprintf("Timer started on: %s", first.Title))
				if resp.StoppedEntry != nil {
					lines = append(lines, fmt.Sprintf("Auto-stopped previous timer on: %s (%s)", resp.StoppedEntry.CardName, resp.StoppedEntry.DurationDisplay))
				}
			}
		}

		if len(failed) > 0 {
			lines = append(lines, fmt.Sprintf("\n%d of %d failed:", len(failed), len(cardIDs)))
			for _, f := range failed {
				lines = append(lines, "  "+f)
			}
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_update_task
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_update_task",
		Description: "Update one or more fields on a task (title, icon, status, priority, assignee). Generalizes skate_update_status. Only the fields you pass are changed.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id"},
			"properties": map[string]any{
				"task_id":     map[string]any{"type": "string", "description": "Task/card ID"},
				"title":       map[string]any{"type": "string", "description": "New title"},
				"icon":        map[string]any{"type": "string", "description": "New icon (emoji)"},
				"status":      map[string]any{"type": "string", "description": "New status — call skate_statuses for valid values"},
				"priority":    map[string]any{"type": "string", "description": "New priority"},
				"assignee":    map[string]any{"type": "string", "description": "Assignee — username (resolved via skate_users) or raw user ID"},
				"start_timer": map[string]any{"type": "boolean", "description": "Start timer after updating (default: false)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		cardID := getStr(input, "task_id")
		if cardID == "" {
			return errResult(fmt.Errorf("task_id required")), nil, nil
		}

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		board, err := svc.GetBoard(card.BoardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		defs := boards.ParsePropertyDefs(board)

		patch := &boards.CardPatch{}
		var changes []string

		if v, ok := input["title"].(string); ok && v != "" {
			patch.Title = &v
			changes = append(changes, fmt.Sprintf("title=%q", v))
		}
		if v, ok := input["icon"].(string); ok && v != "" {
			patch.Icon = &v
			changes = append(changes, fmt.Sprintf("icon=%q", v))
		}

		props := map[string]any{}
		if status := getStr(input, "status"); status != "" {
			p := boards.FindPropertyByName(defs, "Status")
			if p == nil {
				return errResult(fmt.Errorf("board has no Status property")), nil, nil
			}
			o := boards.FindOptionByValue(p, status)
			if o == nil {
				return errResult(fmt.Errorf("invalid status %q", status)), nil, nil
			}
			props[p.ID] = o.ID
			changes = append(changes, fmt.Sprintf("status=%q", status))
		}
		if priority := getStr(input, "priority"); priority != "" {
			p := boards.FindPropertyByName(defs, "Priority")
			if p == nil {
				return errResult(fmt.Errorf("board has no Priority property")), nil, nil
			}
			o := boards.FindOptionByValue(p, priority)
			if o == nil {
				return errResult(fmt.Errorf("invalid priority %q", priority)), nil, nil
			}
			props[p.ID] = o.ID
			changes = append(changes, fmt.Sprintf("priority=%q", priority))
		}
		if assignee := getStr(input, "assignee"); assignee != "" {
			p := boards.FindPropertyByName(defs, "Assignee")
			if p == nil {
				p = boards.FindPropertyByName(defs, "Assignees")
			}
			if p == nil {
				return errResult(fmt.Errorf("board has no Assignee property")), nil, nil
			}
			resolved, err := svc.ResolveUserRef(cfg.TeamID, assignee)
			if err != nil {
				return errResult(fmt.Errorf("resolving assignee: %w", err)), nil, nil
			}
			props[p.ID] = resolved
			changes = append(changes, fmt.Sprintf("assignee=%q", assignee))
		}
		if len(props) > 0 {
			patch.UpdatedProperties = props
		}

		startTimer, _ := input["start_timer"].(bool)
		if patch.Title == nil && patch.Icon == nil && len(patch.UpdatedProperties) == 0 && !startTimer {
			return errResult(fmt.Errorf("nothing to update — pass at least one of title/icon/status/priority/assignee/start_timer")), nil, nil
		}

		if patch.Title != nil || patch.Icon != nil || len(patch.UpdatedProperties) > 0 {
			if _, err := svc.PatchCard(cardID, patch); err != nil {
				return errResult(err), nil, nil
			}
		}

		msg := fmt.Sprintf("Updated %s: %s", cardID, strings.Join(changes, ", "))
		if startTimer {
			resp, err := svc.StartTimer(card.BoardID, cardID)
			if err != nil {
				msg += "\nTime tracking is not available on this Mattermost instance."
			} else {
				msg += fmt.Sprintf("\nTimer started on: %s", card.Title)
				if resp.StoppedEntry != nil {
					msg += fmt.Sprintf("\nAuto-stopped previous timer on: %s (%s)", resp.StoppedEntry.CardName, resp.StoppedEntry.DurationDisplay)
				}
			}
		}
		return textResult(msg), nil, nil
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
				"assignee":    map[string]any{"type": "string", "description": "Assignee — username (resolved via skate_users) or raw user ID"},
				"description": map[string]any{"type": "string", "description": "Task description"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
		props := make(map[string]any)

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
		if s := getStr(input, "assignee"); s != "" {
			p := boards.FindPropertyByName(defs, "Assignee")
			if p == nil {
				p = boards.FindPropertyByName(defs, "Assignees")
			}
			if p != nil {
				resolved, err := svc.ResolveUserRef(cfg.TeamID, s)
				if err != nil {
					return errResult(fmt.Errorf("resolving assignee: %w", err)), nil, nil
				}
				props[p.ID] = resolved
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
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
				Fields:   map[string]any{"fileId": fileID},
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

	// skate_edit_block
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_edit_block",
		Description: "Replace the text of a content block, comment, or heading. For h1/h2/h3 headings, include the markdown prefix in text. Find block IDs via skate_task (json).",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id", "block_id", "text"},
			"properties": map[string]any{
				"task_id":  map[string]any{"type": "string", "description": "Task/card ID the block belongs to"},
				"block_id": map[string]any{"type": "string", "description": "Block ID to edit"},
				"text":     map[string]any{"type": "string", "description": "New block text"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		cardID := getStr(input, "task_id")
		blockID := getStr(input, "block_id")
		text := getStr(input, "text")
		if blockID == "" {
			return errResult(fmt.Errorf("block_id required")), nil, nil
		}

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		if err := svc.UpdateBlockTitle(card.BoardID, blockID, text); err != nil {
			return errResult(fmt.Errorf("editing block: %w", err)), nil, nil
		}
		return textResult(fmt.Sprintf("Edited block %s", blockID)), nil, nil
	})

	// skate_delete_block
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_delete_block",
		Description: "Delete a single block (content, comment, or attachment) from a task. Use to fix mistakes — e.g. removing a wrong comment or an outdated content block. Find block IDs via skate_task (json) or skate_task_files.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id", "block_id"},
			"properties": map[string]any{
				"task_id":  map[string]any{"type": "string", "description": "Task/card ID the block belongs to"},
				"block_id": map[string]any{"type": "string", "description": "Block ID to delete"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		cardID := getStr(input, "task_id")
		blockID := getStr(input, "block_id")
		if blockID == "" {
			return errResult(fmt.Errorf("block_id required")), nil, nil
		}

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		if err := svc.DeleteBlock(card.BoardID, cardID, blockID); err != nil {
			return errResult(fmt.Errorf("deleting block: %w", err)), nil, nil
		}
		return textResult(fmt.Sprintf("Deleted block %s", blockID)), nil, nil
	})

	// skate_attach
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_attach",
		Description: "Upload a local file and attach it to a task. Use for logs, configs, screenshots, generated artifacts, or plan documents that should travel with the task.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"task_id", "file_path"},
			"properties": map[string]any{
				"task_id":   map[string]any{"type": "string", "description": "Task/card ID"},
				"file_path": map[string]any{"type": "string", "description": "Absolute path to the local file to upload. Use absolute paths to avoid working-directory issues."},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		cardID := getStr(input, "task_id")
		filePath := getStr(input, "file_path")
		if filePath == "" {
			return errResult(fmt.Errorf("file_path required")), nil, nil
		}

		card, err := svc.GetCard(cardID)
		if err != nil {
			return errResult(err), nil, nil
		}
		fileID, err := svc.UploadFile(cfg.TeamID, card.BoardID, filePath)
		if err != nil {
			return errResult(fmt.Errorf("uploading file: %w", err)), nil, nil
		}

		now := time.Now().UnixMilli()
		block := &boards.Block{
			ParentID: cardID,
			BoardID:  card.BoardID,
			Type:     "attachment",
			Title:    filePath,
			Fields:   map[string]any{"fileId": fileID},
			CreateAt: now,
			UpdateAt: now,
		}
		if _, err := svc.CreateBlock(card.BoardID, []*boards.Block{block}); err != nil {
			return errResult(fmt.Errorf("creating attachment block: %w", err)), nil, nil
		}
		return textResult(fmt.Sprintf("File attached (fileId: %s).", fileID)), nil, nil
	})

	// skate_download
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_download",
		Description: "Download an attached file. If output_path is given, the file is saved to disk and the path is returned. Otherwise the file content is returned inline (text files only — binary files require output_path).",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"file_id"},
			"properties": map[string]any{
				"file_id":     map[string]any{"type": "string", "description": "File ID (from skate_task_files)"},
				"board_id":    map[string]any{"type": "string", "description": "Board ID (optional, uses default from config)"},
				"output_path": map[string]any{"type": "string", "description": "Absolute path to save the file. If omitted, the content is returned inline (text files only)."},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		fileID := getStr(input, "file_id")
		if fileID == "" {
			return errResult(fmt.Errorf("file_id required")), nil, nil
		}
		boardID := getStr(input, "board_id")
		if boardID == "" {
			boardID = cfg.BoardID
		}
		if boardID == "" {
			return errResult(fmt.Errorf("board_id required")), nil, nil
		}

		data, err := svc.DownloadFile(cfg.TeamID, boardID, fileID)
		if err != nil {
			return errResult(fmt.Errorf("downloading file: %w", err)), nil, nil
		}

		if outPath := getStr(input, "output_path"); outPath != "" {
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				return errResult(fmt.Errorf("writing file: %w", err)), nil, nil
			}
			return textResult(fmt.Sprintf("Saved %d bytes to %s", len(data), outPath)), nil, nil
		}

		const maxInline = 256 * 1024
		if len(data) > maxInline {
			return errResult(fmt.Errorf("file is %d bytes (>%d KiB); pass output_path to save it to disk", len(data), maxInline/1024)), nil, nil
		}
		if !utf8.Valid(data) {
			return errResult(fmt.Errorf("file is not valid UTF-8 (likely binary); pass output_path to save it to disk")), nil, nil
		}
		return textResult(string(data)), nil, nil
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
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
				"task_id":      map[string]any{"type": "string", "description": "Task/card ID"},
				"no_translate": map[string]any{"type": "boolean", "description": "Skip translation even if enabled in config (default: false)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
		var tr *translate.Translator
		if noTr, _ := input["no_translate"].(bool); !noTr {
			tr = translate.New(cfg.Translate)
		}
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
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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

	// skate_state
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_state",
		Description: "Snapshot of your current working state on the configured board: who you are, the timer that's running (if any), and the In Progress tasks assigned to you. Useful as a session-resume preamble.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"board_id": map[string]any{"type": "string", "description": "Board ID (optional, uses default from config)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		boardID := getStr(input, "board_id")
		if boardID == "" {
			boardID = cfg.BoardID
		}
		if boardID == "" {
			return errResult(fmt.Errorf("board_id required")), nil, nil
		}

		me, _ := svc.GetMe()
		timer, _ := svc.GetRunningTimer()

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

		var inProgress []boards.ResolvedCard
		for _, rc := range resolved {
			if strings.EqualFold(rc.Status, "In Progress") {
				inProgress = append(inProgress, rc)
			}
		}
		if me != nil {
			inProgress = boards.FilterMine(inProgress, me)
		}

		var sb strings.Builder
		if me != nil {
			fmt.Fprintf(&sb, "User: @%s (%s)\n", me.Username, me.ID)
		}
		if timer != nil {
			elapsed := boards.FormatDuration((time.Now().UnixMilli() - timer.StartTime) / 1000)
			fmt.Fprintf(&sb, "Running timer: %s (%s elapsed) [card: %s]\n", timer.CardName, elapsed, timer.CardID)
		} else {
			sb.WriteString("Running timer: none\n")
		}
		sb.WriteString("\nIn Progress (yours):\n")
		if len(inProgress) == 0 {
			sb.WriteString("  (none)\n")
		} else {
			for _, rc := range inProgress {
				fmt.Fprintf(&sb, "  - [%s] %s | Priority: %s\n", rc.ID, rc.Title, rc.Priority)
			}
		}
		return textResult(sb.String()), nil, nil
	})

	// skate_users
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_users",
		Description: "List team members. Use to look up a user ID before assigning a task — assignee fields can take a username (resolved to ID) or a raw user ID.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Optional substring to filter by username or full name (case-insensitive)"},
			},
		},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		users, err := svc.ListUsers(cfg.TeamID)
		if err != nil {
			return errResult(err), nil, nil
		}
		if q := strings.ToLower(getStr(input, "query")); q != "" {
			filtered := users[:0]
			for _, u := range users {
				if strings.Contains(strings.ToLower(u.Username), q) ||
					strings.Contains(strings.ToLower(u.FirstName+" "+u.LastName), q) {
					filtered = append(filtered, u)
				}
			}
			users = filtered
		}
		if len(users) == 0 {
			return textResult("No users found."), nil, nil
		}
		var lines []string
		for _, u := range users {
			name := strings.TrimSpace(u.FirstName + " " + u.LastName)
			if name != "" {
				lines = append(lines, fmt.Sprintf("- @%s (%s) | id: %s", u.Username, name, u.ID))
			} else {
				lines = append(lines, fmt.Sprintf("- @%s | id: %s", u.Username, u.ID))
			}
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_me
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_me",
		Description: "Identify the Mattermost user this server is authenticated as. Useful for self-attribution in comments and for filtering 'my' tasks.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
		me, err := svc.GetMe()
		if err != nil {
			return errResult(err), nil, nil
		}
		lines := []string{
			fmt.Sprintf("id: %s", me.ID),
			fmt.Sprintf("username: %s", me.Username),
		}
		if name := strings.TrimSpace(me.FirstName + " " + me.LastName); name != "" {
			lines = append(lines, fmt.Sprintf("name: %s", name))
		}
		if me.Nickname != "" {
			lines = append(lines, fmt.Sprintf("nickname: %s", me.Nickname))
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})

	// skate_config
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "skate_config",
		Description: "Show effective skate configuration (mentions, translate, board settings)",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input map[string]any) (*mcpsdk.CallToolResult, any, error) {
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
