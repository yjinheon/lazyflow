package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type ConnectionsView struct {
	*tview.Table
}

func NewConnectionsView() *ConnectionsView {
	v := &ConnectionsView{
		Table: tview.NewTable(),
	}
	v.SetBorder(true).SetTitle(" Connections ")
	v.SetSelectable(true, false).SetFixed(1, 0)
	v.renderHeaders()
	return v
}

func (v *ConnectionsView) renderHeaders() {
	for i, h := range []string{"Conn ID", "Type", "Host", "Port", "Schema", "Login"} {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false)
		v.SetCell(0, i, cell)
	}
}

func (v *ConnectionsView) Update(conns []models.Connection) {
	v.Clear()
	v.renderHeaders()

	for i, c := range conns {
		row := i + 1
		v.SetCell(row, 0, tview.NewTableCell(c.ConnId).SetTextColor(tcell.ColorWhite).SetExpansion(1))
		v.SetCell(row, 1, tview.NewTableCell(c.ConnType).SetTextColor(tcell.ColorAqua))
		v.SetCell(row, 2, tview.NewTableCell(c.Host).SetTextColor(tcell.ColorWhite))
		v.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", c.Port)).SetTextColor(tcell.ColorWhite))
		v.SetCell(row, 4, tview.NewTableCell(c.Schema).SetTextColor(tcell.ColorWhite))
		v.SetCell(row, 5, tview.NewTableCell(c.Login).SetTextColor(tcell.ColorWhite))
	}
}

func (v *ConnectionsView) Root() *tview.Table {
	return v.Table
}
