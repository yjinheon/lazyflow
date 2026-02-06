package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type RunsView struct {
	*tview.Table
	runs       []models.DAGRun
	onSelected func(runId string)
}

func NewRunsView() *RunsView {
	v := &RunsView{
		Table: tview.NewTable(),
	}
	v.setup()
	return v
}

func (v *RunsView) setup() {
	v.SetBorder(true).SetTitle(" DAG Runs ")
	v.SetSelectable(true, false)
	v.SetFixed(1, 0)
	v.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.DefaultDarkTheme.TableSelected).
		Foreground(theme.DefaultDarkTheme.PrimaryText).
		Attributes(tcell.AttrBold))
	v.SetFocusFunc(func() { v.SetBorderColor(theme.DefaultDarkTheme.BorderFocused) })
	v.SetBlurFunc(func() { v.SetBorderColor(theme.DefaultDarkTheme.BorderColor) })

	headers := []string{"Run ID", "State", "Start", "End", "Duration", "Type"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if i == 0 {
			cell.SetExpansion(1)
		}
		v.SetCell(0, i, cell)
	}

	v.SetSelectedFunc(func(row, column int) {
		if row > 0 && row <= len(v.runs) {
			if v.onSelected != nil {
				v.onSelected(v.runs[row-1].RunId)
			}
		}
	})
}

func (v *RunsView) SetOnSelected(handler func(runId string)) {
	v.onSelected = handler
}

func (v *RunsView) Update(runs []models.DAGRun) {
	v.runs = runs
	v.Clear()
	v.setup()

	t := theme.DefaultDarkTheme
	for i, run := range runs {
		row := i + 1
		bg := t.PrimaryBg
		if row%2 == 0 {
			bg = t.TableRowAlt
		}

		v.SetCell(row, 0, tview.NewTableCell(run.RunId).
			SetTextColor(tcell.ColorWhite).SetExpansion(1).SetBackgroundColor(bg))

		symbol, color := t.StatusStyle(run.State)
		v.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%s %s", symbol, run.State)).
			SetTextColor(color).SetBackgroundColor(bg))

		startStr := ""
		if !run.StartDate.IsZero() {
			startStr = run.StartDate.Format("01-02 15:04:05")
		}
		v.SetCell(row, 2, tview.NewTableCell(startStr).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		endStr := ""
		if !run.EndDate.IsZero() {
			endStr = run.EndDate.Format("01-02 15:04:05")
		}
		v.SetCell(row, 3, tview.NewTableCell(endStr).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		v.SetCell(row, 4, tview.NewTableCell(formatDuration(run.Duration())).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		v.SetCell(row, 5, tview.NewTableCell(run.RunType).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))
	}
}

func (v *RunsView) Root() *tview.Table {
	return v.Table
}
