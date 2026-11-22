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

// DetailTaskID identifies the sample task featured in the detail frames.
const DetailTaskID = "a3f91c07e2b84d6f9a1c3e5b7d902468"

type taskSpec struct {
	id, title, desc string
	age             time.Duration
	whiteboards     int
}

// Workspace reproduces the data shown in 01/default: project "home" with 14
// tasks over four columns, and a daily board holding 3 tasks.
func Workspace(now time.Time) *domain.Workspace {
	return workspace(project("home", now, homeTasks()), dailyWithTasks(now))
}

// TaskDetail is the home board with the full task description and whiteboard
// metadata shown in the detail frames.
func TaskDetail(now time.Time) *domain.Workspace {
	w := Workspace(now)
	t := w.ActiveProject().Board.Tasks[DetailTaskID]
	t.Description = "Undo the last move or delete with u. Keep a short history in memory only, so a mistaken H/L or delete can be reverted without touching the store format. Limit to the last 20 actions; older ones drop off. Show what was undone in the status line, e.g. “undid move → doing”. Redo with ctrl+r while the history is intact. Clear it on project switch and on live reload so undo never runs against a board that changed underneath. Nothing about the stack is written to disk; quitting drops it. If the target task no longer exists, skip the entry and say so in the status line instead of failing silently."
	t.Whiteboards[0].Name = "undo-flow"
	t.Whiteboards[0].UpdatedAt = now.Add(-72 * time.Hour)
	t.Whiteboards[1].Name = "history-stack"
	t.Whiteboards[1].Format = "rnote"
	date := now.UTC().AddDate(0, 0, -8)
	t.Whiteboards[1].UpdatedAt = time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, time.Local)
	return w
}

func TaskDetailWithoutWhiteboards(now time.Time) *domain.Workspace {
	w := TaskDetail(now)
	w.ActiveProject().Board.Tasks[DetailTaskID].Whiteboards = nil
	return w
}

// SearchFilter is the home board as page 05 Search and filter shows it: the
// two tasks matching "CLI" carry the descriptions and ages its frames show.
func SearchFilter(now time.Time) *domain.Workspace {
	tasks := homeTasks()
	tasks["doing"][0] = taskSpec{id: "e2d81b4a6c094f1e8b3d5a7c9e1f2468", title: "Live reload from CLI writes", desc: "Repaint on data file change", age: 30 * time.Hour}
	tasks["done"][1] = taskSpec{id: "d8140e2b7a3c4d5e9f1a2b3c4d5e6f70", title: "Agent skill for CLI", desc: "Document the agent workflow", age: 5 * 24 * time.Hour}
	return workspace(project("home", now, tasks), dailyWithTasks(now))
}

// DailyBoard is the home workspace with the daily board 11/daily shows.
func DailyBoard(now time.Time) *domain.Workspace {
	daily := dailyProject(now, map[domain.Status][]taskSpec{
		domain.StatusWaiting: {{title: "Hear back from IT on laptop", age: 3 * time.Hour}, {title: "Design review sign-off", age: 2 * time.Hour}},
		domain.StatusActive:  {{title: "Reply to standup thread", age: 90 * time.Minute}, {id: "d41e2b90c7a54f1e8b3d5a7c9e1f2468", title: "Fix flaky CI test", desc: "Retries hide a race in the store", age: time.Hour}},
		domain.StatusNext:    {{title: "Book dentist", age: 30 * time.Minute}},
	})
	return workspace(project("home", now, homeTasks()), daily)
}

// ArchiveOld is the home board with every done task untouched for over 30
// days, as 09/archive-old counts them.
func ArchiveOld(now time.Time) *domain.Workspace {
	tasks := homeTasks()
	for i := range tasks["done"] {
		tasks["done"][i].age += 30 * 24 * time.Hour
	}
	return workspace(project("home", now, tasks), dailyWithTasks(now))
}

// WithWorkProject adds project "work" to the home workspace: 14 tasks, 5 of
// them archived, as 09/delete-project describes it.
func WithWorkProject(now time.Time) *domain.Workspace {
	specs := map[domain.Status][]taskSpec{}
	for i := 1; i <= 14; i++ {
		status := Columns[i%len(Columns)]
		specs[status] = append(specs[status], taskSpec{title: fmt.Sprintf("work task %d", i), age: time.Duration(i) * 24 * time.Hour})
	}
	work := project("work", now, specs)
	for _, status := range Columns[:2] {
		ids := work.Board.Order[status]
		n := min(len(ids), 5-archivedCount(work.Board))
		for _, id := range ids[:n] {
			at := now.Add(-time.Hour)
			task := work.Board.Tasks[id]
			task.ArchivedAt, task.ArchivedFrom = &at, task.Status
		}
		work.Board.Order[status] = ids[n:]
	}
	return workspace(project("home", now, homeTasks()), work, dailyWithTasks(now))
}

