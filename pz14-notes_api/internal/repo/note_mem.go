package repo

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"example.com/pz14-notes_api/internal/core"
)

// NoteRepoMem in-memory реализация репозитория
type NoteRepoMem struct {
	mu     sync.RWMutex
	notes  map[int64]*core.Note
	nextID int64
}

// NewNoteRepoMem создает новый in-memory репозиторий
func NewNoteRepoMem() *NoteRepoMem {
	return &NoteRepoMem{
		notes:  make(map[int64]*core.Note),
		nextID: 1,
	}
}

// Create создает новую заметку
func (r *NoteRepoMem) Create(ctx context.Context, n core.ReqNote) (*core.Note, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	now := core.Now()
	note := &core.Note{
		ID:        r.nextID,
		Title:     n.Title,
		Content:   n.Content,
		CreatedAt: now,
		UpdatedAt: &now,
	}

	r.notes[note.ID] = note
	r.nextID++

	return note, nil
}

// Get получает заметку по ID
func (r *NoteRepoMem) Get(ctx context.Context, id int64) (*core.Note, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	note, ok := r.notes[id]
	if !ok {
		return nil, core.ErrNotFound
	}

	return note, nil
}

// GetMany получает несколько заметок (батчинг)
func (r *NoteRepoMem) GetMany(ctx context.Context, ids []int64) ([]*core.Note, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if len(ids) == 0 {
		return []*core.Note{}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*core.Note
	for _, id := range ids {
		if note, ok := r.notes[id]; ok {
			result = append(result, note)
		}
	}

	return result, nil
}

// ListWithKeysetPagination keyset-пагинация
func (r *NoteRepoMem) ListWithKeysetPagination(
	ctx context.Context,
	params ListParams,
) ([]*core.Note, *KeysetCursor, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	// Копируем все заметки внутри блокировки
	r.mu.RLock()
	all := make([]*core.Note, 0, len(r.notes))
	for _, note := range r.notes {
		all = append(all, note)
	}
	r.mu.RUnlock()

	// Сортировка вне блокировки
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].ID > all[j].ID
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	// Применяем cursor
	start := 0
	if params.Cursor != nil {
		for i, note := range all {
			if note.CreatedAt.Before(params.Cursor.Timestamp) ||
				(note.CreatedAt.Equal(params.Cursor.Timestamp) && note.ID < params.Cursor.ID) {
				start = i
				break
			}
		}
	}

	// Возвращаем страницу
	end := start + params.PageSize + 1
	if end > len(all) {
		end = len(all)
	}

	pageResult := all[start:end]

	var nextCursor *KeysetCursor
	// ✅ ПРАВИЛЬНО: Берём СЛЕДУЮЩУЮ запись (индекс PageSize = 20)
	if len(pageResult) > params.PageSize {
		nextNote := pageResult[params.PageSize] // Индекс 20 (21-й элемент)
		nextCursor = &KeysetCursor{
			Timestamp: nextNote.CreatedAt,
			ID:        nextNote.ID,
		}
		pageResult = pageResult[:params.PageSize] // Возвращаем только 20
	}

	return pageResult, nextCursor, nil
}

// Search полнотекстовый поиск
func (r *NoteRepoMem) Search(ctx context.Context, query string) ([]*core.Note, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedQuery := strings.ToLower(query)
	var result []*core.Note

	for _, note := range r.notes {
		if strings.Contains(strings.ToLower(note.Title), normalizedQuery) ||
			strings.Contains(strings.ToLower(note.Content), normalizedQuery) {
			result = append(result, note)
		}
	}

	return result, nil
}

// Update обновляет заметку
func (r *NoteRepoMem) Update(ctx context.Context, n *core.Note) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.notes[n.ID]
	if !ok {
		return core.ErrNotFound
	}

	now := core.Now()

	existing.Title = n.Title
	existing.Content = n.Content
	existing.UpdatedAt = &now

	return nil
}

// Delete удаляет заметку
func (r *NoteRepoMem) Delete(ctx context.Context, id int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.notes[id]; !ok {
		return core.ErrNotFound
	}

	delete(r.notes, id)
	return nil
}
