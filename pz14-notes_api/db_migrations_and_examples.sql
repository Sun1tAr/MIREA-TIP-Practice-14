-- Таблица заметок

CREATE TABLE IF NOT EXISTS notes (
                                     id BIGSERIAL PRIMARY KEY,
                                     title TEXT NOT NULL,
                                     content TEXT NOT NULL,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

-- Индекс для keyset-пагинации

CREATE INDEX IF NOT EXISTS idx_notes_created_id
    ON notes (created_at DESC, id DESC);

-- GIN индекс для полнотекстового поиска

CREATE INDEX IF NOT EXISTS idx_notes_title_gin
    ON notes USING GIN (to_tsvector('simple', title));

-- B-tree индекс для точного поиска

CREATE INDEX IF NOT EXISTS idx_notes_id ON notes (id);

-- Убрали problemный partial индекс с volatile функцией now()
-- Вместо этого простой индекс по полю updated_at

CREATE INDEX IF NOT EXISTS idx_notes_updated_desc ON notes (updated_at DESC);

-- Включаем pg_stat_statements для мониторинга

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- ПРИМЕРЫ EXPLAIN/ANALYZE

-- 1. Простой запрос (Index Scan)

-- EXPLAIN (ANALYZE, BUFFERS)

-- SELECT id, title FROM notes WHERE id = 123;

-- 2. Keyset-пагинация (Index Scan Backward)

-- EXPLAIN (ANALYZE, BUFFERS)

-- SELECT id, title, content, created_at

-- FROM notes

-- WHERE (created_at, id) < ('2025-01-01 12:00:00+00:00'::timestamptz, 123::bigint)

-- ORDER BY created_at DESC, id DESC

-- LIMIT 20;

-- 3. Полнотекстовый поиск (GIN Index Scan)

-- EXPLAIN (ANALYZE, BUFFERS)

-- SELECT id, title FROM notes

-- WHERE to_tsvector('simple', title) @@ plainto_tsquery('simple', 'test');

-- 4. Батчинг (Index Scan)

-- EXPLAIN (ANALYZE, BUFFERS)

-- SELECT id, title FROM notes WHERE id = ANY(ARRAY[1,2,3,4,5]);

-- Топ медленных запросов

-- SELECT query, calls, total_time, mean_time, rows

-- FROM pg_stat_statements

-- ORDER BY total_time DESC

-- LIMIT 10;
