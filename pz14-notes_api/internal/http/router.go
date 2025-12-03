package http

import (
	"example.com/pz14-notes_api/internal/http/handlers"
	"github.com/gin-gonic/gin"
)

// SetupRoutes настраивает все маршруты
func SetupRoutes(router *gin.Engine, handler *handlers.Handler) {
	api := router.Group("/api/v1")
	{
		// Оптимизированные маршруты
		api.GET("/notes", handler.ListNotes)
		api.GET("/notes/batch", handler.GetManyNotes)
		api.GET("/notes/search", handler.SearchNotes)

		// CRUD операции
		api.GET("/notes/:id", handler.GetNote)
		api.POST("/notes", handler.CreateNote)
		api.PATCH("/notes/:id", handler.UpdateNote)
		api.DELETE("/notes/:id", handler.DeleteNote)
	}
}
