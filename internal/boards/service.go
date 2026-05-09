package boards

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"skate/internal/client"
)

type Service struct {
	client     *client.Client
	teamID     string
	boardCache map[string]*Board
}

func NewService(c *client.Client, teamID string) *Service {
	return &Service{
		client:     c,
		teamID:     teamID,
		boardCache: make(map[string]*Board),
	}
}

func (s *Service) GetMe() (*User, error) {
	data, err := s.client.Get("/users/me")
	if err != nil {
		return nil, err
	}
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("parsing user: %w", err)
	}
	return &user, nil
}

func (s *Service) ListUsers(teamID string) ([]*User, error) {
	data, err := s.client.Get("/teams/" + teamID + "/users")
	if err != nil {
		return nil, err
	}
	var users []*User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("parsing users: %w", err)
	}
	return users, nil
}

// ResolveUserRef returns a user ID for a reference that is either an existing
// user ID or a username. Lookup is case-insensitive on username. Returns the
// raw ref unchanged if no users match — the caller decides whether to treat
// that as an error or as a pre-resolved ID.
func (s *Service) ResolveUserRef(teamID, ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	users, err := s.ListUsers(teamID)
	if err != nil {
		return ref, err
	}
	lower := strings.ToLower(ref)
	for _, u := range users {
		if u.ID == ref {
			return u.ID, nil
		}
	}
	for _, u := range users {
		if strings.ToLower(u.Username) == lower {
			return u.ID, nil
		}
	}
	return ref, fmt.Errorf("no user matches %q (try 'skate users' to list)", ref)
}

func (s *Service) ListBoards() ([]*Board, error) {
	data, err := s.client.Get("/teams/" + s.teamID + "/boards")
	if err != nil {
		return nil, err
	}
	var boards []*Board
	if err := json.Unmarshal(data, &boards); err != nil {
		return nil, fmt.Errorf("parsing boards: %w", err)
	}
	// Filter out templates
	result := make([]*Board, 0, len(boards))
	for _, b := range boards {
		if !b.IsTemplate {
			result = append(result, b)
		}
	}
	return result, nil
}

func (s *Service) GetBoard(boardID string) (*Board, error) {
	if b, ok := s.boardCache[boardID]; ok {
		return b, nil
	}
	data, err := s.client.Get("/boards/" + boardID)
	if err != nil {
		return nil, err
	}
	var board Board
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("parsing board: %w", err)
	}
	s.boardCache[boardID] = &board
	return &board, nil
}

func (s *Service) ListCards(boardID string) ([]*Card, error) {
	data, err := s.client.Get("/boards/" + boardID + "/cards?per_page=200")
	if err != nil {
		return nil, err
	}
	var cards []*Card
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("parsing cards: %w", err)
	}
	// Filter out templates
	result := make([]*Card, 0, len(cards))
	for _, c := range cards {
		if !c.IsTemplate {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *Service) GetCard(cardID string) (*Card, error) {
	data, err := s.client.Get("/cards/" + cardID)
	if err != nil {
		return nil, err
	}
	var card Card
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("parsing card: %w", err)
	}
	return &card, nil
}

func (s *Service) GetBlocks(boardID, parentID string) ([]*Block, error) {
	path := "/boards/" + boardID + "/blocks"
	if parentID != "" {
		path += "?parent_id=" + url.QueryEscape(parentID)
	}
	data, err := s.client.Get(path)
	if err != nil {
		return nil, err
	}
	var blocks []*Block
	if err := json.Unmarshal(data, &blocks); err != nil {
		return nil, fmt.Errorf("parsing blocks: %w", err)
	}
	return blocks, nil
}

func (s *Service) PatchCard(cardID string, patch *CardPatch) (*Card, error) {
	data, err := s.client.Patch("/cards/"+cardID, patch)
	if err != nil {
		return nil, err
	}
	var card Card
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("parsing card: %w", err)
	}
	return &card, nil
}

