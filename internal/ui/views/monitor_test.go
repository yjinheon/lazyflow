package views

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func drawMonitor(t *testing.T, v *MonitorView, width, height int) string {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("init simulation screen: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(width, height)
	v.SetRect(0, 0, width, height)
	v.Draw(screen)
	screen.Show()

	cells, cellWidth, _ := screen.GetContents()
	var b strings.Builder
	for i, cell := range cells {
		if i > 0 && i%cellWidth == 0 {
			b.WriteByte('\n')
		}
		if len(cell.Runes) == 0 {
			b.WriteByte(' ')
		} else {
			b.WriteRune(cell.Runes[0])
		}
	}
	return b.String()
}

func TestMonitorView_wideDashboardShowsOperationalPanels(t *testing.T) {
	now := time.Now()
	runs := []models.DAGRun{
		{State: "success", RunAfter: now.Add(-time.Hour), StartDate: now.Add(-70 * time.Minute), EndDate: now.Add(-time.Hour)},
		{State: "failed", RunAfter: now.Add(-2 * time.Hour), StartDate: now.Add(-135 * time.Minute), EndDate: now.Add(-2 * time.Hour)},
	}
	queued, started := now.Add(-3*time.Minute), now.Add(-2*time.Minute)
	v := NewMonitorView()
	v.Update("etl_daily", runs, []models.TaskInstance{{QueuedDttm: &queued, StartDate: &started}})

	got := drawMonitor(t, v, 120, 28)
	for _, want := range []string{
		"etl_daily", "Success rate", "Failed", "P90 duration", "Queue avg",
		"Run duration & outcome", "Reliability", "Recent runs",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("wide dashboard missing %q:\n%s", want, got)
		}
	}
}

func TestRenderDurationColumnsShowsP90Guide(t *testing.T) {
	now := time.Now()
	runs := []models.DAGRun{
		{State: "success", RunAfter: now.Add(-2 * time.Hour), StartDate: now.Add(-2*time.Hour - 5*time.Minute), EndDate: now.Add(-2 * time.Hour)},
		{State: "failed", RunAfter: now.Add(-time.Hour), StartDate: now.Add(-time.Hour - 10*time.Minute), EndDate: now.Add(-time.Hour)},
	}

	got := renderDurationColumns(runs, 60, 10)
	if !strings.Contains(got, "p90") {
		t.Fatalf("duration chart missing p90 guide:\n%s", got)
	}
}

func TestMonitorView_loadingStateIsExplicit(t *testing.T) {
	v := NewMonitorView()
	v.SetLoading("etl_daily")

	got := drawMonitor(t, v, 100, 18)
	if !strings.Contains(got, "Loading history") {
		t.Fatalf("loading state is not visible:\n%s", got)
	}
}

func TestMonitorView_errorKeepsLastSnapshot(t *testing.T) {
	now := time.Now()
	runs := []models.DAGRun{{
		State: "success", RunAfter: now.Add(-time.Hour),
		StartDate: now.Add(-65 * time.Minute), EndDate: now.Add(-time.Hour),
	}}
	v := NewMonitorView()
	v.Update("etl_daily", runs, nil)
	v.SetError("etl_daily", "cache unavailable")

	got := drawMonitor(t, v, 120, 28)
	for _, want := range []string{"100%", "stale", "cache unavailable"} {
		if !strings.Contains(got, want) {
			t.Errorf("error dashboard missing %q:\n%s", want, got)
		}
	}
}

func TestMonitorView_compactLayoutFallsBackToSummary(t *testing.T) {
	now := time.Now()
	v := NewMonitorView()
	v.Update("etl_daily", []models.DAGRun{{
		State: "success", RunAfter: now.Add(-time.Hour),
		StartDate: now.Add(-65 * time.Minute), EndDate: now.Add(-time.Hour),
	}}, nil)

	got := drawMonitor(t, v, 79, 15)
	for _, want := range []string{"etl_daily", "Reliability", "Latency", "Recent", "24h"} {
		if !strings.Contains(got, want) {
			t.Errorf("compact dashboard missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Success rate") {
		t.Fatalf("compact dashboard unexpectedly rendered wide KPI cards:\n%s", got)
	}
}

func TestMonitorView_noSamplesAreNotRenderedAsZero(t *testing.T) {
	v := NewMonitorView()
	v.Update("etl_daily", nil, nil)

	got := drawMonitor(t, v, 120, 28)
	for _, want := range []string{"—", "No terminal duration samples", "No runs in this window"} {
		if !strings.Contains(got, want) {
			t.Errorf("empty dashboard missing %q:\n%s", want, got)
		}
	}
	compact := drawMonitor(t, v, 79, 15)
	if strings.Contains(compact, "success 0") || strings.Contains(compact, "failed 0") {
		t.Fatalf("compact empty state rendered missing observations as zero:\n%s", compact)
	}
}

func TestMonitorView_noSelection(t *testing.T) {
	v := NewMonitorView()
	v.Update("", nil, nil)
	if !strings.Contains(v.GetText(true), "Select a DAG") {
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
