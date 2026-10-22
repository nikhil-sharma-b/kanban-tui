package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

// The view types are the CLI's output contract: JSON field names here are
// what agents and scripts depend on, so change them deliberately.

type projectView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Daily     bool      `json:"daily"`
	Active    bool      `json:"active"`
	Tasks     int       `json:"tasks"`
	Columns   []string  `json:"columns"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newProjectView(w *domain.Workspace, project *domain.Project) projectView {
	columns := make([]string, 0, len(project.Board.Statuses()))
	for _, status := range project.Board.Statuses() {
		columns = append(columns, string(status))
	}
	return projectView{
		ID:        project.ID,
		Name:      project.Name,
		Daily:     project.Daily,
		Active:    project.ID == w.ActiveProjectID,
		Tasks:     project.Board.ActiveTaskCount(),
		Columns:   columns,
		UpdatedAt: project.UpdatedAt,
	}
}

func (v projectView) printText(w io.Writer) {
	fmt.Fprintf(w, "%s  %s%s\n", shortID(v.ID), v.Name, projectMarkers(v))
	fmt.Fprintf(w, "columns: %s\n", strings.Join(v.Columns, ", "))
	fmt.Fprintf(w, "tasks: %d\n", v.Tasks)
}

func projectMarkers(v projectView) string {
	var markers []string
	if v.Active {
		markers = append(markers, "active")
	}
	if v.Daily {
		markers = append(markers, "daily")
	}
	if len(markers) == 0 {
		return ""
	}
	return " (" + strings.Join(markers, ", ") + ")"
}

type projectListView []projectView

func (v projectListView) printText(w io.Writer) {
	for _, project := range v {
		fmt.Fprintf(w, "%s  %-24s %3d tasks%s\n", shortID(project.ID), project.Name, project.Tasks, projectMarkers(project))
	}
}

type taskView struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Column       string     `json:"column"`
	ColumnTitle  string     `json:"column_title"`
	Position     int        `json:"position,omitempty"`
	ProjectID    string     `json:"project_id"`
	Project      string     `json:"project"`
	Archived     bool       `json:"archived"`
	ArchivedAt   *time.Time `json:"archived_at,omitempty"`
	ArchivedFrom string     `json:"archived_from,omitempty"`
	Whiteboards  []string   `json:"whiteboards,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func newTaskView(project *domain.Project, task *domain.Task) taskView {
	view := taskView{
		ID:           task.ID,
		Title:        task.Title,
		Description:  task.Description,
		Column:       string(task.Status),
		ColumnTitle:  task.Status.Title(),
		ProjectID:    project.ID,
		Project:      project.Name,
		Archived:     task.Archived(),
		ArchivedAt:   task.ArchivedAt,
		ArchivedFrom: string(task.ArchivedFrom),
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
	if !view.Archived {
		for i, id := range project.Board.Order[task.Status] {
			if id == task.ID {
				view.Position = i + 1
				break
			}
		}
	}
	for _, whiteboard := range task.Whiteboards {
		view.Whiteboards = append(view.Whiteboards, whiteboard.Path)
	}
	return view
}

func (v taskView) printText(w io.Writer) {
	fmt.Fprintf(w, "%s  %s\n", v.ID, v.Title)
	fmt.Fprintf(w, "project: %s\n", v.Project)
	if v.Archived {
		fmt.Fprintf(w, "status:  archived from %s\n", v.ArchivedFrom)
	} else {
		fmt.Fprintf(w, "column:  %s (#%d)\n", v.ColumnTitle, v.Position)
	}
	fmt.Fprintf(w, "updated: %s\n", v.UpdatedAt.Local().Format("2006-01-02 15:04"))
	for _, path := range v.Whiteboards {
		fmt.Fprintf(w, "whiteboard: %s\n", path)
	}
	if v.Description != "" {
		fmt.Fprintf(w, "\n%s\n", v.Description)
	}
}

type columnView struct {
	Name  string     `json:"name"`
	Title string     `json:"title"`
	Tasks []taskView `json:"tasks"`
}

type boardView struct {
	ProjectID string       `json:"project_id"`
	Project   string       `json:"project"`
	Columns   []columnView `json:"columns"`
}

func (v boardView) printText(w io.Writer) {
	fmt.Fprintf(w, "%s\n", v.Project)
	for _, column := range v.Columns {
		fmt.Fprintf(w, "\n%s (%d)\n", column.Title, len(column.Tasks))
		for _, task := range column.Tasks {
			fmt.Fprintf(w, "  %s  %s\n", shortID(task.ID), singleLine(task.Title))
		}
	}
}

type archiveView struct {
	ProjectID string     `json:"project_id"`
	Project   string     `json:"project"`
	Tasks     []taskView `json:"tasks"`
}

func (v archiveView) printText(w io.Writer) {
	fmt.Fprintf(w, "%s archive (%d)\n", v.Project, len(v.Tasks))
	for _, task := range v.Tasks {
		fmt.Fprintf(w, "  %s  %s  (from %s)\n", shortID(task.ID), singleLine(task.Title), task.ArchivedFrom)
	}
}

type columnListView struct {
	ProjectID string        `json:"project_id"`
	Project   string        `json:"project"`
	Columns   []columnBrief `json:"columns"`
}

type columnBrief struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Tasks int    `json:"tasks"`
}

func newColumnListView(project *domain.Project) columnListView {
	view := columnListView{ProjectID: project.ID, Project: project.Name, Columns: []columnBrief{}}
	for _, status := range project.Board.Statuses() {
		view.Columns = append(view.Columns, columnBrief{
			Name:  string(status),
			Title: status.Title(),
			Tasks: project.Board.Count(status),
		})
	}
	return view
}

func (v columnListView) printText(w io.Writer) {
	for i, column := range v.Columns {
		fmt.Fprintf(w, "%d. %-20s %3d tasks\n", i+1, column.Name, column.Tasks)
	}
}

// deletedView reports a removed task or project.
type deletedView struct {
	Deleted string `json:"deleted"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
}

func (v deletedView) printText(w io.Writer) {
	fmt.Fprintf(w, "deleted %s %s (%s)\n", v.Kind, v.Name, shortID(v.Deleted))
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
