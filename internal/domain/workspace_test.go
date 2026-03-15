package domain

import "testing"

func TestWorkspaceProjectCRUD(t *testing.T) {
	workspace := NewWorkspace()
	original := workspace.ActiveProject()
	if original == nil {
		t.Fatal("expected default project")
	}

	project, err := workspace.CreateProject("Work")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if workspace.ActiveProjectID != project.ID {
		t.Fatalf("expected active project to switch to new project")
	}

	renamed, err := workspace.RenameProject(project.ID, "Client Work")
	if err != nil {
		t.Fatalf("rename project: %v", err)
	}
	if renamed.Name != "Client Work" {
		t.Fatalf("unexpected renamed project: %q", renamed.Name)
	}

	if err := workspace.DeleteProject(original.ID); err != nil {
		t.Fatalf("delete original project: %v", err)
	}
	if workspace.ProjectByID(original.ID) != nil {
		t.Fatal("expected original project to be removed")
	}

	if err := workspace.DeleteProject(project.ID); err == nil {
		t.Fatal("expected deleting last project to fail")
	}
}

func TestWorkspaceNormalizeRepairsMissingActiveProject(t *testing.T) {
	workspace := &Workspace{
		Projects: []*Project{{Name: "One", Board: NewBoard()}},
	}

	if err := workspace.Normalize(); err != nil {
		t.Fatalf("normalize workspace: %v", err)
	}
	if workspace.ActiveProject() == nil {
		t.Fatal("expected active project after normalize")
	}
}
