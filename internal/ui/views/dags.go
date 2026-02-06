package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type DagListView struct {
	*tview.Table
	allDags     []models.DAG // unfiltered
	dags        []models.DAG // currently displayed (filtered)
	filterMode  string       // "all", "active", "failed"
	searchQuery string
	onSelected  func(dagId string)
}

func NewDagListView() *DagListView {
	v := &DagListView{
		Table:      tview.NewTable(),
		filterMode: "all",
	}
	v.setup()
	return v
}

func (v *DagListView) setup() {
	v.SetBorder(true).SetTitle(v.titleText()).SetBorderColor(tcell.ColorGray)
	v.SetSelectable(true, false)
	v.SetFixed(1, 0)
	v.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.DefaultDarkTheme.TableSelected).
		Foreground(theme.DefaultDarkTheme.PrimaryText).
		Attributes(tcell.AttrBold))
	v.SetFocusFunc(func() { v.SetBorderColor(theme.DefaultDarkTheme.BorderFocused) })
	v.SetBlurFunc(func() { v.SetBorderColor(theme.DefaultDarkTheme.BorderColor) })

	v.renderHeaders()

	v.SetSelectedFunc(func(row, column int) {
		if row > 0 && row <= len(v.dags) {
			if v.onSelected != nil {
				v.onSelected(v.dags[row-1].DagId)
			}
		}
	})
}

func (v *DagListView) renderHeaders() {
	headers := []string{"DAG ID", "State", "Schedule", "Owners"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		v.SetCell(0, i, cell)
	}
}

func (v *DagListView) titleText() string {
	title := fmt.Sprintf(" DAGs <%s>", v.filterMode)
	if v.searchQuery != "" {
		title += fmt.Sprintf(" [yellow]/%s[-]", v.searchQuery)
	}
	title += " "
	return title
}

func (v *DagListView) SetOnSelected(handler func(dagId string)) {
	v.onSelected = handler
}

// SetFilter changes the active filter and re-renders.
func (v *DagListView) SetFilter(mode string) {
	v.filterMode = mode
	v.SetTitle(v.titleText())
	v.applyFilter()
	v.render()
}

// Search filters the DAG list by a query string.
func (v *DagListView) Search(query string) {
	v.searchQuery = query
	v.SetTitle(v.titleText())
	v.applyFilter()
	v.render()
}

// Update stores new data and re-renders with current filter.
func (v *DagListView) Update(dags []models.DAG) {
	v.allDags = dags
	v.applyFilter()
	v.render()
}

func (v *DagListView) applyFilter() {
	var filtered []models.DAG
	switch v.filterMode {
	case "active":
		for _, d := range v.allDags {
			if !d.IsPaused {
				filtered = append(filtered, d)
			}
		}
	case "failed":
		for _, d := range v.allDags {
			if d.LastRunState == "failed" {
				filtered = append(filtered, d)
			}
		}
	default:
		filtered = v.allDags
	}

	if v.searchQuery != "" {
		q := strings.ToLower(v.searchQuery)
		var matched []models.DAG
		for _, d := range filtered {
			if strings.Contains(strings.ToLower(d.DagId), q) ||
				strings.Contains(strings.ToLower(d.DisplayName()), q) {
				matched = append(matched, d)
			}
		}
		filtered = matched
	}

	v.dags = filtered
}

func (v *DagListView) render() {
	v.Clear()
	v.renderHeaders()

	t := theme.DefaultDarkTheme
	for i, dag := range v.dags {
		row := i + 1
		bg := t.PrimaryBg
		if row%2 == 0 {
			bg = t.TableRowAlt
		}

		v.SetCell(row, 0, tview.NewTableCell(dag.DagId).
			SetTextColor(tcell.ColorWhite).SetExpansion(1).SetBackgroundColor(bg))

		stateStr := "Active"
		stateColor := tcell.ColorGreen
		if dag.IsPaused {
			stateStr = "Paused"
			stateColor = tcell.ColorDimGray
		}
		v.SetCell(row, 1, tview.NewTableCell(stateStr).SetTextColor(stateColor).SetBackgroundColor(bg))

		v.SetCell(row, 2, tview.NewTableCell(dag.Schedule()).SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		ownerStr := ""
		if len(dag.Owners) > 0 {
			ownerStr = dag.Owners[0]
		}
		v.SetCell(row, 3, tview.NewTableCell(ownerStr).SetTextColor(tcell.ColorBlue).SetBackgroundColor(bg))
	}
}

func (v *DagListView) Root() *tview.Table {
	return v.Table
}
