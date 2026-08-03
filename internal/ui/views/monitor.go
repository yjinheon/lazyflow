package views

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/metrics"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

var monitorWindows = []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 30 * 24 * time.Hour}

const (
	sparklineRuns       = 20
	monitorWideWidth    = 110
	monitorWideHeight   = 20
	monitorMediumWidth  = 80
	monitorMediumHeight = 16
	monitorRecentLimit  = 20
)

type monitorLayoutMode int

const (
	monitorCompact monitorLayoutMode = iota
	monitorMedium
	monitorWide
)

type MonitorView struct {
	*tview.Flex

	mu        sync.Mutex
	windowIdx int

	header        *tview.TextView
	kpis          *tview.Flex
	kpiCards      []*tview.TextView
	body          *tview.Flex
	chart         *tview.TextView
	reliability   *tview.TextView
	recent        *tview.TextView
	mediumSummary *tview.TextView
	compact       *tview.TextView
	empty         *tview.TextView

	dagID        string
	runs         []models.DAGRun
	tasks        []models.TaskInstance
	state        string
	errorMessage string
	layoutMode   monitorLayoutMode
}

func NewMonitorView() *MonitorView {
	v := &MonitorView{
		Flex:          tview.NewFlex().SetDirection(tview.FlexRow),
		header:        monitorText(),
		kpis:          tview.NewFlex().SetDirection(tview.FlexColumn),
		body:          tview.NewFlex().SetDirection(tview.FlexColumn),
		chart:         monitorPanel(" Run duration & outcome "),
		reliability:   monitorPanel(" Reliability "),
		recent:        monitorPanel(" Recent runs "),
		mediumSummary: monitorText(),
		compact:       monitorText(),
		empty:         monitorText().SetTextAlign(tview.AlignCenter),
	}
	v.SetBorder(true).SetTitle(" Monitor ")
	for _, title := range []string{" Success rate ", " Failed ", " P90 duration ", " Queue avg "} {
		card := monitorPanel(title).SetTextAlign(tview.AlignCenter)
		v.kpiCards = append(v.kpiCards, card)
		v.kpis.AddItem(card, 0, 1, false)
	}
	v.body.AddItem(v.chart, 0, 7, false).AddItem(v.reliability, 0, 3, false)
	v.Update("", nil, nil)
	return v
}

func monitorText() *tview.TextView {
	return tview.NewTextView().SetDynamicColors(true).SetWrap(false)
}

func monitorPanel(title string) *tview.TextView {
	v := monitorText()
	v.SetBorder(true).SetTitle(title).SetTitleColor(theme.ActiveTheme().SectionHeader)
	v.SetBorderColor(theme.ActiveTheme().BorderColor)
	return v
}

func (v *MonitorView) Window() time.Duration {
	v.mu.Lock()
	defer v.mu.Unlock()
	return monitorWindows[v.windowIdx]
}

func (v *MonitorView) CycleWindow(delta int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	n := len(monitorWindows)
	v.windowIdx = ((v.windowIdx+delta)%n + n) % n
}

func (v *MonitorView) Update(dagID string, runs []models.DAGRun, tasks []models.TaskInstance) {
	v.dagID = dagID
	v.runs = append(v.runs[:0], runs...)
	v.tasks = append(v.tasks[:0], tasks...)
	v.state = ""
	v.errorMessage = ""
	v.renderSnapshot(80, 10)
	v.rebuildLayout()
}

// SetLoading makes a cache/API refresh visible. When a snapshot for the same
// DAG already exists it remains on screen and is marked as updating.
func (v *MonitorView) SetLoading(dagID string) {
	if dagID != v.dagID {
		v.runs = v.runs[:0]
		v.tasks = v.tasks[:0]
	}
	v.dagID = dagID
	v.state = "loading"
	v.errorMessage = ""
	v.renderSnapshot(80, 10)
	v.rebuildLayout()
}

