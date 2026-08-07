package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/debugutil"
	"github.com/yjinheon/lazyflow/internal/state"
	"github.com/yjinheon/lazyflow/internal/ui/layout"
)

var tabNames = []struct {
	key  rune
	name string
}{
	{'1', "runs"},
	{'2', "tasks"},
	{'3', "logs"},
	{'4', "code"},
	{'5', "lineage"},
	{'6', "monitor"},
	{'7', "backfills"},
	{'8', "connections"},
	{'9', "variables"},
	{'0', "config"},
	{'?', "help"},
}

// tabForRune resolves a digit key to its tab name.
func tabForRune(r rune) (string, bool) {
	for _, t := range tabNames {
		if t.key == r && r != '?' {
			return t.name, true
		}
	}
	return "", false
}

type KeyBindings struct {
	app    *tview.Application
	layout *layout.MainLayout
	store  *state.Store

	onRefresh         func()
	onTrigger         func(dagId string)
	onPause           func(dagId string)
	onBackfill        func(dagId string)
	onBackfillCancel  func(id int)
	onBackfillPause   func(id int)
	onBackfillUnpause func(id int)
	onMonitorWindow   func(delta int)
	onMonitorRefresh  func()
}

func NewKeyBindings(app *tview.Application, l *layout.MainLayout, s *state.Store) *KeyBindings {
	return &KeyBindings{app: app, layout: l, store: s}
}

func (kb *KeyBindings) SetOnRefresh(fn func())            { kb.onRefresh = fn }
func (kb *KeyBindings) SetOnTrigger(fn func(string))      { kb.onTrigger = fn }
func (kb *KeyBindings) SetOnPause(fn func(string))        { kb.onPause = fn }
func (kb *KeyBindings) SetOnBackfill(fn func(string))     { kb.onBackfill = fn }
func (kb *KeyBindings) SetOnBackfillCancel(fn func(int))  { kb.onBackfillCancel = fn }
func (kb *KeyBindings) SetOnBackfillPause(fn func(int))   { kb.onBackfillPause = fn }
func (kb *KeyBindings) SetOnBackfillUnpause(fn func(int)) { kb.onBackfillUnpause = fn }
func (kb *KeyBindings) SetOnMonitorWindow(fn func(int))   { kb.onMonitorWindow = fn }
func (kb *KeyBindings) SetOnMonitorRefresh(fn func())     { kb.onMonitorRefresh = fn }

// Install registers the global input capture on the tview application.
func (kb *KeyBindings) Install() {
	kb.app.SetInputCapture(kb.handle)
}

