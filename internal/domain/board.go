package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Board struct {
	Version int                 `json:"version"`
	Columns []Status            `json:"columns"`
	Tasks   map[string]*Task    `json:"tasks"`
	Order   map[Status][]string `json:"order"`
}

func NewBoard() *Board {
	columns := append([]Status(nil), StatusOrder...)
	board := &Board{
		Version: 1,
		Columns: columns,
		Tasks:   make(map[string]*Task),
		Order:   make(map[Status][]string, len(columns)),
	}

	for _, status := range columns {
		board.Order[status] = []string{}
	}

	return board
}

func (b *Board) Clone() *Board {
	clone := NewBoard()
	clone.Version = b.Version
	clone.Columns = append([]Status{}, b.Columns...)

	clone.Order = make(map[Status][]string, len(b.Order))
	for status, ids := range b.Order {
		clone.Order[status] = append([]string{}, ids...)
	}

	for _, status := range clone.Columns {
		if _, ok := clone.Order[status]; !ok {
			clone.Order[status] = []string{}
		}
	}

	for id, task := range b.Tasks {
		copied := *task
		if task.Whiteboards != nil {
			copied.Whiteboards = append([]Whiteboard(nil), task.Whiteboards...)
		}
		clone.Tasks[id] = &copied
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

	b.Columns = normalizeColumns(b.Columns)
	if len(b.Columns) == 0 {
		b.Columns = append([]Status{}, StatusOrder...)
	}

	normalizedOrder := make(map[Status][]string, len(b.Columns))
	for _, status := range b.Columns {
		normalizedOrder[status] = []string{}
	}

	columnSet := make(map[Status]struct{}, len(b.Columns))
	for _, status := range b.Columns {
		columnSet[status] = struct{}{}
	}

	seenTasks := make(map[string]struct{}, len(b.Tasks))
	defaultStatus := b.Columns[0]

	ensureStatus := func(status Status) {
		status = normalizeStatus(status)
		if status == "" {
			status = defaultStatus
		}

		if _, ok := normalizedOrder[status]; !ok {
			normalizedOrder[status] = []string{}
		}
		if _, ok := columnSet[status]; !ok {
			b.Columns = append(b.Columns, status)
			columnSet[status] = struct{}{}
		}
	}

	processOrder := func(ids []string) error {
		for _, id := range ids {
			task, ok := b.Tasks[id]
			if !ok || task == nil {
				continue
			}
			if strings.TrimSpace(task.Title) == "" {
				return fmt.Errorf("task %s has empty title", id)
			}
			if err := normalizeWhiteboards(task); err != nil {
				return fmt.Errorf("task %s whiteboards: %w", id, err)
			}
			if !task.Status.Valid() {
				task.Status = defaultStatus
			}
			task.Status = normalizeStatus(task.Status)
			if task.Status == "" {
				task.Status = defaultStatus
			}
			if _, dup := seenTasks[id]; dup {
				continue
			}

			ensureStatus(task.Status)
			normalizedOrder[task.Status] = append(normalizedOrder[task.Status], id)
			seenTasks[id] = struct{}{}
		}
		return nil
	}

	for _, status := range b.Columns {
		if err := processOrder(b.Order[status]); err != nil {
			return err
		}
	}

	for status, ids := range b.Order {
		if _, ok := columnSet[normalizeStatus(status)]; ok {
			continue
		}
		if err := processOrder(ids); err != nil {
			return err
		}
	}

	for id, task := range b.Tasks {
		if task == nil {
			delete(b.Tasks, id)
			continue
		}
		if _, seen := seenTasks[id]; seen {
			continue
		}
		if strings.TrimSpace(task.Title) == "" {
			return fmt.Errorf("task %s has empty title", id)
		}
		if err := normalizeWhiteboards(task); err != nil {
			return fmt.Errorf("task %s whiteboards: %w", id, err)
		}
		if !task.Status.Valid() {
			task.Status = defaultStatus
		}
		task.Status = normalizeStatus(task.Status)
		if task.Status == "" {
			task.Status = defaultStatus
		}

		ensureStatus(task.Status)
		normalizedOrder[task.Status] = append(normalizedOrder[task.Status], id)
		seenTasks[id] = struct{}{}
	}

	b.Order = normalizedOrder
	for _, status := range b.Columns {
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
	if b.Tasks == nil {
		b.Tasks = make(map[string]*Task)
	}
	if b.Order == nil {
		b.Order = make(map[Status][]string)
	}
	if len(b.Columns) == 0 {
		b.Columns = append([]Status{}, StatusOrder...)
	}

	task, err := NewTask(title, description)
	if err != nil {
		return nil, err
	}

	task.Status = b.Columns[0]
	b.Tasks[task.ID] = task
	b.Order[task.Status] = append(b.Order[task.Status], task.ID)
	return task, nil
}

func (b *Board) UpdateTask(id, title, description string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	task, ok := b.Tasks[id]
	if !ok || task == nil {
		return nil, fmt.Errorf("task %s not found", id)
	}

	task.Title = title
	task.Description = strings.TrimSpace(description)
	task.Touch()

	return task, nil
}

func (b *Board) Whiteboard(taskID, whiteboardID string) (*Whiteboard, error) {
	task, ok := b.Tasks[taskID]
	if !ok || task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	for i := range task.Whiteboards {
		if task.Whiteboards[i].ID == whiteboardID {
			return &task.Whiteboards[i], nil
		}
	}

	return nil, fmt.Errorf("whiteboard %s not found", whiteboardID)
}

func (b *Board) AddWhiteboard(taskID, name, path string) (*Whiteboard, error) {
	task, ok := b.Tasks[taskID]
	if !ok || task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	if name == "" {
		return nil, fmt.Errorf("whiteboard name cannot be empty")
	}
	if path == "" {
		return nil, fmt.Errorf("whiteboard path cannot be empty")
	}
	if hasWhiteboardName(task.Whiteboards, name, "") {
		return nil, fmt.Errorf("whiteboard %s already exists", name)
	}

	now := timeNowUTC()
	whiteboard := Whiteboard{
		ID:        newTaskID(),
		Name:      name,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	}
	task.Whiteboards = append(task.Whiteboards, whiteboard)
	task.Touch()
	return &task.Whiteboards[len(task.Whiteboards)-1], nil
}

func (b *Board) RenameWhiteboard(taskID, whiteboardID, name string) (*Whiteboard, error) {
	task, ok := b.Tasks[taskID]
	if !ok || task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("whiteboard name cannot be empty")
	}
	if hasWhiteboardName(task.Whiteboards, name, whiteboardID) {
		return nil, fmt.Errorf("whiteboard %s already exists", name)
	}

	for i := range task.Whiteboards {
		if task.Whiteboards[i].ID != whiteboardID {
			continue
		}
		task.Whiteboards[i].Name = name
		task.Whiteboards[i].UpdatedAt = timeNowUTC()
		task.Touch()
		return &task.Whiteboards[i], nil
	}

	return nil, fmt.Errorf("whiteboard %s not found", whiteboardID)
}

func (b *Board) DeleteWhiteboard(taskID, whiteboardID string) (*Whiteboard, error) {
	task, ok := b.Tasks[taskID]
	if !ok || task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	for i := range task.Whiteboards {
		if task.Whiteboards[i].ID != whiteboardID {
			continue
		}
		removed := task.Whiteboards[i]
		task.Whiteboards = append(task.Whiteboards[:i], task.Whiteboards[i+1:]...)
		task.Touch()
		return &removed, nil
	}

	return nil, fmt.Errorf("whiteboard %s not found", whiteboardID)
}

func (b *Board) NextWhiteboardName(taskID string) string {
	task, ok := b.Tasks[taskID]
	if !ok || task == nil {
		return "Whiteboard 1"
	}

	used := make(map[int]struct{}, len(task.Whiteboards))
	for _, whiteboard := range task.Whiteboards {
		trimmed := strings.TrimSpace(whiteboard.Name)
		if !strings.HasPrefix(trimmed, "Whiteboard ") {
			continue
		}
		number, err := strconv.Atoi(strings.TrimPrefix(trimmed, "Whiteboard "))
		if err != nil || number < 1 {
			continue
		}
		used[number] = struct{}{}
	}

	for i := 1; ; i++ {
		if _, exists := used[i]; !exists {
			return fmt.Sprintf("Whiteboard %d", i)
		}
	}
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
	if _, ok := b.Order[next]; !ok {
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

	currentIndex := b.StatusIndex(task.Status)
	if currentIndex < 0 {
		return false
	}

	nextIndex := currentIndex + delta
	if nextIndex < 0 || nextIndex >= len(b.Columns) {
		return false
	}

	nextStatus := b.Columns[nextIndex]
	if _, ok := b.Order[nextStatus]; !ok {
		return false
	}

	return b.MoveTask(id, nextStatus, len(b.Order[nextStatus]))
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

func (b *Board) Statuses() []Status {
	return append([]Status{}, b.Columns...)
}

func (b *Board) StatusIndex(status Status) int {
	for i, candidate := range b.Columns {
		if candidate == status {
			return i
		}
	}
	return -1
}

func (b *Board) AddColumn(name string) (Status, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("column name cannot be empty")
	}

	status := Status(name)
	if len(b.Columns) == 0 {
		b.Columns = append([]Status{}, StatusOrder...)
	}
	if b.Order == nil {
		b.Order = make(map[Status][]string)
	}

	for _, existing := range b.Columns {
		if existing == status {
			return status, fmt.Errorf("column %s already exists", status)
		}
	}

	b.Columns = append(b.Columns, status)
	b.Order[status] = []string{}
	return status, nil
}

func (b *Board) MoveColumn(from, to int) bool {
	if from < 0 || from >= len(b.Columns) || to < 0 || to >= len(b.Columns) || from == to {
		return false
	}

	status := b.Columns[from]
	columns := make([]Status, 0, len(b.Columns))
	columns = append(columns, b.Columns[:from]...)
	columns = append(columns, b.Columns[from+1:]...)

	if to > from {
		to--
	}
	b.Columns = insertStatus(columns, to, status)
	return true
}

func (b *Board) RenameColumn(oldName, newName string) (Status, error) {
	oldStatus := normalizeStatus(Status(oldName))
	if oldStatus == "" {
		return "", fmt.Errorf("column name cannot be empty")
	}

	newName = strings.TrimSpace(newName)
	if newName == "" {
		return oldStatus, fmt.Errorf("column name cannot be empty")
	}

	newStatus := Status(newName)
	index := b.StatusIndex(oldStatus)
	if index < 0 {
		return oldStatus, fmt.Errorf("column %s not found", oldStatus)
	}

	if oldStatus == newStatus {
		return oldStatus, nil
	}

	for _, status := range b.Columns {
		if status == newStatus {
			return oldStatus, fmt.Errorf("column %s already exists", newStatus)
		}
	}

	if b.Order == nil {
		b.Order = make(map[Status][]string)
	}

	for _, task := range b.Tasks {
		if task != nil && task.Status == oldStatus {
			task.Status = newStatus
		}
	}

	if order, ok := b.Order[oldStatus]; ok {
		b.Order[newStatus] = append([]string{}, order...)
	} else if _, ok := b.Order[newStatus]; !ok {
		b.Order[newStatus] = []string{}
	}
	delete(b.Order, oldStatus)

	b.Columns[index] = newStatus
	return newStatus, nil
}

func (b *Board) DeleteColumn(status Status) error {
	status = normalizeStatus(status)
	if status == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	if len(b.Columns) <= 1 {
		return fmt.Errorf("cannot delete the last column")
	}

	index := b.StatusIndex(status)
	if index < 0 {
		return fmt.Errorf("column %s not found", status)
	}

	replacement := fallbackColumn(b.Columns, index)
	if replacement == "" {
		return fmt.Errorf("cannot determine replacement column for %s", status)
	}

	if _, ok := b.Order[replacement]; !ok {
		b.Order[replacement] = []string{}
	}

	if tasks := b.Order[status]; len(tasks) > 0 {
		b.Order[replacement] = append(b.Order[replacement], tasks...)
		for _, taskID := range tasks {
			if task, ok := b.Tasks[taskID]; ok && task != nil {
				task.Status = replacement
			}
		}
	}

	delete(b.Order, status)
	b.Columns = append(b.Columns[:index], b.Columns[index+1:]...)

	return nil
}

func normalizeColumns(statuses []Status) []Status {
	seen := make(map[Status]struct{}, len(statuses))
	result := make([]Status, 0, len(statuses))

	for _, status := range statuses {
		status = normalizeStatus(status)
		if status == "" {
			continue
		}
		if _, ok := seen[status]; ok {
			continue
		}

		seen[status] = struct{}{}
		result = append(result, status)
	}

	if len(result) == 0 {
		return append([]Status{}, StatusOrder...)
	}

	return result
}

func normalizeStatus(status Status) Status {
	return Status(strings.TrimSpace(string(status)))
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

func insertStatus(statuses []Status, index int, status Status) []Status {
	if index < 0 {
		index = 0
	}
	if index > len(statuses) {
		index = len(statuses)
	}

	out := make([]Status, 0, len(statuses)+1)
	out = append(out, statuses[:index]...)
	out = append(out, status)
	out = append(out, statuses[index:]...)
	return out
}

func fallbackColumn(columns []Status, deletedIndex int) Status {
	if len(columns) <= 1 || deletedIndex < 0 || deletedIndex >= len(columns) {
		return ""
	}

	if deletedIndex > 0 {
		return columns[deletedIndex-1]
	}

	return columns[1]
}

func hasWhiteboardName(whiteboards []Whiteboard, name, excludeID string) bool {
	nameKey := strings.ToLower(strings.TrimSpace(name))
	for _, whiteboard := range whiteboards {
		if whiteboard.ID == excludeID {
			continue
		}
		if strings.ToLower(strings.TrimSpace(whiteboard.Name)) == nameKey {
			return true
		}
	}
	return false
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}

func normalizeWhiteboards(task *Task) error {
	if task == nil || len(task.Whiteboards) == 0 {
		return nil
	}

	seenIDs := make(map[string]struct{}, len(task.Whiteboards))
	seenNames := make(map[string]struct{}, len(task.Whiteboards))
	normalized := make([]Whiteboard, 0, len(task.Whiteboards))
	for _, whiteboard := range task.Whiteboards {
		whiteboard.Name = strings.TrimSpace(whiteboard.Name)
		whiteboard.Path = strings.TrimSpace(whiteboard.Path)
		if whiteboard.Name == "" {
			return fmt.Errorf("whiteboard name cannot be empty")
		}
		if whiteboard.ID == "" {
			whiteboard.ID = newTaskID()
		}
		if _, exists := seenIDs[whiteboard.ID]; exists {
			return fmt.Errorf("duplicate whiteboard id %s", whiteboard.ID)
		}
		nameKey := strings.ToLower(whiteboard.Name)
		if _, exists := seenNames[nameKey]; exists {
			return fmt.Errorf("duplicate whiteboard name %s", whiteboard.Name)
		}
		if whiteboard.CreatedAt.IsZero() {
			whiteboard.CreatedAt = task.CreatedAt
		}
		if whiteboard.CreatedAt.IsZero() {
			whiteboard.CreatedAt = timeNowUTC()
		}
		if whiteboard.UpdatedAt.IsZero() {
			whiteboard.UpdatedAt = whiteboard.CreatedAt
		}

		seenIDs[whiteboard.ID] = struct{}{}
		seenNames[nameKey] = struct{}{}
		normalized = append(normalized, whiteboard)
	}

	task.Whiteboards = normalized
	return nil
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
