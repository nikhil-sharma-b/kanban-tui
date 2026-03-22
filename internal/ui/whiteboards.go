package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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
	command, err := exec.LookPath("rnote-cli")
	if err != nil {
		return fmt.Errorf("rnote-cli not found in PATH")
	}
	output, runErr := exec.Command(command, "create", path).CombinedOutput()
	if runErr != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return fmt.Errorf("rnote-cli create failed: %w", runErr)
		}
		return fmt.Errorf("rnote-cli create failed: %s", message)
	}
	return nil
}

var removeWhiteboardFile = os.Remove

func resolveWhiteboardRoot(dataPath string) string {
	if root := strings.TrimSpace(os.Getenv("KANBAN_TUI_WHITEBOARD_DIR")); root != "" {
		return root
	}
	return filepath.Join(filepath.Dir(dataPath), "whiteboards")
}

func buildWhiteboardPath(root, projectName, taskID, whiteboardName string) string {
	return filepath.Join(root, slugifyProjectName(projectName), taskID, slugifyWhiteboardName(whiteboardName)+".rnote")
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

func whiteboardLaunchCommand(path string) (string, []string, error) {
	if custom := strings.TrimSpace(os.Getenv("KANBAN_TUI_WHITEBOARD_CMD")); custom != "" {
		parts := strings.Fields(custom)
		if len(parts) == 0 {
			return "", nil, fmt.Errorf("KANBAN_TUI_WHITEBOARD_CMD is empty")
		}
		return parts[0], append(parts[1:], path), nil
	}
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
		if command, err := exec.LookPath("rnote"); err == nil {
			return command, []string{path}, nil
		}
		return "", nil, fmt.Errorf("no Linux whiteboard launcher found; tried xdg-open, flatpak, and rnote")
	case "windows":
		return "cmd", []string{"/c", "start", "", path}, nil
	default:
		return "", nil, fmt.Errorf("no whiteboard launcher configured for %s", runtime.GOOS)
	}
}
