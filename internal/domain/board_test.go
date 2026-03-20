package domain

import (
	"testing"
	"time"
)

func TestMoveWithinReordersTasks(t *testing.T) {
	board := NewBoard()
	first, _ := board.AddTask("first", "")
	second, _ := board.AddTask("second", "")
	third, _ := board.AddTask("third", "")

	if !board.MoveWithin(StatusBacklog, 0, 2) {
		t.Fatalf("expected reorder to succeed")
	}

	got := board.Order[StatusBacklog]
	want := []string{second.ID, third.ID, first.ID}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected order at %d: got %v want %v", i, got, want)
		}
	}
}

func TestShiftTaskMovesAcrossColumns(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")

	if !board.ShiftTask(task.ID, 1) {
		t.Fatalf("expected shift to succeed")
	}

	if task.Status != StatusInProgress {
		t.Fatalf("unexpected status: got %s want %s", task.Status, StatusInProgress)
	}
	if len(board.Order[StatusBacklog]) != 0 {
		t.Fatalf("expected backlog to be empty")
	}
	if len(board.Order[StatusInProgress]) != 1 || board.Order[StatusInProgress][0] != task.ID {
		t.Fatalf("unexpected in-progress order: %v", board.Order[StatusInProgress])
	}
}

func TestAddColumnAllowsCustomColumnAndShift(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("task", "")

	custom, err := board.AddColumn("Review")
	if err != nil {
		t.Fatalf("add column: %v", err)
	}

	if !board.ShiftTask(task.ID, 1) {
		t.Fatalf("expected first shift")
	}
	if !board.ShiftTask(task.ID, 1) {
		t.Fatalf("expected second shift")
	}
	if !board.ShiftTask(task.ID, 1) {
		t.Fatalf("expected third shift into custom column")
	}

	if task.Status != custom {
		t.Fatalf("unexpected status: got %q want %q", task.Status, custom)
	}
	if got := board.Order[custom]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected custom column order: %v", got)
	}
}

func TestMoveColumnReordersColumns(t *testing.T) {
	board := NewBoard()

	if _, err := board.AddColumn("Review"); err != nil {
		t.Fatalf("add column: %v", err)
	}

	if !board.MoveColumn(0, 2) {
		t.Fatalf("expected column move to succeed")
	}

	want := []Status{StatusInProgress, StatusBacklog, StatusDone, Status("Review")}
	if len(board.Columns) != len(want) {
		t.Fatalf("unexpected columns length: got %d want %d", len(board.Columns), len(want))
	}
	for i := range want {
		if board.Columns[i] != want[i] {
			t.Fatalf("unexpected column at %d: got %s want %s", i, board.Columns[i], want[i])
		}
	}
}

func TestRenameColumnMovesTasksAndOrder(t *testing.T) {
	board := NewBoard()
	review, err := board.AddColumn("Review")
	if err != nil {
		t.Fatalf("add column: %v", err)
	}

	task, _ := board.AddTask("task", "")
	if !board.MoveTask(task.ID, review, 0) {
		t.Fatalf("move to review")
	}

	renameTarget, err := board.RenameColumn(string(review), "QA")
	if err != nil {
		t.Fatalf("rename column: %v", err)
	}
	if renameTarget != Status("QA") {
		t.Fatalf("unexpected rename result: %q", renameTarget)
	}

	if task.Status != Status("QA") {
		t.Fatalf("unexpected task status: got %s want %s", task.Status, "QA")
	}

	if got := board.Order[Status("QA")]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected renamed column order: %v", got)
	}
}

func TestDeleteColumnMovesTasksToAdjacentColumn(t *testing.T) {
	board := NewBoard()
	review, err := board.AddColumn("Review")
	if err != nil {
		t.Fatalf("add column: %v", err)
	}

	task, _ := board.AddTask("task", "")
	if !board.MoveTask(task.ID, review, 0) {
		t.Fatalf("move task to review")
	}

	if err := board.DeleteColumn(review); err != nil {
		t.Fatalf("delete column: %v", err)
	}

	if task.Status != StatusDone {
		t.Fatalf("unexpected task status after delete: %s", task.Status)
	}
	if got := board.Order[StatusDone]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected done order after delete: %v", got)
	}

	for _, status := range board.Columns {
		if status == review {
			t.Fatalf("deleted column still exists in columns: %s", status)
		}
	}
}

func TestRenameColumnRejectsExistingName(t *testing.T) {
	board := NewBoard()
	if _, err := board.AddColumn("Review"); err != nil {
		t.Fatalf("add column: %v", err)
	}

	if _, err := board.RenameColumn("Review", string(StatusInProgress)); err == nil {
		t.Fatal("expected rename to existing name to fail")
	}
}

func TestUpdateTaskChangesTitleAndDescription(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("old title", "old description")

	updated, err := board.UpdateTask(task.ID, "new title", "new description")
	if err != nil {
		t.Fatalf("update task: %v", err)
	}

	if updated.Title != "new title" {
		t.Fatalf("unexpected title: got %q want %q", updated.Title, "new title")
	}
	if updated.Description != "new description" {
		t.Fatalf("unexpected description: got %q want %q", updated.Description, "new description")
	}
}

func TestUpdateTaskRejectsEmptyTitle(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("title", "")

	if _, err := board.UpdateTask(task.ID, "   ", "desc"); err == nil {
		t.Fatal("expected update with empty title to fail")
	}
}

