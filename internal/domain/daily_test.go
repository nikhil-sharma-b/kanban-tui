package domain

import "testing"

func TestNewWorkspaceHasSingleDailyBoard(t *testing.T) {
	workspace := NewWorkspace()
	daily := workspace.DailyProject()
	if daily == nil {
		t.Fatal("expected daily project")
	}
	if daily.Name != DailyProjectName {
		t.Fatalf("daily name = %q, want %q", daily.Name, DailyProjectName)
	}
	if got := daily.Board.Statuses(); len(got) != 3 || got[0] != StatusWaiting || got[1] != StatusActive || got[2] != StatusNext {
		t.Fatalf("unexpected daily columns: %v", got)
	}
	if len(workspace.RegularProjects()) != 1 {
		t.Fatalf("regular projects = %d, want 1", len(workspace.RegularProjects()))
	}
	if workspace.ActiveProjectID == daily.ID {
		t.Fatal("daily board should not be the initial active project")
	}
}

func TestPromoteDailyTaskMovesTaskToProjectColumn(t *testing.T) {
	workspace := NewWorkspace()
	daily := workspace.DailyProject()
	target := workspace.ActiveProject()
	task, err := daily.Board.AddTask("Plan release", "Write the checklist")
	if err != nil {
		t.Fatalf("add daily task: %v", err)
	}
	task.Whiteboards = []Whiteboard{{ID: "whiteboard-1", Name: "Plan"}}
	createdAt := task.CreatedAt

	promoted, err := workspace.PromoteDailyTask(task.ID, target.ID, StatusInProgress)
	if err != nil {
		t.Fatalf("promote daily task: %v", err)
	}
	if promoted != task {
		t.Fatal("promotion should preserve the task")
	}
	if daily.Board.Tasks[task.ID] != nil {
		t.Fatal("task remained on daily board")
	}
	if target.Board.Tasks[task.ID] != task {
		t.Fatal("task missing from target project")
	}
	if task.Status != StatusInProgress {
		t.Fatalf("status = %q, want %q", task.Status, StatusInProgress)
	}
	if len(target.Board.Order[StatusInProgress]) != 1 || target.Board.Order[StatusInProgress][0] != task.ID {
		t.Fatalf("unexpected target order: %v", target.Board.Order[StatusInProgress])
	}
	if task.Title != "Plan release" || task.Description != "Write the checklist" || !task.CreatedAt.Equal(createdAt) || len(task.Whiteboards) != 1 {
		t.Fatalf("task data not preserved: %+v", task)
	}
	if !task.UpdatedAt.After(createdAt) && !task.UpdatedAt.Equal(createdAt) {
		t.Fatal("updated timestamp moved backwards")
	}
}

func TestPromoteDailyTaskRejectsInvalidTargetWithoutMovingTask(t *testing.T) {
	workspace := NewWorkspace()
	daily := workspace.DailyProject()
	task, err := daily.Board.AddTask("Keep daily", "")
	if err != nil {
		t.Fatalf("add daily task: %v", err)
	}
	originalUpdatedAt := task.UpdatedAt

	if _, err := workspace.PromoteDailyTask(task.ID, "missing", StatusBacklog); err == nil {
		t.Fatal("expected missing target error")
	}
	if daily.Board.Tasks[task.ID] != task {
		t.Fatal("failed promotion removed task")
	}
	if !task.UpdatedAt.Equal(originalUpdatedAt) {
		t.Fatalf("failed promotion changed timestamp from %v to %v", originalUpdatedAt, task.UpdatedAt)
	}
	if _, err := workspace.PromoteDailyTask(task.ID, workspace.ActiveProject().ID, Status("Missing")); err == nil {
		t.Fatal("expected missing column error")
	}
	if _, err := workspace.PromoteDailyTask(task.ID, daily.ID, StatusWaiting); err == nil {
		t.Fatal("expected daily target error")
	}
}

