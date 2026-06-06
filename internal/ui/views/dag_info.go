package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// filterPanelHeight is the fixed height of the run-filter list (4 items + border).
const filterPanelHeight = 6

// filterDefs defines the selectable run-state filters, in display order.
// An empty state means "All" (no filter).
var filterDefs = []struct {
	state string
	label string
	color string
	glyph string
}{
	{"", "All", "white", "•"},
	{"success", "Success", "green", "✓"},
	{"failed", "Failed", "red", "✗"},
	{"running", "Running", "blue", "⟳"},
}

// DagInfoView shows the selected DAG's metadata plus a focusable mini panel of
// run-state filters. Selecting a filter narrows the Runs view (via onFilter).
type DagInfoView struct {
	*tview.Flex

	meta   *tview.TextView
	filter *tview.List

	dag *models.DAG

	hasStats    bool
	running     int
	success     int
	failed      int
	spark       string
	windowLabel string

	onFilter func(state string)
}

func NewDagInfoView() *DagInfoView {
	v := &DagInfoView{
		Flex:   tview.NewFlex().SetDirection(tview.FlexRow),
		meta:   tview.NewTextView(),
		filter: tview.NewList(),
	}

	v.meta.SetBorder(true).SetTitle(" DAG Info ")
	v.meta.SetDynamicColors(true).SetScrollable(true)
	v.meta.SetText("[gray]Select a DAG to view details")
	v.meta.SetFocusFunc(func() { v.meta.SetBorderColor(theme.DefaultDarkTheme.BorderFocused) })
	v.meta.SetBlurFunc(func() { v.meta.SetBorderColor(theme.DefaultDarkTheme.BorderColor) })

	v.filter.ShowSecondaryText(false)
	v.filter.SetHighlightFullLine(true)
	v.filter.SetSelectedFocusOnly(false)
	// Match the DagList selection colour so the active filter highlight is
	// visually consistent across panels.
	v.filter.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.DefaultDarkTheme.TableSelected).
		Foreground(theme.DefaultDarkTheme.PrimaryText).
		Attributes(tcell.AttrBold))
	v.filter.SetBorder(true).SetTitle(" Runs filter ")
	v.filter.SetFocusFunc(func() { v.filter.SetBorderColor(theme.DefaultDarkTheme.BorderFocused) })
	v.filter.SetBlurFunc(func() { v.filter.SetBorderColor(theme.DefaultDarkTheme.BorderColor) })
	v.filter.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if v.onFilter != nil && idx >= 0 && idx < len(filterDefs) {
			v.onFilter(filterDefs[idx].state)
		}
	})
	v.rebuildFilterItems()

	v.Flex.
		AddItem(v.meta, 0, 1, false).
		AddItem(v.filter, filterPanelHeight, 0, false)

	v.renderMeta()
	return v
}

// FilterList returns the focusable run-filter list (focus target for 'i').
func (v *DagInfoView) FilterList() *tview.List { return v.filter }

// Meta returns the scrollable metadata panel (a focus-ring stop).
func (v *DagInfoView) Meta() *tview.TextView { return v.meta }

// SetOnFilterSelected registers the callback fired when a run-state filter is
// chosen. An empty state means "All" (clear filter).
func (v *DagInfoView) SetOnFilterSelected(fn func(state string)) { v.onFilter = fn }

// Update caches DAG metadata for the newly selected DAG and re-renders. Run
// stats reset (they belong to the previous selection) and the filter returns to
// "All"; stats arrive shortly after via UpdateRunStats.
func (v *DagInfoView) Update(dag models.DAG) {
	d := dag
	v.dag = &d
	v.hasStats = false
	v.running, v.success, v.failed, v.spark = 0, 0, 0, ""
	v.filter.SetCurrentItem(0)
	v.rebuildFilterItems()
	v.renderMeta()
}

// SetWindowLabel sets the lookback-window label (e.g. "7d") shown in the filter
// panel title. Set once at startup.
func (v *DagInfoView) SetWindowLabel(label string) {
	v.windowLabel = label
	v.filter.SetTitle(fmt.Sprintf(" Runs filter (%s) ", label))
}

