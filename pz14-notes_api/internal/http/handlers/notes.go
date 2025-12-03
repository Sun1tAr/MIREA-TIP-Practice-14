package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/pz14-notes_api/internal/core"
	"example.com/pz14-notes_api/internal/repo"
	"github.com/gin-gonic/gin"
)

// Handler HTTP обработчик для заметок
type Handler struct {
	Repo repo.NoteRepository
}

// CreateNote создает новую заметку
func (h *Handler) CreateNote(c *gin.Context) {
	ctx := c.Request.Context()

	var req core.ReqNote
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	note, err := h.Repo.Create(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": note})
}

// GetNote получает заметку по ID
func (h *Handler) GetNote(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	note, err := h.Repo.Get(ctx, id)
	if err != nil {
		if err == core.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": note})
}

// ListNotes список с keyset-пагинацией
func (h *Handler) ListNotes(c *gin.Context) {
	ctx := c.Request.Context()

	pageSize := 20
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	var cursor *repo.KeysetCursor
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		parts := strings.Split(cursorStr, ":")
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				if id, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					cursor = &repo.KeysetCursor{Timestamp: t, ID: id}
				}
			}
		}
	}

	notes, nextCursor, err := h.Repo.ListWithKeysetPagination(ctx, repo.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	response := gin.H{"data": notes}
	if nextCursor != nil {
		nextCursorStr := fmt.Sprintf("%s:%d", nextCursor.Timestamp.Format(time.RFC3339), nextCursor.ID)
		response["next_cursor"] = nextCursorStr
	}

	c.JSON(http.StatusOK, response)
}

// GetManyNotes батчинг получение
func (h *Handler) GetManyNotes(c *gin.Context) {
	ctx := c.Request.Context()

	idsStr := c.Query("ids")
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids parameter required"})
		return
	}

	var ids []int64
	for _, idStr := range strings.Split(idsStr, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
			return
		}
		ids = append(ids, id)
	}

	notes, err := h.Repo.GetMany(ctx, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": notes})
}

// SearchNotes полнотекстовый поиск
func (h *Handler) SearchNotes(c *gin.Context) {
	ctx := c.Request.Context()

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q parameter required"})
		return
	}

	notes, err := h.Repo.Search(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"query": query, "count": len(notes), "data": notes})
}

// UpdateNote обновляет заметку
func (h *Handler) UpdateNote(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req core.ReqNote
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	note := &core.Note{
		ID:      id,
		Title:   req.Title,
		Content: req.Content,
	}

	if err := h.Repo.Update(ctx, note); err != nil {
		if err == core.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": note})
}

// DeleteNote удаляет заметку
func (h *Handler) DeleteNote(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.Repo.Delete(ctx, id); err != nil {
		if err == core.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
