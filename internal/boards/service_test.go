package boards

import (
	"testing"
)

func TestParsePropertyDefs(t *testing.T) {
	board := &Board{
		CardProperties: []map[string]any{
			{
				"id":   "prop-1",
				"name": "Status",
				"type": "select",
				"options": []any{
					map[string]any{"id": "opt-1", "value": "To Do", "color": "red"},
					map[string]any{"id": "opt-2", "value": "Done", "color": "green"},
				},
			},
			{
				"id":   "prop-2",
				"name": "Assignee",
				"type": "person",
			},
		},
	}

	defs := ParsePropertyDefs(board)
	if len(defs) != 2 {
		t.Fatalf("expected 2 defs, got %d", len(defs))
	}

	if defs[0].Name != "Status" || defs[0].Type != "select" {
		t.Errorf("first def: got %s/%s", defs[0].Name, defs[0].Type)
	}
	if len(defs[0].Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(defs[0].Options))
	}
	if defs[0].Options[0].Value != "To Do" {
		t.Errorf("first option: got %q", defs[0].Options[0].Value)
	}
}

func TestFindPropertyByName(t *testing.T) {
	defs := []PropertyDef{
		{ID: "p1", Name: "Status", Type: "select"},
		{ID: "p2", Name: "Priority", Type: "select"},
	}

	p := FindPropertyByName(defs, "status") // case-insensitive
	if p == nil || p.ID != "p1" {
		t.Error("should find Status property")
	}

	p = FindPropertyByName(defs, "PRIORITY")
	if p == nil || p.ID != "p2" {
		t.Error("should find Priority property")
	}

	p = FindPropertyByName(defs, "nonexistent")
	if p != nil {
		t.Error("should return nil for missing property")
	}
}

func TestFindOptionByValue(t *testing.T) {
	def := &PropertyDef{
		Options: []PropertyOption{
			{ID: "o1", Value: "Not Started"},
			{ID: "o2", Value: "In Progress"},
			{ID: "o3", Value: "Done"},
		},
	}

	o := FindOptionByValue(def, "in progress") // case-insensitive
	if o == nil || o.ID != "o2" {
		t.Error("should find In Progress option")
	}

	o = FindOptionByValue(def, "DONE")
	if o == nil || o.ID != "o3" {
		t.Error("should find Done option")
	}

	o = FindOptionByValue(def, "invalid")
	if o != nil {
		t.Error("should return nil for missing option")
	}
}

