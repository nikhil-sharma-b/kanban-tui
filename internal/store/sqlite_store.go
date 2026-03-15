package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

const timestampLayout = time.RFC3339Nano

type SQLiteStore struct {
	path string
	db   *sql.DB
	mu   sync.Mutex
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	dsn := sqliteDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{
		path: path,
		db:   db,
	}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) Load() (*domain.Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspace, err := s.loadWorkspaceBlob()
	if err != nil {
		return nil, err
	}
	if workspace != nil {
		return workspace, nil
	}

	board, err := s.loadLegacyBoard()
	if err != nil {
		return nil, err
	}
	return domain.WorkspaceFromBoard(board), nil
}

func (s *SQLiteStore) Save(workspace *domain.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if workspace == nil {
		workspace = domain.NewWorkspace()
	}
	clone := workspace.Clone()
	if err := clone.Normalize(); err != nil {
		return fmt.Errorf("normalize workspace: %w", err)
	}

	payload, err := json.Marshal(clone)
	if err != nil {
		return fmt.Errorf("encode workspace: %w", err)
	}

	if _, err := s.db.Exec(`
		INSERT INTO meta (key, value) VALUES ('workspace', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, string(payload)); err != nil {
		return fmt.Errorf("store workspace: %w", err)
	}

	return nil
}

func (s *SQLiteStore) loadWorkspaceBlob() (*domain.Workspace, error) {
	var raw string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'workspace'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load workspace: %w", err)
	}

	workspace := domain.NewWorkspace()
	if err := json.Unmarshal([]byte(raw), workspace); err != nil {
		return nil, fmt.Errorf("decode workspace: %w", err)
	}
	if err := workspace.Normalize(); err != nil {
		return nil, fmt.Errorf("normalize workspace: %w", err)
	}

	return workspace, nil
}

func (s *SQLiteStore) loadLegacyBoard() (*domain.Board, error) {
	board := domain.NewBoard()

	if columns, err := s.loadColumns(); err != nil {
		return nil, fmt.Errorf("load columns: %w", err)
	} else if len(columns) > 0 {
		board.Columns = append([]domain.Status{}, columns...)
	}

	version, err := s.loadVersion()
	if err != nil {
		return nil, err
	}
	if version > 0 {
		board.Version = version
	}

	rows, err := s.db.Query(`
		SELECT id, title, description, status, created_at, updated_at
		FROM tasks
		ORDER BY position ASC, created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			task      domain.Task
			status    string
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		task.Status = domain.Status(status)
		task.CreatedAt, err = time.Parse(timestampLayout, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at for %s: %w", task.ID, err)
		}
		task.UpdatedAt, err = time.Parse(timestampLayout, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse updated_at for %s: %w", task.ID, err)
		}

		board.Tasks[task.ID] = &task
		board.Order[task.Status] = append(board.Order[task.Status], task.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	if err := board.Normalize(); err != nil {
		return nil, fmt.Errorf("normalize board: %w", err)
	}

	return board, nil
}

func (s *SQLiteStore) loadColumns() ([]domain.Status, error) {
	var raw string
	if err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'columns'`).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load columns: %w", err)
	}

	var columns []domain.Status
	if err := json.Unmarshal([]byte(raw), &columns); err != nil {
		return nil, fmt.Errorf("decode columns: %w", err)
	}

	return columns, nil
}

func (s *SQLiteStore) init() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			status TEXT NOT NULL,
			position INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status_position ON tasks(status, position)`,
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	for _, statement := range schema {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("init sqlite schema: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) loadVersion() (int, error) {
	var version int
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'version'`).Scan(&version)
	if err == nil {
		return version, nil
	}
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return 0, fmt.Errorf("load board version: %w", err)
}

func sqliteDSN(path string) string {
	u := &url.URL{
		Scheme: "file",
		Path:   path,
	}

	query := u.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	u.RawQuery = query.Encode()

	return u.String()
}
