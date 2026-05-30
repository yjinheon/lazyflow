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

	if kb.layout.IsExecutionVisible() {
		switch event.Key() {
		case tcell.KeyCtrlC:
			kb.app.Stop()
			return nil
		case tcell.KeyEsc:
			kb.layout.HideExecution()
			return nil
		default:
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
		// Reset focus to DAG list
		kb.app.SetFocus(kb.layout.DagList())
		return nil
	case tcell.KeyTab:
		focused := kb.app.GetFocus()
		if focused == kb.layout.DagList() {
			kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
		} else {
			kb.app.SetFocus(kb.layout.DagList())
		}
		return nil
	case tcell.KeyLeft:
		kb.cycleTab(-1)
		return nil
	case tcell.KeyRight:
		kb.cycleTab(1)
		return nil
	}

	// Rune keys
	switch event.Rune() {
	// Tab switching (0-9)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		for _, t := range tabNames {
			if event.Rune() == t.key {
				kb.layout.SwitchTab(t.name)
				kb.store.SetActiveTab(t.name)
				kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
				return nil
			}
		}

	// Tab aliases and toggles
	case 'B':
		kb.layout.SwitchTab("backfills")
		kb.store.SetActiveTab("backfills")
		kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
		return nil
	case 'g':
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
		}
		return nil

	// DAG filters
	case 'a':
		kb.layout.DagList().SetFilter("active")
		return nil
	case 'A':
		kb.layout.DagList().SetFilter("all")
		return nil
	case 'f':
		kb.layout.DagList().SetFilter("failed")
		return nil

	// Focus movement
	case 'd':
		kb.app.SetFocus(kb.layout.DagList())
		return nil
	case 'i':
		kb.app.SetFocus(kb.layout.DagInfo())
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

func (kb *KeyBindings) cycleTab(delta int) {
	cur := kb.store.ActiveTab()
	idx := 0
	for i, t := range tabNames {
		if t.name == cur {
			idx = i
			break
		}
	}
	n := len(tabNames)
	idx = ((idx+delta)%n + n) % n
	next := tabNames[idx].name
	kb.layout.SwitchTab(next)
	kb.store.SetActiveTab(next)
	kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
}
