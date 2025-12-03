package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "time"

	"example.com/pz14-notes_api/internal/core"
	"github.com/lib/pq"
)

// NoteRepoPostgres PostgreSQL реализация репозитория
type NoteRepoPostgres struct {
	db *sql.DB
}

// NewNoteRepoPostgres создает новый PostgreSQL репозиторий
func NewNoteRepoPostgres(db *sql.DB) *NoteRepoPostgres {
	return &NoteRepoPostgres{db: db}
}

// Create создает новую заметку
func (r *NoteRepoPostgres) Create(ctx context.Context, n core.ReqNote) (*core.Note, error) {
	stmt, err := r.db.PrepareContext(ctx, `
		INSERT INTO notes (title, content, created_at, updated_at)
		VALUES ($1, $2, now(), now())
		RETURNING id, title, content, created_at, updated_at
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	note := &core.Note{}
	err = stmt.QueryRowContext(ctx, n.Title, n.Content).Scan(
		&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return note, nil
}

// Get получает заметку по ID
func (r *NoteRepoPostgres) Get(ctx context.Context, id int64) (*core.Note, error) {
	stmt, err := r.db.PrepareContext(ctx, `
		SELECT id, title, content, created_at, updated_at
		FROM notes WHERE id = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	note := &core.Note{}
	err = stmt.QueryRowContext(ctx, id).Scan(
		&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return note, nil
}

// GetMany получает несколько заметок (батчинг)
func (r *NoteRepoPostgres) GetMany(ctx context.Context, ids []int64) ([]*core.Note, error) {
	if len(ids) == 0 {
		return []*core.Note{}, nil
	}

	stmt, err := r.db.PrepareContext(ctx, `
		SELECT id, title, content, created_at, updated_at
		FROM notes WHERE id = ANY($1)
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var notes []*core.Note
	for rows.Next() {
		note := &core.Note{}
		err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		notes = append(notes, note)
	}

	return notes, rows.Err()
}

// ListWithKeysetPagination keyset-пагинация
func (r *NoteRepoPostgres) ListWithKeysetPagination(
	ctx context.Context,
	params ListParams,
) ([]*core.Note, *KeysetCursor, error) {
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	var rows *sql.Rows
	var err error

	if params.Cursor == nil {
		stmt, err := r.db.PrepareContext(ctx, `
			SELECT id, title, content, created_at, updated_at
			FROM notes
			ORDER BY created_at DESC, id DESC
			LIMIT $1
		`)
		if err != nil {
			return nil, nil, fmt.Errorf("prepare failed: %w", err)
		}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, params.PageSize+1)
	} else {
		stmt, err := r.db.PrepareContext(ctx, `
			SELECT id, title, content, created_at, updated_at
			FROM notes
			WHERE (created_at, id) < ($1, $2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
		`)
		if err != nil {
			return nil, nil, fmt.Errorf("prepare failed: %w", err)
		}
		defer stmt.Close()
		rows, err = stmt.QueryContext(ctx, params.Cursor.Timestamp, params.Cursor.ID, params.PageSize+1)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var notes []*core.Note
	for rows.Next() {
		note := &core.Note{}
		err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("scan failed: %w", err)
		}
		notes = append(notes, note)
	}

	// Проверяем наличие следующей страницы
	var nextCursor *KeysetCursor
	if len(notes) > params.PageSize {
		lastNote := notes[params.PageSize-1]
		nextCursor = &KeysetCursor{
			Timestamp: lastNote.CreatedAt,
			ID:        lastNote.ID,
		}
		notes = notes[:params.PageSize]
	}

	return notes, nextCursor, nil
}

// Search полнотекстовый поиск
func (r *NoteRepoPostgres) Search(ctx context.Context, query string) ([]*core.Note, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	stmt, err := r.db.PrepareContext(ctx, `
		SELECT id, title, content, created_at, updated_at
		FROM notes
		WHERE to_tsvector('simple', title) @@ plainto_tsquery('simple', $1)
		   OR to_tsvector('simple', content) @@ plainto_tsquery('simple', $1)
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var notes []*core.Note
	for rows.Next() {
		note := &core.Note{}
		err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		notes = append(notes, note)
	}

	return notes, rows.Err()
}

// Update обновляет заметку
func (r *NoteRepoPostgres) Update(ctx context.Context, n *core.Note) error {
	stmt, err := r.db.PrepareContext(ctx, `
		UPDATE notes
		SET title = $1, content = $2, updated_at = now()
		WHERE id = $3
		RETURNING updated_at
	`)
	if err != nil {
		return fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(ctx, n.Title, n.Content, n.ID).Scan(&n.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.ErrNotFound
		}
		return fmt.Errorf("query failed: %w", err)
	}

	return nil
}

// Delete удаляет заметку
func (r *NoteRepoPostgres) Delete(ctx context.Context, id int64) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM notes WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return core.ErrNotFound
	}

	return nil
}