func (kb *KeyBindings) handle(event *tcell.EventKey) *tcell.EventKey {
	tStart := time.Now()
	defer func() {
		if d := time.Since(tStart); d > 30*time.Millisecond {
			debugutil.Tag("FZ-key", "handle SLOW key=%v rune=%q elapsed=%v",
				event.Key(), event.Rune(), d)
		}
	}()
	debugutil.Tag("FZ-key", "handle key=%v rune=%q", event.Key(), event.Rune())

	if kb.layout.IsSearchVisible() {
		switch event.Key() {
		case tcell.KeyCtrlC:
			kb.app.Stop()
			return nil
		case tcell.KeyEsc:
			// Cancel the search: clear the filter and close the overlay.
			kb.layout.DagList().Search("")
			kb.layout.HideSearch()
			return nil
		default:
			// Let the focused search input field handle typing/Enter.
			return event
		}
	}

	if kb.layout.IsModalVisible() {
		switch event.Key() {
		case tcell.KeyCtrlC:
			kb.app.Stop()
			return nil
		case tcell.KeyEsc:
			kb.layout.DismissModal()
			return nil
		default:
			return event
		}
	}

	// Special keys
	switch event.Key() {
	case tcell.KeyCtrlC:
		kb.app.Stop()
		return nil
	case tcell.KeyF5:
		if kb.onRefresh != nil {
			kb.onRefresh()
		}
		return nil
	case tcell.KeyEsc:
		// Inside the tab area Esc unwinds the drill-down chain; elsewhere it
		// parks focus back on the DAG list.
		if kb.inTabArea() {
			switch kb.store.ActiveTab() {
			case "logs":
				kb.switchToTab("tasks")
				return nil
			case "tasks":
				kb.switchToTab("runs")
				return nil
			}
		}
		kb.app.SetFocus(kb.layout.DagList())
		return nil
	case tcell.KeyTab:
		kb.cycleFocus(1)
		return nil
	case tcell.KeyBacktab:
		kb.cycleFocus(-1)
		return nil
	case tcell.KeyLeft:
		// Only Shift+Left cycles tabs; a bare Left belongs to the focused widget.
		if event.Modifiers()&tcell.ModShift != 0 {
			kb.cycleTab(-1)
			return nil
		}
		return event
	case tcell.KeyRight:
		if event.Modifiers()&tcell.ModShift != 0 {
			kb.cycleTab(1)
			return nil
		}
		return event
	}

	// Rune keys
	switch event.Rune() {
	// Tab switching (0-9)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if name, ok := tabForRune(event.Rune()); ok {
			kb.switchToTab(name)
			return nil
		}

	// Tab cycling; rune fallback for terminals that swallow Shift+arrows.
	case '<':
		kb.cycleTab(-1)
		return nil
	case '>':
		kb.cycleTab(1)
		return nil

	// Tab aliases and toggles
	case 'B':
		kb.layout.SwitchTab("backfills")
		kb.store.SetActiveTab("backfills")
		kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
		return nil
	case 'g':
		// Consumed only where it toggles a view; elsewhere it stays tview's jump-to-top.
		switch kb.store.ActiveTab() {
		case "tasks":
			kb.store.SetGanttMode(!kb.store.GanttMode())
		case "lineage":
			on := !kb.layout.Lineage().IsGraphMode()
			kb.layout.Lineage().SetGraphMode(on)
			if on {
				dagId := kb.store.SelectedDAG()
				runId := kb.store.SelectedRun()
				stateByTask := map[string]string{}
				for _, ti := range kb.store.GetTaskInstances(dagId, runId) {
					stateByTask[ti.TaskId] = ti.State
				}
				kb.layout.Lineage().UpdateGraph(stateByTask)
			}
		default:
			return event
		}
		return nil

	// DAG filters
	case 'a':
		kb.layout.KpiBar().SelectFilter("active")
		return nil
	case 'A':
		kb.layout.KpiBar().SelectFilter("all")
		return nil
	case 'f':
		kb.layout.KpiBar().SelectFilter("failed")
		return nil

	// Focus movement
	case 'd':
		kb.app.SetFocus(kb.layout.DagList())
		return nil
	case 'i':
		kb.app.SetFocus(kb.layout.DagInfo().Meta())
		return nil

	// Cluster panel: focus, or toggle pool view when already focused
	case 'o':
		if kb.app.GetFocus() == kb.layout.ClusterInfo() {
			kb.layout.ClusterInfo().ToggleView()
		} else {
			kb.app.SetFocus(kb.layout.ClusterInfo())
		}
		return nil

	// DAG actions
	case 't':
		if dagId := kb.store.SelectedDAG(); dagId != "" && kb.onTrigger != nil {
			kb.onTrigger(dagId)
		}
		return nil
	case 'p':
		if kb.store.ActiveTab() == "backfills" {
			if id := kb.store.SelectedBackfill(); id > 0 && kb.onBackfillPause != nil {
				kb.onBackfillPause(id)
			}
			return nil
		}
		if dagId := kb.store.SelectedDAG(); dagId != "" && kb.onPause != nil {
			kb.onPause(dagId)
		}
		return nil
	case 'b':
		if dagId := kb.store.SelectedDAG(); dagId != "" && kb.onBackfill != nil {
			kb.onBackfill(dagId)
		}
		return nil
	case 'c':
		if kb.store.ActiveTab() == "backfills" {
			if id := kb.store.SelectedBackfill(); id > 0 && kb.onBackfillCancel != nil {
				kb.onBackfillCancel(id)
			}
		}
		return nil
	case 'u':
		if kb.store.ActiveTab() == "backfills" {
			if id := kb.store.SelectedBackfill(); id > 0 && kb.onBackfillUnpause != nil {
				kb.onBackfillUnpause(id)
			}
		}
		return nil

	// Monitor tab window / refresh (only while the monitor tab is active)
	case '[':
		if kb.store.ActiveTab() == "monitor" && kb.onMonitorWindow != nil {
			kb.onMonitorWindow(-1)
		}
		return nil
	case ']':
		if kb.store.ActiveTab() == "monitor" && kb.onMonitorWindow != nil {
			kb.onMonitorWindow(1)
		}
		return nil
	case 'r':
		if kb.store.ActiveTab() == "monitor" && kb.onMonitorRefresh != nil {
			kb.onMonitorRefresh()
			return nil
		}
		return event

	// Search
	case '/':
		kb.layout.ShowSearch()
		return nil

	// Help
	case '?':
		kb.layout.ShowHelp()
		kb.store.SetActiveTab("help")
		return nil
	}

	return event
}

