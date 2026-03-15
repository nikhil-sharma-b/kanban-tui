package domain

import "testing"

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
