package cache

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func newTestSQLite(t *testing.T, retention time.Duration) Cache {
	t.Helper()
	c, err := NewSQLite(filepath.Join(t.TempDir(), "cache.db"), Options{
		Retention:   retention,
		WriteBuffer: 64,
	})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	return c
}

func TestSQLite_schemaMigration(t *testing.T) {
	c := newTestSQLite(t, 24*time.Hour)
	defer c.Close()
	sc, ok := c.(*sqliteCache)
	if !ok {
		t.Fatal("expected sqliteCache")
	}

	var name string
	if err := sc.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='dag_runs'").Scan(&name); err != nil {
		t.Fatalf("dag_runs table missing: %v", err)
	}
}

func TestSQLite_putDAGRunsHistoryUpsert(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "cache.db")
	c, err := NewSQLite(path, Options{Retention: 24 * time.Hour, WriteBuffer: 16})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	c.PutDAGRuns("etl", []models.DAGRun{
		{DagId: "etl", RunId: "r1", State: "success", RunAfter: now.Add(-time.Hour)},
		{DagId: "etl", RunId: "r2", State: "running", RunAfter: now},
	})
	c.PutDAGRuns("etl", []models.DAGRun{
		{DagId: "etl", RunId: "r2", State: "failed", RunAfter: now},
	})
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := NewSQLite(path, Options{Retention: 24 * time.Hour, WriteBuffer: 16})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	got, ok := reopened.GetDAGRunsHistory("etl", now.Add(-2*time.Hour), 10)
	if !ok || len(got) != 2 {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
	if got[0].RunId != "r2" || got[0].State != "failed" {
		t.Fatalf("newest/upsert mismatch: %+v", got[0])
	}
}

func TestSQLite_dashboardAggregates(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "cache.db")
	c, err := NewSQLite(path, Options{Retention: 24 * time.Hour, WriteBuffer: 16})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	queued := now.Add(-2 * time.Minute)
	start := now.Add(-time.Minute)
	end := now
	c.PutDAGRuns("etl", []models.DAGRun{
		{DagId: "etl", RunId: "r1", State: "failed", RunAfter: now, StartDate: start, EndDate: end},
	})
	c.PutTaskInstances("etl", "r1", []models.TaskInstance{
		{DagId: "etl", RunId: "r1", TaskId: "extract", State: "failed", QueuedDttm: &queued, StartDate: &start, EndDate: &end, TryNumber: 2},
	})
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := NewSQLite(path, Options{Retention: 24 * time.Hour, WriteBuffer: 16})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	tasks, ok := reopened.GetTaskInstancesHistory("etl", now.Add(-time.Hour), 10)
	if !ok || len(tasks) != 1 || tasks[0].Duration != 60 {
		t.Fatalf("tasks=%+v ok=%v", tasks, ok)
	}
	rows, ok := reopened.GetDagDashboardRows(now.Add(-time.Hour), 10)
	if !ok || len(rows) != 1 {
		t.Fatalf("rows=%+v ok=%v", rows, ok)
	}
	if rows[0].Failed != 1 || rows[0].FailedTasks != 1 || rows[0].RetriedTasks != 1 || rows[0].AvgQueueTime != time.Minute {
		t.Fatalf("unexpected dashboard row: %+v", rows[0])
	}
}

func TestSQLite_retentionRemovesOldRows(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "cache.db")
	c, err := NewSQLite(path, Options{Retention: time.Hour, WriteBuffer: 16})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	c.PutDAGRuns("etl", []models.DAGRun{
		{DagId: "etl", RunId: "old", State: "success", RunAfter: now.Add(-2 * time.Hour)},
		{DagId: "etl", RunId: "new", State: "success", RunAfter: now},
	})
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := NewSQLite(path, Options{Retention: time.Hour, WriteBuffer: 16})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	got, ok := reopened.GetDAGRunsHistory("etl", now.Add(-3*time.Hour), 10)
	if !ok || len(got) != 1 || got[0].RunId != "new" {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
}

func TestSQLite_concurrentPutCloseDrains(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	c, err := NewSQLite(path, Options{Retention: 24 * time.Hour, WriteBuffer: 256})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	now := time.Now().UTC()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.PutDAGRuns("etl", []models.DAGRun{{DagId: "etl", RunId: string(rune('a' + i)), State: "success", RunAfter: now.Add(time.Duration(i) * time.Second)}})
		}(i)
	}
	wg.Wait()
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := NewSQLite(path, Options{Retention: 24 * time.Hour, WriteBuffer: 16})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	got, ok := reopened.GetDAGRunsHistory("etl", now.Add(-time.Hour), 100)
	if !ok || len(got) != 20 {
		t.Fatalf("got len=%d ok=%v", len(got), ok)
	}
}
