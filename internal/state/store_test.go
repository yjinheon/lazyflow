package state

import (
	"sync/atomic"
	"testing"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestSetDAGRuns_notifies(t *testing.T) {
	s := NewStore()
	var got atomic.Int32
	s.Subscribe(EventDAGRunsUpdated, func(_ any) { got.Add(1) })
	s.SetDAGRuns("etl", []models.DAGRun{{}})
	if got.Load() != 1 {
		t.Fatalf("expected 1 notify, got %d", got.Load())
	}
}

func TestSetTaskInstances_notifies(t *testing.T) {
	s := NewStore()
	var got atomic.Int32
	s.Subscribe(EventTaskInstancesUpdated, func(_ any) { got.Add(1) })
	s.SetTaskInstances("etl", "run1", []models.TaskInstance{{}})
	if got.Load() != 1 {
		t.Fatalf("expected 1 notify")
	}
}

func TestSetTasks_notifies(t *testing.T) {
	s := NewStore()
	var got atomic.Int32
	s.Subscribe(EventLineageUpdated, func(_ any) { got.Add(1) })
	s.SetTasks("etl", []models.Task{{TaskId: "a"}})
	if got.Load() != 1 {
		t.Fatalf("expected 1 notify")
	}
	if tasks := s.GetTasks("etl"); len(tasks) != 1 {
		t.Fatalf("GetTasks=%+v", tasks)
	}
}

func TestBackfills_setGetSelect(t *testing.T) {
	s := NewStore()
	var bfEvt, selEvt atomic.Int32
	s.Subscribe(EventBackfillsUpdated, func(_ any) { bfEvt.Add(1) })
	s.Subscribe(EventBackfillSelected, func(_ any) { selEvt.Add(1) })

	s.SetBackfills("etl", []models.Backfill{{ID: 1}, {ID: 2}})
	if got := s.GetBackfills("etl"); len(got) != 2 {
		t.Fatalf("got=%+v", got)
	}
	if bfEvt.Load() != 1 {
		t.Fatalf("backfills notify=%d", bfEvt.Load())
	}

	s.SelectBackfill(2)
	if s.SelectedBackfill() != 2 {
		t.Fatal("select did not stick")
	}
	if selEvt.Load() != 1 {
		t.Fatal("select notify missing")
	}
}

func TestGanttMode_toggle(t *testing.T) {
	s := NewStore()
	var got atomic.Int32
	s.Subscribe(EventGanttModeChanged, func(_ any) { got.Add(1) })
	if s.GanttMode() {
		t.Fatal("default should be false")
	}
	s.SetGanttMode(true)
	if !s.GanttMode() || got.Load() != 1 {
		t.Fatal("toggle failed")
	}
	// Setting same value should NOT notify again (skip-on-no-change).
	s.SetGanttMode(true)
	if got.Load() != 1 {
		t.Fatalf("expected no extra notify on no-change, got %d", got.Load())
	}
}

func TestCriticalPath_setAndQuery(t *testing.T) {
	s := NewStore()
	var got atomic.Int32
	s.Subscribe(EventCriticalPathChanged, func(_ any) { got.Add(1) })
	s.SetCriticalPath(map[string]bool{"a": true, "b": true})
	if !s.IsOnCriticalPath("a") || s.IsOnCriticalPath("c") {
		t.Fatalf("critical path lookup wrong")
	}
	if got.Load() != 1 {
		t.Fatalf("expected 1 notify")
	}
	// Same content → no notify.
	s.SetCriticalPath(map[string]bool{"a": true, "b": true})
	if got.Load() != 1 {
		t.Fatalf("expected no notify on equal map, got %d", got.Load())
	}
	cp := s.GetCriticalPath()
	if len(cp) != 2 {
		t.Fatalf("GetCriticalPath=%+v", cp)
	}
}
