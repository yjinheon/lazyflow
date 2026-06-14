package views

import (
	"testing"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestSummarize(t *testing.T) {
	tis := []models.TaskInstance{
		{TaskId: "a", State: "success"},
		{TaskId: "b", State: "failed"},
		{TaskId: "c", State: "queued"},
		{TaskId: "d", State: "running"},
		{TaskId: "e", State: "skipped"},
	}
	s := summarize(tis)
	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.Done != 2 {
		t.Errorf("Done = %d, want 2", s.Done)
	}
	if s.Failed != 1 {
		t.Errorf("Failed = %d, want 1", s.Failed)
	}
	if s.Queued != 1 {
		t.Errorf("Queued = %d, want 1", s.Queued)
	}
	if s.Running != 1 {
		t.Errorf("Running = %d, want 1", s.Running)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := summarize(nil)
	if s.Total != 0 || s.Done != 0 {
		t.Errorf("empty summary not zeroed: %+v", s)
	}
}

// TestUpdateRunPreservesSelection guards the fix for the bug where a periodic
// "tasks" poll snapped the task-list selection back to the first row.
func TestUpdateRunPreservesSelection(t *testing.T) {
	v := NewExecutionView()
	run := models.DAGRun{RunId: "run-1"}
	tis := []models.TaskInstance{
		{TaskId: "a", State: "success"},
		{TaskId: "b", State: "running"},
		{TaskId: "c", State: "queued"},
	}

	// First load selects the first row.
	v.UpdateRun(run, tis, nil, nil)
	if r, _ := v.taskList.GetSelection(); r != 1 {
		t.Fatalf("initial selection row = %d, want 1", r)
	}

	// User moves to the third task.
	v.taskList.Select(3, 0)

	// A poll refresh of the same run must keep the user on task "c".
	v.UpdateRun(run, tis, nil, nil)
	if r, _ := v.taskList.GetSelection(); r != 3 {
		t.Errorf("selection after refresh = %d, want 3 (preserved)", r)
	}

	// Even if the task order changes, selection follows the TaskId.
	reordered := []models.TaskInstance{
		{TaskId: "c", State: "queued"},
		{TaskId: "a", State: "success"},
		{TaskId: "b", State: "running"},
	}
	v.UpdateRun(run, reordered, nil, nil)
	if r, _ := v.taskList.GetSelection(); r != 1 {
		t.Errorf("selection after reorder = %d, want 1 (follows TaskId 'c')", r)
	}

	// Switching to a different run resets to the first row.
	v.UpdateRun(models.DAGRun{RunId: "run-2"}, tis, nil, nil)
	if r, _ := v.taskList.GetSelection(); r != 1 {
		t.Errorf("selection after run change = %d, want 1 (reset)", r)
	}
}
