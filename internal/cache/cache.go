package cache

import "github.com/yjinheon/lazyflow/pkg/airflow/models"

// Cache is the Phase 1 abstraction over backfill caching. The interface is
// designed so a Phase 2 SQLite-backed implementation can swap in without
// touching call sites.
//
// Contract:
//   - Put returns immediately. In-memory impl is O(1); SQLite impl will use
//     an internal channel and writer goroutine.
//   - Get returns a defensive copy that callers may mutate freely.
type Cache interface {
	GetBackfills(dagId string) ([]models.Backfill, bool)
	PutBackfills(dagId string, bfs []models.Backfill)
	Close() error
}
