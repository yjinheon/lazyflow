package ui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/state"
	"github.com/yjinheon/lazyflow/internal/ui/layout"
)

func newKB(t *testing.T) (*KeyBindings, *layout.MainLayout, *state.Store) {
	t.Helper()
	app := tview.NewApplication()
	l := layout.NewMainLayout(app)
	s := state.NewStore()
	return NewKeyBindings(app, l, s), l, s
}

func key(k tcell.Key, r rune, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(k, r, mod)
}

// Bare arrows belong to the focused widget; only Shift+arrow cycles tabs.
func TestArrowsPassThroughAndShiftCyclesTabs(t *testing.T) {
	kb, _, s := newKB(t)
	s.SetActiveTab("runs")

	for _, k := range []tcell.Key{tcell.KeyLeft, tcell.KeyRight} {
		if got := kb.handle(key(k, 0, tcell.ModNone)); got == nil {
			t.Errorf("bare %v was swallowed; it must reach the focused widget", k)
		}
	}
	if s.ActiveTab() != "runs" {
		t.Errorf("bare arrows changed the tab to %q", s.ActiveTab())
	}

	if got := kb.handle(key(tcell.KeyRight, 0, tcell.ModShift)); got != nil {
		t.Error("Shift+Right should be consumed for tab cycling")
	}
	if s.ActiveTab() != "tasks" {
		t.Errorf("Shift+Right → %q, want tasks", s.ActiveTab())
	}
	if got := kb.handle(key(tcell.KeyLeft, 0, tcell.ModShift)); got != nil {
		t.Error("Shift+Left should be consumed for tab cycling")
	}
	if s.ActiveTab() != "runs" {
		t.Errorf("Shift+Left → %q, want runs", s.ActiveTab())
	}
}

// '<' / '>' are the terminal-safe fallback for Shift+arrows.
func TestRuneTabCycling(t *testing.T) {
	kb, _, s := newKB(t)
	s.SetActiveTab("runs")

	if got := kb.handle(key(tcell.KeyRune, '>', tcell.ModNone)); got != nil {
		t.Error("'>' should be consumed")
	}
	if s.ActiveTab() != "tasks" {
		t.Errorf("'>' → %q, want tasks", s.ActiveTab())
	}
	if kb.handle(key(tcell.KeyRune, '<', tcell.ModNone)) != nil {
		t.Error("'<' should be consumed")
	}
	if s.ActiveTab() != "runs" {
		t.Errorf("'<' → %q, want runs", s.ActiveTab())
	}
}

// Tab cycling must never land on the help page, which is '?'-only.
func TestCycleTabSkipsHelp(t *testing.T) {
	kb, _, s := newKB(t)
	s.SetActiveTab("runs")
	for i := 0; i < 2*len(tabNames)+1; i++ {
		kb.cycleTab(1)
		if s.ActiveTab() == "help" {
			t.Fatalf("cycleTab landed on help after %d steps", i+1)
		}
	}
	for _, name := range cycleTabNames {
		if name == "help" {
			t.Fatal("cycleTabNames still contains help")
		}
	}
}

// 'g' toggles a view only on tasks/lineage; elsewhere it stays tview's jump-to-top.
func TestGIsConditionallyConsumed(t *testing.T) {
	kb, _, s := newKB(t)

	for _, tab := range []string{"tasks", "lineage"} {
		s.SetActiveTab(tab)
		if got := kb.handle(key(tcell.KeyRune, 'g', tcell.ModNone)); got != nil {
			t.Errorf("'g' on %q should be consumed", tab)
		}
	}
	for _, tab := range []string{"runs", "logs", "connections", "config"} {
		s.SetActiveTab(tab)
		if got := kb.handle(key(tcell.KeyRune, 'g', tcell.ModNone)); got == nil {
			t.Errorf("'g' on %q was swallowed; jump-to-top is lost", tab)
		}
	}
}

// 'r' refreshes only on the monitor tab; elsewhere it reaches the widget.
func TestRIsConditionallyConsumed(t *testing.T) {
	kb, _, s := newKB(t)
	refreshed := 0
	kb.SetOnMonitorRefresh(func() { refreshed++ })

	s.SetActiveTab("monitor")
	if got := kb.handle(key(tcell.KeyRune, 'r', tcell.ModNone)); got != nil {
		t.Error("'r' on monitor should be consumed")
	}
	if refreshed != 1 {
		t.Errorf("monitor refresh called %d times, want 1", refreshed)
	}

	s.SetActiveTab("runs")
	if got := kb.handle(key(tcell.KeyRune, 'r', tcell.ModNone)); got == nil {
		t.Error("'r' off monitor was swallowed")
	}
	if refreshed != 1 {
		t.Errorf("monitor refresh fired off-tab (%d)", refreshed)
	}
}

// Regression guard for the tview v0.42.0 header-only Table infinite loop: every
// navigation key must terminate on an empty table. The loop only reproduces
// after a Draw (it needs the draw-time selection clamp), so each target is drawn
// on a simulation screen first — without that this test cannot fail.
func TestEmptyTableNavigationDoesNotHang(t *testing.T) {
	kb, l, _ := newKB(t)

	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatalf("init simulation screen: %v", err)
	}
	defer scr.Fini()
	scr.SetSize(120, 40)

	events := []*tcell.EventKey{
		key(tcell.KeyUp, 0, tcell.ModNone),
		key(tcell.KeyDown, 0, tcell.ModNone),
		key(tcell.KeyLeft, 0, tcell.ModNone),
		key(tcell.KeyRight, 0, tcell.ModNone),
		key(tcell.KeyHome, 0, tcell.ModNone),
		key(tcell.KeyEnd, 0, tcell.ModNone),
		key(tcell.KeyPgUp, 0, tcell.ModNone),
		key(tcell.KeyPgDn, 0, tcell.ModNone),
		key(tcell.KeyRune, 'j', tcell.ModNone),
		key(tcell.KeyRune, 'k', tcell.ModNone),
		key(tcell.KeyRune, 'h', tcell.ModNone),
		key(tcell.KeyRune, 'l', tcell.ModNone),
		key(tcell.KeyRune, 'g', tcell.ModNone),
		key(tcell.KeyRune, 'G', tcell.ModNone),
	}

	targets := []tview.Primitive{
		l.DagList(),
		l.Runs().Root(),
		l.Connections().Root(),
		l.Variables().Root(),
		l.DagInfo().FilterList(),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		noFocus := func(tview.Primitive) {}
		for _, target := range targets {
			handler := target.InputHandler()
			if handler == nil {
				continue
			}
			target.SetRect(0, 0, 120, 40)
			target.Draw(scr) // arms the draw-time clamp that triggers the loop
			for _, ev := range events {
				// Mirror runtime order: global capture first, widget second.
				if out := kb.handle(ev); out != nil {
					handler(out, noFocus)
				}
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("navigation keys hung on an empty table (tview header-only loop)")
	}
}
