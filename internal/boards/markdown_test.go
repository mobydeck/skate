package boards

import (
	"strings"
	"testing"
	"time"
)

func TestRenderCardMarkdown_Basic(t *testing.T) {
	card := &Card{
		ID:    "card-1",
		Title: "Test Task",
		Icon:  "🎯",
		Properties: map[string]any{
			"p1": "o1",
		},
	}
	board := &Board{
		CardProperties: []map[string]any{
			{
				"id":   "p1",
				"name": "Status",
				"type": "select",
				"options": []any{
					map[string]any{"id": "o1", "value": "In Progress"},
				},
			},
		},
	}
	blocks := []*Block{
		{Type: "text", Title: "Description text here"},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	if !strings.Contains(md, "# 🎯 Test Task") {
		t.Error("should contain title with icon")
	}
	if !strings.Contains(md, "In Progress") {
		t.Error("should contain resolved status")
	}
	if !strings.Contains(md, "Description text here") {
		t.Error("should contain text block content")
	}
}

func TestRenderCardMarkdown_Comments(t *testing.T) {
	card := &Card{Title: "Test"}
	board := &Board{}
	blocks := []*Block{
		{Type: "comment", Title: "A comment", CreatedBy: "user-1", CreateAt: time.Now().UnixMilli()},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	if !strings.Contains(md, "## Comments") {
		t.Error("should have comments section")
	}
	if !strings.Contains(md, "A comment") {
		t.Error("should contain comment text")
	}
}

func TestRenderCardMarkdown_Attachments(t *testing.T) {
	card := &Card{Title: "Test"}
	board := &Board{}
	blocks := []*Block{
		{Type: "attachment", Title: "file.pdf", Fields: map[string]any{"fileId": "f123"}},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	if !strings.Contains(md, "## Attachments") {
		t.Error("should have attachments section")
	}
	if !strings.Contains(md, "file.pdf") {
		t.Error("should contain filename")
	}
}

func TestRenderCardMarkdown_InlineImage(t *testing.T) {
	card := &Card{Title: "Test"}
	board := &Board{}
	blocks := []*Block{
		{Type: "text", Title: "Some text before"},
		{Type: "image", Title: "screenshot.png", Fields: map[string]any{"fileId": "img123"}},
		{Type: "text", Title: "Some text after"},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	if !strings.Contains(md, "## Description") {
		t.Error("should have description section")
	}
	if !strings.Contains(md, "![screenshot.png](fileId: img123)") {
		t.Error("should render inline image with fileId")
	}
	if !strings.Contains(md, "Some text before") || !strings.Contains(md, "Some text after") {
		t.Error("should contain surrounding text blocks")
	}
	if strings.Contains(md, "## Attachments") {
		t.Error("image blocks should not appear in attachments section")
	}
}

func TestRenderCardMarkdown_ContentOrder(t *testing.T) {
	card := &Card{
		Title: "Test",
		ContentOrder: []any{
			"b1", "b2", "b3",
		},
	}
	board := &Board{}
	// Blocks arrive from the API in a different order than ContentOrder.
	blocks := []*Block{
		{ID: "b3", Type: "text", Title: "Third"},
		{ID: "b1", Type: "text", Title: "First"},
		{ID: "b2", Type: "text", Title: "Second"},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	first := strings.Index(md, "First")
	second := strings.Index(md, "Second")
	third := strings.Index(md, "Third")
	if first < 0 || second < 0 || third < 0 {
		t.Fatalf("missing content blocks in output:\n%s", md)
	}
	if !(first < second && second < third) {
		t.Errorf("content blocks should follow ContentOrder, got positions First=%d Second=%d Third=%d\n%s", first, second, third, md)
	}
}

func TestRenderCardMarkdown_ContentOrderNestedRows(t *testing.T) {
	// Focalboard groups inline content as nested arrays inside ContentOrder.
	card := &Card{
		Title: "Test",
		ContentOrder: []any{
			"b1",
			[]any{"b2", "b3"},
			"b4",
		},
	}
	board := &Board{}
	blocks := []*Block{
		{ID: "b4", Type: "text", Title: "Fourth"},
		{ID: "b2", Type: "text", Title: "Second"},
		{ID: "b1", Type: "text", Title: "First"},
		{ID: "b3", Type: "text", Title: "Third"},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	pos := []int{
		strings.Index(md, "First"),
		strings.Index(md, "Second"),
		strings.Index(md, "Third"),
		strings.Index(md, "Fourth"),
	}
	for i := 1; i < len(pos); i++ {
		if pos[i-1] < 0 || pos[i] < 0 || pos[i-1] >= pos[i] {
			t.Errorf("nested ContentOrder not flattened in order, positions=%v\n%s", pos, md)
			break
		}
	}
}

func TestRenderCardMarkdown_ContentOrderUnlistedBlocksTrail(t *testing.T) {
	card := &Card{
		Title:        "Test",
		ContentOrder: []any{"b1"},
	}
	board := &Board{}
	blocks := []*Block{
		{ID: "borphan", Type: "text", Title: "Orphan"},
		{ID: "b1", Type: "text", Title: "Listed"},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	listed := strings.Index(md, "Listed")
	orphan := strings.Index(md, "Orphan")
	if listed < 0 || orphan < 0 {
		t.Fatalf("missing block in output:\n%s", md)
	}
	if listed > orphan {
		t.Errorf("blocks present in ContentOrder should appear before unlisted ones; got Listed=%d Orphan=%d\n%s", listed, orphan, md)
	}
}

func TestRenderCardMarkdown_CreatedBy(t *testing.T) {
	card := &Card{Title: "Test", CreatedBy: "user-abc"}
	board := &Board{}

	md := RenderCardMarkdown(card, board, nil, nil, nil, nil)

	if !strings.Contains(md, "| Created By | @user-abc |") {
		t.Error("should show Created By with @ prefix")
	}
}

func TestRenderCardMarkdown_PersonProperty(t *testing.T) {
	card := &Card{
		Title: "Test",
		Properties: map[string]any{
			"p1": "user-id-1",
		},
	}
	board := &Board{
		CardProperties: []map[string]any{
			{
				"id":   "p1",
				"name": "Assignee",
				"type": "person",
			},
		},
	}

	// Without UserCache, person property shows raw ID (no @ prefix, no resolution)
	md := RenderCardMarkdown(card, board, nil, nil, nil, nil)
	if !strings.Contains(md, "| Assignee | user-id-1 |") {
		t.Errorf("should show person property value, got:\n%s", md)
	}
	if strings.Contains(md, "@user-id-1") {
		t.Error("should not add @ prefix without UserCache")
	}
}

func TestRenderCardMarkdown_TimeTracking(t *testing.T) {
	card := &Card{Title: "Test"}
	board := &Board{}
	summaries := []*TimeEntrySummary{
		{UserID: "user-1", TotalSeconds: 3600, TotalDisplay: "01:00"},
	}

	md := RenderCardMarkdown(card, board, nil, summaries, nil, nil)

	if !strings.Contains(md, "## Time Tracking") {
		t.Error("should have time tracking section")
	}
	if !strings.Contains(md, "01:00") {
		t.Error("should contain time display")
	}
}

func TestRenderCardMarkdown_RunningTimer(t *testing.T) {
	card := &Card{Title: "Test"}
	board := &Board{}
	now := time.Now().UnixMilli()
	summaries := []*TimeEntrySummary{
		{
			UserID:       "user-1",
			TotalSeconds: 0,
			TotalDisplay: "00:00",
			RunningEntry: &TimeEntry{StartTime: now - 120000}, // 2 minutes ago
		},
	}

	md := RenderCardMarkdown(card, board, nil, summaries, nil, nil)

	if !strings.Contains(md, "running") {
		t.Error("should indicate running timer")
	}
}

func TestRenderCardMarkdown_WithTranslator(t *testing.T) {
	card := &Card{Title: "Original Title"}
	board := &Board{}
	blocks := []*Block{
		{Type: "text", Title: "Original text"},
	}

	tr := &mockTranslator{prefix: "TR:"}
	md := RenderCardMarkdown(card, board, blocks, nil, nil, tr)

	if !strings.Contains(md, "TR:Original Title") {
		t.Error("title should be translated")
	}
	if !strings.Contains(md, "TR:Original text") {
		t.Error("text block should be translated")
	}
}

type mockTranslator struct {
	prefix string
}

func (m *mockTranslator) Translate(text string) string {
	return m.prefix + text
}

func TestRenderCardMarkdown_MaxComments(t *testing.T) {
	card := &Card{Title: "Test"}
	board := &Board{}
	now := time.Now().UnixMilli()
	blocks := []*Block{
		{Type: "comment", Title: "Comment 1", CreatedBy: "u1", CreateAt: now - 5000},
		{Type: "comment", Title: "Comment 2", CreatedBy: "u2", CreateAt: now - 4000},
		{Type: "comment", Title: "Comment 3", CreatedBy: "u3", CreateAt: now - 3000},
		{Type: "comment", Title: "Comment 4", CreatedBy: "u4", CreateAt: now - 2000},
		{Type: "comment", Title: "Comment 5", CreatedBy: "u5", CreateAt: now - 1000},
	}

	// Limit to 2 comments: should show last 2 and hide 3
	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil, 2)

	if !strings.Contains(md, "Comment 4") || !strings.Contains(md, "Comment 5") {
		t.Error("should show last 2 comments")
	}
	if strings.Contains(md, "Comment 1") || strings.Contains(md, "Comment 2") {
		t.Error("should hide earlier comments")
	}
	if !strings.Contains(md, "3 earlier comments not shown") {
		t.Error("should show hidden count")
	}

	// No limit (0): should show all
	md = RenderCardMarkdown(card, board, blocks, nil, nil, nil)
	if !strings.Contains(md, "Comment 1") || !strings.Contains(md, "Comment 5") {
		t.Error("should show all comments when no limit")
	}
}

func TestRenderCardMarkdown_CommentsSortedByDate(t *testing.T) {
	card := &Card{Title: "Test"}
	board := &Board{}
	now := time.Now().UnixMilli()
	blocks := []*Block{
		{Type: "comment", Title: "Third", CreatedBy: "u1", CreateAt: now},
		{Type: "comment", Title: "First", CreatedBy: "u2", CreateAt: now - 2000},
		{Type: "comment", Title: "Second", CreatedBy: "u3", CreateAt: now - 1000},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	firstIdx := strings.Index(md, "First")
	secondIdx := strings.Index(md, "Second")
	thirdIdx := strings.Index(md, "Third")

	if firstIdx > secondIdx || secondIdx > thirdIdx {
		t.Errorf("comments should be sorted chronologically: First(%d) Second(%d) Third(%d)", firstIdx, secondIdx, thirdIdx)
	}
}

func TestRenderComments(t *testing.T) {
	now := time.Now().UnixMilli()
	blocks := []*Block{
		{Type: "text", Title: "Not a comment"},
		{Type: "comment", Title: "Comment B", CreatedBy: "u1", CreateAt: now},
		{Type: "comment", Title: "Comment A", CreatedBy: "u2", CreateAt: now - 1000},
	}

	md := RenderComments(blocks, nil, nil)

	if !strings.Contains(md, "Comment A") || !strings.Contains(md, "Comment B") {
		t.Error("should contain both comments")
	}
	if strings.Contains(md, "Not a comment") {
		t.Error("should not contain non-comment blocks")
	}

	// Verify sorted chronologically (A before B)
	aIdx := strings.Index(md, "Comment A")
	bIdx := strings.Index(md, "Comment B")
	if aIdx > bIdx {
		t.Error("comments should be sorted chronologically (A before B)")
	}
}

func TestRenderComments_Empty(t *testing.T) {
	blocks := []*Block{
		{Type: "text", Title: "Just text"},
	}
	md := RenderComments(blocks, nil, nil)
	if md != "No comments.\n" {
		t.Errorf("expected 'No comments.', got %q", md)
	}
}

func TestComputeElapsed(t *testing.T) {
	now := time.Now().UnixMilli()

	got := computeElapsed(0)
	if got != 0 {
		t.Errorf("zero start should return 0, got %d", got)
	}

	got = computeElapsed(now - 60000) // 60 seconds ago
	if got < 59 || got > 61 {
		t.Errorf("expected ~60 seconds, got %d", got)
	}
}
