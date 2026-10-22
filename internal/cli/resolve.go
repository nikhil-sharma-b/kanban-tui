package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

// resolveProject finds a project by ID or case-insensitive name. An empty ref
// means the workspace's active project.
func resolveProject(w *domain.Workspace, ref string) (*domain.Project, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		if project := w.ActiveProject(); project != nil {
			return project, nil
		}
		return nil, fmt.Errorf("no active project")
	}
	if project := w.ProjectByID(ref); project != nil {
		return project, nil
	}
	for _, project := range w.Projects {
		if strings.EqualFold(project.Name, ref) {
			return project, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", ref)
}

// resolveRegularProject is resolveProject for commands the daily board does
// not support.
func resolveRegularProject(w *domain.Workspace, ref, action string) (*domain.Project, error) {
	project, err := resolveProject(w, ref)
	if err != nil {
		return nil, err
	}
	if project.Daily {
		return nil, fmt.Errorf("cannot %s on the daily board", action)
	}
	return project, nil
}

type taskRef struct {
	project *domain.Project
	task    *domain.Task
}

// resolveTask finds a task in any project by full ID or unique ID prefix.
// Task IDs are unique across the workspace, so no project is needed.
func resolveTask(w *domain.Workspace, ref string) (taskRef, error) {
	ref = strings.ToLower(strings.TrimSpace(ref))
	if ref == "" {
		return taskRef{}, usagef("task ID cannot be empty")
	}

	var matches []taskRef
	for _, project := range w.Projects {
		if project.Board == nil {
			continue
		}
		if task := project.Board.Tasks[ref]; task != nil {
			return taskRef{project: project, task: task}, nil
		}
		for id, task := range project.Board.Tasks {
			if task != nil && strings.HasPrefix(id, ref) {
				matches = append(matches, taskRef{project: project, task: task})
			}
		}
	}

	switch len(matches) {
	case 0:
		return taskRef{}, fmt.Errorf("task %q not found", ref)
	case 1:
		return matches[0], nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].task.ID < matches[j].task.ID })
	lines := make([]string, len(matches))
	for i, match := range matches {
		lines[i] = fmt.Sprintf("  %s  %s / %s", match.task.ID, match.project.Name, singleLine(match.task.Title))
	}
	return taskRef{}, fmt.Errorf("task ID %q is ambiguous:\n%s", ref, strings.Join(lines, "\n"))
}

// resolveColumn finds a column by its key or title, ignoring case and
// treating spaces, dashes and underscores alike: "in progress",
// "in-progress" and "in_progress" all match the In Progress column.
func resolveColumn(board *domain.Board, ref string) (domain.Status, error) {
	want := columnKey(ref)
	if want == "" {
		return "", usagef("column cannot be empty")
	}
	for _, status := range board.Statuses() {
		if columnKey(string(status)) == want || columnKey(status.Title()) == want {
			return status, nil
		}
	}
	names := make([]string, 0, len(board.Statuses()))
	for _, status := range board.Statuses() {
		names = append(names, string(status))
	}
	return "", fmt.Errorf("column %q not found (columns: %s)", ref, strings.Join(names, ", "))
}

func columnKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.NewReplacer(" ", "", "_", "", "-", "").Replace(value)
}

// position converts a 1-based position flag into a board index; 0 means the
// end of the column.
func position(value int) (int, error) {
	if value < 0 {
		return 0, usagef("--position must be 1 or more")
	}
	if value == 0 {
		return -1, nil
	}
	return value - 1, nil
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
