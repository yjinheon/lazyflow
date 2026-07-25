package views

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

// Enter commits a selection while the cursor keeps moving, so the two need
// different marks: the cursor is the selection bar, the committed row this glyph.
const (
	activeMarker   = "▸ "
	inactiveMarker = "  "
)

func rowLabel(text string, active bool) string {
	if active {
		return activeMarker + text
	}
	return inactiveMarker + text
}

func rowLabelColor(active bool) tcell.Color {
	if active {
		return theme.ActiveTheme().Accent
	}
	return theme.ActiveTheme().PrimaryText
}

// setEmptyHint draws a non-selectable hint where data rows would go. Callers
// must leave the table non-selectable (tview header-only Down-arrow loop guard).
func setEmptyHint(tbl *tview.Table, msg string) {
	tbl.SetCell(1, 0, tview.NewTableCell(msg).
		SetTextColor(theme.ActiveTheme().MutedText).
		SetSelectable(false).
		SetExpansion(1))
}
