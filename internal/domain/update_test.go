package domain

import (
	"testing"
	"time"
)

func TestUpdateTaskLeavesUnchangedTaskUntouched(t *testing.T) {
	board := NewBoard()
	task, err := board.AddTask("Ship", "notes")
	if err != nil {
		t.Fatal(err)
	}
	then := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task.UpdatedAt = then

	if _, err := board.UpdateTask(task.ID, " Ship ", "notes\n"); err != nil {
		t.Fatal(err)
	}
	if !task.UpdatedAt.Equal(then) {
		t.Fatalf("unchanged save moved UpdatedAt to %v", task.UpdatedAt)
	}
	if _, err := board.UpdateTask(task.ID, "Ship it", "notes"); err != nil {
		t.Fatal(err)
	}
	if task.UpdatedAt.Equal(then) {
		t.Fatal("a real edit did not touch UpdatedAt")
	}
}
