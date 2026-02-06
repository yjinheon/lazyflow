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

	onRefresh func()
}

func NewKeyBindings(app *tview.Application, l *layout.MainLayout, s *state.Store) *KeyBindings {
	return &KeyBindings{app: app, layout: l, store: s}
}

// SetOnRefresh registers a callback for F5 manual refresh.
func (kb *KeyBindings) SetOnRefresh(fn func()) {
	kb.onRefresh = fn
}

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
		// Cycle focus: dagList → runs/tasks table → back
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
