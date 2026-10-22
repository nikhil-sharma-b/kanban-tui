package cli

import (
	"flag"
	"fmt"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/whiteboardfs"
)

var groups = map[string]map[string]command{
	"project": {
		"list":   {"list", projectList},
		"show":   {"show [PROJECT]", projectShow},
		"create": {"create NAME", projectCreate},
		"rename": {"rename PROJECT NAME", projectRename},
		"delete": {"delete PROJECT --yes", projectDelete},
		"use":    {"use PROJECT", projectUse},
	},
	"column": {
		"list":   {"list [-p PROJECT]", columnList},
		"add":    {"add NAME [-p PROJECT]", columnAdd},
		"rename": {"rename COLUMN NAME [-p PROJECT]", columnRename},
		"move":   {"move COLUMN --position N [-p PROJECT]", columnMove},
		"delete": {"delete COLUMN [-p PROJECT]", columnDelete},
	},
	"task": {
		"list":    {"list [-p PROJECT] [--column COLUMN] [--search TEXT] [--archived]", taskList},
		"show":    {"show TASK", taskShow},
		"add":     {"add TITLE [-d DESCRIPTION] [-p PROJECT] [--column COLUMN]", taskAdd},
		"edit":    {"edit TASK [--title TITLE] [-d DESCRIPTION]", taskEdit},
		"move":    {"move TASK [--column COLUMN] [--position N]", taskMove},
		"archive": {"archive TASK", taskArchive},
		"restore": {"restore TASK", taskRestore},
		"delete":  {"delete TASK", taskDelete},
	},
	"daily": {
		"show":    {"show", dailyShow},
		"add":     {"add TITLE [-d DESCRIPTION] [--column COLUMN]", dailyAdd},
		"promote": {"promote TASK --to PROJECT [--column COLUMN]", dailyPromote},
	},
}

var groupOrder = map[string][]string{
	"project": {"list", "show", "create", "rename", "delete", "use"},
	"column":  {"list", "add", "rename", "move", "delete"},
	"task":    {"list", "show", "add", "edit", "move", "archive", "restore", "delete"},
	"daily":   {"show", "add", "promote"},
}

// --- projects ---

func projectList(a *app, args []string) error {
	if _, err := parse(a.flags("project list"), args, 0); err != nil {
		return err
	}
	return a.view(func(w *domain.Workspace) (any, error) {
		list := projectListView{}
		for _, project := range w.Projects {
			list = append(list, newProjectView(w, project))
		}
		return list, nil
	})
}

func projectShow(a *app, args []string) error {
	fs := a.flags("project show")
	ref := ""
	positional, err := parseOptional(fs, args, "PROJECT")
	if err != nil {
		return err
	}
	if len(positional) == 1 {
		ref = positional[0]
	}
	return a.view(func(w *domain.Workspace) (any, error) {
		project, err := resolveProject(w, ref)
		if err != nil {
			return nil, err
		}
		return newProjectView(w, project), nil
	})
}

func projectCreate(a *app, args []string) error {
	positional, err := parse(a.flags("project create"), args, 1, "NAME")
	if err != nil {
		return err
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		// CreateProject activates the new project; creating from the CLI
		// should not move the user's TUI default.
		active := w.ActiveProjectID
		project, err := w.CreateProject(positional[0])
		if err != nil {
			return change{}, err
		}
		w.SetActiveProject(active)
		return change{result: newProjectView(w, project)}, nil
	})
}

func projectRename(a *app, args []string) error {
	positional, err := parse(a.flags("project rename"), args, 2, "PROJECT", "NAME")
	if err != nil {
		return err
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		project, err := resolveProject(w, positional[0])
		if err != nil {
			return change{}, err
		}
		previous := whiteboardfs.SnapshotProjectPaths(project, a.dbPath)
		if _, err := w.RenameProject(project.ID, positional[1]); err != nil {
			return change{}, err
		}
		return change{
			result: newProjectView(w, project),
			commit: func() error { return whiteboardfs.RelocateProjectFiles(project, a.dbPath, previous) },
		}, nil
	})
}

