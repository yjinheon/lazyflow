package state

import (
	"sync"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// Event names used for subscriptions.
const (
	EventDAGsUpdated     = "dags_updated"
	EventDAGSelected     = "dag_selected"
	EventRunSelected     = "run_selected"
	EventTaskSelected    = "task_selected"
	EventTabChanged      = "tab_changed"
	EventHealthUpdated   = "health_updated"
)

type Store struct {
	mu sync.RWMutex

	// Data cache
	dags          []models.DAG
	dagRuns       map[string][]models.DAGRun       // dagId -> runs
	taskInstances map[string][]models.TaskInstance  // "dagId/runId" -> tasks
	health        *models.HealthInfo

	// Selection state
	selectedDAG  string
	selectedRun  string
	selectedTask string
	activeTab    string

	// Cache metadata
	lastRefresh map[string]time.Time
	cacheTTL    time.Duration

	// Observer pattern
	subscribers map[string][]func(any)
}

func NewStore() *Store {
	return &Store{
		dags:          make([]models.DAG, 0),
		dagRuns:       make(map[string][]models.DAGRun),
		taskInstances: make(map[string][]models.TaskInstance),
		lastRefresh:   make(map[string]time.Time),
		cacheTTL:      5 * time.Second,
		subscribers:   make(map[string][]func(any)),
	}
}

// ---------- Subscribe / Notify ----------

func (s *Store) Subscribe(event string, handler func(any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[event] = append(s.subscribers[event], handler)
}

func (s *Store) notify(event string, data any) {
	s.mu.RLock()
	handlers := make([]func(any), len(s.subscribers[event]))
	copy(handlers, s.subscribers[event])
	s.mu.RUnlock()

	for _, h := range handlers {
		h(data)
	}
}

// ---------- DAGs ----------

func (s *Store) SetDAGs(dags []models.DAG) {
	s.mu.Lock()
	s.dags = dags
	s.lastRefresh["dags"] = time.Now()
	s.mu.Unlock()

	s.notify(EventDAGsUpdated, dags)
}

func (s *Store) GetDAGs() []models.DAG {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.DAG, len(s.dags))
	copy(out, s.dags)
	return out
}

// ---------- DAG Runs ----------

func (s *Store) SetDAGRuns(dagId string, runs []models.DAGRun) {
	s.mu.Lock()
	s.dagRuns[dagId] = runs
	s.lastRefresh["runs:"+dagId] = time.Now()
	s.mu.Unlock()
}

func (s *Store) GetDAGRuns(dagId string) []models.DAGRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runs := s.dagRuns[dagId]
	out := make([]models.DAGRun, len(runs))
	copy(out, runs)
	return out
}

// ---------- Task Instances ----------

func (s *Store) SetTaskInstances(dagId, runId string, tasks []models.TaskInstance) {
	key := dagId + "/" + runId
	s.mu.Lock()
	s.taskInstances[key] = tasks
	s.lastRefresh["tasks:"+key] = time.Now()
	s.mu.Unlock()
}

func (s *Store) GetTaskInstances(dagId, runId string) []models.TaskInstance {
	key := dagId + "/" + runId
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := s.taskInstances[key]
	out := make([]models.TaskInstance, len(tasks))
	copy(out, tasks)
	return out
}

// ---------- Health ----------

func (s *Store) SetHealth(h *models.HealthInfo) {
	s.mu.Lock()
	s.health = h
	s.lastRefresh["health"] = time.Now()
	s.mu.Unlock()

	s.notify(EventHealthUpdated, h)
}

func (s *Store) GetHealth() *models.HealthInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.health
}

// ---------- Selection ----------

func (s *Store) SelectDAG(dagId string) {
	s.mu.Lock()
	s.selectedDAG = dagId
	s.selectedRun = ""
	s.selectedTask = ""
	s.mu.Unlock()

	s.notify(EventDAGSelected, dagId)
}

func (s *Store) SelectedDAG() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedDAG
}

func (s *Store) SelectRun(runId string) {
	s.mu.Lock()
	s.selectedRun = runId
	s.selectedTask = ""
	s.mu.Unlock()

	s.notify(EventRunSelected, runId)
}

func (s *Store) SelectedRun() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedRun
}

func (s *Store) SelectTask(taskId string) {
	s.mu.Lock()
	s.selectedTask = taskId
	s.mu.Unlock()

	s.notify(EventTaskSelected, taskId)
}

func (s *Store) SelectedTask() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedTask
}

// ---------- Tab ----------

func (s *Store) SetActiveTab(tab string) {
	s.mu.Lock()
	s.activeTab = tab
	s.mu.Unlock()

	s.notify(EventTabChanged, tab)
}

func (s *Store) ActiveTab() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeTab
}

// ---------- Cache ----------

func (s *Store) NeedsRefresh(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.lastRefresh[key]
	if !ok {
		return true
	}
	return time.Since(t) > s.cacheTTL
}
