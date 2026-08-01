package views

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// DagInfoView shows the selected DAG's metadata and recent-run sparkline.
type DagInfoView struct {
	*tview.Flex

	meta *tview.TextView

	dag *models.DAG

	spark string
}

func NewDagInfoView() *DagInfoView {
	v := &DagInfoView{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
		meta: tview.NewTextView(),
	}

	v.meta.SetBorder(true).SetTitle(" DAG Info ")
	v.meta.SetDynamicColors(true).SetScrollable(true)
	v.meta.SetText("[gray]Select a DAG to view details")
	v.meta.SetFocusFunc(func() { v.meta.SetBorderColor(theme.ActiveTheme().BorderFocused) })
	v.meta.SetBlurFunc(func() { v.meta.SetBorderColor(theme.ActiveTheme().BorderColor) })

	v.Flex.AddItem(v.meta, 0, 1, true)

	v.renderMeta()
	return v
}

// Meta returns the scrollable metadata panel (a focus-ring stop).
func (v *DagInfoView) Meta() *tview.TextView { return v.meta }

// Update caches DAG metadata for the newly selected DAG and re-renders. Run
// recent-run data resets because it belongs to the previous selection.
func (v *DagInfoView) Update(dag models.DAG) {
	d := dag
	v.dag = &d
	v.spark = ""
	v.renderMeta()
}

// UpdateRecentRuns fills in the selected DAG's recent-run sparkline.
func (v *DagInfoView) UpdateRecentRuns(spark string) {
	v.spark = spark
	v.renderMeta()
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
	fmt.Fprintf(
		&b,
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
