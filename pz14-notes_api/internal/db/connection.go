package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Config настройки connection pool
type Config struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewDB инициализирует базу данных с пулом соединений
func NewDB(databaseURL string, cfg *Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	// Значения по умолчанию
	if cfg == nil {
		cfg = &Config{
			MaxOpenConns:    20,
			MaxIdleConns:    10,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		}
	}

	// Применяем настройки пула
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	log.Println("✅ Database connected successfully")
	PrintPoolStats(db)

	// Инициализируем схему
	if err := InitSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return db, nil
}

// PrintPoolStats выводит статистику пула
func PrintPoolStats(db *sql.DB) {
	stats := db.Stats()
	log.Printf("=== Connection Pool Stats ===")
	log.Printf("OpenConnections: %d", stats.OpenConnections)
	log.Printf("InUse: %d", stats.InUse)
	log.Printf("Idle: %d", stats.Idle)
	log.Printf("WaitCount: %d", stats.WaitCount)
	log.Printf("==============================")
}

// InitSchema создает таблицы и индексы
func InitSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS notes (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	content TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notes_created_id
ON notes (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_notes_title_gin
ON notes USING GIN (to_tsvector('simple', title));

CREATE INDEX IF NOT EXISTS idx_notes_id ON notes (id);

-- Убрали проблемный частичный индекс с функцией now()

CREATE INDEX IF NOT EXISTS idx_notes_updated_desc ON notes (updated_at DESC);

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
`

	_, err := db.Exec(schema)
	if err != nil {
		// Игнорируем ошибку связанного с volatile функциями, если осталась
		if strings.Contains(err.Error(), "functions in index predicate must be marked IMMUTABLE") {
			log.Println("⚠️  Ignored IMMUTABLE function error in index predicate")
			return nil
		}
		return fmt.Errorf("failed to exec schema: %w", err)
	}

	log.Println("✅ Schema initialized")
	return nil
}
