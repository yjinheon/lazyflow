package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
	{'5', "config"},
	{'6', "connections"},
	{'7', "variables"},
	{'8', "monitor"},
	{'9', "lineage"},
}

type KeyBindings struct {
	app    *tview.Application
	layout *layout.MainLayout
	store  *state.Store

	onRefresh  func()
	onTrigger  func(dagId string)
	onPause    func(dagId string)
	onBackfill func(dagId string)
}

func NewKeyBindings(app *tview.Application, l *layout.MainLayout, s *state.Store) *KeyBindings {
	return &KeyBindings{app: app, layout: l, store: s}
}

func (kb *KeyBindings) SetOnRefresh(fn func())        { kb.onRefresh = fn }
func (kb *KeyBindings) SetOnTrigger(fn func(string))  { kb.onTrigger = fn }
func (kb *KeyBindings) SetOnPause(fn func(string))    { kb.onPause = fn }
func (kb *KeyBindings) SetOnBackfill(fn func(string)) { kb.onBackfill = fn }

// Install registers the global input capture on the tview application.
func (kb *KeyBindings) Install() {
	kb.app.SetInputCapture(kb.handle)
}

func (kb *KeyBindings) handle(event *tcell.EventKey) *tcell.EventKey {
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
	}

	// Rune keys
	switch event.Rune() {
	// Tab switching (1-9)
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		for _, t := range tabNames {
			if event.Rune() == t.key {
				kb.layout.SwitchTab(t.name)
				kb.store.SetActiveTab(t.name)
				kb.app.SetFocus(kb.layout.ActiveTabPrimitive())
				return nil
			}
		}

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
		if dagId := kb.store.SelectedDAG(); dagId != "" && kb.onPause != nil {
			kb.onPause(dagId)
		}
		return nil
	case 'b':
		if dagId := kb.store.SelectedDAG(); dagId != "" && kb.onBackfill != nil {
			kb.onBackfill(dagId)
		}
		return nil

	// Search
	case '/':
		kb.layout.ShowSearch()
		return nil

	// Help
	case '?':
		kb.layout.ShowHelp()
		return nil
	}

	return event
}
