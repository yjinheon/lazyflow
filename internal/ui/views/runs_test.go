package views

import (
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestRunsViewStateFilter(t *testing.T) {
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	since := now.Add(-7 * 24 * time.Hour)
	runs := []models.DAGRun{
		{RunId: "s1", State: "success", RunAfter: now.Add(-time.Hour)},
		{RunId: "s_old", State: "success", RunAfter: now.Add(-30 * 24 * time.Hour)}, // outside window
		{RunId: "f1", State: "failed", RunAfter: now.Add(-2 * time.Hour)},
		{RunId: "r_old", State: "running", RunAfter: now.Add(-40 * 24 * time.Hour)}, // running ignores window
	}

	v := NewRunsView()
	v.Update(runs)
	if len(v.runs) != 4 {
		t.Fatalf("no filter: displayed = %d, want 4", len(v.runs))
	}

	v.SetStateFilter("success", since)
	if len(v.runs) != 1 || v.runs[0].RunId != "s1" {
		t.Fatalf("success+window filter = %v, want [s1]", runIDs(v.runs))
	}

	v.SetStateFilter("running", since)
	if len(v.runs) != 1 || v.runs[0].RunId != "r_old" {
		t.Fatalf("running filter should ignore window, got %v", runIDs(v.runs))
	}

	v.ClearFilter()
	if len(v.runs) != 4 {
		t.Fatalf("clear filter: displayed = %d, want 4", len(v.runs))
	}
}

func runIDs(runs []models.DAGRun) []string {
	ids := make([]string, len(runs))
	for i, r := range runs {
		ids[i] = r.RunId
	}
	return ids
}
