package layout

import (
	"fmt"
	"strings"

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
		"Vars": "variables", "Config": "config",
	}
	for _, tab := range tabLabels {
		if tab.name == "Conns" {
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
