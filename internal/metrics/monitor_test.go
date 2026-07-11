package metrics

import (
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func mkRun(state string, ago time.Duration) models.DAGRun {
	t := time.Now().Add(-ago)
	return models.DAGRun{State: state, RunAfter: t}
}

func TestSuccessFailed_ignoresNonTerminal(t *testing.T) {
	runs := []models.DAGRun{
		mkRun("success", 1*time.Hour), mkRun("failed", 2*time.Hour),
		mkRun("running", 0), mkRun("queued", 0), mkRun("success", 3*time.Hour),
	}
	s, f := SuccessFailed(runs)
	if s != 2 || f != 1 {
		t.Fatalf("got success=%d failed=%d, want 2/1", s, f)
	}
}

func TestFailureStreak_countsFromNewestSkippingRunning(t *testing.T) {
	runs := []models.DAGRun{
		mkRun("running", 0),
		mkRun("failed", 1*time.Hour),
		mkRun("failed", 2*time.Hour),
		mkRun("success", 3*time.Hour),
		mkRun("failed", 4*time.Hour),
	}
	if got := FailureStreak(runs); got != 2 {
		t.Fatalf("got streak=%d, want 2", got)
	}
}

func TestFlakyTasks_countsRetriedThenSucceeded(t *testing.T) {
	tasks := []models.TaskInstance{
		{State: "success", TryNumber: 2},
		{State: "success", TryNumber: 1},
		{State: "failed", TryNumber: 3},
	}
	if got := FlakyTasks(tasks); got != 1 {
		t.Fatalf("got flaky=%d, want 1", got)
	}
}

func runDur(state string, ago, dur time.Duration) models.DAGRun {
	end := time.Now().Add(-ago)
	return models.DAGRun{State: state, RunAfter: end, StartDate: end.Add(-dur), EndDate: end}
}

func TestPercentiles_terminalOnlyNearestRank(t *testing.T) {
	var runs []models.DAGRun
	for i := 1; i <= 10; i++ {
		runs = append(runs, runDur("success", time.Duration(i)*time.Hour, time.Duration(i)*time.Minute))
	}
	runs = append(runs, models.DAGRun{State: "running", RunAfter: time.Now(), StartDate: time.Now().Add(-99 * time.Hour)})
	p50, p90, p99 := Percentiles(runs)
	if p50 != 5*time.Minute || p90 != 9*time.Minute || p99 != 10*time.Minute {
		t.Fatalf("got p50=%v p90=%v p99=%v", p50, p90, p99)
	}
}

func TestPercentiles_emptyIsZero(t *testing.T) {
	p50, p90, p99 := Percentiles(nil)
	if p50 != 0 || p90 != 0 || p99 != 0 {
		t.Fatalf("want all zero, got %v/%v/%v", p50, p90, p99)
	}
}

func TestAvgQueueTime(t *testing.T) {
	q := time.Now().Add(-5 * time.Minute)
	s := q.Add(1 * time.Minute)
	tasks := []models.TaskInstance{
		{QueuedDttm: &q, StartDate: &s},
		{QueuedDttm: nil, StartDate: &s},
	}
	if got := AvgQueueTime(tasks); got != time.Minute {
		t.Fatalf("got %v, want 1m", got)
	}
}

func TestTrend(t *testing.T) {
	var runs []models.DAGRun
	for i := 0; i < 4; i++ {
		runs = append(runs, mkRun("failed", time.Duration(20+i)*time.Hour))
	}
	for i := 0; i < 4; i++ {
		runs = append(runs, mkRun("success", time.Duration(1+i)*time.Hour))
	}
	if got := Trend(runs); got != TrendImproving {
		t.Fatalf("got %v, want improving", got)
	}
	if got := Trend(runs[:3]); got != TrendNA {
		t.Fatalf("small sample: got %v, want n/a", got)
	}
}
