// Package whiteboardfs maps whiteboards to files on disk and keeps those
// files in step when tasks or projects change, so every front end (TUI, CLI)
// moves them the same way.
package whiteboardfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)
var dashRe = regexp.MustCompile(`-+`)

var moveFile = os.Rename

type stagedFile struct {
	oldPath string
	newPath string
}

func Root(dataPath string) string {
	if root := strings.TrimSpace(os.Getenv("KANBAN_TUI_WHITEBOARD_DIR")); root != "" {
		return root
	}
	return filepath.Join(filepath.Dir(dataPath), "whiteboards")
}

func BuildPath(root, projectName, taskID, whiteboardName, ext string) string {
	return filepath.Join(root, slugifyProjectName(projectName), taskID, slugifyWhiteboardName(whiteboardName)+ext)
}

func Path(dataPath, projectName, taskID, whiteboardName, ext string) string {
	return BuildPath(Root(dataPath), projectName, taskID, whiteboardName, ext)
}

func AssignWorkspacePaths(workspace *domain.Workspace, dataPath string) {
	if workspace == nil {
		return
	}
	for _, project := range workspace.Projects {
		AssignProjectPaths(project, dataPath)
	}
}

func AssignProjectPaths(project *domain.Project, dataPath string) {
	if project == nil || project.Board == nil {
		return
	}
	for taskID, task := range project.Board.Tasks {
		if task == nil {
			continue
		}
		for i := range task.Whiteboards {
			task.Whiteboards[i].Path = Path(dataPath, project.Name, taskID, task.Whiteboards[i].Name, task.Whiteboards[i].Extension())
		}
	}
}

func SnapshotProjectPaths(project *domain.Project, dataPath string) map[string]string {
	paths := map[string]string{}
	if project == nil || project.Board == nil {
		return paths
	}
	for taskID, task := range project.Board.Tasks {
		if task == nil {
			continue
		}
		for _, whiteboard := range task.Whiteboards {
			paths[whiteboard.ID] = Path(dataPath, project.Name, taskID, whiteboard.Name, whiteboard.Extension())
		}
	}
	return paths
}

func RelocateProjectFiles(project *domain.Project, dataPath string, previous map[string]string) error {
	if project == nil || project.Board == nil {
		return nil
	}
	for taskID, task := range project.Board.Tasks {
		if task == nil {
			continue
		}
		for i := range task.Whiteboards {
			newPath := Path(dataPath, project.Name, taskID, task.Whiteboards[i].Name, task.Whiteboards[i].Extension())
			oldPath := previous[task.Whiteboards[i].ID]
			if oldPath != "" && oldPath != newPath {
				if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
					return err
				}
				if err := moveFile(oldPath, newPath); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			task.Whiteboards[i].Path = newPath
		}
	}
	return nil
}

// StageTaskFiles copies a task's files to their future project paths.
// The caller commits by removing old files only after workspace persistence.
func StageTaskFiles(task *domain.Task, dataPath, projectName string) (commit func() error, rollback func() error, err error) {
	if task == nil {
		return func() error { return nil }, func() error { return nil }, nil
	}
	staged := make([]stagedFile, 0, len(task.Whiteboards))
	paths := make([]string, len(task.Whiteboards))

	for i, whiteboard := range task.Whiteboards {
		newPath := Path(dataPath, projectName, task.ID, whiteboard.Name, whiteboard.Extension())
		paths[i] = newPath
		if whiteboard.Path == newPath {
			continue
		}
		if whiteboard.Path == "" {
			return nil, nil, fmt.Errorf("whiteboard %s has no source path", whiteboard.Name)
		}
		if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			return nil, nil, stagedError(err, removeStaged(staged))
		}
		if err := copyFile(whiteboard.Path, newPath); err != nil {
			return nil, nil, stagedError(err, removeStaged(staged))
		}
		staged = append(staged, stagedFile{oldPath: whiteboard.Path, newPath: newPath})
	}
	for i := range task.Whiteboards {
		task.Whiteboards[i].Path = paths[i]
	}
	return func() error { return removeOriginals(staged) }, func() error { return removeStaged(staged) }, nil
}

func copyFile(source, target string) (err error) {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dst.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(target)
		}
	}()
	_, err = io.Copy(dst, src)
	return err
}

func removeStaged(staged []stagedFile) error {
	var firstErr error
	for i := len(staged) - 1; i >= 0; i-- {
		if err := os.Remove(staged[i].newPath); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func removeOriginals(staged []stagedFile) error {
	var firstErr error
	for _, file := range staged {
		if err := os.Remove(file.oldPath); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func stagedError(stageErr, cleanupErr error) error {
	if cleanupErr != nil {
		return fmt.Errorf("%v; clean staged files: %w", stageErr, cleanupErr)
	}
	return stageErr
}

func slugifyProjectName(name string) string {
	return slugify(name, "project")
}

func slugifyWhiteboardName(name string) string {
	return slugify(name, "whiteboard")
}

func slugify(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, string(filepath.Separator), "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.Join(strings.Fields(value), "-")
	value = slugRe.ReplaceAllString(value, "-")
	value = dashRe.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return fallback
	}
	return value
}
