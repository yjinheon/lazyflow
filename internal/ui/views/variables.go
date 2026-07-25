package views

import (
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
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
	// See RunsView.setup: start non-selectable to avoid tview Table's
	// infinite-loop on Down arrow when no data rows exist.
	v.SetSelectable(false, false).SetFixed(1, 0)
	v.renderHeaders()
	return v
}

func (v *VariablesView) renderHeaders() {
	for i, h := range []string{"Key", "Value", "Description"} {
		cell := tview.NewTableCell(h).
			SetTextColor(theme.ActiveTheme().TableHeaderText).
			SetSelectable(false)
		v.SetCell(0, i, cell)
	}
}

func (v *VariablesView) Update(vars []models.Variable) {
	v.Clear()
	v.renderHeaders()
	if len(vars) == 0 {
		v.SetSelectable(false, false)
		setEmptyHint(v.Table, "No variables defined in this Airflow instance.")
		return
	}
	v.SetSelectable(true, false)

	for i, vr := range vars {
		row := i + 1
		v.SetCell(row, 0, tview.NewTableCell(vr.Key).SetTextColor(theme.ActiveTheme().PrimaryText).SetExpansion(1))
		v.SetCell(row, 1, tview.NewTableCell(vr.Value).SetTextColor(theme.ActiveTheme().PrimaryText).SetExpansion(2))
		v.SetCell(row, 2, tview.NewTableCell(vr.Description).SetTextColor(theme.ActiveTheme().MutedText))
	}
}

func (v *VariablesView) Root() *tview.Table {
	return v.Table
}