func archivedCount(b *domain.Board) int {
	n := 0
	for _, t := range b.Tasks {
		if t.Archived() {
			n++
		}
	}
	return n
}

func homeTasks() map[domain.Status][]taskSpec {
	return map[domain.Status][]taskSpec{
		"backlog": {
			{title: "Export board as markdown", age: 9 * 24 * time.Hour},
			{title: "Recurring daily tasks", age: 8 * 24 * time.Hour},
			{title: "Sync whiteboards to iCloud", age: 7 * 24 * time.Hour},
			{title: "Due dates (maybe)", age: 6 * 24 * time.Hour},
			{title: "Theme config file", age: 5 * 24 * time.Hour},
		},
		"todo": {
			{title: "Archive search by date", age: 4 * 24 * time.Hour},
			{id: DetailTaskID, title: "Undo for move and delete", desc: "Undo the last move or delete", age: 2 * 24 * time.Hour, whiteboards: 2},
			{title: "Column WIP limits", age: 3 * 24 * time.Hour},
		},
		"doing": {
			{id: "7be20d14c6a94f0e8b3d5a7c9e1f2468", title: "Live reload from CLI writes", desc: "Watch board.db and refresh", age: 5 * time.Hour},
			{title: "Revamp the TUI", age: 1 * time.Hour},
		},
		"done": {
			{title: "Copy task ID keybind", age: 10 * 24 * time.Hour},
			{title: "Agent skill for CLI", age: 11 * 24 * time.Hour},
			{title: "Makefile install", age: 12 * 24 * time.Hour},
			{title: "Wider projects dialog", age: 13 * 24 * time.Hour},
		},
	}
}

func dailyWithTasks(now time.Time) *domain.Project {
	return dailyProject(now, map[domain.Status][]taskSpec{
		domain.StatusWaiting: {{title: "Reply to design feedback", age: 2 * time.Hour}},
		domain.StatusActive:  {{title: "Review the frame ledger", age: time.Hour}},
		domain.StatusNext:    {{title: "Plan tomorrow", age: 30 * time.Minute}},
	})
}

// EmptyColumn is 01/empty-column: 01/default with the doing column emptied.
func EmptyColumn(now time.Time) *domain.Workspace {
	tasks := homeTasks()
	tasks["doing"] = nil
	return workspace(project("home", now, tasks), dailyWithTasks(now))
}

// OverflowColumn is 01/overflow-column: 01/default with 16 backlog tasks, the
// 9th of which has a description and a whiteboard.
func OverflowColumn(now time.Time) *domain.Workspace {
	tasks := homeTasks()
	day := 24 * time.Hour
	backlog := []taskSpec{
		{title: "Keyboard macro recorder", age: 20 * day},
		{title: "Board statistics view", age: 19 * day},
	}
	backlog = append(backlog, tasks["backlog"]...)
	backlog = append(backlog,
		taskSpec{title: "Card colour labels", age: 4 * day},
		taskSpec{id: "b7c24f19d03e4a5b8c6d7e8f90a1b2c3", title: "Import from Trello JSON", desc: "Lists → columns, cards and checklists → descriptions", age: 5 * day, whiteboards: 1},
		taskSpec{title: "Export to CSV", age: 3 * day},
		taskSpec{title: "Bulk move with visual mode", age: 3 * day},
		taskSpec{title: "Pin a task to the top", age: 2 * day},
		taskSpec{title: "Task templates", age: 2 * day},
		taskSpec{title: "Snooze a task until a date", age: day},
		taskSpec{title: "Per-column sort order", age: day},
		taskSpec{title: "Print the board", age: 12 * time.Hour},
	)
	tasks["backlog"] = backlog
	return workspace(project("home", now, tasks), dailyWithTasks(now))
}

// MoreColumns is 01/more-columns: 01/default with a column before backlog and
// one after done, six in all, so only four fit at 120 cols.
func MoreColumns(now time.Time) *domain.Workspace {
	tasks := homeTasks()
	tasks["ideas"] = []taskSpec{{title: "Calendar view", age: 15 * 24 * time.Hour}}
	tasks["shipped"] = []taskSpec{{title: "SQLite storage", age: 30 * 24 * time.Hour}}
	home := project("home", now, tasks)
	home.Board.Columns = []domain.Status{"ideas", "backlog", "todo", "doing", "done", "shipped"}
	home.Board.Tasks = map[string]*domain.Task{}
	fill(home.Board, now, "home", tasks)
	return workspace(home, dailyWithTasks(now))
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
