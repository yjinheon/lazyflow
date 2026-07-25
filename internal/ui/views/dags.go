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
	activeDagId string // committed via Enter; distinct from the cursor row
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
	v.SetBorder(true).SetTitle(v.titleText()).SetBorderColor(theme.ActiveTheme().BorderColor)
	// Selectable is toggled in render() based on whether any data rows
	// remain after filtering. With a header-only table, tview's draw-time
	// clamp pushes selectedRow out of bounds, then the next Down arrow
	// enters an infinite loop in Table.InputHandler down→backwards
	// (table.go:1428).
	v.SetSelectable(false, false)
	v.SetFixed(1, 0)
	v.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.ActiveTheme().TableSelected).
		Foreground(theme.ActiveTheme().PrimaryText).
		Attributes(tcell.AttrBold))
	v.SetFocusFunc(func() { v.SetBorderColor(theme.ActiveTheme().BorderFocused) })
	v.SetBlurFunc(func() { v.SetBorderColor(theme.ActiveTheme().BorderColor) })

	v.renderHeaders()

	v.SetSelectedFunc(func(row, column int) {
		if row > 0 && row <= len(v.dags) {
			v.setActiveDag(v.dags[row-1].DagId)
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
			SetTextColor(theme.ActiveTheme().TableHeaderText).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		v.SetCell(0, i, cell)
	}
}

func (v *DagListView) emptyHint() string {
	if v.searchQuery != "" {
		return "No DAGs match this search — press Esc then / to search again."
	}
	if v.filterMode != "all" {
		return "No DAGs in this filter — press A for all DAGs."
	}
	return "No DAGs loaded yet — press F5 to refresh."
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

// setActiveDag re-marks the committed row in place. It avoids Clear/SetSelectable
// because it runs inside the table's own input handler.
func (v *DagListView) setActiveDag(dagId string) {
	if v.activeDagId == dagId {
		return
	}
	v.activeDagId = dagId
	for i, d := range v.dags {
		cell := v.GetCell(i+1, 0)
		if cell == nil {
			continue
		}
		active := d.DagId == dagId
		cell.SetText(rowLabel(d.DagId, active)).SetTextColor(rowLabelColor(active))
	}
}

func (v *DagListView) render() {
	v.Clear()
	v.renderHeaders()
	if len(v.dags) == 0 {
		v.SetSelectable(false, false)
		setEmptyHint(v.Table, v.emptyHint())
		return
	}
	v.SetSelectable(true, false)

	t := theme.ActiveTheme()
	for i, dag := range v.dags {
		row := i + 1
		bg := t.PrimaryBg
		if row%2 == 0 {
			bg = t.TableRowAlt
		}

		active := dag.DagId == v.activeDagId
		v.SetCell(row, 0, tview.NewTableCell(rowLabel(dag.DagId, active)).
			SetTextColor(rowLabelColor(active)).SetExpansion(1).SetBackgroundColor(bg))

		stateStr := "Active"
		stateColor := t.StatusSuccess
		if dag.IsPaused {
			stateStr = "Paused"
			stateColor = t.MutedText
		}
		v.SetCell(row, 1, tview.NewTableCell(stateStr).SetTextColor(stateColor).SetBackgroundColor(bg))

		v.SetCell(row, 2, tview.NewTableCell(dag.Schedule()).SetTextColor(t.PrimaryText).SetBackgroundColor(bg))

		ownerStr := ""
		if len(dag.Owners) > 0 {
			ownerStr = dag.Owners[0]
		}
		v.SetCell(row, 3, tview.NewTableCell(ownerStr).SetTextColor(t.Accent).SetBackgroundColor(bg))
	}
}

func (v *DagListView) Root() *tview.Table {
	return v.Table
}
