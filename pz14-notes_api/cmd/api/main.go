package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/pz14-notes_api/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	apihttp "example.com/pz14-notes_api/internal/http" // алиас для http-сервера с маршрутами
	"example.com/pz14-notes_api/internal/http/handlers"
	"example.com/pz14-notes_api/internal/repo"
	_ "github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load("C:/0_DATA/MyDATA/Coding/Common/.env")

	if err != nil {
		log.Println("Warning: .env file not found")
	}
	usePostgres := err == nil

	var repository repo.NoteRepository

	if usePostgres {
		databaseURL := os.Getenv("POSTGRES_URL_GO")
		if databaseURL == "" {
			databaseURL = "postgres://notes_user:notes_password@localhost:5432/notes_db?sslmode=disable"
		}

		sqlDB, err := db.NewDB(databaseURL, &db.Config{
			MaxOpenConns:    20,
			MaxIdleConns:    10,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		})
		if err != nil {
			log.Fatalf("failed to init db: %v", err)
		}

		repository = repo.NewNoteRepoPostgres(sqlDB)
		log.Println("✅ Using PostgreSQL")

		defer sqlDB.Close()
	} else {
		repository = repo.NewNoteRepoMem()
		log.Println("✅ Using in-memory")
	}

	router := gin.Default()
	h := &handlers.Handler{Repo: repository}

	// Вызов SetupRoutes из правильного пакета (алиас apihttp)
	apihttp.SetupRoutes(router, h)

	// Graceful shutdown
	server := &http.Server{Addr: ":8080", Handler: router}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
