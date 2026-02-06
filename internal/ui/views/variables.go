package views

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type VariablesView struct {
	*tview.Table
}

func NewVariablesView() *VariablesView {
	v := &VariablesView{
		Table: tview.NewTable(),
	}
	v.SetBorder(true).SetTitle(" Variables ")
	v.SetSelectable(true, false).SetFixed(1, 0)
	v.renderHeaders()
	return v
}

func (v *VariablesView) renderHeaders() {
	for i, h := range []string{"Key", "Value", "Description"} {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false)
		v.SetCell(0, i, cell)
	}
}

func (v *VariablesView) Update(vars []models.Variable) {
	v.Clear()
	v.renderHeaders()

	for i, vr := range vars {
		row := i + 1
		v.SetCell(row, 0, tview.NewTableCell(vr.Key).SetTextColor(tcell.ColorWhite).SetExpansion(1))
		v.SetCell(row, 1, tview.NewTableCell(vr.Value).SetTextColor(tcell.ColorWhite).SetExpansion(2))
		v.SetCell(row, 2, tview.NewTableCell(vr.Description).SetTextColor(tcell.ColorGray))
	}
}

func (v *VariablesView) Root() *tview.Table {
	return v.Table
}
