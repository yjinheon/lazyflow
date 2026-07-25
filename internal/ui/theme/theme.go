// internal/ui/theme/theme.go
package theme

import (
	"fmt"

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
	StatusSkipped  tcell.Color
	CriticalPath   tcell.Color

	Accent    tcell.Color
	AccentDim tcell.Color

	BorderColor   tcell.Color
	BorderFocused tcell.Color

	TableHeader     tcell.Color
	TableHeaderText tcell.Color
	TableSelected   tcell.Color
	TableRowAlt     tcell.Color

	// SectionHeader styles grouping labels that are not table headers
	// (help sections, config section names, connection types).
	SectionHeader tcell.Color
}

// TokyoNightStorm is the sole UI theme, taken from the Tokyo Night Storm palette.
var TokyoNightStorm = Theme{
	Name: "tokyo-night-storm",

	PrimaryBg:   hex(0x24283b),
	SecondaryBg: hex(0x1f2335),
	TertiaryBg:  hex(0x292e42),

	PrimaryText:   hex(0xc0caf5),
	SecondaryText: hex(0xa9b1d6),
	MutedText:     hex(0x8089b3),

	StatusRunning:  hex(0x7aa2f7), // blue
	StatusSuccess:  hex(0x73daca), // green
	StatusFailed:   hex(0xf7768e), // red
	StatusPaused:   hex(0xe0af68), // yellow
	StatusQueued:   hex(0xbb9af7), // magenta
	StatusUpstream: hex(0x8089b3),
	StatusSkipped:  hex(0x414868),
	CriticalPath:   hex(0xff9e64), // orange

	Accent:    hex(0x7aa2f7),
	AccentDim: hex(0x6183bb),

	BorderColor:   hex(0x3b4261),
	BorderFocused: hex(0x7aa2f7),

	TableHeader:     hex(0x292e42),
	TableHeaderText: hex(0xe0af68),
	TableSelected:   hex(0x3b4261),
	TableRowAlt:     hex(0x1f2335),

	SectionHeader: hex(0x7dcfff), // cyan
}

func hex(v int32) tcell.Color { return tcell.NewHexColor(v) }

// active tracks the most recently applied theme so helpers can resolve
// theme tokens (e.g. GanttMarkupColor) without taking explicit args.
var active = TokyoNightStorm

// ActiveTheme returns the theme most recently passed to ApplyTheme.
func ActiveTheme() Theme { return active }

// ApplyTheme sets the global tview styles and rebinds tview's markup colour
// names to this theme.
func ApplyTheme(t Theme) {
	active = t
	applyMarkupNames(t)
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

// applyMarkupNames rebinds tview markup colour names (e.g. "[yellow]x[-]") to
// theme tokens; tview resolves them via the exported tcell.ColorNames map.
func applyMarkupNames(t Theme) {
	for name, c := range map[string]tcell.Color{
		"black":  t.PrimaryBg,
		"white":  t.PrimaryText,
		"gray":   t.MutedText,
		"yellow": t.TableHeaderText,
		"red":    t.StatusFailed,
		"green":  t.StatusSuccess,
		"blue":   t.StatusRunning,
		"aqua":   t.SectionHeader,
	} {
		tcell.ColorNames[name] = c
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
	case "skipped", "removed":
		return "○", t.StatusSkipped
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

// GanttMarkupColor maps the Gantt renderer's color tokens to tview markup hex strings.
// The renderer emits tokens like "success" / "queued" / "failed" / "running" /
// "skipped" / "upstream" / "critical". tview understands hex strings like
// "#22c55e" inside markup tags: "[#22c55e]…[-]".
func GanttMarkupColor(token string) string {
	t := ActiveTheme()
	switch token {
	case "success":
		return colorHex(t.StatusSuccess)
	case "failed":
		return colorHex(t.StatusFailed)
	case "running":
		return colorHex(t.StatusRunning)
	case "queued":
		return colorHex(t.StatusQueued)
	case "skipped":
		return colorHex(t.StatusSkipped)
	case "upstream":
		return colorHex(t.StatusUpstream)
	case "critical":
		return colorHex(t.CriticalPath)
	default:
		return colorHex(t.PrimaryText)
	}
}

func colorHex(c tcell.Color) string {
	r, g, b := c.RGB()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// MarkupHex converts a tcell color to a "#rrggbb" string usable inside tview
// dynamic-color markup tags, e.g. "[#22c55e]text[-]".
func MarkupHex(c tcell.Color) string {
	return colorHex(c)
}
