package store

import (
	"database/sql"
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

func (s *SQLiteStore) Load() (*domain.Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	board := domain.NewBoard()

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
		ORDER BY
			CASE status
				WHEN 'backlog' THEN 0
				WHEN 'in_progress' THEN 1
				WHEN 'done' THEN 2
				ELSE 3
			END,
			position ASC,
			created_at ASC
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

func (s *SQLiteStore) Save(board *domain.Board) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM tasks`); err != nil {
		return fmt.Errorf("clear tasks: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO tasks (id, title, description, status, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, status := range domain.StatusOrder {
		for position, id := range board.Order[status] {
			task := board.Tasks[id]
			if task == nil {
				continue
			}
			if _, err = stmt.Exec(
				task.ID,
				task.Title,
				task.Description,
				string(status),
				position,
				task.CreatedAt.UTC().Format(timestampLayout),
				task.UpdatedAt.UTC().Format(timestampLayout),
			); err != nil {
				return fmt.Errorf("insert task %s: %w", task.ID, err)
			}
		}
	}

	if _, err = tx.Exec(`
		INSERT INTO meta (key, value) VALUES ('version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, board.Version); err != nil {
		return fmt.Errorf("store version: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
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