// focusRing is the ordered set of panels Tab / Shift+Tab cycle through, so every
// panel (and thus every feature) is reachable by keyboard alone. The active
// bottom tab is resolved dynamically since it changes with the selected tab.
func (kb *KeyBindings) focusRing() []tview.Primitive {
	return []tview.Primitive{
		kb.layout.DagList(),
		kb.layout.KpiBar().FocusPrimitive(),
		kb.layout.DagInfo().Meta(),
		kb.layout.ClusterInfo(),
		kb.layout.ActiveTabPrimitive(),
	}
}

// inTabArea reports whether focus sits in the bottom tab area rather than on
// one of the top panels (the ring's last stop is the active tab).
func (kb *KeyBindings) inTabArea() bool {
	focused := kb.app.GetFocus()
	if focused == nil {
		return false
	}
	ring := kb.focusRing()
	for _, p := range ring[:len(ring)-1] {
		if p == focused {
			return false
		}
	}
	return true
}

// cycleFocus moves focus by delta (+1 Tab, -1 Shift+Tab) around focusRing. If the
// current focus isn't in the ring (e.g. an overlay), it lands on the first stop.
func (kb *KeyBindings) cycleFocus(delta int) {
	ring := kb.focusRing()
	focused := kb.app.GetFocus()
	idx := 0
	found := false
	for i, p := range ring {
		if p == focused {
			idx = i
			found = true
			break
		}
	}
	n := len(ring)
	if found {
		idx = ((idx+delta)%n + n) % n
	}
	kb.app.SetFocus(ring[idx])
}

// cycleTabNames is tabNames minus help, which is reachable via '?' only.
var cycleTabNames = func() []string {
	out := make([]string, 0, len(tabNames))
	for _, t := range tabNames {
		if t.name != "help" {
			out = append(out, t.name)
		}
	}
	return out
}()

// switchToTab activates a bottom tab and moves focus into it.
func (kb *KeyBindings) switchToTab(name string) {
	kb.layout.SwitchTab(name)
	kb.store.SetActiveTab(name)
	kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
}

func (kb *KeyBindings) cycleTab(delta int) {
	cur := kb.store.ActiveTab()
	idx := 0
	for i, name := range cycleTabNames {
		if name == cur {
			idx = i
			break
		}
	}
	n := len(cycleTabNames)
	idx = ((idx+delta)%n + n) % n
	next := cycleTabNames[idx]
	kb.layout.SwitchTab(next)
	kb.store.SetActiveTab(next)
	kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
}
