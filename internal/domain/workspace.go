package domain

import (
	"fmt"
	"strings"
	"time"
)

const DefaultProjectName = "Personal"

// DailyProjectName is the name of the single, always-present daily board.
const DailyProjectName = "Daily"

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Daily marks the one and only daily board. It has fixed
	// waiting/active/next columns, cannot be renamed or deleted, and is
	// hidden from the project manager.
	Daily     bool      `json:"daily,omitempty"`
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

func NewDailyProject() *Project {
	now := time.Now().UTC()
	return &Project{
		ID:        newTaskID(),
		Name:      DailyProjectName,
		Daily:     true,
		Board:     NewDailyBoard(),
		CreatedAt: now,
		UpdatedAt: now,
	}
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
		Projects:        []*Project{project, NewDailyProject()},
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
	dailySeen := false

	for _, project := range w.Projects {
		if project == nil {
			continue
		}

		// Only the first daily project stays daily; any extra one becomes a
		// regular project so the invariant "exactly one daily board" holds.
		if project.Daily {
			if dailySeen {
				project.Daily = false
			} else {
				dailySeen = true
			}
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

		// The daily board is hidden from the project list, so its name never
		// takes part in the uniqueness check.
		if !project.Daily {
			nameKey := strings.ToLower(project.Name)
			if _, ok := seenNames[nameKey]; ok {
				return fmt.Errorf("duplicate project name %s", project.Name)
			}
			seenNames[nameKey] = struct{}{}
		}

		if project.Board == nil {
			if project.Daily {
				project.Board = NewDailyBoard()
			} else {
				project.Board = NewBoard()
			}
		}
		if project.Daily {
			normalizeDailyBoard(project.Board)
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
		normalized = append(normalized, project)
	}

	if len(normalized) == 0 {
		fresh := NewWorkspace()
		*w = *fresh
		return nil
	}

	if !dailySeen {
		normalized = append(normalized, NewDailyProject())
	}
	if !hasRegularProject(normalized) {
		project, err := NewProject(DefaultProjectName)
		if err != nil {
			return err
		}
		normalized = append([]*Project{project}, normalized...)
	}

	w.Projects = normalized
	if w.ActiveProjectID == "" || w.ProjectByID(w.ActiveProjectID) == nil {
		w.ActiveProjectID = w.Projects[0].ID
	}

	return nil
}

// normalizeDailyBoard pins the daily board to its fixed columns and moves any
// task with a foreign status back to Waiting.
func normalizeDailyBoard(board *Board) {
	if board == nil {
		return
	}

	board.Columns = append([]Status(nil), DailyStatusOrder...)
	for _, task := range board.Tasks {
		if task == nil {
			continue
		}
		if !IsDailyStatus(normalizeStatus(task.Status)) {
			task.Status = StatusWaiting
		}
		if task.Archived() && !IsDailyStatus(normalizeStatus(task.ArchivedFrom)) {
			task.ArchivedFrom = StatusWaiting
		}
	}

	if board.Order == nil {
		board.Order = make(map[Status][]string, len(DailyStatusOrder))
	}
	for status := range board.Order {
		if !IsDailyStatus(status) {
			delete(board.Order, status)
		}
	}
	for _, status := range DailyStatusOrder {
		if _, ok := board.Order[status]; !ok {
			board.Order[status] = []string{}
		}
	}
}

func hasRegularProject(projects []*Project) bool {
	for _, project := range projects {
		if project != nil && !project.Daily {
			return true
		}
	}
	return false
}

// DailyProject returns the single daily board project, or nil if the
// workspace has not been normalized yet.
func (w *Workspace) DailyProject() *Project {
	for _, project := range w.Projects {
		if project != nil && project.Daily {
			return project
		}
	}
	return nil
}

// RegularProjects returns every project except the daily board.
func (w *Workspace) RegularProjects() []*Project {
	projects := make([]*Project, 0, len(w.Projects))
	for _, project := range w.Projects {
		if project != nil && !project.Daily {
			projects = append(projects, project)
		}
	}
	return projects
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
	if project.Daily {
		return nil, fmt.Errorf("cannot rename the daily board")
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
	index := w.ProjectIndex(id)
	if index < 0 {
		return fmt.Errorf("project not found")
	}
	if w.Projects[index].Daily {
		return fmt.Errorf("cannot delete the daily board")
	}
	if len(w.RegularProjects()) <= 1 {
		return fmt.Errorf("cannot delete the last project")
	}

	w.Projects = append(w.Projects[:index], w.Projects[index+1:]...)
	if w.ActiveProjectID == id {
		w.ActiveProjectID = w.fallbackProjectID(index)
	}

	return nil
}

// fallbackProjectID picks the regular project that should become active after
// the project at index was removed.
func (w *Workspace) fallbackProjectID(index int) string {
	for i := index; i < len(w.Projects); i++ {
		if !w.Projects[i].Daily {
			return w.Projects[i].ID
		}
	}
	for i := min(index, len(w.Projects)) - 1; i >= 0; i-- {
		if !w.Projects[i].Daily {
			return w.Projects[i].ID
		}
	}
	return w.Projects[0].ID
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
		if project == nil || project.ID == exceptID || project.Daily {
			continue
		}
		if strings.ToLower(project.Name) == needle {
			return true
		}
	}
	return false
}