func (s *Service) CreateCard(boardID string, card *Card) (*Card, error) {
	data, err := s.client.Post("/boards/"+boardID+"/cards", card)
	if err != nil {
		return nil, err
	}
	var created Card
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, fmt.Errorf("parsing card: %w", err)
	}
	return &created, nil
}

func (s *Service) CreateBlock(boardID string, blocks []*Block) ([]*Block, error) {
	data, err := s.client.Post("/boards/"+boardID+"/blocks", blocks)
	if err != nil {
		return nil, err
	}
	var created []*Block
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, fmt.Errorf("parsing blocks: %w", err)
	}
	return created, nil
}

// CreateContentBlock creates a block and appends it to the card's contentOrder
// so it appears in the frontend. Comments don't need this -- only content blocks
// (text, h1-h3, divider, checkbox, image).
func (s *Service) CreateContentBlock(boardID, cardID string, block *Block) (*Block, error) {
	created, err := s.CreateBlock(boardID, []*Block{block})
	if err != nil {
		return nil, err
	}
	if len(created) == 0 {
		return nil, fmt.Errorf("no block returned")
	}

	// Update card's contentOrder to include the new block
	card, err := s.GetCard(cardID)
	if err != nil {
		return created[0], nil // block created but contentOrder not updated
	}
	newOrder := append(card.ContentOrder, any(created[0].ID))
	s.PatchCard(cardID, &CardPatch{ContentOrder: newOrder})

	return created[0], nil
}

// UpdateBlockTitle PATCHes a block's title (the body text for text/comment
// blocks, the heading for h1-h3, the filename for attachments). Other block
// fields are unchanged.
func (s *Service) UpdateBlockTitle(boardID, blockID, title string) error {
	payload := map[string]any{"title": title}
	_, err := s.client.Patch("/boards/"+boardID+"/blocks/"+blockID, payload)
	return err
}

// DeleteBlock removes a block (content, comment, or attachment) from a board.
// If cardID is non-empty and the block was a content block, the card's
// contentOrder is patched to drop the deleted ID so the web UI doesn't show a
// phantom slot.
func (s *Service) DeleteBlock(boardID, cardID, blockID string) error {
	if _, err := s.client.Delete("/boards/" + boardID + "/blocks/" + blockID); err != nil {
		return err
	}
	if cardID == "" {
		return nil
	}
	card, err := s.GetCard(cardID)
	if err != nil {
		return nil // block deleted; contentOrder cleanup is best-effort
	}
	newOrder, changed := removeFromContentOrder(card.ContentOrder, blockID)
	if changed {
		s.PatchCard(cardID, &CardPatch{ContentOrder: newOrder})
	}
	return nil
}

// removeFromContentOrder strips blockID from a card's contentOrder, preserving
// the nested-row structure Focalboard supports. Returns (newOrder, changed).
func removeFromContentOrder(order []any, blockID string) ([]any, bool) {
	out := make([]any, 0, len(order))
	changed := false
	for _, e := range order {
		switch v := e.(type) {
		case string:
			if v == blockID {
				changed = true
				continue
			}
			out = append(out, v)
		case []any:
			row := make([]any, 0, len(v))
			for _, inner := range v {
				if s, ok := inner.(string); ok && s == blockID {
					changed = true
					continue
				}
				row = append(row, inner)
			}
			if len(row) > 0 {
				out = append(out, row)
			} else {
				changed = true // dropped an empty row
			}
		case []string:
			row := make([]string, 0, len(v))
			for _, s := range v {
				if s == blockID {
					changed = true
					continue
				}
				row = append(row, s)
			}
			if len(row) > 0 {
				out = append(out, row)
			} else {
				changed = true
			}
		default:
			out = append(out, e)
		}
	}
	return out, changed
}

func (s *Service) UploadFile(teamID, boardID, filePath string) (string, error) {
	data, err := s.client.Upload("/teams/"+teamID+"/"+boardID+"/files", filePath)
	if err != nil {
		return "", err
	}
	var resp FileUploadResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parsing upload response: %w", err)
	}
	return resp.FileID, nil
}

