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
		Properties: map[string]interface{}{
			"p1": "o1",
		},
	}
	board := &Board{
		CardProperties: []map[string]interface{}{
			{
				"id":   "p1",
				"name": "Status",
				"type": "select",
				"options": []interface{}{
					map[string]interface{}{"id": "o1", "value": "In Progress"},
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
		{Type: "attachment", Title: "file.pdf", Fields: map[string]interface{}{"fileId": "f123"}},
	}

	md := RenderCardMarkdown(card, board, blocks, nil, nil, nil)

	if !strings.Contains(md, "## Attachments") {
		t.Error("should have attachments section")
	}
	if !strings.Contains(md, "file.pdf") {
		t.Error("should contain filename")
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
