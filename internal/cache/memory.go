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

// memoryCache is a TTL-based in-memory cache and fallback history cache.
type memoryCache struct {
	mu            sync.RWMutex
	ttl           time.Duration
	backfills     map[string]cachedBackfills
	dagRuns       map[string][]models.DAGRun
	taskInstances map[string][]models.TaskInstance
}

func NewMemory(ttl time.Duration) Cache {
	return &memoryCache{
		ttl:           ttl,
		backfills:     make(map[string]cachedBackfills),
		dagRuns:       make(map[string][]models.DAGRun),
		taskInstances: make(map[string][]models.TaskInstance),
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

func (m *memoryCache) PutDAGRuns(dagId string, runs []models.DAGRun) {
	dup := make([]models.DAGRun, len(runs))
	copy(dup, runs)
	m.mu.Lock()
	m.dagRuns[dagId] = mergeDAGRuns(m.dagRuns[dagId], dup)
	m.mu.Unlock()
}

func (m *memoryCache) GetDAGRunsHistory(dagId string, since time.Time, limit int) ([]models.DAGRun, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return filterDAGRuns(m.dagRuns[dagId], since, limit)
}

func (m *memoryCache) GetAllDAGRunsHistory(since time.Time, limit int) ([]models.DAGRun, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := make([]models.DAGRun, 0)
	for _, runs := range m.dagRuns {
		all = append(all, runs...)
	}
	return filterDAGRuns(all, since, limit)
}

func (m *memoryCache) PutTaskInstances(dagId, runId string, tasks []models.TaskInstance) {
	key := dagId + "/" + runId
	dup := make([]models.TaskInstance, len(tasks))
	copy(dup, tasks)
	m.mu.Lock()
	m.taskInstances[key] = mergeTaskInstances(m.taskInstances[key], dup)
	m.mu.Unlock()
}

func (m *memoryCache) GetTaskInstancesHistory(dagId string, since time.Time, limit int) ([]models.TaskInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := make([]models.TaskInstance, 0)
	for _, tasks := range m.taskInstances {
		for _, ti := range tasks {
			if ti.DagId == dagId {
				all = append(all, ti)
			}
		}
	}
	return filterTaskInstances(all, since, limit)
}

func (m *memoryCache) GetDagDashboardRows(since time.Time, limit int) ([]DagDashboardRow, bool) {
	runs, ok := m.GetAllDAGRunsHistory(since, 0)
	if !ok {
		return nil, false
	}
	m.mu.RLock()
	tasks := make([]models.TaskInstance, 0)
	for _, tis := range m.taskInstances {
		tasks = append(tasks, tis...)
	}
	m.mu.RUnlock()
	rows := dashboardRows(runs, tasks, since, limit)
	return rows, len(rows) > 0
}

func (m *memoryCache) Close() error { return nil }