func TestAddWhiteboardsToTask(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("title", "")

	first, err := board.AddWhiteboard(task.ID, "Whiteboard 1", "/tmp/wb-1.rnote")
	if err != nil {
		t.Fatalf("add first whiteboard: %v", err)
	}
	second, err := board.AddWhiteboard(task.ID, "Whiteboard 2", "/tmp/wb-2.rnote")
	if err != nil {
		t.Fatalf("add second whiteboard: %v", err)
	}

	if len(task.Whiteboards) != 2 {
		t.Fatalf("unexpected whiteboard count: got %d want 2", len(task.Whiteboards))
	}
	if task.Whiteboards[0].ID != first.ID || task.Whiteboards[1].ID != second.ID {
		t.Fatalf("unexpected whiteboard ids: %+v", task.Whiteboards)
	}
}

func TestAddWhiteboardRejectsDuplicateNames(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("title", "")

	if _, err := board.AddWhiteboard(task.ID, "Notes", "/tmp/notes-1.rnote"); err != nil {
		t.Fatalf("add whiteboard: %v", err)
	}
	if _, err := board.AddWhiteboard(task.ID, " notes ", "/tmp/notes-2.rnote"); err == nil {
		t.Fatal("expected duplicate whiteboard name to fail")
	}
}

func TestRenameWhiteboardUpdatesNameOnly(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("title", "")

	whiteboard, err := board.AddWhiteboard(task.ID, "Sketches", "/tmp/sketches.rnote")
	if err != nil {
		t.Fatalf("add whiteboard: %v", err)
	}

	originalCreated := whiteboard.CreatedAt
	originalPath := whiteboard.Path
	time.Sleep(time.Millisecond)

	renamed, err := board.RenameWhiteboard(task.ID, whiteboard.ID, "Wireframes")
	if err != nil {
		t.Fatalf("rename whiteboard: %v", err)
	}
	if renamed.Name != "Wireframes" {
		t.Fatalf("unexpected whiteboard name: got %q want %q", renamed.Name, "Wireframes")
	}
	if renamed.Path != originalPath {
		t.Fatalf("rename should preserve path: got %q want %q", renamed.Path, originalPath)
	}
	if !renamed.UpdatedAt.After(originalCreated) {
		t.Fatalf("expected updated timestamp after created timestamp")
	}
}

func TestDeleteWhiteboardRemovesOnlyTarget(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("title", "")

	first, _ := board.AddWhiteboard(task.ID, "First", "/tmp/first.rnote")
	second, _ := board.AddWhiteboard(task.ID, "Second", "/tmp/second.rnote")

	removed, err := board.DeleteWhiteboard(task.ID, first.ID)
	if err != nil {
		t.Fatalf("delete whiteboard: %v", err)
	}
	if removed.ID != first.ID {
		t.Fatalf("unexpected removed whiteboard: got %s want %s", removed.ID, first.ID)
	}
	if len(task.Whiteboards) != 1 || task.Whiteboards[0].ID != second.ID {
		t.Fatalf("unexpected remaining whiteboards: %+v", task.Whiteboards)
	}
}

func TestNextWhiteboardNameFindsNextFreeSlot(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("title", "")

	if _, err := board.AddWhiteboard(task.ID, "Whiteboard 1", "/tmp/one.rnote"); err != nil {
		t.Fatalf("add whiteboard 1: %v", err)
	}
	if _, err := board.AddWhiteboard(task.ID, "Custom", "/tmp/custom.rnote"); err != nil {
		t.Fatalf("add custom whiteboard: %v", err)
	}
	if _, err := board.AddWhiteboard(task.ID, "Whiteboard 3", "/tmp/three.rnote"); err != nil {
		t.Fatalf("add whiteboard 3: %v", err)
	}

	if got, want := board.NextWhiteboardName(task.ID), "Whiteboard 2"; got != want {
		t.Fatalf("NextWhiteboardName() = %q, want %q", got, want)
	}
}

func TestUpdateTaskPreservesWhiteboards(t *testing.T) {
	board := NewBoard()
	task, _ := board.AddTask("title", "")
	whiteboard, _ := board.AddWhiteboard(task.ID, "Sketch", "/tmp/sketch.rnote")

	updated, err := board.UpdateTask(task.ID, "new title", "new description")
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if len(updated.Whiteboards) != 1 || updated.Whiteboards[0].ID != whiteboard.ID {
		t.Fatalf("whiteboards not preserved: %+v", updated.Whiteboards)
	}
}

func TestDeleteColumnDisallowsRemovingLastColumn(t *testing.T) {
	board := NewBoard()

	if err := board.DeleteColumn(StatusDone); err != nil {
		t.Fatalf("delete %q: %v", StatusDone, err)
	}
	if err := board.DeleteColumn(StatusInProgress); err != nil {
		t.Fatalf("delete %q: %v", StatusInProgress, err)
	}

	if err := board.DeleteColumn(StatusBacklog); err == nil {
		t.Fatal("expected deleting last column to fail")
	}

	if got := len(board.Columns); got != 1 {
		t.Fatalf("unexpected column count: got %d want 1", got)
	}
}

func TestMoveColumnBounds(t *testing.T) {
	board := NewBoard()
	if _, err := board.AddColumn("Review"); err != nil {
		t.Fatalf("add column: %v", err)
	}

	if board.MoveColumn(-1, 1) {
		t.Fatal("expected invalid source index to fail")
	}
	if board.MoveColumn(0, len(board.Columns)) {
		t.Fatal("expected invalid destination index to fail")
	}
	if board.MoveColumn(0, 0) {
		t.Fatal("expected same source and destination to fail")
	}
}
