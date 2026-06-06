package metrics

import (
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func run(dag, state string, runAfter time.Time) models.DAGRun {
	return models.DAGRun{DagId: dag, State: state, RunAfter: runAfter}
}

func TestRollupLatestState(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	runs := []models.DAGRun{
		run("a", "success", base),
		run("a", "failed", base.Add(2*time.Hour)), // newer → wins
		run("a", "success", base.Add(time.Hour)),
		run("b", "running", base),
		run("c", "queued", base),
	}

	got := RollupLatestState(runs)
	want := map[string]string{"a": "failed", "b": "running", "c": "queued"}
	if len(got) != len(want) {
		t.Fatalf("rollup size = %d, want %d (%v)", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("rollup[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestRollupLatestStateOrderIndependent(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	forward := []models.DAGRun{
		run("a", "success", base),
		run("a", "failed", base.Add(time.Hour)),
	}
	reversed := []models.DAGRun{forward[1], forward[0]}
	if RollupLatestState(forward)["a"] != RollupLatestState(reversed)["a"] {
		t.Fatalf("rollup depends on input order")
	}
	if RollupLatestState(reversed)["a"] != "failed" {
		t.Fatalf("latest run not selected, got %q", RollupLatestState(reversed)["a"])
	}
}

func TestRollupFallsBackToLogicalDate(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	runs := []models.DAGRun{
		{DagId: "a", State: "success", LogicalDate: base},
		{DagId: "a", State: "failed", LogicalDate: base.Add(time.Hour)},
	}
	if got := RollupLatestState(runs)["a"]; got != "failed" {
		t.Fatalf("logical_date fallback failed, got %q", got)
	}
}

func TestCountByState(t *testing.T) {
	rollup := map[string]string{
		"a": "running", "b": "success", "c": "success",
		"d": "failed", "e": "queued", // queued ignored
	}
	running, success, failed := CountByState(rollup)
	if running != 1 || success != 2 || failed != 1 {
		t.Fatalf("CountByState = (%d,%d,%d), want (1,2,1)", running, success, failed)
	}
}

func TestCountWindowStates(t *testing.T) {
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	since := now.Add(-7 * 24 * time.Hour)
	runs := []models.DAGRun{
		run("a", "success", now.Add(-time.Hour)),         // in window
		run("a", "failed", now.Add(-2*time.Hour)),        // in window
		run("a", "success", now.Add(-30*24*time.Hour)),   // outside window → excluded
		run("a", "running", now.Add(-40*24*time.Hour)),   // running counts regardless of window
	}
	running, success, failed := CountWindowStates(runs, since)
	if running != 1 || success != 1 || failed != 1 {
		t.Fatalf("CountWindowStates = (%d,%d,%d), want (1,1,1)", running, success, failed)
	}
}

func TestCountWindowStatesNoWindow(t *testing.T) {
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	runs := []models.DAGRun{
		run("a", "success", now.Add(-1000*24*time.Hour)),
		run("a", "failed", now),
	}
	_, success, failed := CountWindowStates(runs, time.Time{})
	if success != 1 || failed != 1 {
		t.Fatalf("no-window count = (success %d, failed %d), want (1,1)", success, failed)
	}
}