func TestResolvePropertyValue(t *testing.T) {
	defs := []PropertyDef{
		{
			ID:   "p1",
			Name: "Status",
			Type: "select",
			Options: []PropertyOption{
				{ID: "o1", Value: "To Do"},
				{ID: "o2", Value: "Done"},
			},
		},
		{
			ID:   "p2",
			Name: "Notes",
			Type: "text",
		},
	}

	// Select property resolves option ID to value
	got := ResolvePropertyValue(defs, "p1", "o2")
	if got != "Done" {
		t.Errorf("expected 'Done', got %q", got)
	}

	// Text property returns raw value
	got = ResolvePropertyValue(defs, "p2", "some text")
	if got != "some text" {
		t.Errorf("expected 'some text', got %q", got)
	}

	// Unknown option ID returns raw
	got = ResolvePropertyValue(defs, "p1", "unknown-option")
	if got != "unknown-option" {
		t.Errorf("expected raw value, got %q", got)
	}

	// Nil value
	got = ResolvePropertyValue(defs, "p1", nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestResolveCards_PersonProperty(t *testing.T) {
	defs := []PropertyDef{
		{ID: "p1", Name: "Status", Type: "select", Options: []PropertyOption{{ID: "o-todo", Value: "Not Started"}}},
		{ID: "p2", Name: "Assignee", Type: "person"},
	}
	cards := []*Card{
		{ID: "c1", Properties: map[string]any{"p1": "o-todo", "p2": "user-id-123"}},
		{ID: "c2", Properties: map[string]any{"p2": ""}},  // empty assignee
		{ID: "c3", Properties: map[string]any{}},          // no property at all
	}

	// Pre-populated cache so Resolve() doesn't try the API.
	uc := &UserCache{
		users: map[string]*User{
			"user-id-123": {ID: "user-id-123", Username: "alice"},
		},
		path: "", // empty path; save() bails early when nothing dirty
	}

	got := ResolveCards(cards, defs, uc)
	if len(got) != 3 {
		t.Fatalf("got %d cards", len(got))
	}
	if got[0].Assignee != "alice" {
		t.Errorf("c1 Assignee = %q, want %q", got[0].Assignee, "alice")
	}
	if got[0].Status != "Not Started" {
		t.Errorf("c1 Status = %q, want %q", got[0].Status, "Not Started")
	}
	if got[1].Assignee != "" {
		t.Errorf("c2 (empty value) Assignee = %q, want empty", got[1].Assignee)
	}
	if got[2].Assignee != "" {
		t.Errorf("c3 (missing prop) Assignee = %q, want empty", got[2].Assignee)
	}

	// nil UserCache → raw user ID passes through unchanged (back-compat).
	got2 := ResolveCards(cards, defs, nil)
	if got2[0].Assignee != "user-id-123" {
		t.Errorf("with nil uc, expected raw ID, got %q", got2[0].Assignee)
	}
}

func TestResolveCards_MultiPerson(t *testing.T) {
	defs := []PropertyDef{
		{ID: "p_assignee", Name: "Assignee", Type: "person"},
		{ID: "p_assignees", Name: "Assignees", Type: "multiPerson"},
	}
	uc := &UserCache{
		users: map[string]*User{
			"id-alice": {ID: "id-alice", Username: "alice"},
			"id-bob":   {ID: "id-bob", Username: "bob"},
			"id-carol": {ID: "id-carol", Username: "carol"},
		},
	}

	t.Run("multi-list joined with comma", func(t *testing.T) {
		card := &Card{ID: "c1", Properties: map[string]any{
			"p_assignees": []any{"id-alice", "id-bob"},
		}}
		got := ResolveCards([]*Card{card}, defs, uc)
		if got[0].Assignee != "alice, bob" {
			t.Errorf("got %q, want %q", got[0].Assignee, "alice, bob")
		}
	})

	t.Run("multi wins over empty single", func(t *testing.T) {
		// Both Assignee and Assignees are present; Assignees has the data.
		card := &Card{ID: "c2", Properties: map[string]any{
			"p_assignee":  "",
			"p_assignees": []any{"id-carol"},
		}}
		got := ResolveCards([]*Card{card}, defs, uc)
		if got[0].Assignee != "carol" {
			t.Errorf("got %q, want %q", got[0].Assignee, "carol")
		}
	})

	t.Run("single person passes through", func(t *testing.T) {
		card := &Card{ID: "c3", Properties: map[string]any{
			"p_assignee": "id-alice",
		}}
		got := ResolveCards([]*Card{card}, defs, uc)
		if got[0].Assignee != "alice" {
			t.Errorf("got %q, want %q", got[0].Assignee, "alice")
		}
	})

	t.Run("empty multiPerson array", func(t *testing.T) {
		card := &Card{ID: "c4", Properties: map[string]any{
			"p_assignees": []any{},
		}}
		got := ResolveCards([]*Card{card}, defs, uc)
		if got[0].Assignee != "" {
			t.Errorf("expected empty for empty array, got %q", got[0].Assignee)
		}
	})

	t.Run("nil uc returns raw IDs joined", func(t *testing.T) {
		card := &Card{ID: "c5", Properties: map[string]any{
			"p_assignees": []any{"id-alice", "id-bob"},
		}}
		got := ResolveCards([]*Card{card}, defs, nil)
		if got[0].Assignee != "id-alice, id-bob" {
			t.Errorf("got %q, want raw ids", got[0].Assignee)
		}
	})
}

func TestAtPrefix(t *testing.T) {
	if got := AtPrefix(""); got != "" {
		t.Errorf("AtPrefix(\"\") = %q, want empty", got)
	}
	if got := AtPrefix("alice"); got != "@alice" {
		t.Errorf("AtPrefix(\"alice\") = %q, want @alice", got)
	}
}

func TestSortByPriority(t *testing.T) {
	cards := []ResolvedCard{
		{Priority: "3. Low"},
		{Priority: "1. High"},
		{Priority: ""},
		{Priority: "2. Medium"},
	}

	SortByPriority(cards)

	expected := []string{"1. High", "2. Medium", "3. Low", ""}
	for i, want := range expected {
		if cards[i].Priority != want {
			t.Errorf("index %d: got %q, want %q", i, cards[i].Priority, want)
		}
	}
}

func TestRemoveFromContentOrder(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		got, changed := removeFromContentOrder([]any{"a", "b", "c"}, "b")
		if !changed {
			t.Error("changed should be true")
		}
		if len(got) != 2 || got[0] != "a" || got[1] != "c" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("missing-id-no-op", func(t *testing.T) {
		_, changed := removeFromContentOrder([]any{"a", "b"}, "z")
		if changed {
			t.Error("changed should be false when id absent")
		}
	})

	t.Run("nested-row-keeps-others", func(t *testing.T) {
		got, changed := removeFromContentOrder([]any{"a", []any{"b", "c"}, "d"}, "b")
		if !changed {
			t.Error("changed should be true")
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 entries, got %v", got)
		}
		row, ok := got[1].([]any)
		if !ok || len(row) != 1 || row[0] != "c" {
			t.Errorf("nested row not handled: got %v", got[1])
		}
	})

	t.Run("nested-row-empties", func(t *testing.T) {
		got, changed := removeFromContentOrder([]any{"a", []any{"b"}, "c"}, "b")
		if !changed {
			t.Error("changed should be true")
		}
		if len(got) != 2 || got[0] != "a" || got[1] != "c" {
			t.Errorf("empty row should be dropped, got %v", got)
		}
	})

	t.Run("string-slice-form", func(t *testing.T) {
		// Defensive: handle the typed form too.
		got, changed := removeFromContentOrder([]any{[]string{"x", "y"}, "z"}, "x")
		if !changed {
			t.Error("changed should be true")
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 entries, got %v", got)
		}
	})
}

func TestFilterMine(t *testing.T) {
	me := &User{ID: "u1", Username: "alice"}
	cards := []ResolvedCard{
		{Card: Card{ID: "c1"}, Assignee: "u1"},         // exact ID
		{Card: Card{ID: "c2"}, Assignee: "alice"},      // exact username
		{Card: Card{ID: "c3"}, Assignee: "Alice"},      // case-insensitive substring (multi-assignee)
		{Card: Card{ID: "c4"}, Assignee: "bob, alice"}, // substring inside multi-name
		{Card: Card{ID: "c5"}, Assignee: "bob"},        // not me
		{Card: Card{ID: "c6"}, Assignee: ""},           // unassigned
	}

	got := FilterMine(cards, me)
	wantIDs := []string{"c1", "c2", "c3", "c4"}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d cards, want %d", len(got), len(wantIDs))
	}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Errorf("position %d: got %s, want %s", i, got[i].ID, id)
		}
	}

	// Nil user is a no-op.
	if out := FilterMine(cards, nil); len(out) != len(cards) {
		t.Errorf("FilterMine(nil) should return all cards, got %d/%d", len(out), len(cards))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int64
		want    string
	}{
		{0, "00:00"},
		{59, "00:01"}, // ceiled
		{60, "00:01"},
		{61, "00:02"}, // ceiled
		{3600, "01:00"},
		{3661, "01:02"},
		{7200, "02:00"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.seconds)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	got := FormatTimestamp(0)
	if got != "" {
		t.Errorf("expected empty for 0, got %q", got)
	}

	got = FormatTimestamp(1712016000000) // some timestamp
	if got == "" {
		t.Error("expected non-empty for valid timestamp")
	}
}
