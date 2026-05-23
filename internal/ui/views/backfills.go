package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// BackfillsView is the active page of the "backfills" tab. It contains a
// left-hand list (Table) and a right-hand detail (TextView).
type BackfillsView struct {
	*tview.Flex
	list       *tview.Table
	detail     *tview.TextView
	backfills  []models.Backfill
	onSelected func(id int)
}

func NewBackfillsView() *BackfillsView {
	v := &BackfillsView{
		Flex:   tview.NewFlex().SetDirection(tview.FlexColumn),
		list:   tview.NewTable(),
		detail: tview.NewTextView(),
	}
	v.list.SetBorders(false).SetSelectable(true, false).SetFixed(1, 0)
	v.list.SetBorder(true).SetTitle(" Backfills ")
	v.detail.SetBorder(true).SetTitle(" Detail ")
	v.detail.SetDynamicColors(true).SetScrollable(true)
	v.AddItem(v.list, 0, 1, true)
	v.AddItem(v.detail, 0, 1, false)
	v.list.SetSelectionChangedFunc(func(row, _ int) {
		idx := row - 1 // header offset
		if idx < 0 || idx >= len(v.backfills) {
			return
		}
		if v.onSelected != nil {
			v.onSelected(v.backfills[idx].ID)
		}
	})
	return v
}

// SetOnSelected registers a callback fired when the list cursor changes.
func (v *BackfillsView) SetOnSelected(fn func(id int)) { v.onSelected = fn }

// UpdateList re-renders the list table.
func (v *BackfillsView) UpdateList(bfs []models.Backfill) {
	v.backfills = bfs
	v.list.Clear()
	headers := []string{"ID", "DAG", "State", "Range", "Progress", "Created"}
	for c, h := range headers {
		v.list.SetCell(0, c, tview.NewTableCell("[::b]"+h).SetSelectable(false))
	}
	for i, bf := range bfs {
		row := i + 1
		stateColorHex := theme.GanttMarkupColor(stateTokenForBackfill(bf.State()))
		cells := []string{
			fmt.Sprintf("%d", bf.ID),
			truncate(bf.DagId, 12),
			fmt.Sprintf("[%s]%s[-]", stateColorHex, bf.State()),
			fmt.Sprintf("%s -> %s", bf.FromDate.Format("2006-01-02"), bf.ToDate.Format("2006-01-02")),
			fmt.Sprintf("%d/%d", bf.CompletedRuns, bf.TotalRuns),
			ago(bf.CreatedAt),
		}
		for c, s := range cells {
			v.list.SetCell(row, c, tview.NewTableCell(s))
		}
	}
}

// UpdateDetail re-renders the right-hand pane for the currently selected
// backfill. Pass nil to clear.
func (v *BackfillsView) UpdateDetail(bf *models.Backfill) {
	if bf == nil {
		v.detail.SetText("[gray]No backfill selected.")
		return
	}
	pct := 0
	if bf.TotalRuns > 0 {
		pct = bf.CompletedRuns * 100 / bf.TotalRuns
	}
	bar := renderProgressBar(bf, 20)

	var b strings.Builder
	fmt.Fprintf(&b, "[::b]Backfill #%d[::-]  [gray]%s[-]\n", bf.ID, bf.DagId)
	fmt.Fprintf(&b, "Range:           %s -> %s\n", bf.FromDate.Format("2006-01-02"), bf.ToDate.Format("2006-01-02"))
	fmt.Fprintf(&b, "MaxActiveRuns:   %d\n", bf.MaxActiveRuns)
	fmt.Fprintf(&b, "ReprocessBeh.:   %s\n", bf.ReprocessBehavior)
	fmt.Fprintf(&b, "State:           [%s]%s[-]\n", theme.GanttMarkupColor(stateTokenForBackfill(bf.State())), bf.State())
	fmt.Fprintf(&b, "\nProgress: %s %d%% (%d/%d)\n", bar, pct, bf.CompletedRuns, bf.TotalRuns)
	fmt.Fprintf(&b, "\n[gray]Actions: [c]ancel  [p]ause  [u]npause[-]\n")
	v.detail.SetText(b.String())
}

// List returns the inner table (so keybindings can focus it).
func (v *BackfillsView) List() *tview.Table { return v.list }

// Root for layout registration.
func (v *BackfillsView) Root() tview.Primitive { return v.Flex }

// ---------- helpers ----------

func stateTokenForBackfill(state string) string {
	switch state {
	case "running":
		return "running"
	case "paused":
		return "queued" // amber-ish from theme; visually distinct from green
	case "completed":
		return "success"
	default:
		return "skipped"
	}
}

func ago(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func renderProgressBar(bf *models.Backfill, width int) string {
	if bf.TotalRuns == 0 {
		return strings.Repeat(string([]rune{0x2591}), width)
	}
	filled := bf.CompletedRuns * width / bf.TotalRuns
	if filled > width {
		filled = width
	}
	var b strings.Builder
	b.WriteString("[")
	b.WriteString(theme.GanttMarkupColor("success"))
	b.WriteString("]")
	b.WriteString(strings.Repeat(string([]rune{0x2588}), filled))
	b.WriteString("[-]")
	b.WriteString(strings.Repeat(string([]rune{0x2591}), width-filled))
	return b.String()
}
