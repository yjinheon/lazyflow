package cache

import (
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestMemory_putGet(t *testing.T) {
	c := NewMemory(50 * time.Millisecond)
	bfs := []models.Backfill{{ID: 1}, {ID: 2}}
	c.PutBackfills("etl", bfs)
	got, ok := c.GetBackfills("etl")
	if !ok || len(got) != 2 || got[0].ID != 1 {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
}

func TestMemory_ttl(t *testing.T) {
	c := NewMemory(20 * time.Millisecond)
	c.PutBackfills("x", []models.Backfill{{ID: 9}})
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.GetBackfills("x"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestMemory_missForUnknownKey(t *testing.T) {
	c := NewMemory(time.Second)
	if _, ok := c.GetBackfills("nope"); ok {
		t.Fatal("expected miss")
	}
}

func TestMemory_close(t *testing.T) {
	c := NewMemory(time.Second)
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestMemory_dagRunsHistory(t *testing.T) {
	c := NewMemory(time.Second)
	now := time.Now().UTC()
	c.PutDAGRuns("etl", []models.DAGRun{
		{DagId: "etl", RunId: "old", State: "success", RunAfter: now.Add(-2 * time.Hour)},
		{DagId: "etl", RunId: "new", State: "failed", RunAfter: now},
	})

	got, ok := c.GetDAGRunsHistory("etl", now.Add(-time.Hour), 10)
	if !ok || len(got) != 1 || got[0].RunId != "new" {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}

	got[0].State = "mutated"
	again, _ := c.GetDAGRunsHistory("etl", now.Add(-time.Hour), 10)
	if again[0].State != "failed" {
		t.Fatalf("cache returned mutable DAG run state: %q", again[0].State)
	}
}

func TestMemory_taskInstancesHistoryAndDashboard(t *testing.T) {
	c := NewMemory(time.Second)
	now := time.Now().UTC()
	queued := now.Add(-2 * time.Minute)
	start := now.Add(-time.Minute)
	end := now
	c.PutDAGRuns("etl", []models.DAGRun{
		{DagId: "etl", RunId: "r1", State: "failed", RunAfter: now, StartDate: start, EndDate: end},
	})
	c.PutTaskInstances("etl", "r1", []models.TaskInstance{
		{DagId: "etl", RunId: "r1", TaskId: "extract", State: "failed", QueuedDttm: &queued, StartDate: &start, EndDate: &end, TryNumber: 2},
	})

	tasks, ok := c.GetTaskInstancesHistory("etl", now.Add(-time.Hour), 10)
	if !ok || len(tasks) != 1 || tasks[0].TaskId != "extract" {
		t.Fatalf("tasks=%+v ok=%v", tasks, ok)
	}
	rows, ok := c.GetDagDashboardRows(now.Add(-time.Hour), 10)
	if !ok || len(rows) != 1 {
		t.Fatalf("rows=%+v ok=%v", rows, ok)
	}
	if rows[0].Failed != 1 || rows[0].FailedTasks != 1 || rows[0].RetriedTasks != 1 || rows[0].AvgQueueTime != time.Minute {
		t.Fatalf("unexpected dashboard row: %+v", rows[0])
	}
}
