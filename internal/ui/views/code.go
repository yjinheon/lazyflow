package views

import (
	"github.com/rivo/tview"
)

type CodeView struct {
	*tview.TextView
}

func NewCodeView() *CodeView {
	v := &CodeView{
		TextView: tview.NewTextView(),
	}
	v.SetBorder(true).SetTitle(" DAG Code ")
	v.SetDynamicColors(true).SetScrollable(true).SetWrap(false)
	v.SetText("[gray]Select a DAG to view source code")
	return v
}

func (v *CodeView) SetContent(code string) {
	v.SetText(tview.Escape(code))
	v.ScrollToBeginning()
}

// SetHighlighted takes pre-rendered tview markup (see HighlightPython), which
// is produced off the tview goroutine because highlighting is CPU-bound.
func (v *CodeView) SetHighlighted(markup string) {
	v.SetText(markup)
	v.ScrollToBeginning()
}

func (v *CodeView) SetMessage(msg string) {
	v.SetText("[gray]" + tview.Escape(msg))
	v.ScrollToBeginning()
}

func (v *CodeView) SetError(msg string) {
	v.SetText("[red]" + tview.Escape(msg))
	v.ScrollToBeginning()
}

func (v *CodeView) Root() *tview.TextView {
	return v.TextView
}
