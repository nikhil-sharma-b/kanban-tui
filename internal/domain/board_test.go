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
