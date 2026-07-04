package cache

import (
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// DagDashboardRow is the lightweight aggregate used by the future Monitor DAG
// dashboard. More expensive trend/percentile calculations can read raw history.
type DagDashboardRow struct {
	DagId        string
	Runs         int
	Success      int
	Failed       int
	Running      int
	Queued       int
	LastState    string
	LastRunAfter time.Time
	AvgDuration  time.Duration
	MaxDuration  time.Duration
	AvgQueueTime time.Duration
	FailedTasks  int
	RetriedTasks int
}

// Options configures persistent cache implementations.
type Options struct {
	Retention   time.Duration
	WriteBuffer int
}

// Cache is the history-cache abstraction over backfills, DAG runs, and task
// instances. SQLite is the persistent implementation; memory is the fallback.
//
// Contract:
//   - Put returns immediately. SQLite writes through an internal writer
//     goroutine, so reads may lag the latest Put by a short interval.
//   - Get returns a defensive copy that callers may mutate freely.
type Cache interface {
	GetBackfills(dagId string) ([]models.Backfill, bool)
	PutBackfills(dagId string, bfs []models.Backfill)

	PutDAGRuns(dagId string, runs []models.DAGRun)
	GetDAGRunsHistory(dagId string, since time.Time, limit int) ([]models.DAGRun, bool)
	GetAllDAGRunsHistory(since time.Time, limit int) ([]models.DAGRun, bool)

	PutTaskInstances(dagId, runId string, tasks []models.TaskInstance)
	GetTaskInstancesHistory(dagId string, since time.Time, limit int) ([]models.TaskInstance, bool)

	GetDagDashboardRows(since time.Time, limit int) ([]DagDashboardRow, bool)
	Close() error
}