// SetError preserves the last good snapshot and marks it stale. If no snapshot
// exists, the error is shown as an explicit empty state.
func (v *MonitorView) SetError(dagID, message string) {
	if dagID != v.dagID {
		v.runs = v.runs[:0]
		v.tasks = v.tasks[:0]
	}
	v.dagID = dagID
	v.state = "error"
	v.errorMessage = tview.Escape(message)
	v.renderSnapshot(80, 10)
	v.rebuildLayout()
}

// Draw switches layouts using this primitive's actual rectangle, not the
// application's global terminal size.
func (v *MonitorView) Draw(screen tcell.Screen) {
	_, _, width, height := v.GetRect()
	mode := monitorCompact
	if width >= monitorWideWidth && height >= monitorWideHeight {
		mode = monitorWide
	} else if width >= monitorMediumWidth && height >= monitorMediumHeight {
		mode = monitorMedium
	}
	v.renderSnapshot(width, height)
	if mode != v.layoutMode || v.GetItemCount() == 0 {
		v.layoutMode = mode
		v.renderSnapshot(width, height)
		v.rebuildLayout()
	}
	v.Flex.Draw(screen)
}

func (v *MonitorView) rebuildLayout() {
	v.Clear().SetDirection(tview.FlexRow)
	if v.dagID == "" || (v.state != "" && len(v.runs) == 0) {
		v.AddItem(v.empty, 0, 1, false)
		return
	}
	if v.layoutMode == monitorCompact {
		v.AddItem(v.compact, 0, 1, false)
		return
	}
	if v.layoutMode == monitorMedium {
		v.body.Clear().SetDirection(tview.FlexColumn).AddItem(v.chart, 0, 1, false)
		v.AddItem(v.header, 2, 0, false).
			AddItem(v.kpis, 4, 0, false).
			AddItem(v.body, 0, 1, false).
			AddItem(v.mediumSummary, 2, 0, false)
		return
	}
	v.body.Clear().SetDirection(tview.FlexColumn).
		AddItem(v.chart, 0, 7, false).
		AddItem(v.reliability, 0, 3, false)
	v.AddItem(v.header, 2, 0, false).
		AddItem(v.kpis, 5, 0, false).
		AddItem(v.body, 0, 1, false).
		AddItem(v.recent, 5, 0, false)
}

func (v *MonitorView) renderSnapshot(width, height int) {
	if v.dagID == "" {
		v.empty.SetText("\n[gray]Select a DAG from the list above to view run health.[-]\n[yellow]No DAG selected[-]")
		v.compact.SetText("[gray]Select a DAG from the list")
		return
	}
	if v.state == "loading" && len(v.runs) == 0 {
		v.empty.SetText(fmt.Sprintf("\n[blue]Loading history…[-]\n[gray]%s · %s[-]", v.dagID, monitorWindowLabel(v.Window())))
		v.compact.SetText(fmt.Sprintf("[blue]Loading history…[-]  [gray]%s[-]", v.dagID))
		return
	}
	if v.state == "error" && len(v.runs) == 0 {
		v.empty.SetText(fmt.Sprintf("\n[red]History unavailable[-]\n[gray]%s[-]\n%s", v.dagID, v.errorMessage))
		v.compact.SetText(fmt.Sprintf("[red]History unavailable[-]  %s", v.errorMessage))
		return
	}

	success, failed := metrics.SuccessFailed(v.runs)
	terminal := success + failed
	streak := metrics.FailureStreak(v.runs)
	flaky := metrics.FlakyTasks(v.tasks)
	_, p90, _ := metrics.Percentiles(v.runs)
	queueAvg := metrics.AvgQueueTime(v.tasks)
	trend := metrics.Trend(v.runs)

	badge := ""
	if v.state == "loading" {
		badge = "    [blue]updating…[-]"
	} else if v.state == "error" {
		badge = fmt.Sprintf("    [red]stale · %s[-]", v.errorMessage)
	}
	v.header.SetText(fmt.Sprintf(" [white::b]%s[-::-]    [gray]window:[-] %s%s", v.dagID, v.windowSelector(), badge))
	v.renderKPIs(success, failed, terminal, p90, queueAvg)

	innerWidth := max(width-2, 1)
	chartWidth := innerWidth
	chartHeight := max(height-12, 3)
	if v.layoutMode == monitorWide {
		bodyHeight := max(height-14, 5)
		chartWidth = max(innerWidth*7/10, 20)
		chartHeight = bodyHeight - 2
	}
	v.chart.SetText(renderDurationColumns(v.runs, chartWidth-2, chartHeight))
	v.reliability.SetText(renderReliability(success, failed, terminal, streak, flaky, len(v.tasks), trend, max(innerWidth*3/10-4, 8)))
	recentLimit := min(monitorRecentLimit, max((innerWidth-2)/7, 1))
	v.recent.SetText(renderRecentRuns(v.runs, recentLimit))
	v.mediumSummary.SetText(renderMediumSummary(success, failed, terminal, streak, flaky, len(v.tasks), trend, v.runs))
	v.compact.SetText(renderCompactMonitor(v.dagID, v.windowSelector(), v.runs, v.tasks))
	if v.state == "error" {
		v.compact.SetText(fmt.Sprintf("[red]stale · %s[-]\n%s", v.errorMessage, v.compact.GetText(false)))
	}
}

