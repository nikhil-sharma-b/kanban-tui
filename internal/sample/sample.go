// Package sample builds the fixture workspaces the Dusk frames show, with
// timestamps relative to a fixed clock so relative times render the same
// every run.
package sample

import (
	"fmt"
	"time"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

// Now is the fixed clock every frame scenario uses.
var Now = time.Date(2026, time.November, 20, 10, 0, 0, 0, time.UTC)

// Columns of the "home" project in every frame.
var Columns = []domain.Status{"backlog", "todo", "doing", "done"}

type taskSpec struct {
	id, title, desc string
	age             time.Duration
	whiteboards     int
}

// Workspace reproduces the data shown in 01/default: project "home" with 14
// tasks over four columns, and a daily board holding 3 tasks.
func Workspace(now time.Time) *domain.Workspace {
	home := project("home", now, map[domain.Status][]taskSpec{
		"backlog": {
			{title: "Export board as markdown", age: 9 * 24 * time.Hour},
			{title: "Recurring daily tasks", age: 8 * 24 * time.Hour},
			{title: "Sync whiteboards to iCloud", age: 7 * 24 * time.Hour},
			{title: "Due dates (maybe)", age: 6 * 24 * time.Hour},
			{title: "Theme config file", age: 5 * 24 * time.Hour},
		},
		"todo": {
			{title: "Archive search by date", age: 4 * 24 * time.Hour},
			{id: "a3f91c07e2b84d6f9a1c3e5b7d902468", title: "Undo for move and delete", desc: "Undo the last move or delete with u, from a toast", age: 2 * 24 * time.Hour, whiteboards: 2},
			{title: "Column WIP limits", age: 3 * 24 * time.Hour},
		},
		"doing": {
			{title: "Live reload from CLI writes", age: 3 * time.Hour},
			{title: "Revamp the TUI", age: 1 * time.Hour},
		},
		"done": {
			{title: "Copy task ID keybind", age: 10 * 24 * time.Hour},
			{title: "Agent skill for CLI", age: 11 * 24 * time.Hour},
			{title: "Makefile install", age: 12 * 24 * time.Hour},
			{title: "Wider projects dialog", age: 13 * 24 * time.Hour},
		},
	})
	daily := dailyProject(now, map[domain.Status][]taskSpec{
		domain.StatusWaiting: {{title: "Reply to design feedback", age: 2 * time.Hour}},
		domain.StatusActive:  {{title: "Review the frame ledger", age: time.Hour}},
		domain.StatusNext:    {{title: "Plan tomorrow", age: 30 * time.Minute}},
	})
	return workspace(home, daily)
}

// EmptyBoard is 01/empty-board: project "home" with its four columns and no
// tasks anywhere, the daily board included.
func EmptyBoard(now time.Time) *domain.Workspace {
	return workspace(project("home", now, nil), dailyProject(now, nil))
}

func workspace(projects ...*domain.Project) *domain.Workspace {
	return &domain.Workspace{Version: 1, Projects: projects, ActiveProjectID: projects[0].ID}
}

func project(name string, now time.Time, tasks map[domain.Status][]taskSpec) *domain.Project {
	board := &domain.Board{
		Version: 1,
		Columns: append([]domain.Status(nil), Columns...),
		Tasks:   map[string]*domain.Task{},
		Order:   map[domain.Status][]string{},
	}
	fill(board, now, name, tasks)
	created := now.Add(-30 * 24 * time.Hour)
	return &domain.Project{ID: "p-" + name, Name: name, Board: board, CreatedAt: created, UpdatedAt: now}
}

func dailyProject(now time.Time, tasks map[domain.Status][]taskSpec) *domain.Project {
	board := domain.NewDailyBoard()
	fill(board, now, "daily", tasks)
	created := now.Add(-30 * 24 * time.Hour)
	return &domain.Project{ID: "p-daily", Name: domain.DailyProjectName, Daily: true, Board: board, CreatedAt: created, UpdatedAt: now}
}

func fill(board *domain.Board, now time.Time, prefix string, tasks map[domain.Status][]taskSpec) {
	n := 0
	for _, status := range board.Columns {
		board.Order[status] = []string{}
		for _, spec := range tasks[status] {
			n++
			id := spec.id
			if id == "" {
				id = fmt.Sprintf("%08x%024d", hash(prefix)+uint32(n), n)
			}
			at := now.Add(-spec.age)
			task := &domain.Task{ID: id, Title: spec.title, Description: spec.desc, Status: status, CreatedAt: at, UpdatedAt: at}
			for w := 1; w <= spec.whiteboards; w++ {
				task.Whiteboards = append(task.Whiteboards, domain.Whiteboard{
					ID: fmt.Sprintf("%s-wb%d", id[:8], w), Name: fmt.Sprintf("sketch %d", w), Format: "xopp", CreatedAt: at, UpdatedAt: at,
				})
			}
			board.Tasks[id] = task
			board.Order[status] = append(board.Order[status], id)
		}
	}
}

func hash(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h
}
