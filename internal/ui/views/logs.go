package views

import (
	"github.com/rivo/tview"
)

type LogsView struct {
	*tview.TextView
}

func NewLogsView() *LogsView {
	v := &LogsView{
		TextView: tview.NewTextView(),
	}
	v.SetBorder(true).SetTitle(" Task Logs ")
	v.SetDynamicColors(true).SetScrollable(true)
	v.SetText("[gray]Select a task to view logs")
	return v
}

func (v *LogsView) SetContent(text string) {
	v.SetText(tview.Escape(text))
	v.ScrollToEnd()
}

func (v *LogsView) SetError(msg string) {
	v.SetText("[red]" + tview.Escape(msg))
	v.ScrollToEnd()
}

func (v *LogsView) Root() *tview.TextView {
	return v.TextView
}
