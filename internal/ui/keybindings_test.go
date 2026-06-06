package ui

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/state"
	"github.com/yjinheon/lazyflow/internal/ui/layout"
)

// TestCycleFocusRing verifies Tab / Shift+Tab walk every panel so the whole UI
// is reachable by keyboard alone.
func TestCycleFocusRing(t *testing.T) {
	app := tview.NewApplication()
	l := layout.NewMainLayout(app)
	kb := NewKeyBindings(app, l, state.NewStore())

	ring := kb.focusRing()
	if len(ring) != 5 {
		t.Fatalf("focus ring length = %d, want 5", len(ring))
	}

	// Start on the DAG list, then Tab forward through the whole ring and back.
	app.SetFocus(l.DagList())
	forward := make([]tview.Primitive, 0, len(ring))
	for range ring {
		kb.cycleFocus(1)
		forward = append(forward, app.GetFocus())
	}
	// After len(ring) forward steps we should be back on the DAG list.
	if app.GetFocus() != tview.Primitive(l.DagList()) {
		t.Fatalf("cycling forward did not return to DAG list")
	}

	// Each ring stop should be visited exactly once over a full lap.
	seen := map[tview.Primitive]bool{}
	for _, p := range forward {
		seen[p] = true
	}
	if len(seen) != len(ring) {
		t.Fatalf("forward lap visited %d distinct panels, want %d", len(seen), len(ring))
	}

	// Shift+Tab from the DAG list lands on the last ring stop (active tab).
	app.SetFocus(l.DagList())
	kb.cycleFocus(-1)
	if app.GetFocus() != ring[len(ring)-1] {
		t.Fatalf("Shift+Tab from DAG list should land on the last panel")
	}
}
