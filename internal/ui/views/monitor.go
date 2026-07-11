package views

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/metrics"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// monitorWindows are the selectable lookback windows, cycled by '[' and ']'.
var monitorWindows = []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 30 * 24 * time.Hour}

const sparklineRuns = 20

// MonitorView renders per-DAG operational statistics for the currently selected
// DAG over a lookback window. It is a passive view: callers hand it runs/tasks
// and it derives all stats via internal/metrics.
type MonitorView struct {
	*tview.TextView
	mu        sync.Mutex // guards windowIdx (read off-main via refreshMonitor, written on main via CycleWindow)
	windowIdx int
}

func NewMonitorView() *MonitorView {
	v := &MonitorView{TextView: tview.NewTextView()}
	v.SetBorder(true).SetTitle(" Monitor ")
	v.SetDynamicColors(true).SetScrollable(true)
	v.SetText("[gray]상단 DagList에서 DAG를 선택하세요")
	return v
}

// Window returns the currently selected lookback duration. Safe to call from
// any goroutine.
func (v *MonitorView) Window() time.Duration {
	v.mu.Lock()
	defer v.mu.Unlock()
	return monitorWindows[v.windowIdx]
}

// CycleWindow advances the window selection by delta, wrapping around. Safe to
// call from any goroutine.
func (v *MonitorView) CycleWindow(delta int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	n := len(monitorWindows)
	v.windowIdx = ((v.windowIdx+delta)%n + n) % n
}

// Update renders reliability/latency/recent summaries for one DAG. An empty
// dagId shows the "no selection" hint. runs/tasks are the window-scoped history.
func (v *MonitorView) Update(dagId string, runs []models.DAGRun, tasks []models.TaskInstance) {
	if dagId == "" {
		v.SetText("[gray]상단 DagList에서 DAG를 선택하세요")
		return
	}

	success, failed := metrics.SuccessFailed(runs)
	terminal := success + failed
	pct := 0
	if terminal > 0 {
		pct = success * 100 / terminal
	}
	streak := metrics.FailureStreak(runs)
	flaky := metrics.FlakyTasks(tasks)
	p50, p90, p99 := metrics.Percentiles(runs)
	queueAvg := metrics.AvgQueueTime(tasks)
	trend := metrics.Trend(runs)

	var b strings.Builder
	fmt.Fprintf(&b, "[white::b]%s[-::-]   [gray]window: %s[-]\n\n", dagId, monitorWindowLabel(v.Window()))
	fmt.Fprintf(&b, "[yellow]Reliability[-]   runs %d | success %d (%d%%) | failed %d | streak %d | flaky %d\n",
		len(runs), success, pct, failed, streak, flaky)
	fmt.Fprintf(&b, "[yellow]Latency[-]       p50 %s | p90 %s | p99 %s | queue avg %s\n",
		formatDuration(p50), formatDuration(p90), formatDuration(p99), formatDuration(queueAvg))
	fmt.Fprintf(&b, "[yellow]Recent[-]        %s   trend %s\n", renderSparkline(runs), trend)

	v.SetText(b.String())
}

func (v *MonitorView) Root() *tview.TextView { return v.TextView }

// monitorWindowLabel formats a window duration as "24h" / "7d" / "30d".
func monitorWindowLabel(d time.Duration) string {
	h := int(d.Hours())
	if h%24 == 0 {
		return fmt.Sprintf("%dd", h/24)
	}
	return fmt.Sprintf("%dh", h)
}

// renderSparkline draws up to the last sparklineRuns runs oldest→newest as
// colored glyphs: ✓ success, ✗ failed, ○ other (running/queued).
func renderSparkline(runs []models.DAGRun) string {
	n := len(runs)
	if n == 0 {
		return "[gray](no runs)[-]"
	}
	if n > sparklineRuns {
		runs = runs[:sparklineRuns] // newest-first: take most recent N
		n = sparklineRuns
	}
	var b strings.Builder
	// runs are newest-first; render oldest→newest for left-to-right time flow.
	for i := n - 1; i >= 0; i-- {
		switch runs[i].State {
		case "success":
			b.WriteString("[green]✓[-] ")
		case "failed":
			b.WriteString("[red]✗[-] ")
		default:
			b.WriteString("[gray]○[-] ")
		}
	}
	return strings.TrimRight(b.String(), " ")
}
