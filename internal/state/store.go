package state

import (
	"maps"
	"sync"
	"time"

	"github.com/yjinheon/lazyflow/internal/debugutil"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// Event names used for subscriptions.
const (
	EventDAGsUpdated          = "dags_updated"
	EventDAGRunsUpdated       = "dag_runs_updated"
	EventDAGSelected          = "dag_selected"
	EventRunSelected          = "run_selected"
	EventTaskSelected         = "task_selected"
	EventTaskInstancesUpdated = "task_instances_updated"
	EventTabChanged           = "tab_changed"
	EventHealthUpdated        = "health_updated"
	EventBackfillsUpdated     = "backfills_updated"
	EventBackfillSelected     = "backfill_selected"
	EventBackfillsActioned    = "backfills_actioned"
	EventGanttModeChanged     = "tasks_gantt_mode"
	EventLineageUpdated       = "lineage_updated"
	EventCriticalPathChanged  = "critical_path_changed"
	EventPoolsUpdated         = "pools_updated"
	EventDAGStateRollupUpdated = "dag_state_rollup_updated"
)

type Store struct {
	mu sync.RWMutex

	// Data cache
	dags             []models.DAG
	dagRuns          map[string][]models.DAGRun       // dagId -> runs
	taskInstances    map[string][]models.TaskInstance // "dagId/runId" -> tasks
	health           *models.HealthInfo
	tasks            map[string][]models.Task // dagId → lineage
	backfills        map[string][]models.Backfill
	selectedBackfill int
	ganttMode        bool
	criticalPath     map[string]bool
	pools            []models.Pool
	dagStateRollup   map[string]string // dagId -> latest run state (cluster-wide)

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
		dags:             make([]models.DAG, 0),
		dagRuns:          make(map[string][]models.DAGRun),
		taskInstances:    make(map[string][]models.TaskInstance),
		lastRefresh:      make(map[string]time.Time),
		cacheTTL:         5 * time.Second,
		subscribers:      make(map[string][]func(any)),
		tasks:            make(map[string][]models.Task),
		backfills:        make(map[string][]models.Backfill),
		selectedBackfill: -1,
		criticalPath:     make(map[string]bool),
		dagStateRollup:   make(map[string]string),
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

	debugutil.Tag("FZ-store", "notify event=%s handlers=%d START", event, len(handlers))
	tStart := time.Now()
	for i, h := range handlers {
		hStart := time.Now()
		h(data)
		if d := time.Since(hStart); d > 50*time.Millisecond {
			debugutil.Tag("FZ-store", "notify event=%s handler[%d] SLOW elapsed=%v", event, i, d)
		}
	}
	debugutil.Tag("FZ-store", "notify event=%s END elapsed=%v", event, time.Since(tStart))
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

	s.notify(EventDAGRunsUpdated, dagId)
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

	s.notify(EventTaskInstancesUpdated, key)
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

// ---------- Pools ----------

func (s *Store) SetPools(pools []models.Pool) {
	s.mu.Lock()
	s.pools = pools
	s.lastRefresh["pools"] = time.Now()
	s.mu.Unlock()

	s.notify(EventPoolsUpdated, pools)
}

func (s *Store) GetPools() []models.Pool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Pool, len(s.pools))
	copy(out, s.pools)
	return out
}

// ---------- DAG state rollup ----------

// SetDAGStateRollup replaces the cluster-wide dagId→latest-run-state map.
func (s *Store) SetDAGStateRollup(rollup map[string]string) {
	s.mu.Lock()
	s.dagStateRollup = rollup
	s.lastRefresh["dag_state_rollup"] = time.Now()
	s.mu.Unlock()

	s.notify(EventDAGStateRollupUpdated, rollup)
}

// GetDAGStateRollup returns a defensive copy of the dagId→latest-run-state map.
func (s *Store) GetDAGStateRollup() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.dagStateRollup))
	maps.Copy(out, s.dagStateRollup)
	return out
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

// ---------- Tasks (lineage) ----------

// tasks cache (DAG-level lineage). Keyed by dag_id.
// Used by critical-path computation in the Gantt view.

func (s *Store) SetTasks(dagId string, tasks []models.Task) {
	s.mu.Lock()
	if s.tasks == nil {
		s.tasks = make(map[string][]models.Task)
	}
	s.tasks[dagId] = tasks
	s.mu.Unlock()

	s.notify(EventLineageUpdated, dagId)
}

func (s *Store) GetTasks(dagId string) []models.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	in := s.tasks[dagId]
	out := make([]models.Task, len(in))
	copy(out, in)
	return out
}

// ---------- Backfills ----------

func (s *Store) SetBackfills(dagId string, bfs []models.Backfill) {
	s.mu.Lock()
	s.backfills[dagId] = bfs
	s.mu.Unlock()

	s.notify(EventBackfillsUpdated, dagId)
}

func (s *Store) GetBackfills(dagId string) []models.Backfill {
	s.mu.RLock()
	defer s.mu.RUnlock()
	in := s.backfills[dagId]
	out := make([]models.Backfill, len(in))
	copy(out, in)
	return out
}

func (s *Store) SelectBackfill(id int) {
	s.mu.Lock()
	s.selectedBackfill = id
	s.mu.Unlock()

	s.notify(EventBackfillSelected, id)
}

func (s *Store) SelectedBackfill() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedBackfill
}

// ---------- Gantt mode ----------

func (s *Store) SetGanttMode(on bool) {
	s.mu.Lock()
	changed := s.ganttMode != on
	s.ganttMode = on
	s.mu.Unlock()

	if changed {
		s.notify(EventGanttModeChanged, on)
	}
}

func (s *Store) GanttMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ganttMode
}

// ---------- Critical path ----------

// SetCriticalPath replaces the critical-path set and notifies only when changed.
// This prevents needless redraws when running-task duration "moves" but the
// path remains the same.
func (s *Store) SetCriticalPath(set map[string]bool) {
	s.mu.Lock()
	changed := !mapsEqual(s.criticalPath, set)
	s.criticalPath = set
	s.mu.Unlock()

	if changed {
		s.notify(EventCriticalPathChanged, set)
	}
}

func (s *Store) IsOnCriticalPath(taskId string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.criticalPath[taskId]
}

// GetCriticalPath returns a defensive copy of the critical-path set.
func (s *Store) GetCriticalPath() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]bool, len(s.criticalPath))
	maps.Copy(out, s.criticalPath)
	return out
}

func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
