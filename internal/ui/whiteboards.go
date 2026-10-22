package ui

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nikhilsharma/kanban-tui/internal/whiteboardfs"
)

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

var (
	resolveWhiteboardPath          = whiteboardfs.Path
	assignWorkspaceWhiteboardPaths = whiteboardfs.AssignWorkspacePaths
	snapshotProjectWhiteboardPaths = whiteboardfs.SnapshotProjectPaths
	relocateProjectWhiteboardFiles = whiteboardfs.RelocateProjectFiles
	stageTaskWhiteboardFiles       = whiteboardfs.StageTaskFiles
)

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
