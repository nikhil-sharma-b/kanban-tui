package domain

import (
	"fmt"
	"strings"
)

type Board struct {
	Version int                 `json:"version"`
	Tasks   map[string]*Task    `json:"tasks"`
	Order   map[Status][]string `json:"order"`
}

func NewBoard() *Board {
	board := &Board{
		Version: 1,
		Tasks:   make(map[string]*Task),
		Order:   make(map[Status][]string, len(StatusOrder)),
	}

	for _, status := range StatusOrder {
		board.Order[status] = []string{}
	}

	return board
}

func (b *Board) Clone() *Board {
	clone := NewBoard()
	clone.Version = b.Version

	for id, task := range b.Tasks {
		copied := *task
		clone.Tasks[id] = &copied
	}

	for _, status := range StatusOrder {
		clone.Order[status] = append([]string{}, b.Order[status]...)
	}

	return clone
}

func (b *Board) Normalize() error {
	if b.Tasks == nil {
		b.Tasks = make(map[string]*Task)
	}
	if b.Order == nil {
		b.Order = make(map[Status][]string, len(StatusOrder))
	}

	seen := make(map[string]struct{}, len(b.Tasks))
	for _, status := range StatusOrder {
		order := b.Order[status]
		filtered := make([]string, 0, len(order))
		for _, id := range order {
			task, ok := b.Tasks[id]
			if !ok {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			task.Status = status
			filtered = append(filtered, id)
			seen[id] = struct{}{}
		}
		b.Order[status] = filtered
	}

	for id, task := range b.Tasks {
		if task == nil {
			delete(b.Tasks, id)
			continue
		}
		if strings.TrimSpace(task.Title) == "" {
			return fmt.Errorf("task %s has empty title", id)
		}
		if !task.Status.Valid() {
			task.Status = StatusBacklog
		}
		if _, ok := seen[id]; ok {
			continue
		}
		b.Order[task.Status] = append(b.Order[task.Status], id)
	}

	for _, status := range StatusOrder {
		if _, ok := b.Order[status]; !ok {
			b.Order[status] = []string{}
		}
	}

	if b.Version == 0 {
		b.Version = 1
	}

	return nil
}

func (b *Board) AddTask(title, description string) (*Task, error) {
	task, err := NewTask(title, description)
	if err != nil {
		return nil, err
	}

	b.Tasks[task.ID] = task
	b.Order[task.Status] = append(b.Order[task.Status], task.ID)
	return task, nil
}

func (b *Board) DeleteTask(id string) bool {
	task, ok := b.Tasks[id]
	if !ok {
		return false
	}

	b.Order[task.Status] = removeID(b.Order[task.Status], id)
	delete(b.Tasks, id)
	return true
}

func (b *Board) MoveTask(id string, next Status, index int) bool {
	task, ok := b.Tasks[id]
	if !ok || !next.Valid() {
		return false
	}

	current := task.Status
	currentOrder := b.Order[current]
	currentIndex := indexOf(currentOrder, id)
	if currentIndex == -1 {
		return false
	}
	currentOrder = removeAt(currentOrder, currentIndex)
	b.Order[current] = currentOrder

	targetOrder := b.Order[next]
	if current == next && index > currentIndex {
		index--
	}
	if index < 0 || index > len(targetOrder) {
		index = len(targetOrder)
	}
	b.Order[next] = insertAt(targetOrder, index, id)

	task.Status = next
	task.Touch()
	return true
}

func (b *Board) ShiftTask(id string, delta int) bool {
	task, ok := b.Tasks[id]
	if !ok {
		return false
	}

	currentIndex := statusIndex(task.Status)
	nextIndex := currentIndex + delta
	if nextIndex < 0 || nextIndex >= len(StatusOrder) {
		return false
	}

	return b.MoveTask(id, StatusOrder[nextIndex], len(b.Order[StatusOrder[nextIndex]]))
}

func (b *Board) MoveWithin(status Status, from, to int) bool {
	order := b.Order[status]
	if from < 0 || from >= len(order) || to < 0 || to >= len(order) || from == to {
		return false
	}

	id := order[from]
	order = removeAt(order, from)
	b.Order[status] = insertAt(order, to, id)

	if task, ok := b.Tasks[id]; ok {
		task.Touch()
	}
	return true
}

func (b *Board) Count(status Status) int {
	return len(b.Order[status])
}

func statusIndex(status Status) int {
	for i, candidate := range StatusOrder {
		if candidate == status {
			return i
		}
	}
	return 0
}

func removeID(ids []string, target string) []string {
	for i, id := range ids {
		if id != target {
			continue
		}
		return removeAt(ids, i)
	}
	return ids
}

func removeAt(ids []string, index int) []string {
	if index < 0 || index >= len(ids) {
		return ids
	}

	out := make([]string, 0, len(ids)-1)
	out = append(out, ids[:index]...)
	out = append(out, ids[index+1:]...)
	return out
}

func insertAt(ids []string, index int, id string) []string {
	if index < 0 {
		index = 0
	}
	if index > len(ids) {
		index = len(ids)
	}

	out := make([]string, 0, len(ids)+1)
	out = append(out, ids[:index]...)
	out = append(out, id)
	out = append(out, ids[index:]...)
	return out
}

func indexOf(ids []string, target string) int {
	for i, id := range ids {
		if id == target {
			return i
		}
	}
	return -1
}
