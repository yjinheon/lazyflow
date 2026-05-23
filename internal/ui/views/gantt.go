package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// GanttView renders a per-run Gantt chart inside a tview.TextView.
// The renderer functions are pure (gantt_renderer.go); this view owns
// only the markup assembly and the TextView container.
type GanttView struct {
	*tview.TextView
}

func NewGanttView() *GanttView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(false)
	tv.SetBorder(true).SetTitle(" Gantt ")
	return &GanttView{TextView: tv}
}

// Update redraws the Gantt for the given run.
// onCritical (task_id → bool) is used for critical-path bold highlighting;
// pass nil or empty map to disable highlighting.
func (v *GanttView) Update(runId string, tis []models.TaskInstance, onCritical map[string]bool) {
	if len(tis) == 0 {
		v.SetText("[gray]No tasks have started yet.")
		return
	}
	_, _, innerW, _ := v.GetInnerRect()
	const labelCol = 25
	barW := innerW - labelCol - 1
	if barW < 10 {
		v.SetText(fmt.Sprintf("[gray]Terminal too narrow (need >=%d cols).", labelCol+11))
		return
	}

	now := time.Now()
	buckets, tMin, tMax := ComputeBuckets(tis, barW, now)
	if buckets == nil {
		v.SetText("[gray]No tasks have started yet.")
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[gray]Gantt -- %s   %s -> %s   [g] back to table\n",
		runId, formatTick(tMin, tMax), formatTick(tMax, tMax))

	// Sort tasks alphabetically by task_id for deterministic order.
	sorted := append([]models.TaskInstance(nil), tis...)
	sortByTaskID(sorted)

	for _, ti := range sorted {
		cells := RenderCells(ti, buckets, now)
		bold := onCritical[ti.TaskId]
		label := truncate(ti.TaskId, labelCol-2)
		fmt.Fprintf(&b, "%-*s ", labelCol-1, label)
		b.WriteString(EmitRLE(cells, bold))
		b.WriteByte('\n')
	}
	b.WriteString(renderXAxis(tMin, tMax, barW, labelCol))
	v.SetText(b.String())
}

func (v *GanttView) Root() tview.Primitive { return v.TextView }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "..."
}

// Simple insertion sort by TaskId (no extra deps, deterministic).
func sortByTaskID(tis []models.TaskInstance) {
	for i := 1; i < len(tis); i++ {
		for j := i; j > 0 && tis[j-1].TaskId > tis[j].TaskId; j-- {
			tis[j-1], tis[j] = tis[j], tis[j-1]
		}
	}
}

func formatTick(t, tMax time.Time) string {
	if tMax.IsZero() || t.IsZero() {
		return "--"
	}
	if tMax.Sub(t) >= 24*time.Hour {
		return t.Format("01-02 15:04")
	}
	return t.Format("15:04:05")
}

// renderXAxis draws a best-effort tick label row under the bars.
// Phase 1: approximate alignment; precise pixel alignment is a polish concern.
func renderXAxis(tMin, tMax time.Time, barW, labelCol int) string {
	if barW < 10 {
		return ""
	}
	dur := tMax.Sub(tMin)
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", labelCol))
	ticks := 4
	hex := theme.GanttMarkupColor("running")
	for i := 0; i < ticks; i++ {
		t := tMin.Add(dur * time.Duration(i) / time.Duration(ticks-1))
		s := formatTick(t, tMax)
		pos := i * (barW - 1) / (ticks - 1)
		current := b.Len() - labelCol
		if pos > current {
			b.WriteString(strings.Repeat(" ", pos-current))
		}
		fmt.Fprintf(&b, "[%s]%s[-]", hex, s)
	}
	return b.String()
}
