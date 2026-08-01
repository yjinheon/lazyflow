package layout

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

// ---------- Header ----------

type Header struct {
	*tview.TextView
}

func NewHeader() *Header {
	h := &Header{
		TextView: tview.NewTextView(),
	}
	h.SetDynamicColors(true)
	h.SetText(" [::b]lazyflow[::-] v0.1.0 | ?: Help | /: Search")
	return h
}

func (h *Header) SetConnection(host string, ok bool) {
	h.SetInfo(host, ok, 0)
}

func (h *Header) SetInfo(host string, connected bool, dagCount int) {
	status := fmt.Sprintf("[green]%s[-]", host)
	if !connected {
		status = fmt.Sprintf("[red]%s (disconnected)[-]", host)
	}
	extra := ""
	if dagCount > 0 {
		extra = fmt.Sprintf(" | DAGs: [yellow]%d[-]", dagCount)
	}
	h.SetText(fmt.Sprintf(" [::b]lazyflow[::-] v0.1.0 | %s%s | [gray]?[-]:Help [gray]/[-]:Search", status, extra))
}

func (h *Header) Root() *tview.TextView {
	return h.TextView
}

// ---------- KPI Bar ----------

// KpiBar is the cluster-wide overview and DAG-filter tab strip. Its cards show
// counts and selecting one filters the DAG list:
//   - all: every DAG
//   - active/inactive: paused vs unpaused DAGs
//   - running/success/failed: DAGs bucketed by their latest run's state
type KpiBar struct {
	root       *tview.Flex
	cards      map[string]*tview.TextView
	order      []string
	titles     map[string]string
	active     string
	onSelected func(string)

	activeDAGs   int
	inactiveDAGs int
	runningDAGs  int
	successDAGs  int
	failedDAGs   int
}

func NewKpiBar() *KpiBar {
	k := &KpiBar{
		root:   tview.NewFlex().SetDirection(tview.FlexColumn),
		cards:  make(map[string]*tview.TextView),
		titles: make(map[string]string),
		active: "all",
	}
	k.addCard("all", "All", theme.ActiveTheme().PrimaryText)
	k.addCard("active", "Active", theme.ActiveTheme().StatusSuccess)
	k.addCard("paused", "Paused", theme.ActiveTheme().StatusPaused)
	k.addCard("running", "Running", theme.ActiveTheme().StatusRunning)
	k.addCard("success", "Success", theme.ActiveTheme().StatusSuccess)
	k.addCard("failed", "Failed", theme.ActiveTheme().StatusFailed)
	k.refresh()
	return k
}

func (k *KpiBar) addCard(key, title string, color tcell.Color) {
	card := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	card.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleColor(color).
		SetBorderColor(color)
	k.cards[key] = card
	k.order = append(k.order, key)
	k.titles[key] = title
	card.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			k.selectRelative(-1)
			return nil
		case tcell.KeyRight:
			k.selectRelative(1)
			return nil
		case tcell.KeyEnter:
			k.selectFilter(k.active)
			return nil
		}
		return event
	})
	k.root.AddItem(card, 0, 1, key == "all")
}

// SetOnSelected registers the callback invoked when a filter tab is selected.
func (k *KpiBar) SetOnSelected(fn func(string)) { k.onSelected = fn }

// ActiveFilter returns the selected DAG filter.
func (k *KpiBar) ActiveFilter() string { return k.active }

// FocusPrimitive returns the card that currently represents the filter strip
// in the application's focus ring.
func (k *KpiBar) FocusPrimitive() tview.Primitive { return k.cards["all"] }

// SelectFilter selects a card programmatically (for keyboard aliases).
func (k *KpiBar) SelectFilter(filter string) { k.selectFilter(filter) }

func (k *KpiBar) selectRelative(delta int) {
	idx := 0
	for i, key := range k.order {
		if key == k.active {
			idx = i
			break
		}
	}
	k.selectFilter(k.order[(idx+delta+len(k.order))%len(k.order)])
}

func (k *KpiBar) selectFilter(filter string) {
	if _, ok := k.cards[filter]; !ok {
		return
	}
	k.active = filter
	k.refresh()
	if k.onSelected != nil {
		k.onSelected(filter)
	}
}

func (k *KpiBar) SetDAGCounts(active, inactive int) {
	k.activeDAGs = active
	k.inactiveDAGs = inactive
	k.refresh()
}

