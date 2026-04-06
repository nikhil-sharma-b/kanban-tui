package ui

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

var whiteboardSlugRe = regexp.MustCompile(`[^a-z0-9-]+`)
var whiteboardDashRe = regexp.MustCompile(`-+`)

var launchWhiteboard = func(path string) error {
	command, args, err := whiteboardLaunchCommand(path)
	if err != nil {
		return err
	}
	return exec.Command(command, args...).Start()
}

var createWhiteboardFile = func(path string) error {
	const emptyXopp = `<?xml version="1.0" standalone="no"?>
<xournal creator="Xournal++ 1.2.3" fileversion="4">
<page width="612.00" height="792.00">
<background type="solid" color="#ffffffff" style="lined" />
<layer/>
</page>
</xournal>
`
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(emptyXopp)); err != nil {
		return fmt.Errorf("compress xopp: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("compress xopp: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

var removeWhiteboardFile = os.Remove
var moveWhiteboardFile = os.Rename

func resolveWhiteboardRoot(dataPath string) string {
	if root := strings.TrimSpace(os.Getenv("KANBAN_TUI_WHITEBOARD_DIR")); root != "" {
		return root
	}
	return filepath.Join(filepath.Dir(dataPath), "whiteboards")
}

func buildWhiteboardPath(root, projectName, taskID, whiteboardName, ext string) string {
	return filepath.Join(root, slugifyProjectName(projectName), taskID, slugifyWhiteboardName(whiteboardName)+ext)
}

func resolveWhiteboardPath(dataPath, projectName, taskID, whiteboardName, ext string) string {
	return buildWhiteboardPath(resolveWhiteboardRoot(dataPath), projectName, taskID, whiteboardName, ext)
}

func assignWorkspaceWhiteboardPaths(workspace *domain.Workspace, dataPath string) {
	if workspace == nil {
		return
	}
	for _, project := range workspace.Projects {
		assignProjectWhiteboardPaths(project, dataPath)
	}
}

func assignProjectWhiteboardPaths(project *domain.Project, dataPath string) {
	if project == nil || project.Board == nil {
		return
	}
	for taskID, task := range project.Board.Tasks {
		if task == nil {
			continue
		}
		for i := range task.Whiteboards {
			task.Whiteboards[i].Path = resolveWhiteboardPath(dataPath, project.Name, taskID, task.Whiteboards[i].Name, task.Whiteboards[i].Extension())
		}
	}
}

func snapshotProjectWhiteboardPaths(project *domain.Project, dataPath string) map[string]string {
	paths := map[string]string{}
	if project == nil || project.Board == nil {
		return paths
	}
	for taskID, task := range project.Board.Tasks {
		if task == nil {
			continue
		}
		for _, whiteboard := range task.Whiteboards {
			paths[whiteboard.ID] = resolveWhiteboardPath(dataPath, project.Name, taskID, whiteboard.Name, whiteboard.Extension())
		}
	}
	return paths
}

func relocateProjectWhiteboardFiles(project *domain.Project, dataPath string, previous map[string]string) error {
	if project == nil || project.Board == nil {
		return nil
	}
	for taskID, task := range project.Board.Tasks {
		if task == nil {
			continue
		}
		for i := range task.Whiteboards {
			newPath := resolveWhiteboardPath(dataPath, project.Name, taskID, task.Whiteboards[i].Name, task.Whiteboards[i].Extension())
			oldPath := previous[task.Whiteboards[i].ID]
			if oldPath != "" && oldPath != newPath {
				if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
					return err
				}
				if err := moveWhiteboardFile(oldPath, newPath); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			task.Whiteboards[i].Path = newPath
		}
	}
	return nil
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
	value = whiteboardSlugRe.ReplaceAllString(value, "-")
	value = whiteboardDashRe.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return fallback
	}
	return value
}

func isRnoteFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".rnote")
}

func whiteboardLaunchCommand(path string) (string, []string, error) {
	if custom := strings.TrimSpace(os.Getenv("KANBAN_TUI_WHITEBOARD_CMD")); custom != "" {
		parts := strings.Fields(custom)
		if len(parts) == 0 {
			return "", nil, fmt.Errorf("KANBAN_TUI_WHITEBOARD_CMD is empty")
		}
		return parts[0], append(parts[1:], path), nil
	}

	// Legacy .rnote files should still open with rnote.
	if isRnoteFile(path) {
		return whiteboardLaunchRnote(path)
	}

	return whiteboardLaunchXournalpp(path)
}

func whiteboardLaunchXournalpp(path string) (string, []string, error) {
	if command, err := exec.LookPath("xournalpp"); err == nil {
		return command, []string{path}, nil
	}

	switch runtime.GOOS {
	case "darwin":
		return "open", []string{"-a", "Xournal++", "--args", path}, nil
	case "linux":
		if command, err := exec.LookPath("xdg-open"); err == nil {
			return command, []string{path}, nil
		}
		if command, err := exec.LookPath("flatpak"); err == nil {
			return command, []string{"run", "com.github.xournalpp.xournalpp", path}, nil
		}
		return "", nil, fmt.Errorf("no Linux whiteboard launcher found; tried xournalpp, xdg-open, and flatpak")
	case "windows":
		return "cmd", []string{"/c", "start", "", path}, nil
	default:
		return "", nil, fmt.Errorf("no whiteboard launcher configured for %s", runtime.GOOS)
	}
}

func whiteboardLaunchRnote(path string) (string, []string, error) {
	if command, err := exec.LookPath("rnote"); err == nil {
		return command, []string{path}, nil
	}

	switch runtime.GOOS {
	case "darwin":
		return "open", []string{"-a", "Rnote", "--args", path}, nil
	case "linux":
		if command, err := exec.LookPath("xdg-open"); err == nil {
			return command, []string{path}, nil
		}
		if command, err := exec.LookPath("flatpak"); err == nil {
			return command, []string{"run", "com.github.flxzt.rnote", path}, nil
		}
		return "", nil, fmt.Errorf("no Linux rnote launcher found; tried rnote, xdg-open, and flatpak")
	case "windows":
		return "cmd", []string{"/c", "start", "", path}, nil
	default:
		return "", nil, fmt.Errorf("no whiteboard launcher configured for %s", runtime.GOOS)
	}
}
