package boards

type Board struct {
	ID             string           `json:"id"`
	TeamID         string           `json:"teamId"`
	ChannelID      string           `json:"channelId,omitempty"`
	CreatedBy      string           `json:"createdBy"`
	ModifiedBy     string           `json:"modifiedBy"`
	Type           string           `json:"type"`
	MinimumRole    string           `json:"minimumRole"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Icon           string           `json:"icon"`
	IsTemplate     bool             `json:"isTemplate"`
	CardProperties []map[string]any `json:"cardProperties"`
	CreateAt       int64            `json:"createAt"`
	UpdateAt       int64            `json:"updateAt"`
	DeleteAt       int64            `json:"deleteAt"`
}

type Card struct {
	ID           string         `json:"id"`
	BoardID      string         `json:"boardId"`
	CreatedBy    string         `json:"createdBy"`
	ModifiedBy   string         `json:"modifiedBy"`
	Title        string         `json:"title"`
	ContentOrder []any          `json:"contentOrder"`
	Icon         string         `json:"icon"`
	IsTemplate   bool           `json:"isTemplate"`
	Properties   map[string]any `json:"properties"`
	CreateAt     int64          `json:"createAt"`
	UpdateAt     int64          `json:"updateAt"`
	DeleteAt     int64          `json:"deleteAt"`
}

type CardPatch struct {
	Title             *string        `json:"title,omitempty"`
	Icon              *string        `json:"icon,omitempty"`
	ContentOrder      []any          `json:"contentOrder,omitempty"`
	UpdatedProperties map[string]any `json:"updatedProperties,omitempty"`
}

type Block struct {
	ID         string         `json:"id"`
	ParentID   string         `json:"parentId"`
	CreatedBy  string         `json:"createdBy"`
	ModifiedBy string         `json:"modifiedBy"`
	Schema     int64          `json:"schema"`
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Fields     map[string]any `json:"fields"`
	CreateAt   int64          `json:"createAt"`
	UpdateAt   int64          `json:"updateAt"`
	DeleteAt   int64          `json:"deleteAt"`
	BoardID    string         `json:"boardId"`
}

type TimeEntry struct {
	ID              string `json:"id"`
	BoardID         string `json:"boardId"`
	CardID          string `json:"cardId"`
	UserID          string `json:"userId"`
	BoardName       string `json:"boardName"`
	CardName        string `json:"cardName"`
	StartTime       int64  `json:"startTime"`
	EndTime         int64  `json:"endTime"`
	DurationSeconds int64  `json:"durationSeconds"`
	DurationMinutes int64  `json:"durationMinutes"`
	DurationDisplay string `json:"durationDisplay"`
	IsManual        bool   `json:"isManual"`
	IsRunning       bool   `json:"isRunning"`
	Notes           string `json:"notes"`
	CreateAt        int64  `json:"createAt"`
	UpdateAt        int64  `json:"updateAt"`
	DeleteAt        int64  `json:"deleteAt"`
}

type StartTimerResponse struct {
	Entry        *TimeEntry `json:"entry"`
	StoppedEntry *TimeEntry `json:"stoppedEntry,omitempty"`
}

type TimeEntrySummary struct {
	UserID       string     `json:"userId"`
	TotalSeconds int64      `json:"totalSeconds"`
	TotalMinutes int64      `json:"totalMinutes"`
	TotalDisplay string     `json:"totalDisplay"`
	RunningEntry *TimeEntry `json:"runningEntry,omitempty"`
}

type User struct {
	ID        string `json:"id" yaml:"id"`
	Username  string `json:"username" yaml:"username,omitempty"`
	Nickname  string `json:"nickname" yaml:"nickname,omitempty"`
	FirstName string `json:"firstname" yaml:"firstname,omitempty"`
	LastName  string `json:"lastname" yaml:"lastname,omitempty"`
}

type FileUploadResponse struct {
	FileID string `json:"fileId"`
}

type FileInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`
}

// ResolvedCard is a card with human-readable property values.
type ResolvedCard struct {
	Card
	Status   string
	Priority string
	Assignee string
	DueDate  string
}

// PropertyDef represents a board property definition.
type PropertyDef struct {
	ID      string
	Name    string
	Type    string
	Options []PropertyOption
}

// PropertyOption represents a select option.
type PropertyOption struct {
	ID    string
	Value string
	Color string
	Index int
}