func (s *Service) DownloadFile(teamID, boardID, filename string) ([]byte, error) {
	return s.client.Download("/files/teams/" + teamID + "/" + boardID + "/" + url.PathEscape(filename))
}

func (s *Service) StartTimer(boardID, cardID string) (*StartTimerResponse, error) {
	data, err := s.client.Post("/boards/"+boardID+"/cards/"+cardID+"/timer/start", nil)
	if err != nil {
		return nil, err
	}
	var resp StartTimerResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing timer response: %w", err)
	}
	return &resp, nil
}

func (s *Service) StopTimer(entryID, notes string) (*TimeEntry, error) {
	payload := map[string]string{"notes": notes}
	data, err := s.client.Post("/time-entries/"+entryID+"/stop", payload)
	if err != nil {
		return nil, err
	}
	var entry TimeEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing timer: %w", err)
	}
	return &entry, nil
}

func (s *Service) GetRunningTimer() (*TimeEntry, error) {
	data, err := s.client.Get("/me/timer")
	if err != nil {
		return nil, err
	}
	if string(data) == "null" {
		return nil, nil
	}
	var entry TimeEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing timer: %w", err)
	}
	return &entry, nil
}

func (s *Service) AddManualTime(boardID, cardID string, durationSeconds, date int64, notes string) (*TimeEntry, error) {
	payload := map[string]any{
		"durationSeconds": durationSeconds,
		"date":            date,
		"notes":           notes,
	}
	data, err := s.client.Post("/boards/"+boardID+"/cards/"+cardID+"/time-entries", payload)
	if err != nil {
		return nil, err
	}
	var entry TimeEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing time entry: %w", err)
	}
	return &entry, nil
}

func (s *Service) GetTimeSummary(boardID, cardID string) ([]*TimeEntrySummary, error) {
	data, err := s.client.Get("/boards/" + boardID + "/cards/" + cardID + "/time-summary")
	if err != nil {
		return nil, err
	}
	var summaries []*TimeEntrySummary
	if err := json.Unmarshal(data, &summaries); err != nil {
		return nil, fmt.Errorf("parsing time summary: %w", err)
	}
	return summaries, nil
}

// Property resolution helpers

func ParsePropertyDefs(board *Board) []PropertyDef {
	var defs []PropertyDef
	for _, raw := range board.CardProperties {
		def := PropertyDef{
			ID:   getString(raw, "id"),
			Name: getString(raw, "name"),
			Type: getString(raw, "type"),
		}
		if opts, ok := raw["options"].([]any); ok {
			for i, o := range opts {
				if om, ok := o.(map[string]any); ok {
					def.Options = append(def.Options, PropertyOption{
						ID:    getString(om, "id"),
						Value: getString(om, "value"),
						Color: getString(om, "color"),
						Index: i,
					})
				}
			}
		}
		defs = append(defs, def)
	}
	return defs
}

func FindPropertyByName(defs []PropertyDef, name string) *PropertyDef {
	lower := strings.ToLower(name)
	for i, d := range defs {
		if strings.ToLower(d.Name) == lower {
			return &defs[i]
		}
	}
	return nil
}

func FindOptionByValue(def *PropertyDef, value string) *PropertyOption {
	lower := strings.ToLower(value)
	for i, o := range def.Options {
		if strings.ToLower(o.Value) == lower {
			return &def.Options[i]
		}
	}
	return nil
}

func ResolvePropertyValue(defs []PropertyDef, propID string, rawValue any) string {
	if rawValue == nil {
		return ""
	}
	for _, d := range defs {
		if d.ID == propID {
			if d.Type == "select" || d.Type == "multiSelect" {
				if s, ok := rawValue.(string); ok {
					for _, o := range d.Options {
						if o.ID == s {
							return o.Value
						}
					}
				}
			}
			break
		}
	}
	if s, ok := rawValue.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", rawValue)
}

