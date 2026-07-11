package views

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestMonitorView_noSelection(t *testing.T) {
	v := NewMonitorView()
	v.Update("", nil, nil)
	if !strings.Contains(v.GetText(true), "DAG를 선택") {
		t.Fatalf("expected selection hint, got %q", v.GetText(true))
	}
}

func TestMonitorView_rendersSections(t *testing.T) {
	now := time.Now()
	runs := []models.DAGRun{
		{State: "success", RunAfter: now.Add(-1 * time.Hour), StartDate: now.Add(-1*time.Hour - time.Minute), EndDate: now.Add(-1 * time.Hour)},
		{State: "failed", RunAfter: now.Add(-2 * time.Hour), StartDate: now.Add(-2*time.Hour - time.Minute), EndDate: now.Add(-2 * time.Hour)},
	}
	v := NewMonitorView()
	v.Update("etl_daily", runs, nil)
	txt := v.GetText(true)
	for _, want := range []string{"etl_daily", "Reliability", "Latency", "Recent", "success 1"} {
		if !strings.Contains(txt, want) {
			t.Fatalf("missing %q in:\n%s", want, txt)
		}
	}
}

// TestMonitorView_windowRace exercises concurrent Window()/CycleWindow() the way
// refreshMonitor (poller goroutine) and the '['/']' keybinding (main goroutine)
// do. Run with -race to catch unsynchronized windowIdx access.
func TestMonitorView_windowRace(t *testing.T) {
	v := NewMonitorView()
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				v.CycleWindow(1)
				_ = v.Window()
			}
		}()
	}
	wg.Wait()
}

func TestMonitorView_cycleWindow(t *testing.T) {
	v := NewMonitorView()
	if v.Window() != 24*time.Hour {
		t.Fatalf("default window = %v, want 24h", v.Window())
	}
	v.CycleWindow(1)
	if v.Window() != 7*24*time.Hour {
		t.Fatalf("after cycle = %v, want 7d", v.Window())
	}
	v.CycleWindow(-1)
	if v.Window() != 24*time.Hour {
		t.Fatalf("wrap back = %v, want 24h", v.Window())
	}
}
