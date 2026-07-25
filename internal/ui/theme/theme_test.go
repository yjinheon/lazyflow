package theme

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ApplyTheme must rebind tview's markup colour names, otherwise the ~100
// "[yellow]…[-]" literals in the views keep rendering in raw ANSI colours.
func TestApplyThemeRebindsMarkupNames(t *testing.T) {
	ApplyTheme(TokyoNightStorm)

	for name, want := range map[string]tcell.Color{
		"yellow": TokyoNightStorm.TableHeaderText,
		"gray":   TokyoNightStorm.MutedText,
		"red":    TokyoNightStorm.StatusFailed,
		"green":  TokyoNightStorm.StatusSuccess,
		"blue":   TokyoNightStorm.StatusRunning,
		"white":  TokyoNightStorm.PrimaryText,
		"black":  TokyoNightStorm.PrimaryBg,
		"aqua":   TokyoNightStorm.SectionHeader,
	} {
		if got := tcell.GetColor(name); got != want {
			t.Errorf("GetColor(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestApplyThemeSetsTviewStyles(t *testing.T) {
	ApplyTheme(TokyoNightStorm)

	if tview.Styles.PrimitiveBackgroundColor != TokyoNightStorm.PrimaryBg {
		t.Error("tview background not taken from the theme")
	}
	if tview.Styles.PrimaryTextColor != TokyoNightStorm.PrimaryText {
		t.Error("tview primary text not taken from the theme")
	}
	if ActiveTheme().Name != TokyoNightStorm.Name {
		t.Errorf("ActiveTheme() = %q, want %q", ActiveTheme().Name, TokyoNightStorm.Name)
	}
}

// Every run/task state must be visually distinguishable.
func TestStatusColoursAreDistinct(t *testing.T) {
	states := []string{"running", "success", "failed", "paused", "queued", "upstream_failed", "skipped"}
	seen := map[tcell.Color]string{}
	for _, s := range states {
		symbol, c := TokyoNightStorm.StatusStyle(s)
		if symbol == "" {
			t.Errorf("state %q has no symbol", s)
		}
		if prev, dup := seen[c]; dup {
			t.Errorf("states %q and %q share colour %v", prev, s, c)
		}
		seen[c] = s
	}
}

func TestMarkupHexRoundTrip(t *testing.T) {
	if got := MarkupHex(TokyoNightStorm.StatusFailed); got != "#f7768e" {
		t.Errorf("MarkupHex(StatusFailed) = %q, want #f7768e", got)
	}
	if got := GanttMarkupColor("success"); got != "#73daca" {
		t.Errorf("GanttMarkupColor(success) = %q, want #73daca", got)
	}
}