func TestNormalizeAddsMissingDailyBoard(t *testing.T) {
	project, err := NewProject("Work")
	if err != nil {
		t.Fatalf("new project: %v", err)
	}
	workspace := &Workspace{Projects: []*Project{project}, ActiveProjectID: project.ID}

	if err := workspace.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if workspace.DailyProject() == nil {
		t.Fatal("expected daily board to be created")
	}
	if workspace.ActiveProjectID != project.ID {
		t.Fatalf("active project changed: %s", workspace.ActiveProjectID)
	}
}

func TestNormalizeKeepsOnlyOneDailyBoard(t *testing.T) {
	first := NewDailyProject()
	second := NewDailyProject()
	second.Name = "Daily copy"
	workspace := &Workspace{Projects: []*Project{first, second}, ActiveProjectID: first.ID}

	if err := workspace.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	dailyCount := 0
	for _, project := range workspace.Projects {
		if project.Daily {
			dailyCount++
		}
	}
	if dailyCount != 1 {
		t.Fatalf("daily projects = %d, want 1", dailyCount)
	}
	if workspace.DailyProject().ID != first.ID {
		t.Fatal("expected the first daily project to stay daily")
	}
	if len(workspace.RegularProjects()) == 0 {
		t.Fatal("expected at least one regular project")
	}
}

func TestNormalizeDailyBoardClampsForeignStatuses(t *testing.T) {
	daily := NewDailyProject()
	task, err := daily.Board.AddTask("stray", "")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	task.Status = StatusInProgress
	daily.Board.Order = map[Status][]string{StatusInProgress: {task.ID}}

	workspace := &Workspace{Projects: []*Project{daily}}
	if err := workspace.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}

	board := workspace.DailyProject().Board
	if got := board.Statuses(); len(got) != 3 {
		t.Fatalf("unexpected daily columns: %v", got)
	}
	if task.Status != StatusWaiting {
		t.Fatalf("task status = %q, want %q", task.Status, StatusWaiting)
	}
	if got := board.Order[StatusWaiting]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected waiting order: %v", got)
	}
}

func TestDailyProjectCannotBeRenamedOrDeleted(t *testing.T) {
	workspace := NewWorkspace()
	daily := workspace.DailyProject()

	if _, err := workspace.RenameProject(daily.ID, "Nope"); err == nil {
		t.Fatal("expected rename to fail")
	}
	if err := workspace.DeleteProject(daily.ID); err == nil {
		t.Fatal("expected delete to fail")
	}
	if workspace.DailyProject() == nil {
		t.Fatal("daily board disappeared")
	}
}

func TestDeleteLastRegularProjectFails(t *testing.T) {
	workspace := NewWorkspace()
	regular := workspace.RegularProjects()[0]

	if err := workspace.DeleteProject(regular.ID); err == nil {
		t.Fatal("expected deleting the last regular project to fail")
	}
}

func TestDeleteActiveProjectFallsBackToRegularProject(t *testing.T) {
	workspace := NewWorkspace()
	first := workspace.RegularProjects()[0]
	if _, err := workspace.CreateProject("Work"); err != nil {
		t.Fatalf("create project: %v", err)
	}
	workspace.SetActiveProject(first.ID)

	if err := workspace.DeleteProject(first.ID); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	active := workspace.ActiveProject()
	if active == nil || active.Daily {
		t.Fatalf("unexpected active project after delete: %+v", active)
	}
}

func TestBoardClearRemovesEveryTask(t *testing.T) {
	board := NewDailyBoard()
	first, err := board.AddTask("one", "")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	if _, err := board.AddTask("two", ""); err != nil {
		t.Fatalf("add task: %v", err)
	}
	if _, err := board.ArchiveTask(first.ID); err != nil {
		t.Fatalf("archive task: %v", err)
	}

	if got := board.Clear(); got != 2 {
		t.Fatalf("cleared = %d, want 2", got)
	}
	if len(board.Tasks) != 0 {
		t.Fatalf("tasks left: %d", len(board.Tasks))
	}
	for _, status := range board.Statuses() {
		if len(board.Order[status]) != 0 {
			t.Fatalf("order for %s not empty", status)
		}
	}
	if len(board.ArchivedTasks()) != 0 {
		t.Fatal("archived tasks left after clear")
	}
}
