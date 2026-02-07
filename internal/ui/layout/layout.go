// internal/ui/layout/layout.go
package layout

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/views"
)

// Layout constants define the fixed proportions of the UI
const (
	HeaderHeight       = 3
	TopSectionRatio    = 40
	BottomSectionRatio = 60

	DagListRatio     = 30
	DagInfoRatio     = 40
	ClusterInfoRatio = 30
)

// MainLayout manages the global grid structure
type MainLayout struct {
	app  *tview.Application
	root *tview.Flex

	header      *Header
	dagList     *views.DagListView
	dagInfo     *views.DagInfoView
	clusterInfo *views.ClusterInfoView
	tabBar      *TabBar
	statusBar   *StatusBar

	// Tab views
	runsView        *views.RunsView
	tasksView       *views.TasksView
	logsView        *views.LogsView
	codeView        *views.CodeView
	configView      *views.ConfigView
	connectionsView *views.ConnectionsView
	variablesView   *views.VariablesView
	monitorView     *views.MonitorView
	lineageView     *views.LineageView

	tabContent *tview.Pages
}

func NewMainLayout(app *tview.Application) *MainLayout {
	m := &MainLayout{
		app:         app,
		header:      NewHeader(),
		dagList:     views.NewDagListView(),
		dagInfo:     views.NewDagInfoView(),
		clusterInfo: views.NewClusterInfoView(),
		tabBar:      NewTabBar(),
		statusBar:   NewStatusBar(),

		runsView:        views.NewRunsView(),
		tasksView:       views.NewTasksView(),
		logsView:        views.NewLogsView(),
		codeView:        views.NewCodeView(),
		configView:      views.NewConfigView(),
		connectionsView: views.NewConnectionsView(),
		variablesView:   views.NewVariablesView(),
		monitorView:     views.NewMonitorView(),
		lineageView:     views.NewLineageView(),

		tabContent: tview.NewPages(),
	}
	m.build()
	return m
}

func (m *MainLayout) build() {
	topSection := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(m.dagList.Root(), 0, DagListRatio, true).
		AddItem(m.dagInfo.Root(), 0, DagInfoRatio, false).
		AddItem(m.clusterInfo.Root(), 0, ClusterInfoRatio, false)

	m.registerTabs()

	m.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(m.header.Root(), HeaderHeight, 0, false).
		AddItem(topSection, 0, TopSectionRatio, true).
		AddItem(m.tabBar.Root(), 1, 0, false).
		AddItem(m.tabContent, 0, BottomSectionRatio, false).
		AddItem(m.statusBar.Root(), 1, 0, false)
}

func (m *MainLayout) registerTabs() {
	m.tabContent.AddPage("runs", m.runsView.Root(), true, true)
	m.tabContent.AddPage("tasks", m.tasksView.Root(), true, false)
	m.tabContent.AddPage("logs", m.logsView.Root(), true, false)
	m.tabContent.AddPage("code", m.codeView.Root(), true, false)
	m.tabContent.AddPage("config", m.configView.Root(), true, false)
	m.tabContent.AddPage("connections", m.connectionsView.Root(), true, false)
	m.tabContent.AddPage("variables", m.variablesView.Root(), true, false)
	m.tabContent.AddPage("monitor", m.monitorView.Root(), true, false)
	m.tabContent.AddPage("lineage", m.lineageView.Root(), true, false)
}

func (m *MainLayout) SwitchTab(name string) {
	m.tabContent.SwitchToPage(name)
	m.tabBar.SetActive(name)
}

// ShowHelp displays a help modal with keybinding reference.
func (m *MainLayout) ShowHelp() {
	helpText := `[yellow::b]Keybindings[-::-]

[white]Navigation[-]
  j/k        Up / Down
  Enter      Select
  Esc        Back to DAG list

[white]Tabs[-]
  1-9        Switch tab

[white]DAG Filters[-]
  a          Active DAGs
  A          All DAGs
  f          Failed DAGs

[white]Focus[-]
  d          DAG list
  i          DAG info

[white]General[-]
  F5         Refresh
  /          Search
  ?          Help
  Ctrl+C     Quit
`
	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(_ int, _ string) {
			m.app.SetRoot(m.root, true)
			m.app.SetFocus(m.dagList)
		})
	overlay := tview.NewPages().
		AddPage("main", m.root, true, true).
		AddPage("help", modal, true, true)
	m.app.SetRoot(overlay, true)
}

// ShowSearch displays a search input overlay for filtering DAGs.
func (m *MainLayout) ShowSearch() {
	input := tview.NewInputField().
		SetLabel(" / ").
		SetFieldWidth(30).
		SetLabelColor(tcell.ColorYellow)
	input.SetBorder(true).SetTitle(" Search DAGs ").SetBorderColor(tcell.ColorBlue)

	input.SetChangedFunc(func(text string) {
		m.dagList.Search(text)
	})

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			m.dagList.Search("")
		}
		m.app.SetRoot(m.root, true)
		m.app.SetFocus(m.dagList)
	})

	overlay := tview.NewPages().
		AddPage("main", m.root, true, true).
		AddPage("search", centerPrimitive(input, 40, 3), true, true)
	m.app.SetRoot(overlay, true)
	m.app.SetFocus(input)
}

func centerPrimitive(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 0, true).
			AddItem(nil, 0, 1, false),
			width, 0, true).
		AddItem(nil, 0, 1, false)
}

// ActiveTabPrimitive returns the currently visible bottom panel widget.
func (m *MainLayout) ActiveTabPrimitive() tview.Primitive {
	name, _ := m.tabContent.GetFrontPage()
	switch name {
	case "runs":
		return m.runsView
	case "tasks":
		return m.tasksView
	case "logs":
		return m.logsView
	case "code":
		return m.codeView
	case "config":
		return m.configView
	case "connections":
		return m.connectionsView
	case "variables":
		return m.variablesView
	case "monitor":
		return m.monitorView
	case "lineage":
		return m.lineageView
	default:
		return m.runsView
	}
}

func (m *MainLayout) Root() *tview.Flex                      { return m.root }
func (m *MainLayout) DagList() *views.DagListView            { return m.dagList }
func (m *MainLayout) DagInfo() *views.DagInfoView            { return m.dagInfo }
func (m *MainLayout) ClusterInfo() *views.ClusterInfoView    { return m.clusterInfo }
func (m *MainLayout) Runs() *views.RunsView                  { return m.runsView }
func (m *MainLayout) Tasks() *views.TasksView                { return m.tasksView }
func (m *MainLayout) Logs() *views.LogsView                  { return m.logsView }
func (m *MainLayout) Code() *views.CodeView                  { return m.codeView }
func (m *MainLayout) Config() *views.ConfigView              { return m.configView }
func (m *MainLayout) Connections() *views.ConnectionsView    { return m.connectionsView }
func (m *MainLayout) Variables() *views.VariablesView        { return m.variablesView }
func (m *MainLayout) Monitor() *views.MonitorView            { return m.monitorView }
func (m *MainLayout) Lineage() *views.LineageView            { return m.lineageView }
func (m *MainLayout) StatusBar() *StatusBar                  { return m.statusBar }
func (m *MainLayout) Header() *Header                        { return m.header }
