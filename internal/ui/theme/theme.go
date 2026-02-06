// internal/ui/theme/theme.go
package theme

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Theme defines the global UI style
type Theme struct {
	Name string

	PrimaryBg   tcell.Color
	SecondaryBg tcell.Color
	TertiaryBg  tcell.Color

	PrimaryText   tcell.Color
	SecondaryText tcell.Color
	MutedText     tcell.Color

	StatusRunning  tcell.Color
	StatusSuccess  tcell.Color
	StatusFailed   tcell.Color
	StatusPaused   tcell.Color
	StatusQueued   tcell.Color
	StatusUpstream tcell.Color

	Accent    tcell.Color
	AccentDim tcell.Color

	BorderColor   tcell.Color
	BorderFocused tcell.Color

	TableHeader   tcell.Color
	TableSelected tcell.Color
	TableRowAlt   tcell.Color
}

// DefaultDarkTheme implements a dark theme inspired by kdash/lazydocker
var DefaultDarkTheme = Theme{
	Name: "dark",

	PrimaryBg:   tcell.NewRGBColor(24, 24, 27), // zinc-900
	SecondaryBg: tcell.NewRGBColor(39, 39, 42), // zinc-800
	TertiaryBg:  tcell.NewRGBColor(63, 63, 70), // zinc-700

	PrimaryText:   tcell.NewRGBColor(250, 250, 250), // zinc-50
	SecondaryText: tcell.NewRGBColor(161, 161, 170), // zinc-400
	MutedText:     tcell.NewRGBColor(113, 113, 122), // zinc-500

	StatusRunning:  tcell.NewRGBColor(59, 130, 246),  // blue-500
	StatusSuccess:  tcell.NewRGBColor(34, 197, 94),   // green-500
	StatusFailed:   tcell.NewRGBColor(239, 68, 68),   // red-500
	StatusPaused:   tcell.NewRGBColor(251, 191, 36),  // amber-400
	StatusQueued:   tcell.NewRGBColor(168, 85, 247),  // purple-500
	StatusUpstream: tcell.NewRGBColor(107, 114, 128), // gray-500

	Accent:    tcell.NewRGBColor(99, 102, 241), // indigo-500
	AccentDim: tcell.NewRGBColor(67, 56, 202),  // indigo-700

	BorderColor:   tcell.NewRGBColor(63, 63, 70),   // zinc-700
	BorderFocused: tcell.NewRGBColor(99, 102, 241), // indigo-500

	TableHeader:   tcell.NewRGBColor(63, 63, 70),
	TableSelected: tcell.NewRGBColor(55, 48, 163), // indigo-800
	TableRowAlt:   tcell.NewRGBColor(30, 30, 35),
}

// ApplyTheme sets the global tview styles
func ApplyTheme(t Theme) {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    t.PrimaryBg,
		ContrastBackgroundColor:     t.SecondaryBg,
		MoreContrastBackgroundColor: t.TertiaryBg,
		BorderColor:                 t.BorderColor,
		TitleColor:                  t.Accent,
		GraphicsColor:               t.BorderColor,
		PrimaryTextColor:            t.PrimaryText,
		SecondaryTextColor:          t.SecondaryText,
		TertiaryTextColor:           t.MutedText,
		InverseTextColor:            t.PrimaryBg,
		ContrastSecondaryTextColor:  t.SecondaryText,
	}
}

// StatusStyle returns the symbol and color for a given status
func (t Theme) StatusStyle(status string) (string, tcell.Color) {
	switch status {
	case "running":
		return "●", t.StatusRunning
	case "success":
		return "●", t.StatusSuccess
	case "failed":
		return "●", t.StatusFailed
	case "paused":
		return "⏸", t.StatusPaused
	case "queued":
		return "◌", t.StatusQueued
	case "upstream_failed", "upstream":
		return "⏸", t.StatusUpstream
	default:
		return "○", t.MutedText
	}
}

// TableCellStyle returns the style for a table cell
func (t Theme) TableCellStyle(row, col int, isSelected bool) tcell.Style {
	style := tcell.StyleDefault.
		Background(t.PrimaryBg).
		Foreground(t.PrimaryText)

	if isSelected {
		style = style.Background(t.TableSelected)
	} else if row%2 == 0 {
		style = style.Background(t.TableRowAlt)
	}

	return style
}