func projectDelete(a *app, args []string) error {
	fs := a.flags("project delete")
	yes := fs.Bool("yes", false, "confirm deleting the project and all its tasks")
	positional, err := parse(fs, args, 1, "PROJECT")
	if err != nil {
		return err
	}
	if !*yes {
		return usagef("deleting a project removes all its tasks; pass --yes to confirm")
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		project, err := resolveProject(w, positional[0])
		if err != nil {
			return change{}, err
		}
		if err := w.DeleteProject(project.ID); err != nil {
			return change{}, err
		}
		return change{result: deletedView{Deleted: project.ID, Kind: "project", Name: project.Name}}, nil
	})
}

func projectUse(a *app, args []string) error {
	positional, err := parse(a.flags("project use"), args, 1, "PROJECT")
	if err != nil {
		return err
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		project, err := resolveProject(w, positional[0])
		if err != nil {
			return change{}, err
		}
		w.SetActiveProject(project.ID)
		return change{result: newProjectView(w, project)}, nil
	})
}

// --- columns ---

func columnList(a *app, args []string) error {
	fs := a.flags("column list")
	projectRef := projectFlag(fs)
	if _, err := parse(fs, args, 0); err != nil {
		return err
	}
	return a.view(func(w *domain.Workspace) (any, error) {
		project, err := resolveProject(w, *projectRef)
		if err != nil {
			return nil, err
		}
		return newColumnListView(project), nil
	})
}

func columnAdd(a *app, args []string) error {
	fs := a.flags("column add")
	projectRef := projectFlag(fs)
	positional, err := parse(fs, args, 1, "NAME")
	if err != nil {
		return err
	}
	return a.mutateColumns(*projectRef, "add columns", func(board *domain.Board) error {
		_, err := board.AddColumn(positional[0])
		return err
	})
}

func columnRename(a *app, args []string) error {
	fs := a.flags("column rename")
	projectRef := projectFlag(fs)
	positional, err := parse(fs, args, 2, "COLUMN", "NAME")
	if err != nil {
		return err
	}
	return a.mutateColumns(*projectRef, "rename columns", func(board *domain.Board) error {
		status, err := resolveColumn(board, positional[0])
		if err != nil {
			return err
		}
		_, err = board.RenameColumn(string(status), positional[1])
		return err
	})
}

func columnMove(a *app, args []string) error {
	fs := a.flags("column move")
	projectRef := projectFlag(fs)
	pos := fs.Int("position", 0, "new 1-based position")
	positional, err := parse(fs, args, 1, "COLUMN")
	if err != nil {
		return err
	}
	if *pos < 1 {
		return usagef("--position must be 1 or more")
	}
	return a.mutateColumns(*projectRef, "move columns", func(board *domain.Board) error {
		status, err := resolveColumn(board, positional[0])
		if err != nil {
			return err
		}
		to := min(*pos, len(board.Statuses())) - 1
		from := board.StatusIndex(status)
		if from != to && !board.MoveColumn(from, to) {
			return fmt.Errorf("could not move column %s", status)
		}
		return nil
	})
}

func columnDelete(a *app, args []string) error {
	fs := a.flags("column delete")
	projectRef := projectFlag(fs)
	positional, err := parse(fs, args, 1, "COLUMN")
	if err != nil {
		return err
	}
	return a.mutateColumns(*projectRef, "delete columns", func(board *domain.Board) error {
		status, err := resolveColumn(board, positional[0])
		if err != nil {
			return err
		}
		return board.DeleteColumn(status)
	})
}

// mutateColumns applies a column change to a regular project and prints the
// resulting column list. The daily board's columns are fixed.
func (a *app) mutateColumns(projectRef, action string, fn func(*domain.Board) error) error {
	return a.mutate(func(w *domain.Workspace) (change, error) {
		project, err := resolveRegularProject(w, projectRef, action)
		if err != nil {
			return change{}, err
		}
		if err := fn(project.Board); err != nil {
			return change{}, err
		}
		project.Touch()
		return change{result: newColumnListView(project)}, nil
	})
}

// --- tasks ---

func taskList(a *app, args []string) error {
	fs := a.flags("task list")
	projectRef := projectFlag(fs)
	columnRef := fs.String("column", "", "only this column")
	search := fs.String("search", "", "only tasks whose title or description contains TEXT")
	archived := fs.Bool("archived", false, "list archived tasks instead")
	if _, err := parse(fs, args, 0); err != nil {
		return err
	}
	return a.view(func(w *domain.Workspace) (any, error) {
		project, err := resolveProject(w, *projectRef)
		if err != nil {
			return nil, err
		}
		if *archived {
			return newArchiveView(project, *search), nil
		}
		return newBoardView(project, *columnRef, *search)
	})
}

