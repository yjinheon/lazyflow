package views

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

// Renders the markup through tview onto a simulation screen and reads the cell
// colours back, so the assertion is what the terminal actually shows.
func TestHighlightLogs_rendersRichColoursInTview(t *testing.T) {
	const line = "{task.stdout} INFO - [cyan]transform_b[/cyan]: rows=[green]4[/green]"
	const visible = "{task.stdout} INFO - transform_b: rows=4"

	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatalf("init screen: %v", err)
	}
	defer scr.Fini()
	scr.SetSize(120, 3)

	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	tv.SetText(HighlightLogs(line))
	tv.SetRect(0, 0, 120, 3)
	tv.Draw(scr)
	scr.Show()

	cells, w, _ := scr.GetContents()
	at := func(col int) (rune, tcell.Color) {
		c := cells[col]
		fg, _, _ := c.Style.Decompose()
		if len(c.Runes) == 0 {
			return ' ', fg
		}
		return c.Runes[0], fg
	}

	// The rendered row must carry the visible text with the tags gone.
	var row strings.Builder
	for i := 0; i < w; i++ {
		r, _ := at(i)
		row.WriteRune(r)
	}
	if got := strings.TrimRight(row.String(), " "); got != visible {
		t.Fatalf("rendered row = %q, want %q", got, visible)
	}

	th := theme.ActiveTheme()
	checks := []struct {
		needle string
		col    int
		want   tcell.Color
	}{
		{"transform_b", strings.Index(visible, "transform_b"), th.SectionHeader},
		{"4", strings.LastIndex(visible, "4"), th.StatusSuccess},
		{"INFO", strings.Index(visible, "INFO"), th.Accent},
	}
	for _, c := range checks {
		r, fg := at(c.col)
		if fg != c.want {
			t.Errorf("%q at col %d rendered as %v (rune %q), want %v", c.needle, c.col, fg, r, c.want)
		}
	}
}
