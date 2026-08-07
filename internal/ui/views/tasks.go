package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// TasksView is the tasks tab. It shows the DAG's task definitions when no run
// is selected, the run dashboard once one is, and the Gantt chart under `g`.
type TasksView struct {
	*tview.Pages

	table *tview.Table
	gantt *GanttView
	run   *ExecutionView

	taskDefinitions []models.Task
	activeTaskId    string
	hasRun          bool
	ganttMode       bool
	onSelected      func(taskId string)
}

const (
	tasksPageDefs  = "definitions"
	tasksPageRun   = "run"
	tasksPageGantt = "gantt"
)

func NewTasksView() *TasksView {
	v := &TasksView{
		Pages: tview.NewPages(),
		table: tview.NewTable(),
		gantt: NewGanttView(),
		run:   NewExecutionView(),
	}
	v.setupTable()
	v.run.SetOnTaskSelected(v.selectTask)
	v.AddPage(tasksPageDefs, v.table, true, true)
	v.AddPage(tasksPageRun, v.run.Root(), true, false)
	v.AddPage(tasksPageGantt, v.gantt, true, false)
	return v
}

func (v *TasksView) setupTable() {
	v.table.SetBorder(true).SetTitle(" DAG Tasks ")
	v.table.SetSelectable(false, false)
	v.table.SetFixed(1, 0)
	v.table.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.ActiveTheme().TableSelected).
		Foreground(theme.ActiveTheme().PrimaryText).
		Attributes(tcell.AttrBold))
	v.table.SetFocusFunc(func() { v.table.SetBorderColor(theme.ActiveTheme().BorderFocused) })
	v.table.SetBlurFunc(func() { v.table.SetBorderColor(theme.ActiveTheme().BorderColor) })

	v.renderHeaders([]string{"Task ID", "Operator", "Owner", "Retries", "Trigger", "Pool", "Queue", "Downstream"})

	v.table.SetSelectedFunc(func(row, column int) {
		if row > 0 && row <= len(v.taskDefinitions) {
			v.selectTask(v.taskDefinitions[row-1].TaskId)
		}
	})
}

func (v *TasksView) renderHeaders(headers []string) {
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(theme.ActiveTheme().TableHeaderText).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if i == 0 {
			cell.SetExpansion(1)
		}
		v.table.SetCell(0, i, cell)
	}
}

func (v *TasksView) selectTask(taskId string) {
	v.setActiveTask(taskId)
	if v.onSelected != nil {
		v.onSelected(taskId)
	}
}

// setActiveTask re-marks the committed row in place; see DagListView.setActiveDag.
func (v *TasksView) setActiveTask(taskId string) {
	if v.activeTaskId == taskId {
		return
	}
	v.activeTaskId = taskId
	for i, t := range v.taskDefinitions {
		cell := v.table.GetCell(i+1, 0)
		if cell == nil {
			continue
		}
		active := t.TaskId == taskId
		cell.SetText(rowLabel(t.TaskId, active)).SetTextColor(rowLabelColor(active))
	}
}

func (v *TasksView) SetOnSelected(handler func(taskId string)) {
	v.onSelected = handler
}

// showActive brings the page matching the current mode to the front.
func (v *TasksView) showActive() {
	switch {
	case v.ganttMode:
		v.SwitchToPage(tasksPageGantt)
	case v.hasRun:
		v.SwitchToPage(tasksPageRun)
	default:
		v.SwitchToPage(tasksPageDefs)
	}
}

// UpdateRun renders the run dashboard for the selected run.
func (v *TasksView) UpdateRun(run models.DAGRun, tis []models.TaskInstance, defs []models.Task, onCritical map[string]bool) {
	v.hasRun = run.RunId != ""
	v.run.UpdateRun(run, tis, defs, onCritical)
	v.showActive()
}

// UpdateDefinitions redraws the DAG-level task definitions, which need no run.
func (v *TasksView) UpdateDefinitions(dagId string, tasks []models.Task) {
	v.hasRun = false
	v.taskDefinitions = tasks
	v.table.Clear()
	v.setupTable()
	if len(tasks) == 0 {
		setEmptyHint(v.table, fmt.Sprintf("No DAG tasks loaded for %s", dagId))
		v.showActive()
		return
	}
	v.table.SetSelectable(true, false)

	t := theme.ActiveTheme()
	for i, task := range tasks {
		row := i + 1
		bg := t.PrimaryBg
		if row%2 == 0 {
			bg = t.TableRowAlt
		}

		active := task.TaskId == v.activeTaskId
		v.table.SetCell(row, 0, tview.NewTableCell(rowLabel(task.TaskId, active)).
			SetTextColor(rowLabelColor(active)).SetExpansion(1).SetBackgroundColor(bg))
		v.table.SetCell(row, 1, tview.NewTableCell(task.Operator).
			SetTextColor(t.PrimaryText).SetBackgroundColor(bg))
		v.table.SetCell(row, 2, tview.NewTableCell(task.Owner).
			SetTextColor(t.PrimaryText).SetBackgroundColor(bg))
		v.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.0f", task.Retries)).
			SetTextColor(t.PrimaryText).SetBackgroundColor(bg))
		v.table.SetCell(row, 4, tview.NewTableCell(task.TriggerRule).
			SetTextColor(t.PrimaryText).SetBackgroundColor(bg))
		v.table.SetCell(row, 5, tview.NewTableCell(task.Pool).
			SetTextColor(t.PrimaryText).SetBackgroundColor(bg))
		v.table.SetCell(row, 6, tview.NewTableCell(task.Queue).
			SetTextColor(t.PrimaryText).SetBackgroundColor(bg))
		v.table.SetCell(row, 7, tview.NewTableCell(fmt.Sprintf("%d", len(task.DownstreamTaskIds))).
			SetTextColor(t.PrimaryText).SetBackgroundColor(bg))
	}
	v.showActive()
}

// SetGanttMode switches which child page is visible.
func (v *TasksView) SetGanttMode(on bool) {
	v.ganttMode = on
	v.showActive()
}

// UpdateGantt forwards a fresh render to the embedded GanttView.
func (v *TasksView) UpdateGantt(runId string, tis []models.TaskInstance, onCritical map[string]bool) {
	v.gantt.Update(runId, tis, onCritical)
}

// Run exposes the run dashboard so callers can push logs into its preview pane.
func (v *TasksView) Run() *ExecutionView { return v.run }

func (v *TasksView) Root() tview.Primitive {
	return v.Pages
}
