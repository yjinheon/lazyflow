package views

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

// renderLines draws markup in a TextView and returns the visible text.
func renderLines(t *testing.T, markup string, w, h int) []string {
	t.Helper()
	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatalf("init screen: %v", err)
	}
	defer scr.Fini()
	scr.SetSize(w, h)

	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	tv.SetText(markup)
	tv.SetRect(0, 0, w, h)
	tv.Draw(scr)
	scr.Show()

	cells, cw, _ := scr.GetContents()
	var sb strings.Builder
	for i, c := range cells {
		if i%cw == 0 && i > 0 {
			sb.WriteByte('\n')
		}
		if len(c.Runes) > 0 {
			sb.WriteRune(c.Runes[0])
		} else {
			sb.WriteByte(' ')
		}
	}
	out := []string{}
	for _, line := range strings.Split(sb.String(), "\n") {
		out = append(out, strings.TrimRight(line, " "))
	}
	return out
}

// The invariant: highlighting changes colours, never the characters shown.
// Bracket-heavy Python is the dangerous case — "[red]" reads as a tview tag.
func TestHighlightPreservesVisibleText(t *testing.T) {
	theme.ApplyTheme(theme.TokyoNightStorm)

	cases := []string{
		"x = [red]",
		"x = [blue]",
		"y = [1, 2]",
		"z = d['k']",
		"w = [[a], [b]]",
		"t = arr[i][j]",
		"c = 1  # note [green] here",
		`s = "[yellow]literal"`,
		"def f(a: list[int]) -> dict[str, int]:",
	}

	for _, src := range cases {
		got := renderLines(t, HighlightPython(src+"\n"), 80, 3)[0]
		if got != src {
			t.Errorf("highlighting altered the text\n  src=%q\n  got=%q", src, got)
		}
	}
}

// Plain (unhighlighted) content must survive the same way.
func TestPlainContentPreservesVisibleText(t *testing.T) {
	theme.ApplyTheme(theme.TokyoNightStorm)
	src := "x = [red]"

	v := NewCodeView()
	v.SetContent(src + "\n")
	got := renderLines(t, v.GetText(false), 80, 3)[0]
	if got != src {
		t.Errorf("SetContent altered the text\n  src=%q\n  got=%q", src, got)
	}
}
