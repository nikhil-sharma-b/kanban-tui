package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

type BoardStore interface {
	Load() (*domain.Board, error)
	Save(*domain.Board) error
}

func ResolvePaths() (dbPath, legacyPath string, err error) {
	if file := os.Getenv("KANBAN_TUI_DATA_FILE"); file != "" {
		dbPath, legacyPath := resolveEnvPaths(file)
		return dbPath, legacyPath, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", "", err
	}

	baseDir := filepath.Join(configDir, "kanban-tui")
	return filepath.Join(baseDir, "board.db"), filepath.Join(baseDir, "board.json"), nil
}

func Open(dbPath, legacyPath string) (BoardStore, error) {
	exists, err := fileExists(dbPath)
	if err != nil {
		return nil, err
	}

	sqliteStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		return nil, err
	}

	if exists {
		return sqliteStore, nil
	}

	if err := importLegacyBoard(sqliteStore, legacyPath); err != nil {
		return nil, err
	}

	return sqliteStore, nil
}

func resolveEnvPaths(file string) (dbPath, legacyPath string) {
	if strings.EqualFold(filepath.Ext(file), ".json") {
		base := strings.TrimSuffix(file, filepath.Ext(file))
		return base + ".db", file
	}

	ext := filepath.Ext(file)
	if ext == "" {
		return file, file + ".json"
	}

	base := strings.TrimSuffix(file, ext)
	return file, base + ".json"
}

func importLegacyBoard(store BoardStore, legacyPath string) error {
	if legacyPath == "" {
		return nil
	}

	exists, err := fileExists(legacyPath)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	board, err := NewJSONStore(legacyPath).Load()
	if err != nil {
		return fmt.Errorf("load legacy board: %w", err)
	}
	if len(board.Tasks) == 0 {
		return nil
	}

	if err := store.Save(board); err != nil {
		return fmt.Errorf("import legacy board: %w", err)
	}

	return nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