func newBoardView(project *domain.Project, columnRef, search string) (boardView, error) {
	statuses := project.Board.Statuses()
	if columnRef != "" {
		status, err := resolveColumn(project.Board, columnRef)
		if err != nil {
			return boardView{}, err
		}
		statuses = []domain.Status{status}
	}
	view := boardView{ProjectID: project.ID, Project: project.Name, Columns: []columnView{}}
	for _, status := range statuses {
		column := columnView{Name: string(status), Title: status.Title(), Tasks: []taskView{}}
		for _, id := range project.Board.Order[status] {
			task := project.Board.Tasks[id]
			if task != nil && matchesSearch(task, search) {
				column.Tasks = append(column.Tasks, newTaskView(project, task))
			}
		}
		view.Columns = append(view.Columns, column)
	}
	return view, nil
}

func newArchiveView(project *domain.Project, search string) archiveView {
	view := archiveView{ProjectID: project.ID, Project: project.Name, Tasks: []taskView{}}
	for _, task := range project.Board.ArchivedTasks() {
		if matchesSearch(task, search) {
			view.Tasks = append(view.Tasks, newTaskView(project, task))
		}
	}
	return view
}

func matchesSearch(task *domain.Task, search string) bool {
	return search == "" || containsFold(task.SearchText(), search)
}

func taskShow(a *app, args []string) error {
	positional, err := parse(a.flags("task show"), args, 1, "TASK")
	if err != nil {
		return err
	}
	return a.view(func(w *domain.Workspace) (any, error) {
		ref, err := resolveTask(w, positional[0])
		if err != nil {
			return nil, err
		}
		return newTaskView(ref.project, ref.task), nil
	})
}

func taskAdd(a *app, args []string) error {
	fs := a.flags("task add")
	projectRef := projectFlag(fs)
	return a.addTask(fs, args, func(w *domain.Workspace) (*domain.Project, error) {
		return resolveProject(w, *projectRef)
	})
}

// addTask is shared by `task add` and `daily add`; target picks the project.
func (a *app) addTask(fs *flag.FlagSet, args []string, target func(*domain.Workspace) (*domain.Project, error)) error {
	description := descriptionFlag(fs)
	columnRef := fs.String("column", "", "column to add to (default: first column)")
	positional, err := parse(fs, args, 1, "TITLE")
	if err != nil {
		return err
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		project, err := target(w)
		if err != nil {
			return change{}, err
		}
		var status domain.Status
		if *columnRef != "" {
			if status, err = resolveColumn(project.Board, *columnRef); err != nil {
				return change{}, err
			}
		}
		task, err := project.Board.AddTask(positional[0], *description)
		if err != nil {
			return change{}, err
		}
		if status != "" && status != task.Status {
			project.Board.MoveTask(task.ID, status, -1)
		}
		project.Touch()
		return change{result: newTaskView(project, task)}, nil
	})
}

func taskEdit(a *app, args []string) error {
	fs := a.flags("task edit")
	title := fs.String("title", "", "new title")
	description := descriptionFlag(fs)
	positional, err := parse(fs, args, 1, "TASK")
	if err != nil {
		return err
	}
	set := setFlags(fs)
	if !set["title"] && !set["d"] && !set["description"] {
		return usagef("task edit needs --title or -d")
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		ref, err := resolveTask(w, positional[0])
		if err != nil {
			return change{}, err
		}
		newTitle, newDescription := ref.task.Title, ref.task.Description
		if set["title"] {
			newTitle = *title
		}
		if set["d"] || set["description"] {
			newDescription = *description
		}
		task, err := ref.project.Board.UpdateTask(ref.task.ID, newTitle, newDescription)
		if err != nil {
			return change{}, err
		}
		ref.project.Touch()
		return change{result: newTaskView(ref.project, task)}, nil
	})
}