func (v *MonitorView) renderKPIs(success, failed, terminal int, p90, queueAvg time.Duration) {
	rate := "—"
	failedValue := "—"
	if terminal > 0 {
		rate = fmt.Sprintf("%d%%", success*100/terminal)
		failedValue = fmt.Sprintf("%d", failed)
	}
	values := []string{rate, failedValue, observedDuration(p90), observedDuration(queueAvg)}
	for i, value := range values {
		v.kpiCards[i].SetText(fmt.Sprintf("\n[white::b]%s[-::-]", value))
	}
}

func (v *MonitorView) windowSelector() string {
	selected := v.Window()
	labels := make([]string, 0, len(monitorWindows))
	for _, window := range monitorWindows {
		label := monitorWindowLabel(window)
		if window == selected {
			label = fmt.Sprintf("[blue::b]%s[-::-]", label)
		} else {
			label = "[gray]" + label + "[-]"
		}
		labels = append(labels, label)
	}
	return strings.Join(labels, "  ")
}

func (v *MonitorView) GetText(stripAllTags bool) string { return v.compact.GetText(stripAllTags) }

func (v *MonitorView) Root() tview.Primitive { return v }

func monitorWindowLabel(d time.Duration) string {
	if d == 24*time.Hour {
		return "24h"
	}
	h := int(d.Hours())
	if h%24 == 0 {
		return fmt.Sprintf("%dd", h/24)
	}
	return fmt.Sprintf("%dh", h)
}

func observedDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	return formatDuration(d)
}

func renderCompactMonitor(dagID, window string, runs []models.DAGRun, tasks []models.TaskInstance) string {
	success, failed := metrics.SuccessFailed(runs)
	terminal := success + failed
	rate := "—"
	successValue, failedValue := "—", "—"
	streakValue, flakyValue := "—", "—"
	if terminal > 0 {
		rate = fmt.Sprintf("%d%%", success*100/terminal)
		successValue = fmt.Sprintf("%d", success)
		failedValue = fmt.Sprintf("%d", failed)
		streakValue = fmt.Sprintf("%d", metrics.FailureStreak(runs))
	}
	if len(tasks) > 0 {
		flakyValue = fmt.Sprintf("%d", metrics.FlakyTasks(tasks))
	}
	p50, p90, p99 := metrics.Percentiles(runs)
	return fmt.Sprintf(
		"[white::b]%s[-::-]   [gray]window:[-] %s\n"+
			"[yellow]Reliability[-]  runs %d | success %s (%s) | failed %s | streak %s | flaky %s\n"+
			"[yellow]Latency[-]      p50 %s | p90 %s | p99 %s | queue avg %s\n"+
			"[yellow]Recent[-]       %s   trend %s",
		dagID, window, len(runs), successValue, rate, failedValue, streakValue, flakyValue,
		observedDuration(p50), observedDuration(p90), observedDuration(p99), observedDuration(metrics.AvgQueueTime(tasks)),
		renderSparkline(runs), metrics.Trend(runs),
	)
}

