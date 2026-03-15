package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

type JSONStore struct {
	path string
	mu   sync.Mutex
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

func (s *JSONStore) Load() (*domain.Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewBoard(), nil
		}
		return nil, err
	}

	board := domain.NewBoard()
	if err := json.Unmarshal(data, board); err != nil {
		return nil, fmt.Errorf("decode board: %w", err)
	}
	if err := board.Normalize(); err != nil {
		return nil, fmt.Errorf("normalize board: %w", err)
	}

	return board, nil
}

func (s *JSONStore) Save(board *domain.Board) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Errorf("encode board: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(s.path), "board-*.json")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return nil
}