// SetDAGStateCounts sets the cluster-wide DAG counts bucketed by latest run
// state. Each DAG contributes to at most one of running/success/failed.
func (k *KpiBar) SetDAGStateCounts(running, success, failed int) {
	k.runningDAGs = running
	k.successDAGs = success
	k.failedDAGs = failed
	k.refresh()
}

func (k *KpiBar) refresh() {
	k.setCard("all", k.activeDAGs+k.inactiveDAGs, "white")
	k.setCard("active", k.activeDAGs, "green")
	k.setCard("paused", k.inactiveDAGs, "yellow")
	k.setCard("running", k.runningDAGs, "blue")
	k.setCard("success", k.successDAGs, "green")
	k.setCard("failed", k.failedDAGs, "red")
	for key, card := range k.cards {
		title := fmt.Sprintf(" %s ", k.titles[key])
		if key == k.active {
			title = fmt.Sprintf(" [::b]%s[::-] ", k.titles[key])
			card.SetBorderColor(theme.ActiveTheme().BorderFocused)
		} else {
			card.SetBorderColor(theme.ActiveTheme().BorderColor)
		}
		card.SetTitle(title)
	}
}

func (k *KpiBar) setCard(key string, value int, color string) {
	card, ok := k.cards[key]
	if !ok {
		return
	}
	card.SetText(fmt.Sprintf("[%s::b]%d[-::-]\n[gray]DAGs[-]", color, value))
}

func (k *KpiBar) Root() *tview.Flex {
	return k.root
}

// ---------- TabBar ----------

var tabLabels = []struct {
	key  string
	name string
}{
	{"1", "Runs"},
	{"2", "Tasks"},
	{"3", "Logs"},
	{"4", "Code"},
	{"5", "Lineage"},
	{"6", "Monitor"},
	{"7", "Backfills"},
	{"8", "Conns"},
	{"9", "Vars"},
	{"0", "Config"},
	{"?", "Help"},
}

type TabBar struct {
	*tview.TextView
	active string
}

func NewTabBar() *TabBar {
	t := &TabBar{
		TextView: tview.NewTextView(),
		active:   "runs",
	}
	t.SetDynamicColors(true)
	t.refresh()
	return t
}

func (t *TabBar) SetActive(name string) {
	t.active = name
	t.refresh()
}

// Active returns the currently highlighted tab id.
func (t *TabBar) Active() string { return t.active }

func (t *TabBar) refresh() {
	var text strings.Builder
	text.WriteString(" ")
	nameMap := map[string]string{
		"Runs": "runs", "Tasks": "tasks", "Logs": "logs",
		"Code": "code", "Lineage": "lineage", "Monitor": "monitor",
		"Backfills": "backfills", "Conns": "connections",
		"Vars": "variables", "Config": "config", "Help": "help",
	}
	for _, tab := range tabLabels {
		if tab.name == "Conns" || tab.name == "Help" {
			text.WriteString("[gray]│[-] ")
		}
		tabID := nameMap[tab.name]
		if tabID == t.active {
			text.WriteString(fmt.Sprintf("[black:white:b] %s:%s [-:-:-] ", tab.key, tab.name))
		} else {
			text.WriteString(fmt.Sprintf("[white:-:-] %s:%s [-:-:-] ", tab.key, tab.name))
		}
	}
	t.SetText(text.String())
}

func (t *TabBar) Root() *tview.TextView {
	return t.TextView
}

// ---------- StatusBar ----------

// StatusBar renders selection context (or a transient message) on the left and
// context-sensitive key hints on the right. The two are kept apart so an action
// result never wipes the hints.
type StatusBar struct {
	*tview.TextView

	dagID, runID, taskID string
	flash                string // transient status/error; outranks the selection info
	tab                  string
	hasDAG               bool
}

func NewStatusBar() *StatusBar {
	s := &StatusBar{TextView: tview.NewTextView(), tab: "runs"}
	s.SetDynamicColors(true)
	return s
}

// Draw recomposes the line for the current width before delegating, so the key
// hints keep their space and the selection info is what shrinks.
func (s *StatusBar) Draw(screen tcell.Screen) {
	_, _, w, _ := s.GetRect()
	s.SetText(s.compose(w))
	s.TextView.Draw(screen)
}

const statusSep = "  │  "