func renderReliability(success, failed, terminal, streak, flaky, taskSamples int, trend metrics.TrendDirection, width int) string {
	if terminal == 0 {
		return "[gray]No terminal runs[-]\n\nsamples 0"
	}
	barWidth := max(width-12, 4)
	flakyValue := "—"
	if taskSamples > 0 {
		flakyValue = fmt.Sprintf("%d", flaky)
	}
	return fmt.Sprintf(
		"success %s %d\nfailed  %s %d\n\nstreak %d  flaky %s\ntrend %s\nsamples %d terminal runs",
		renderHorizontalBar(success, terminal, barWidth, "green"), success,
		renderHorizontalBar(failed, terminal, barWidth, "red"), failed,
		streak, flakyValue, trend, terminal,
	)
}

func renderMediumSummary(success, failed, terminal, streak, flaky, taskSamples int, trend metrics.TrendDirection, runs []models.DAGRun) string {
	rate, successValue, failedValue, streakValue, flakyValue := "—", "—", "—", "—", "—"
	if terminal > 0 {
		rate = fmt.Sprintf("%d%%", success*100/terminal)
		successValue = fmt.Sprintf("%d", success)
		failedValue = fmt.Sprintf("%d", failed)
		streakValue = fmt.Sprintf("%d", streak)
	}
	if taskSamples > 0 {
		flakyValue = fmt.Sprintf("%d", flaky)
	}
	return fmt.Sprintf(
		"[yellow]Reliability[-]  success %s (%s) | failed %s | streak %s | flaky %s | trend %s\n"+
			"[yellow]Recent[-]       %s",
		successValue, rate, failedValue, streakValue, flakyValue, trend, renderSparkline(runs),
	)
}

func renderHorizontalBar(value, total, width int, color string) string {
	if width <= 0 || total <= 0 {
		return ""
	}
	filled := value * width / total
	if value > 0 && filled == 0 {
		filled = 1
	}
	filled = min(filled, width)
	return fmt.Sprintf("[%s]%s[-][gray]%s[-]", color, strings.Repeat("█", filled), strings.Repeat("░", width-filled))
}

func renderDurationColumns(runs []models.DAGRun, width, height int) string {
	if width < 8 || height < 3 {
		return "[gray]Chart unavailable[-]"
	}
	points := append([]models.DAGRun(nil), runs...)
	sort.Slice(points, func(i, j int) bool { return points[i].RunAfter.Before(points[j].RunAfter) })
	usable := points[:0]
	for _, run := range points {
		if run.Duration() > 0 || run.State == "queued" {
			usable = append(usable, run)
		}
	}
	if len(usable) == 0 {
		return "\n[gray]No terminal duration samples[-]\n[gray]Try a wider window with ][-]"
	}

	plotWidth := max(width-7, 1)
	maxPoints := max(plotWidth/2, 1)
	if len(usable) > maxPoints {
		usable = downsampleRuns(usable, maxPoints)
	}
	maxDuration := time.Duration(0)
	for _, run := range usable {
		maxDuration = max(maxDuration, run.Duration())
	}
	if maxDuration <= 0 {
		maxDuration = time.Second
	}
	plotHeight := max(height-2, 1)
	_, p90, _ := metrics.Percentiles(usable)
	p90Row := 0
	if p90 > 0 {
		p90Row = max(1, int(float64(p90)/float64(maxDuration)*float64(plotHeight)))
	}
	columns := make([]int, len(usable))
	for i, run := range usable {
		if d := run.Duration(); d > 0 {
			columns[i] = max(1, int(float64(d)/float64(maxDuration)*float64(plotHeight)))
		}
	}

	var b strings.Builder
	for row := plotHeight; row >= 1; row-- {
		if row == p90Row {
			b.WriteString("p90   ├")
		} else if row == plotHeight {
			fmt.Fprintf(&b, "%-5s │", formatDuration(maxDuration))
		} else {
			b.WriteString("      │")
		}
		for i, run := range usable {
			if columns[i] >= row {
				fmt.Fprintf(&b, "[%s]█[-] ", runColor(run.State))
			} else if run.State == "queued" && row == 1 {
				fmt.Fprintf(&b, "[%s]○[-] ", theme.MarkupHex(theme.ActiveTheme().StatusQueued))
			} else if row == p90Row {
				b.WriteString("┄ ")
			} else {
				b.WriteString("  ")
			}
		}
		b.WriteByte('\n')
	}
	b.WriteString("   0s └")
	b.WriteString(strings.Repeat("─", min(plotWidth, len(usable)*2)))
	b.WriteString("\n[gray]older")
	if plotWidth > 10 {
		b.WriteString(strings.Repeat(" ", plotWidth-10))
	}
	b.WriteString("now[-]")
	return b.String()
}

