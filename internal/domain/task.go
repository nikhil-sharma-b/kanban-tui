package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusBacklog    Status = "backlog"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

var StatusOrder = []Status{
	StatusBacklog,
	StatusInProgress,
	StatusDone,
}

func (s Status) Valid() bool {
	return strings.TrimSpace(string(s)) != ""
}

func (s Status) Title() string {
	trimmed := strings.TrimSpace(string(s))
	switch s {
	case StatusBacklog:
		return "Backlog"
	case StatusInProgress:
		return "In Progress"
	case StatusDone:
		return "Done"
	default:
		if trimmed == "" {
			return "Unknown"
		}
		return strings.ReplaceAll(trimmed, "_", " ")
	}
}

type Task struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      Status       `json:"status"`
	Whiteboards []Whiteboard `json:"whiteboards,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Whiteboard struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewTask(title, description string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	now := time.Now().UTC()
	return &Task{
		ID:          newTaskID(),
		Title:       title,
		Description: strings.TrimSpace(description),
		Status:      StatusBacklog,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (t *Task) Touch() {
	t.UpdatedAt = time.Now().UTC()
}

func (t *Task) SearchText() string {
	return strings.ToLower(t.Title + "\n" + t.Description)
}

func newTaskID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("tsk-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(buf)
}
