package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type ConfigView struct {
	*tview.Table
}

func NewConfigView() *ConfigView {
	v := &ConfigView{
		Table: tview.NewTable(),
	}
	v.SetBorder(true).SetTitle(" Airflow Config ")
	v.SetSelectable(true, false).SetFixed(1, 0)
	v.renderHeaders()
	return v
}

func (v *ConfigView) renderHeaders() {
	for i, h := range []string{"Section", "Key", "Value"} {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false)
		v.SetCell(0, i, cell)
	}
}

func (v *ConfigView) Update(cfg *models.AirflowConfigResponse) {
	v.Clear()
	v.renderHeaders()

	if cfg == nil {
		return
	}

	row := 1
	for _, section := range cfg.Sections {
		for _, opt := range section.Options {
			v.SetCell(row, 0, tview.NewTableCell(section.Section).SetTextColor(tcell.ColorAqua))
			v.SetCell(row, 1, tview.NewTableCell(opt.Key).SetTextColor(tcell.ColorWhite))
			v.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.80s", opt.Value)).SetTextColor(tcell.ColorWhite))
			row++
		}
	}
}

func (v *ConfigView) Root() *tview.Table {
	return v.Table
}