func downsampleRuns(runs []models.DAGRun, limit int) []models.DAGRun {
	if limit <= 0 || len(runs) == 0 {
		return nil
	}
	if len(runs) <= limit {
		return runs
	}
	if limit == 1 {
		return runs[len(runs)-1:]
	}
	sampled := make([]models.DAGRun, 0, limit)
	for i := range limit {
		index := i * (len(runs) - 1) / (limit - 1)
		sampled = append(sampled, runs[index])
	}
	return sampled
}

func runColor(state string) string {
	switch state {
	case "success":
		return "green"
	case "failed":
		return "red"
	case "running":
		return "blue"
	case "queued":
		return theme.MarkupHex(theme.ActiveTheme().StatusQueued)
	default:
		return "gray"
	}
}

func renderRecentRuns(runs []models.DAGRun, limit int) string {
	if len(runs) == 0 {
		return "[gray]No runs in this window · press ] to expand[-]"
	}
	ordered := append([]models.DAGRun(nil), runs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].RunAfter.Before(ordered[j].RunAfter) })
	if len(ordered) > limit {
		ordered = ordered[len(ordered)-limit:]
	}
	var times, glyphs strings.Builder
	for i, run := range ordered {
		label := run.RunAfter.Format("15:04")
		if run.RunAfter.IsZero() {
			label = "  —  "
		}
		fmt.Fprintf(&times, "%-6s ", label)
		glyph, color := "○", "gray"
		switch run.State {
		case "success":
			glyph, color = "✓", "green"
		case "failed":
			glyph, color = "✗", "red"
		case "running":
			glyph, color = "●", "blue"
		case "queued":
			glyph, color = "◌", theme.MarkupHex(theme.ActiveTheme().StatusQueued)
		}
		style := color
		if i == len(ordered)-1 {
			style += "::br"
		}
		fmt.Fprintf(&glyphs, "[%s]%-6s[-:-:-] ", style, glyph)
	}
	return "[gray]" + strings.TrimSpace(times.String()) + "[-]\n " + strings.TrimSpace(glyphs.String())
}

// renderSparkline draws at most 20 runs oldest-to-newest for compact mode.
func renderSparkline(runs []models.DAGRun) string {
	if len(runs) == 0 {
		return "[gray](no runs)[-]"
	}
	ordered := append([]models.DAGRun(nil), runs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].RunAfter.Before(ordered[j].RunAfter) })
	if len(ordered) > sparklineRuns {
		ordered = ordered[len(ordered)-sparklineRuns:]
	}
	var b strings.Builder
	for _, run := range ordered {
		glyph, color := "○", "gray"
		switch run.State {
		case "success":
			glyph, color = "✓", "green"
		case "failed":
			glyph, color = "✗", "red"
		}
		fmt.Fprintf(&b, "[%s]%s[-] ", color, glyph)
	}
	return strings.TrimSpace(b.String())
}
