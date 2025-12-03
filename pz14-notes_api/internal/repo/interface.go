package repo

import (
	"context"
	"time"

	"example.com/pz14-notes_api/internal/core"
)

// NoteRepository интерфейс для работы с заметками
type NoteRepository interface {
	Create(ctx context.Context, n core.ReqNote) (*core.Note, error)
	Get(ctx context.Context, id int64) (*core.Note, error)
	GetMany(ctx context.Context, ids []int64) ([]*core.Note, error)
	ListWithKeysetPagination(ctx context.Context, params ListParams) ([]*core.Note, *KeysetCursor, error)
	Search(ctx context.Context, query string) ([]*core.Note, error)
	Update(ctx context.Context, n *core.Note) error
	Delete(ctx context.Context, id int64) error
}

// KeysetCursor для пагинации
type KeysetCursor struct {
	Timestamp time.Time
	ID        int64
}

// ListParams параметры для пагинации
type ListParams struct {
	PageSize int
	Cursor   *KeysetCursor
}
