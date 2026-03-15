package domain

import (
	"fmt"
	"strings"
	"time"
)

const DefaultProjectName = "Personal"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Board     *Board    `json:"board"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewProject(name string) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	now := time.Now().UTC()
	return &Project{
		ID:        newTaskID(),
		Name:      name,
		Board:     NewBoard(),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p *Project) Touch() {
	p.UpdatedAt = time.Now().UTC()
}

func (p *Project) Clone() *Project {
	if p == nil {
		return nil
	}

	clone := *p
	if p.Board != nil {
		clone.Board = p.Board.Clone()
	} else {
		clone.Board = NewBoard()
	}
	return &clone
}

type Workspace struct {
	Version         int        `json:"version"`
	Projects        []*Project `json:"projects"`
	ActiveProjectID string     `json:"active_project_id"`
}

func NewWorkspace() *Workspace {
	project, _ := NewProject(DefaultProjectName)
	return &Workspace{
		Version:         1,
		Projects:        []*Project{project},
		ActiveProjectID: project.ID,
	}
}

func WorkspaceFromBoard(board *Board) *Workspace {
	workspace := NewWorkspace()
	workspace.Projects[0].Board = board.Clone()
	return workspace
}

func (w *Workspace) Clone() *Workspace {
	clone := &Workspace{
		Version:         w.Version,
		ActiveProjectID: w.ActiveProjectID,
		Projects:        make([]*Project, 0, len(w.Projects)),
	}

	for _, project := range w.Projects {
		if project == nil {
			continue
		}
		clone.Projects = append(clone.Projects, project.Clone())
	}

	if len(clone.Projects) == 0 {
		return NewWorkspace()
	}

	return clone
}

func (w *Workspace) Normalize() error {
	if w.Version == 0 {
		w.Version = 1
	}

	if len(w.Projects) == 0 {
		fresh := NewWorkspace()
		*w = *fresh
		return nil
	}

	seenIDs := make(map[string]struct{}, len(w.Projects))
	seenNames := make(map[string]struct{}, len(w.Projects))
	normalized := make([]*Project, 0, len(w.Projects))

	for _, project := range w.Projects {
		if project == nil {
			continue
		}

		project.Name = strings.TrimSpace(project.Name)
		if project.Name == "" {
			return fmt.Errorf("project name cannot be empty")
		}
		if project.ID == "" {
			project.ID = newTaskID()
		}
		if _, ok := seenIDs[project.ID]; ok {
			return fmt.Errorf("duplicate project id %s", project.ID)
		}

		nameKey := strings.ToLower(project.Name)
		if _, ok := seenNames[nameKey]; ok {
			return fmt.Errorf("duplicate project name %s", project.Name)
		}

		if project.Board == nil {
			project.Board = NewBoard()
		}
		if err := project.Board.Normalize(); err != nil {
			return fmt.Errorf("normalize project %s: %w", project.Name, err)
		}
		if project.CreatedAt.IsZero() {
			project.CreatedAt = time.Now().UTC()
		}
		if project.UpdatedAt.IsZero() {
			project.UpdatedAt = project.CreatedAt
		}

		seenIDs[project.ID] = struct{}{}
		seenNames[nameKey] = struct{}{}
		normalized = append(normalized, project)
	}

	if len(normalized) == 0 {
		fresh := NewWorkspace()
		*w = *fresh
		return nil
	}

	w.Projects = normalized
	if w.ActiveProjectID == "" || w.ProjectByID(w.ActiveProjectID) == nil {
		w.ActiveProjectID = w.Projects[0].ID
	}

	return nil
}

func (w *Workspace) ActiveProject() *Project {
	return w.ProjectByID(w.ActiveProjectID)
}

func (w *Workspace) ProjectByID(id string) *Project {
	for _, project := range w.Projects {
		if project != nil && project.ID == id {
			return project
		}
	}
	return nil
}

func (w *Workspace) ProjectIndex(id string) int {
	for i, project := range w.Projects {
		if project != nil && project.ID == id {
			return i
		}
	}
	return -1
}

func (w *Workspace) CreateProject(name string) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}
	if w.hasProjectName(name, "") {
		return nil, fmt.Errorf("project %s already exists", name)
	}

	project, err := NewProject(name)
	if err != nil {
		return nil, err
	}

	w.Projects = append(w.Projects, project)
	w.ActiveProjectID = project.ID
	return project, nil
}

func (w *Workspace) RenameProject(id, name string) (*Project, error) {
	project := w.ProjectByID(id)
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}
	if strings.EqualFold(project.Name, name) {
		project.Name = name
		project.Touch()
		return project, nil
	}
	if w.hasProjectName(name, id) {
		return nil, fmt.Errorf("project %s already exists", name)
	}

	project.Name = name
	project.Touch()
	return project, nil
}

func (w *Workspace) DeleteProject(id string) error {
	if len(w.Projects) <= 1 {
		return fmt.Errorf("cannot delete the last project")
	}

	index := w.ProjectIndex(id)
	if index < 0 {
		return fmt.Errorf("project not found")
	}

	w.Projects = append(w.Projects[:index], w.Projects[index+1:]...)
	if w.ActiveProjectID == id {
		if index >= len(w.Projects) {
			index = len(w.Projects) - 1
		}
		w.ActiveProjectID = w.Projects[index].ID
	}

	return nil
}

func (w *Workspace) SetActiveProject(id string) bool {
	if w.ProjectByID(id) == nil {
		return false
	}
	w.ActiveProjectID = id
	return true
}

func (w *Workspace) hasProjectName(name, exceptID string) bool {
	needle := strings.ToLower(strings.TrimSpace(name))
	for _, project := range w.Projects {
		if project == nil || project.ID == exceptID {
			continue
		}
		if strings.ToLower(project.Name) == needle {
			return true
		}
	}
	return false
}
