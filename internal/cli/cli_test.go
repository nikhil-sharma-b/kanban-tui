package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

type harness struct {
	t      *testing.T
	dbPath string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dir := t.TempDir()
	return &harness{t: t, dbPath: filepath.Join(dir, "board.db")}
}

// run executes a command and returns stdout, stderr and the exit code.
func (h *harness) run(args ...string) (string, string, int) {
	h.t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr, h.dbPath, filepath.Join(filepath.Dir(h.dbPath), "board.json"))
	return stdout.String(), stderr.String(), code
}

// ok runs a command that must succeed and decodes its JSON output into out.
func (h *harness) ok(out any, args ...string) {
	h.t.Helper()
	stdout, stderr, code := h.run(append(args, "--json")...)
	if code != exitOK {
		h.t.Fatalf("%v: exit %d: %s", args, code, stderr)
	}
	if out != nil {
		if err := json.Unmarshal([]byte(stdout), out); err != nil {
			h.t.Fatalf("%v: decode %q: %v", args, stdout, err)
		}
	}
}

func TestTaskLifecycle(t *testing.T) {
	h := newHarness(t)
	h.ok(nil, "project", "create", "Work")

	var task taskView
	h.ok(&task, "task", "add", "Write docs", "-p", "work", "-d", "README", "--column", "in progress")
	if task.Project != "Work" || task.Column != "in_progress" || task.Description != "README" {
		t.Fatalf("unexpected task %+v", task)
	}

	h.ok(&task, "task", "move", task.ID[:5], "--column", "Done")
	if task.Column != "done" || task.Position != 1 {
		t.Fatalf("expected task at done #1, got %s #%d", task.Column, task.Position)
	}

	h.ok(&task, "task", "edit", task.ID, "--title", "Write the docs")
	if task.Title != "Write the docs" || task.Description != "README" {
		t.Fatalf("edit must keep unset fields, got %+v", task)
	}
	h.ok(&task, "task", "edit", task.ID, "-d", "")
	if task.Description != "" {
		t.Fatalf("explicit empty -d must clear the description, got %q", task.Description)
	}

	h.ok(&task, "task", "archive", task.ID)
	var archive archiveView
	h.ok(&archive, "task", "list", "-p", "Work", "--archived")
	if len(archive.Tasks) != 1 || !archive.Tasks[0].Archived {
		t.Fatalf("expected one archived task, got %+v", archive.Tasks)
	}
	if _, stderr, code := h.run("task", "move", task.ID, "--column", "backlog"); code != exitError || !strings.Contains(stderr, "archived") {
		t.Fatalf("moving an archived task must fail, got %d %q", code, stderr)
	}

	h.ok(&task, "task", "restore", task.ID)
	if task.Column != "done" {
		t.Fatalf("expected restore to done, got %s", task.Column)
	}

	h.ok(nil, "task", "delete", task.ID)
	if _, _, code := h.run("task", "show", task.ID); code != exitError {
		t.Fatalf("expected deleted task to be gone, exit %d", code)
	}
}

func TestTaskListFiltersByColumnAndSearch(t *testing.T) {
	h := newHarness(t)
	h.ok(nil, "task", "add", "alpha one")
	h.ok(nil, "task", "add", "beta", "--column", "done")
	h.ok(nil, "task", "add", "alpha two", "--column", "done")

	var board boardView
	h.ok(&board, "task", "list", "--column", "done", "--search", "ALPHA")
	if len(board.Columns) != 1 || len(board.Columns[0].Tasks) != 1 || board.Columns[0].Tasks[0].Title != "alpha two" {
		t.Fatalf("unexpected filtered board %+v", board)
	}
}

func TestAmbiguousTaskPrefixListsCandidates(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 40; i++ {
		h.ok(nil, "task", "add", "task")
	}
	var board boardView
	h.ok(&board, "task", "list")
	seen := map[byte]string{}
	for _, task := range board.Columns[0].Tasks {
		if other, ok := seen[task.ID[0]]; ok {
			_, stderr, code := h.run("task", "show", task.ID[:1])
			if code != exitError || !strings.Contains(stderr, "ambiguous") || !strings.Contains(stderr, other) {
				t.Fatalf("expected ambiguity error listing %s, got %d %q", other, code, stderr)
			}
			return
		}
		seen[task.ID[0]] = task.ID
	}
	t.Fatal("expected two task IDs sharing a first character")
}

func TestProjectCreateKeepsActiveProject(t *testing.T) {
	h := newHarness(t)
	var created projectView
	h.ok(&created, "project", "create", "Side")
	if created.Active {
		t.Fatal("creating a project must not make it active")
	}
	h.ok(&created, "project", "use", "side")
	if !created.Active {
		t.Fatal("project use must make it active")
	}

	var task taskView
	h.ok(&task, "task", "add", "lands in active")
	if task.Project != "Side" {
		t.Fatalf("expected default project Side, got %s", task.Project)
	}
}

func TestDailyPromote(t *testing.T) {
	h := newHarness(t)
	var task taskView
	h.ok(&task, "daily", "add", "call bob", "--column", "active")
	if task.Project != "Daily" || task.Column != "active" {
		t.Fatalf("unexpected daily task %+v", task)
	}

	h.ok(&task, "daily", "promote", task.ID, "--to", "Personal", "--column", "done")
	if task.Project != "Personal" || task.Column != "done" {
		t.Fatalf("unexpected promoted task %+v", task)
	}

	var daily boardView
	h.ok(&daily, "daily", "show")
	for _, column := range daily.Columns {
		if len(column.Tasks) != 0 {
			t.Fatalf("expected empty daily board, got %+v", daily)
		}
	}
}

func TestDailyColumnsAreFixed(t *testing.T) {
	h := newHarness(t)
	if _, stderr, code := h.run("column", "add", "Later", "-p", "daily"); code != exitError || !strings.Contains(stderr, "daily board") {
		t.Fatalf("expected daily column change to fail, got %d %q", code, stderr)
	}
}

func TestColumnCommands(t *testing.T) {
	h := newHarness(t)
	var columns columnListView
	h.ok(&columns, "column", "add", "Review")
	h.ok(&columns, "column", "move", "review", "--position", "2")
	h.ok(&columns, "column", "rename", "review", "QA")
	names := []string{}
	for _, column := range columns.Columns {
		names = append(names, column.Name)
	}
	if got := strings.Join(names, ","); got != "backlog,QA,in_progress,done" {
		t.Fatalf("unexpected columns %s", got)
	}
}

func TestUsageErrors(t *testing.T) {
	h := newHarness(t)
	cases := [][]string{
		{"bogus"},
		{"task", "bogus"},
		{"task", "add"},
		{"task", "move", "abc"},
		{"project", "delete", "Personal"},
		{"daily", "promote", "abc"},
	}
	for _, args := range cases {
		if _, _, code := h.run(args...); code != exitUsage {
			t.Errorf("%v: expected exit %d, got %d", args, exitUsage, code)
		}
	}
}

func TestHelp(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{{"help"}, {"task"}, {"task", "add", "-h"}} {
		stdout, _, code := h.run(args...)
		if code != exitOK || stdout == "" {
			t.Errorf("%v: expected help output, got %d %q", args, code, stdout)
		}
	}
}