func (s *StatusBar) compose(width int) string {
	hints, hintsW := buildHints(s.tab, s.hasDAG)

	// A flash carries caller-supplied markup of unknown width; let it clip.
	left := s.flash
	if left == "" {
		left = s.infoSegment(width - 1 - len(statusSep) - hintsW)
	}
	return " " + left + statusSep + hints
}

// infoSegment renders as much selection context as budget allows, shrinking the
// long, low-signal run id before dropping anything.
func (s *StatusBar) infoSegment(budget int) string {
	if s.dagID == "" && s.runID == "" && s.taskID == "" {
		return "[green]Ready[-]"
	}

	var segs []seg
	if s.dagID != "" {
		segs = append(segs, seg{"DAG", s.dagID})
	}
	if s.runID != "" {
		segs = append(segs, seg{"Run", s.runID})
	}
	if s.taskID != "" {
		segs = append(segs, seg{"Task", s.taskID})
	}

	// Shrink the run id first; drop it entirely if even that does not fit.
	for attempt := 0; attempt < len(segs); attempt++ {
		out, w := renderSegs(segs)
		if w <= budget || budget <= 0 {
			return out
		}
		if i := indexOfLabel(segs, "Run"); i >= 0 {
			room := len(segs[i].value) - (w - budget)
			if room >= 8 {
				segs[i].value = segs[i].value[:room-1] + "…"
				continue
			}
			segs = append(segs[:i], segs[i+1:]...)
			continue
		}
		break
	}
	out, _ := renderSegs(segs)
	return out
}

// seg is one "Label:value" chunk of the status bar info section.
type seg struct{ label, value string }

func indexOfLabel(segs []seg, label string) int {
	for i, s := range segs {
		if s.label == label {
			return i
		}
	}
	return -1
}

func renderSegs(segs []seg) (string, int) {
	parts := make([]string, 0, len(segs))
	width := 0
	for i, sg := range segs {
		parts = append(parts, fmt.Sprintf("%s:[yellow]%s[-]", sg.label, tview.Escape(sg.value)))
		width += len(sg.label) + 1 + len(sg.value)
		if i > 0 {
			width += 3 // " | "
		}
	}
	return strings.Join(parts, " | "), width
}

func (s *StatusBar) SetStatus(msg string) {
	s.flash = msg
}

func (s *StatusBar) SetError(msg string) {
	s.flash = fmt.Sprintf("[red]Error: %s[-]", msg)
}

func (s *StatusBar) SetInfo(dagId, runId, taskId string) {
	s.dagID, s.runID, s.taskID = dagId, runId, taskId
	s.flash = "" // a new selection supersedes the previous action result
}

// SetContext refreshes the key hints for the active tab and selection state.
func (s *StatusBar) SetContext(tab string, hasDAG bool) {
	s.tab, s.hasDAG = tab, hasDAG
}

// buildHints lists the keys that actually do something right now, returning the
// markup and its visible width. Keep it short: it shares one line with the info.
func buildHints(tab string, hasDAG bool) (string, int) {
	var keys [][2]string // key, label
	if hasDAG {
		keys = append(keys, [2]string{"t", "trigger"}, [2]string{"p", "pause"}, [2]string{"b", "backfill"})
	}
	switch tab {
	case "backfills":
		keys = append(keys, [2]string{"c", "cancel"}, [2]string{"u", "unpause"})
	case "monitor":
		keys = append(keys, [2]string{"[", "prev"}, [2]string{"]", "next"}, [2]string{"r", "refresh"})
	case "tasks":
		keys = append(keys, [2]string{"g", "gantt"})
	case "lineage":
		keys = append(keys, [2]string{"g", "graph"})
	}
	if len(keys) == 0 {
		keys = append(keys, [2]string{"Enter", "select a DAG"})
	}

	parts := make([]string, 0, len(keys))
	width := 0
	for i, k := range keys {
		parts = append(parts, fmt.Sprintf("[yellow]%s[-][gray]:%s[-]", tview.Escape(k[0]), k[1]))
		width += len(k[0]) + 1 + len(k[1])
		if i > 0 {
			width += 2
		}
	}
	return strings.Join(parts, "  "), width
}

// Root returns the StatusBar itself, not the embedded TextView: the wrapper's
// Draw is what composes the line for the current width.
func (s *StatusBar) Root() tview.Primitive {
	return s
}
