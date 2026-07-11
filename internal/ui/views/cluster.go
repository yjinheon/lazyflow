package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type poolViewMode int

const (
	poolCompact poolViewMode = iota
	poolTable
)

const poolBarWidth = 8

type ClusterInfoView struct {
	*tview.TextView
	health   *models.HealthInfo
	pools    []models.Pool
	poolView poolViewMode
}

func NewClusterInfoView() *ClusterInfoView {
	v := &ClusterInfoView{
		TextView: tview.NewTextView(),
		poolView: poolCompact,
	}
	v.SetBorder(true).SetTitle(" Cluster ")
	v.SetDynamicColors(true)
	v.SetScrollable(true)
	v.SetText("[gray]Waiting for health check...")
	return v
}

// Update caches the latest health and re-renders.
func (v *ClusterInfoView) Update(health *models.HealthInfo) {
	v.health = health
	v.render()
}

// UpdatePools caches the latest pools and re-renders.
func (v *ClusterInfoView) UpdatePools(pools []models.Pool) {
	v.pools = pools
	v.render()
}

// ToggleView flips compact <-> table and re-renders from cache (no refetch).
func (v *ClusterInfoView) ToggleView() {
	if v.poolView == poolCompact {
		v.poolView = poolTable
	} else {
		v.poolView = poolCompact
	}
	v.render()
}

func (v *ClusterInfoView) render() {
	var b strings.Builder
	b.WriteString(v.renderHealth())
	if len(v.pools) > 0 {
		b.WriteString("\n[gray]─── Pools ───[-]\n")
		if v.poolView == poolTable {
			b.WriteString(renderPoolTable(v.pools))
		} else {
			b.WriteString(renderPoolCompact(v.pools))
		}
	}
	v.SetText(b.String())
}

func (v *ClusterInfoView) renderHealth() string {
	if v.health == nil {
		return "[red]Health check failed"
	}
	return fmt.Sprintf(
		"[yellow]Scheduler:[-]    %s\n"+
			"[yellow]Metadatabase:[-] %s\n"+
			"[yellow]Triggerer:[-]    %s\n"+
			"[yellow]DAG Proc:[-]     %s",
		formatHealthLag(v.health.Scheduler, heartbeatOf(v.health.Scheduler, "scheduler")),
		formatHealth(v.health.Metadatabase),
		formatHealthLag(v.health.Triggerer, heartbeatOf(v.health.Triggerer, "triggerer")),
		formatHealthLag(v.health.DagProcessor, heartbeatOf(v.health.DagProcessor, "dagprocessor")),
	)
}

// heartbeatOf returns the component-specific heartbeat string from a status.
func heartbeatOf(s *models.HealthStatus, kind string) string {
	if s == nil {
		return ""
	}
	switch kind {
	case "scheduler":
		return s.LatestSchedulerHeartbeat
	case "triggerer":
		return s.LatestTriggererHeartbeat
	case "dagprocessor":
		return s.LatestDagProcessorHeartbeat
	}
	return ""
}

// formatHealthLag renders the base health status plus a color-coded heartbeat
// lag age. Thresholds: <30s green, <120s yellow, else red. When the heartbeat
// is missing or unparseable, only the base status is shown.
func formatHealthLag(s *models.HealthStatus, heartbeat string) string {
	base := formatHealth(s)
	lag, ok := heartbeatLag(heartbeat, time.Now())
	if !ok {
		return base
	}
	color := "green"
	switch {
	case lag >= 120*time.Second:
		color = "red"
	case lag >= 30*time.Second:
		color = "yellow"
	}
	return fmt.Sprintf("%s [%s](%s)[-]", base, color, formatDuration(lag))
}

// heartbeatLag parses an RFC3339 heartbeat timestamp and returns now-hb,
// clamped to >= 0. ok is false when the string is empty or unparseable.
func heartbeatLag(hb string, now time.Time) (time.Duration, bool) {
	if hb == "" {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339, hb)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, hb)
		if err != nil {
			return 0, false
		}
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	return d, true
}

func formatHealth(s *models.HealthStatus) string {
	if s == nil {
		return "[gray]N/A[-]"
	}
	if s.Status == "healthy" {
		return "[green]healthy[-]"
	}
	return fmt.Sprintf("[red]%s[-]", s.Status)
}

func renderPoolCompact(pools []models.Pool) string {
	const nameW = 8
	var b strings.Builder
	for _, p := range pools {
		bar := renderPoolBar(p.OccupiedSlots, p.Slots, poolBarWidth)
		b.WriteString(fmt.Sprintf("%-*s %s %d/%d", nameW, truncateName(p.Name, nameW), bar, p.OccupiedSlots, p.Slots))
		if p.QueuedSlots > 0 {
			b.WriteString(fmt.Sprintf(" [red]⚠%dq[-]", p.QueuedSlots))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderPoolTable(pools []models.Pool) string {
	var b strings.Builder
	b.WriteString("[gray]NAME      USED  Q[-]\n")
	for _, p := range pools {
		used := fmt.Sprintf("%d/%d", p.OccupiedSlots, p.Slots)
		b.WriteString(fmt.Sprintf("%-9s %-5s %d\n", truncateName(p.Name, 9), used, p.QueuedSlots))
	}
	return b.String()
}

func truncateName(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// renderPoolBar draws a fixed-width utilization bar like [████░░] wrapped in a
// tview color tag. Color: green <70%, yellow <90%, red >=90% (or saturated).
// The leading "[[" is tview's escape for a literal "[".
func renderPoolBar(occupied, slots, width int) string {
	if width <= 0 {
		return "[[" + "[-]]"
	}
	var ratio float64
	if slots > 0 {
		ratio = float64(occupied) / float64(slots)
	}
	ratio = min(ratio, 1)
	filled := min(int(ratio*float64(width)+0.5), width)
	color := "green"
	switch {
	case ratio >= 0.9:
		color = "red"
	case ratio >= 0.7:
		color = "yellow"
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	// "[[" -> literal "[", then [color]...[-], then literal "]".
	return fmt.Sprintf("[[[%s]%s[-]]", color, bar)
}

func (v *ClusterInfoView) Root() *tview.TextView {
	return v.TextView
}
