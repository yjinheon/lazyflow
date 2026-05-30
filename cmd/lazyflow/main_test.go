package main

import (
	"testing"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestCountDAGActivity(t *testing.T) {
	dags := []models.DAG{
		{DagId: "active_a"},
		{DagId: "paused_a", IsPaused: true},
		{DagId: "active_b"},
	}

	active, inactive := countDAGActivity(dags)
	if active != 2 || inactive != 1 {
		t.Fatalf("countDAGActivity() active=%d inactive=%d, want active=2 inactive=1", active, inactive)
	}
}

func TestCountRunStates(t *testing.T) {
	runs := []models.DAGRun{
		{State: "running"},
		{State: "success"},
		{State: "failed"},
		{State: "queued"},
		{State: "success"},
	}

	running, success, failed := countRunStates(runs)
	if running != 1 || success != 2 || failed != 1 {
		t.Fatalf("countRunStates() running=%d success=%d failed=%d, want running=1 success=2 failed=1", running, success, failed)
	}
}