func taskMove(a *app, args []string) error {
	fs := a.flags("task move")
	columnRef := fs.String("column", "", "target column (default: current column)")
	pos := fs.Int("position", 0, "1-based position in the column (default: end)")
	positional, err := parse(fs, args, 1, "TASK")
	if err != nil {
		return err
	}
	if *columnRef == "" && *pos == 0 {
		return usagef("task move needs --column or --position")
	}
	index, err := position(*pos)
	if err != nil {
		return err
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		ref, err := resolveTask(w, positional[0])
		if err != nil {
			return change{}, err
		}
		if ref.task.Archived() {
			return change{}, fmt.Errorf("task %s is archived; restore it first", shortID(ref.task.ID))
		}
		status := ref.task.Status
		if *columnRef != "" {
			if status, err = resolveColumn(ref.project.Board, *columnRef); err != nil {
				return change{}, err
			}
		}
		if !ref.project.Board.MoveTask(ref.task.ID, status, index) {
			return change{}, fmt.Errorf("could not move task %s", shortID(ref.task.ID))
		}
		ref.project.Touch()
		return change{result: newTaskView(ref.project, ref.task)}, nil
	})
}

func taskArchive(a *app, args []string) error {
	return a.taskAction("task archive", args, func(board *domain.Board, id string) (*domain.Task, error) {
		return board.ArchiveTask(id)
	})
}

func taskRestore(a *app, args []string) error {
	return a.taskAction("task restore", args, func(board *domain.Board, id string) (*domain.Task, error) {
		return board.RestoreTask(id)
	})
}

func (a *app) taskAction(name string, args []string, fn func(*domain.Board, string) (*domain.Task, error)) error {
	positional, err := parse(a.flags(name), args, 1, "TASK")
	if err != nil {
		return err
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		ref, err := resolveTask(w, positional[0])
		if err != nil {
			return change{}, err
		}
		task, err := fn(ref.project.Board, ref.task.ID)
		if err != nil {
			return change{}, err
		}
		ref.project.Touch()
		return change{result: newTaskView(ref.project, task)}, nil
	})
}

func taskDelete(a *app, args []string) error {
	positional, err := parse(a.flags("task delete"), args, 1, "TASK")
	if err != nil {
		return err
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		ref, err := resolveTask(w, positional[0])
		if err != nil {
			return change{}, err
		}
		if !ref.project.Board.DeleteTask(ref.task.ID) {
			return change{}, fmt.Errorf("could not delete task %s", shortID(ref.task.ID))
		}
		ref.project.Touch()
		return change{result: deletedView{Deleted: ref.task.ID, Kind: "task", Name: ref.task.Title}}, nil
	})
}

// --- daily board ---

func dailyShow(a *app, args []string) error {
	if _, err := parse(a.flags("daily show"), args, 0); err != nil {
		return err
	}
	return a.view(func(w *domain.Workspace) (any, error) {
		daily, err := dailyProject(w)
		if err != nil {
			return nil, err
		}
		return newBoardView(daily, "", "")
	})
}

func dailyAdd(a *app, args []string) error {
	return a.addTask(a.flags("daily add"), args, dailyProject)
}

func dailyPromote(a *app, args []string) error {
	fs := a.flags("daily promote")
	to := fs.String("to", "", "target project")
	columnRef := fs.String("column", "", "target column (default: first column)")
	positional, err := parse(fs, args, 1, "TASK")
	if err != nil {
		return err
	}
	if *to == "" {
		return usagef("daily promote needs --to PROJECT")
	}
	return a.mutate(func(w *domain.Workspace) (change, error) {
		ref, err := resolveTask(w, positional[0])
		if err != nil {
			return change{}, err
		}
		if !ref.project.Daily {
			return change{}, fmt.Errorf("task %s is not on the daily board", shortID(ref.task.ID))
		}
		target, err := resolveRegularProject(w, *to, "promote")
		if err != nil {
			return change{}, err
		}
		status := target.Board.Statuses()[0]
		if *columnRef != "" {
			if status, err = resolveColumn(target.Board, *columnRef); err != nil {
				return change{}, err
			}
		}
		task, err := w.PromoteDailyTask(ref.task.ID, target.ID, status)
		if err != nil {
			return change{}, err
		}
		commit, rollback, err := whiteboardfs.StageTaskFiles(task, a.dbPath, target.Name)
		if err != nil {
			return change{}, fmt.Errorf("stage whiteboards: %w", err)
		}
		return change{result: newTaskView(target, task), commit: commit, rollback: rollback}, nil
	})
}

func dailyProject(w *domain.Workspace) (*domain.Project, error) {
	daily := w.DailyProject()
	if daily == nil {
		return nil, fmt.Errorf("daily board not found")
	}
	return daily, nil
}
