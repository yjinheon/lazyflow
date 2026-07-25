package layout

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// renderBar draws the status bar through Root(), the same primitive the layout
// mounts — drawing the wrapper directly would hide a wrong Root() type.
func renderBar(t *testing.T, s *StatusBar, w int) string {
	t.Helper()
	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatalf("init screen: %v", err)
	}
	defer scr.Fini()
	scr.SetSize(w, 1)

	root := s.Root()
	root.SetRect(0, 0, w, 1)
	root.Draw(scr)
	scr.Show()

	cells, _, _ := scr.GetContents()
	var sb strings.Builder
	for _, c := range cells {
		if len(c.Runes) > 0 {
			sb.WriteRune(c.Runes[0])
		} else {
			sb.WriteByte(' ')
		}
	}
	return strings.TrimRight(sb.String(), " ")
}

// The monitor hints use '[' and ']' as key names — exactly the characters tview
// parses as colour tags. They must survive to the screen.
func TestStatusBarRendersBracketKeys(t *testing.T) {
	s := NewStatusBar()
	s.SetContext("monitor", true)

	got := renderBar(t, s, 160)
	for _, want := range []string{"[:prev", "]:next", "r:refresh"} {
		if !strings.Contains(got, want) {
			t.Errorf("hint %q missing from status bar\n  got=%q", want, got)
		}
	}
}

// An action result must not wipe the hints; that was the point of splitting them.
func TestStatusBarKeepsHintsThroughStatusAndError(t *testing.T) {
	s := NewStatusBar()
	s.SetContext("runs", true)
	s.SetInfo("etl", "", "")

	for _, step := range []struct {
		name string
		do   func()
		want string
	}{
		{"info", func() {}, "DAG:etl"},
		{"status", func() { s.SetStatus("DAG etl triggered") }, "DAG etl triggered"},
		{"error", func() { s.SetError("boom") }, "Error: boom"},
	} {
		step.do()
		got := renderBar(t, s, 160)
		if !strings.Contains(got, step.want) {
			t.Errorf("%s: expected %q\n  got=%q", step.name, step.want, got)
		}
		if !strings.Contains(got, "t:trigger") {
			t.Errorf("%s: hints were wiped\n  got=%q", step.name, got)
		}
	}
}

// A new selection supersedes the previous action result.
func TestStatusBarSelectionClearsFlash(t *testing.T) {
	s := NewStatusBar()
	s.SetContext("runs", true)
	s.SetError("boom")
	s.SetInfo("other", "", "")

	got := renderBar(t, s, 160)
	if strings.Contains(got, "boom") {
		t.Errorf("stale error survived a new selection\n  got=%q", got)
	}
	if !strings.Contains(got, "DAG:other") {
		t.Errorf("new selection not shown\n  got=%q", got)
	}
}

// On a narrow terminal the run id gives way before the key hints do.
func TestStatusBarShrinksInfoBeforeHints(t *testing.T) {
	newBar := func() *StatusBar {
		s := NewStatusBar()
		s.SetContext("tasks", true)
		s.SetInfo("etl_daily", "manual__2026-07-25T10:00:00+00:00", "extract_rows")
		return s
	}

	wide := renderBar(t, newBar(), 120)
	if !strings.Contains(wide, "Run:manual__2026-07-25T10:00:00+00:00") {
		t.Errorf("wide terminal should show the full run id\n  got=%q", wide)
	}

	mid := renderBar(t, newBar(), 100)
	if !strings.Contains(mid, "…") {
		t.Errorf("run id should be shortened at 100 cols\n  got=%q", mid)
	}

	narrow := renderBar(t, newBar(), 80)
	if strings.Contains(narrow, "Run:") {
		t.Errorf("run id should be dropped at 80 cols\n  got=%q", narrow)
	}

	// The keys stay reachable at every width, and DAG/Task context survives.
	for _, out := range []string{wide, mid, narrow} {
		if !strings.Contains(out, "t:trigger") {
			t.Errorf("hints lost while shrinking\n  got=%q", out)
		}
		if !strings.Contains(out, "DAG:etl_daily") || !strings.Contains(out, "Task:extract_rows") {
			t.Errorf("DAG/Task context dropped before the run id\n  got=%q", out)
		}
	}
}

// Hints track the active tab and whether a DAG is selected.
func TestStatusBarHintsAreContextual(t *testing.T) {
	s := NewStatusBar()

	s.SetContext("runs", false)
	noDag := renderBar(t, s, 160)
	if strings.Contains(noDag, "t:trigger") {
		t.Errorf("trigger offered with no DAG selected\n  got=%q", noDag)
	}
	if !strings.Contains(noDag, "Enter:select a DAG") {
		t.Errorf("expected a nudge when nothing is selected\n  got=%q", noDag)
	}

	s.SetContext("backfills", true)
	bf := renderBar(t, s, 160)
	for _, want := range []string{"c:cancel", "u:unpause", "t:trigger"} {
		if !strings.Contains(bf, want) {
			t.Errorf("backfills hint %q missing\n  got=%q", want, bf)
		}
	}

	s.SetContext("tasks", true)
	if got := renderBar(t, s, 160); !strings.Contains(got, "g:gantt") {
		t.Errorf("tasks hint missing\n  got=%q", got)
	}
}
