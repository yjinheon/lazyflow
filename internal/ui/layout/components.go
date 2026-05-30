package layout

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

type KpiBar struct {
	root  *tview.Flex
	cards map[string]*tview.TextView

	activeDAGs   int
	inactiveDAGs int
	runningRuns  int
	successRuns  int
	failedRuns   int
	runScope     string
}

func NewKpiBar() *KpiBar {
	k := &KpiBar{
		root:     tview.NewFlex().SetDirection(tview.FlexColumn),
		cards:    make(map[string]*tview.TextView),
		runScope: "select a DAG",
	}
	k.addCard("active", "Active", tcell.ColorGreen)
	k.addCard("inactive", "Inactive", tcell.ColorYellow)
	k.addCard("running", "Running", tcell.ColorBlue)
	k.addCard("success", "Success", tcell.ColorGreen)
	k.addCard("failed", "Failed", tcell.ColorRed)
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
	k.root.AddItem(card, 0, 1, false)
}

func (k *KpiBar) SetDAGCounts(active, inactive int) {
	k.activeDAGs = active
	k.inactiveDAGs = inactive
	k.refresh()
}

func (k *KpiBar) SetRunCounts(scope string, running, success, failed int) {
	if scope == "" {
		scope = "select a DAG"
	}
	k.runScope = scope
	k.runningRuns = running
	k.successRuns = success
	k.failedRuns = failed
	k.refresh()
}

func (k *KpiBar) refresh() {
	k.setCard("active", "DAGs", k.activeDAGs, "green")
	k.setCard("inactive", "DAGs", k.inactiveDAGs, "yellow")
	k.setCard("running", k.runScope, k.runningRuns, "blue")
	k.setCard("success", k.runScope, k.successRuns, "green")
	k.setCard("failed", k.runScope, k.failedRuns, "red")
}

func (k *KpiBar) setCard(key, scope string, value int, color string) {
	card, ok := k.cards[key]
	if !ok {
		return
	}
	card.SetText(fmt.Sprintf("[%s::b]%d[-::-]\n[gray]%s[-]", color, value, truncateKpiScope(scope, 20)))
}

func truncateKpiScope(scope string, max int) string {
	if len(scope) <= max {
		return scope
	}
	if max <= 3 {
		return scope[:max]
	}
	return scope[:max-3] + "..."
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

type StatusBar struct {
	*tview.TextView
}

func NewStatusBar() *StatusBar {
	s := &StatusBar{
		TextView: tview.NewTextView(),
	}
	s.SetDynamicColors(true)
	s.SetText(" [green]Ready[-]")
	return s
}

func (s *StatusBar) SetStatus(msg string) {
	s.SetText(" " + msg)
}

func (s *StatusBar) SetInfo(dagId, runId, taskId string) {
	parts := make([]string, 0, 4)
	if dagId != "" {
		parts = append(parts, fmt.Sprintf("DAG:[yellow]%s[-]", dagId))
	}
	if runId != "" {
		display := runId
		if len(display) > 30 {
			display = display[:27] + "..."
		}
		parts = append(parts, fmt.Sprintf("Run:[yellow]%s[-]", display))
	}
	if taskId != "" {
		parts = append(parts, fmt.Sprintf("Task:[yellow]%s[-]", taskId))
	}
	if len(parts) == 0 {
		s.SetText(" [green]Ready[-]")
		return
	}
	s.SetText(" " + strings.Join(parts, " | "))
}

func (s *StatusBar) SetError(msg string) {
	s.SetText(fmt.Sprintf(" [red]Error: %s[-]", msg))
}

func (s *StatusBar) Root() *tview.TextView {
	return s.TextView
}
