package cache

import (
	"sync"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type cachedBackfills struct {
	bfs []models.Backfill
	at  time.Time
}

// memoryCache is a TTL-based in-memory cache, Phase 1 impl.
type memoryCache struct {
	mu        sync.RWMutex
	ttl       time.Duration
	backfills map[string]cachedBackfills
}

func NewMemory(ttl time.Duration) Cache {
	return &memoryCache{
		ttl:       ttl,
		backfills: make(map[string]cachedBackfills),
	}
}

func (m *memoryCache) GetBackfills(dagId string) ([]models.Backfill, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.backfills[dagId]
	if !ok || time.Since(c.at) > m.ttl {
		return nil, false
	}
	out := make([]models.Backfill, len(c.bfs))
	copy(out, c.bfs)
	return out, true
}

func (m *memoryCache) PutBackfills(dagId string, bfs []models.Backfill) {
	dup := make([]models.Backfill, len(bfs))
	copy(dup, bfs)
	m.mu.Lock()
	m.backfills[dagId] = cachedBackfills{bfs: dup, at: time.Now()}
	m.mu.Unlock()
}

func (m *memoryCache) Close() error { return nil }
