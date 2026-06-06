package views

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	*tview.Table
}

func NewHelpView() *HelpView {
	v := &HelpView{
		Table: tview.NewTable(),
	}
	v.SetBorder(true).SetTitle(" Keymap ")
	v.SetSelectable(false, false).SetFixed(1, 0)
	v.render()
	return v
}

func (v *HelpView) render() {
	v.SetCell(0, 0, tview.NewTableCell("Key").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetExpansion(1))
	v.SetCell(0, 1, tview.NewTableCell("Action").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetExpansion(4))

	row := 1
	row = v.addSection(row, "Navigation")
	row = v.addBinding(row, "j / k", "Move up / down")
	row = v.addBinding(row, "Enter", "Select / drill down")
	row = v.addBinding(row, "Esc", "Close modal, execution view, or return focus to DAG list")

	row = v.addSection(row+1, "Tabs")
	row = v.addBinding(row, "1-7", "Pipeline tabs: runs, tasks, logs, code, lineage, monitor, backfills")
	row = v.addBinding(row, "8 / 9 / 0", "Global tabs: connections, variables, config")
	row = v.addBinding(row, "Left / Right", "Previous / next tab")
	row = v.addBinding(row, "B", "Backfills")
	row = v.addBinding(row, "g", "Toggle tasks gantt or lineage graph")
	row = v.addBinding(row, "?", "Open this keymap page")

	row = v.addSection(row+1, "DAG Actions")
	row = v.addBinding(row, "t", "Trigger selected DAG run")
	row = v.addBinding(row, "p", "Pause / unpause selected DAG")
	row = v.addBinding(row, "b", "Backfill selected DAG")

	row = v.addSection(row+1, "Modal Actions")
	row = v.addBinding(row, "Esc", "Close without running")
	row = v.addBinding(row, "Enter", "Submit when focused outside a JSON text area")
	row = v.addBinding(row, "Ctrl+J / Ctrl+M", "Submit from anywhere in the form")

	row = v.addSection(row+1, "Backfill Actions")
	row = v.addBinding(row, "p / u", "Pause / unpause selected backfill")
	row = v.addBinding(row, "c", "Cancel selected backfill")

	row = v.addSection(row+1, "DAG Filters")
	row = v.addBinding(row, "a", "Active DAGs")
	row = v.addBinding(row, "A", "All DAGs")
	row = v.addBinding(row, "f", "Failed DAGs")

	row = v.addSection(row+1, "Focus")
	row = v.addBinding(row, "d", "DAG list")
	row = v.addBinding(row, "i", "DAG info")
	row = v.addBinding(row, "o", "Cluster panel (press again to toggle pool compact/table)")

	row = v.addSection(row+1, "General")
	row = v.addBinding(row, "F5", "Refresh")
	row = v.addBinding(row, "/", "Search")
	_ = v.addBinding(row, "Ctrl+C", "Quit")
}

func (v *HelpView) addSection(row int, title string) int {
	cell := tview.NewTableCell(title).
		SetTextColor(tcell.ColorAqua).
		SetSelectable(false).
		SetExpansion(1)
	v.SetCell(row, 0, cell)
	v.SetCell(row, 1, tview.NewTableCell("").SetSelectable(false))
	return row + 1
}

func (v *HelpView) addBinding(row int, key, action string) int {
	v.SetCell(row, 0, tview.NewTableCell(key).
		SetTextColor(tcell.ColorWhite).
		SetSelectable(false).
		SetExpansion(1))
	v.SetCell(row, 1, tview.NewTableCell(action).
		SetTextColor(tcell.ColorWhite).
		SetSelectable(false).
		SetExpansion(4))
	return row + 1
}

func (v *HelpView) Root() *tview.Table {
	return v.Table
}