// UpdateRunStats fills in the per-DAG run counts for the current selection.
// spark is a pre-rendered (colour-tagged) sparkline of recent runs.
func (v *DagInfoView) UpdateRunStats(running, success, failed int, spark string) {
	v.hasStats = true
	v.running = running
	v.success = success
	v.failed = failed
	v.spark = spark
	v.rebuildFilterItems()
	v.renderMeta()
}

func (v *DagInfoView) rebuildFilterItems() {
	cur := v.filter.GetCurrentItem()
	v.filter.Clear()
	for _, d := range filterDefs {
		v.filter.AddItem(v.filterLabel(d.state, d.label, d.color, d.glyph), "", 0, nil)
	}
	if cur >= 0 && cur < v.filter.GetItemCount() {
		v.filter.SetCurrentItem(cur)
	}
}

func (v *DagInfoView) filterLabel(state, label, color, glyph string) string {
	if state == "" {
		return fmt.Sprintf("[%s]%s %s[-]", color, glyph, label)
	}
	if !v.hasStats {
		return fmt.Sprintf("[%s]%s %s  -[-]", color, glyph, label)
	}
	n := 0
	switch state {
	case "success":
		n = v.success
	case "failed":
		n = v.failed
	case "running":
		n = v.running
	}
	return fmt.Sprintf("[%s]%s %s  %d[-]", color, glyph, label, n)
}

func (v *DagInfoView) renderMeta() {
	if v.dag == nil {
		v.meta.SetText("[gray]Select a DAG to view details")
		return
	}
	dag := *v.dag

	tags := make([]string, len(dag.Tags))
	for i, t := range dag.Tags {
		tags[i] = t.Name
	}
	tagStr := "(none)"
	if len(tags) > 0 {
		tagStr = strings.Join(tags, ", ")
	}

	owners := "(none)"
	if len(dag.Owners) > 0 {
		owners = strings.Join(dag.Owners, ", ")
	}

	state := "Active"
	stateColor := "green"
	if dag.IsPaused {
		state = "Paused"
		stateColor = "yellow"
	}

	var b strings.Builder
	fmt.Fprintf(&b,
		"[yellow]DAG ID:[-]    %s\n"+
			"[yellow]State:[-]     [%s]%s[-]\n"+
			"[yellow]Owner:[-]     %s\n"+
			"[yellow]Schedule:[-]  %s\n"+
			"[yellow]Tags:[-]      %s\n"+
			"[yellow]File:[-]      %s\n",
		dag.DagId,
		stateColor, state,
		owners,
		dag.Schedule(),
		tagStr,
		dag.Fileloc,
	)

	if v.spark != "" {
		fmt.Fprintf(&b, "[yellow]Recent:[-]    %s\n", v.spark)
	}

	fmt.Fprintf(&b, "\n[yellow]Description:[-]\n%s", derefStr(dag.Description))

	v.meta.SetText(b.String())
}

func (v *DagInfoView) Root() tview.Primitive { return v.Flex }

// RunSparkline renders up to max recent runs as a colour-tagged glyph strip,
// oldest→newest (left→right). runs are expected newest-first (the order the API
// returns them in).
func RunSparkline(runs []models.DAGRun, max int) string {
	if max <= 0 || len(runs) == 0 {
		return ""
	}
	n := min(len(runs), max)
	var b strings.Builder
	for i := n - 1; i >= 0; i-- {
		b.WriteString(runGlyph(runs[i].State))
	}
	return b.String()
}

func runGlyph(state string) string {
	switch state {
	case "success":
		return "[green]✓[-]"
	case "failed":
		return "[red]✗[-]"
	case "running":
		return "[blue]⟳[-]"
	case "queued", "up_for_retry", "up_for_reschedule", "restarting":
		return "[yellow]●[-]"
	default:
		return "[gray]·[-]"
	}
}