// ResolveCards turns raw cards into ResolvedCard with select-option labels and
// (when uc is non-nil) usernames in place of user IDs for person-type
// properties. Pass uc=nil to skip username resolution — callers that just
// need IDs (filtering, comparisons) save the API round-trip.
//
// When a board has both "Assignee" (person) and "Assignees" (multiPerson),
// non-empty values from later properties overwrite earlier ones, so the
// multiPerson list wins when present. Empty values don't overwrite.
func ResolveCards(cards []*Card, defs []PropertyDef, uc *UserCache) []ResolvedCard {
	resolved := make([]ResolvedCard, 0, len(cards))
	for _, c := range cards {
		rc := ResolvedCard{Card: *c}
		for _, d := range defs {
			val, ok := c.Properties[d.ID]
			if !ok {
				continue
			}
			var rv string
			if d.Type == "person" || d.Type == "multiPerson" {
				rv = resolvePersonValue(val, uc)
			} else {
				rv = ResolvePropertyValue(defs, d.ID, val)
			}
			if rv == "" {
				continue
			}
			switch strings.ToLower(d.Name) {
			case "status":
				rc.Status = rv
			case "priority":
				rc.Priority = rv
			case "assignee", "assignees":
				rc.Assignee = rv
			case "due date", "duedate", "due_date":
				rc.DueDate = rv
			}
		}
		resolved = append(resolved, rc)
	}
	return resolved
}

// resolvePersonValue handles both person (string user ID) and multiPerson
// (array of user IDs) property values. With uc non-nil, IDs are resolved to
// usernames; otherwise raw IDs are returned. Multiple users are joined with
// ", ".
func resolvePersonValue(raw any, uc *UserCache) string {
	var ids []string
	switch v := raw.(type) {
	case string:
		if v != "" {
			ids = append(ids, v)
		}
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok && s != "" {
				ids = append(ids, s)
			}
		}
	case []string:
		for _, s := range v {
			if s != "" {
				ids = append(ids, s)
			}
		}
	}
	if len(ids) == 0 {
		return ""
	}
	if uc == nil {
		return strings.Join(ids, ", ")
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, uc.Resolve(id))
	}
	return strings.Join(names, ", ")
}

// AtPrefix returns "@name" for non-empty input, or "" so empty cells stay
// blank in tabular and list output.
func AtPrefix(name string) string {
	if name == "" {
		return ""
	}
	return "@" + name
}

// FilterMine keeps only cards whose Assignee matches the given user.
// Matches user ID, exact username, or username substring (case-insensitive)
// to handle multi-assignee fields that store joined names.
func FilterMine(cards []ResolvedCard, me *User) []ResolvedCard {
	if me == nil {
		return cards
	}
	username := strings.ToLower(me.Username)
	out := make([]ResolvedCard, 0, len(cards))
	for _, rc := range cards {
		if rc.Assignee == me.ID || rc.Assignee == me.Username ||
			(username != "" && strings.Contains(strings.ToLower(rc.Assignee), username)) {
			out = append(out, rc)
		}
	}
	return out
}

func SortByPriority(cards []ResolvedCard) {
	priorityOrder := map[string]int{
		"urgent": 0, "1. high": 1, "high": 1,
		"2. medium": 2, "medium": 2,
		"3. low": 3, "low": 3,
		"": 99,
	}
	sort.SliceStable(cards, func(i, j int) bool {
		pi := priorityOrder[strings.ToLower(cards[i].Priority)]
		pj := priorityOrder[strings.ToLower(cards[j].Priority)]
		if pi == 0 && pj == 0 {
			return cards[i].Priority < cards[j].Priority
		}
		return pi < pj
	})
}

func FormatDuration(seconds int64) string {
	totalMinutes := int64(math.Ceil(float64(seconds) / 60))
	h := totalMinutes / 60
	m := totalMinutes % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func FormatTimestamp(millis int64) string {
	if millis == 0 {
		return ""
	}
	return time.UnixMilli(millis).Format("Jan 2, 2006 15:04")
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
