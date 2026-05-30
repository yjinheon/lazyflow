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
