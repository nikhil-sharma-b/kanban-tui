package domain

import (
	"testing"
	"time"
)

func TestArchiveTaskRemovesFromOrderKeepsInTasks(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")

	archived, err := board.ArchiveTask(task.ID)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}

	if archived.ArchivedAt == nil {
		t.Fatal("expected ArchivedAt to be set")
	}
	if archived.ArchivedFrom != StatusBacklog {
		t.Fatalf("unexpected ArchivedFrom: %q", archived.ArchivedFrom)
	}
	if archived.Status != StatusBacklog {
		t.Fatalf("expected Status to keep last column, got %q", archived.Status)
	}
	if len(board.Order[StatusBacklog]) != 0 {
		t.Fatalf("expected empty backlog order, got %v", board.Order[StatusBacklog])
	}
	if board.Tasks[task.ID] == nil {
		t.Fatal("expected archived task to stay in Tasks map")
	}
}

func TestArchiveTaskMissingErrors(t *testing.T) {
	board := NewBoard()
	if _, err := board.ArchiveTask("nope"); err == nil {
		t.Fatal("expected error for missing task")
	}
	if _, err := board.RestoreTask("nope"); err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestArchiveTaskIdempotent(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")

	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	firstUpdated := task.UpdatedAt
	firstArchivedAt := *task.ArchivedAt

	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("second archive: %v", err)
	}
	if !task.UpdatedAt.Equal(firstUpdated) {
		t.Fatal("expected second archive not to retouch UpdatedAt")
	}
	if !task.ArchivedAt.Equal(firstArchivedAt) {
		t.Fatal("expected second archive not to change ArchivedAt")
	}
}

func TestRestoreTaskReturnsToOriginalColumn(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")
	board.MoveTask(task.ID, StatusDone, 0)
	other, _ := board.AddTask("other", "")
	board.MoveTask(other.ID, StatusDone, 1)

	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	restored, err := board.RestoreTask(task.ID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}

	if restored.Status != StatusDone {
		t.Fatalf("unexpected status: %q", restored.Status)
	}
	if restored.ArchivedAt != nil || restored.ArchivedFrom != "" {
		t.Fatal("expected archive metadata cleared")
	}
	got := board.Order[StatusDone]
	if len(got) != 2 || got[1] != task.ID {
		t.Fatalf("expected restored task appended once at end, got %v", got)
	}
}

func TestRestoreTaskIdempotent(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")
	updated := task.UpdatedAt

	if _, err := board.RestoreTask(task.ID); err != nil {
		t.Fatalf("restore active: %v", err)
	}
	if !task.UpdatedAt.Equal(updated) {
		t.Fatal("expected restore of active task not to retouch UpdatedAt")
	}
	if got := board.Order[StatusBacklog]; len(got) != 1 {
		t.Fatalf("expected single order entry, got %v", got)
	}
}

func TestRestoreTaskFallsBackToFirstColumn(t *testing.T) {
	board := NewBoard()
	review, _ := board.AddColumn("Review")
	task, _ := board.AddTask("task", "")
	board.MoveTask(task.ID, review, 0)

	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if err := board.DeleteColumn(review); err != nil {
		t.Fatalf("delete column: %v", err)
	}

	restored, err := board.RestoreTask(task.ID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Status != board.Columns[0] {
		t.Fatalf("expected fallback to first column, got %q", restored.Status)
	}
	if got := board.Order[board.Columns[0]]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected first column order: %v", got)
	}
}

func TestArchivedTasksSortedNewestFirst(t *testing.T) {
	board := NewBoard()
	first, _ := board.AddTask("first", "")
	second, _ := board.AddTask("second", "")

	if _, err := board.ArchiveTask(first.ID); err != nil {
		t.Fatalf("archive first: %v", err)
	}
	if _, err := board.ArchiveTask(second.ID); err != nil {
		t.Fatalf("archive second: %v", err)
	}
	older := first.ArchivedAt.Add(-time.Hour)
	first.ArchivedAt = &older

	archived := board.ArchivedTasks()
	if len(archived) != 2 {
		t.Fatalf("unexpected archived count: %d", len(archived))
	}
	if archived[0].ID != second.ID || archived[1].ID != first.ID {
		t.Fatalf("expected newest first, got %s, %s", archived[0].ID, archived[1].ID)
	}
}

func TestNormalizeExcludesArchivedFromOrder(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")
	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	// Simulate a stale order slice that still references the archived task.
	board.Order[StatusBacklog] = append(board.Order[StatusBacklog], task.ID)

	if err := board.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(board.Order[StatusBacklog]) != 0 {
		t.Fatalf("expected archived task removed from order, got %v", board.Order[StatusBacklog])
	}
	if board.Tasks[task.ID] == nil {
		t.Fatal("expected archived task to stay in Tasks map")
	}
}

func TestNormalizeDoesNotRecreateDeletedColumnFromArchived(t *testing.T) {
	board := NewBoard()
	review, _ := board.AddColumn("Review")
	task, _ := board.AddTask("task", "")
	board.MoveTask(task.ID, review, 0)

	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if err := board.DeleteColumn(review); err != nil {
		t.Fatalf("delete column: %v", err)
	}
	if err := board.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if board.StatusIndex(review) >= 0 {
		t.Fatalf("expected deleted column to stay deleted, columns: %v", board.Columns)
	}
}

func TestArchiveDoneOlderThanOnlyOldDone(t *testing.T) {
	board := NewBoard()
	oldDone, _ := board.AddTask("old done", "")
	board.MoveTask(oldDone.ID, StatusDone, 0)
	oldDone.UpdatedAt = time.Now().UTC().Add(-40 * 24 * time.Hour)

	freshDone, _ := board.AddTask("fresh done", "")
	board.MoveTask(freshDone.ID, StatusDone, 1)

	oldBacklog, _ := board.AddTask("old backlog", "")
	oldBacklog.UpdatedAt = time.Now().UTC().Add(-40 * 24 * time.Hour)

	count, err := board.ArchiveDoneOlderThan(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("bulk archive: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected archive count: got %d want 1", count)
	}
	if !oldDone.Archived() {
		t.Fatal("expected old done task archived")
	}
	if freshDone.Archived() || oldBacklog.Archived() {
		t.Fatal("expected fresh done and backlog tasks untouched")
	}
}

func TestActiveTaskCountExcludesArchived(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")
	board.AddTask("other", "")

	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if got := board.ActiveTaskCount(); got != 1 {
		t.Fatalf("unexpected active count: got %d want 1", got)
	}
	if got := board.Count(StatusBacklog); got != 1 {
		t.Fatalf("unexpected backlog count: got %d want 1", got)
	}
}

func TestCloneDeepCopiesArchivedAt(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")
	if _, err := board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	clone := board.Clone()
	cloned := clone.Tasks[task.ID]
	if cloned.ArchivedAt == nil {
		t.Fatal("expected clone to keep ArchivedAt")
	}
	if cloned.ArchivedAt == task.ArchivedAt {
		t.Fatal("expected ArchivedAt pointer to be deep-copied")
	}
	mutated := cloned.ArchivedAt.Add(time.Hour)
	*cloned.ArchivedAt = mutated
	if task.ArchivedAt.Equal(mutated) {
		t.Fatal("mutating clone leaked into original")
	}
}
