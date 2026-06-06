package main

import (
	"testing"
	"time"

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

func TestWindowLabel(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{168 * time.Hour, "7d"},
		{336 * time.Hour, "14d"},
		{36 * time.Hour, "36h0m0s"},
	}
	for _, c := range cases {
		if got := windowLabel(c.in); got != c.want {
			t.Errorf("windowLabel(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWithLastRunState(t *testing.T) {
	dags := []models.DAG{{DagId: "a"}, {DagId: "b"}, {DagId: "c"}}
	rollup := map[string]string{"a": "failed", "b": "success"}
	got := withLastRunState(dags, rollup)
	if got[0].LastRunState != "failed" || got[1].LastRunState != "success" || got[2].LastRunState != "" {
		t.Fatalf("withLastRunState states = %q/%q/%q, want failed/success/empty",
			got[0].LastRunState, got[1].LastRunState, got[2].LastRunState)
	}
}
