package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// TasksView is a Pages wrapper showing either a Table (default) or a Gantt
// view of the same TaskInstance slice. The active page is controlled by
// SetGanttMode (set from the keybindings layer in response to the `g` key).
type TasksView struct {
	*tview.Pages

	table *tview.Table
	gantt *GanttView

	tasks      []models.TaskInstance
	onSelected func(taskId string)
}

const (
	tasksPageTable = "table"
	tasksPageGantt = "gantt"
)

func NewTasksView() *TasksView {
	v := &TasksView{
		Pages: tview.NewPages(),
		table: tview.NewTable(),
		gantt: NewGanttView(),
	}
	v.setupTable()
	v.AddPage(tasksPageTable, v.table, true, true)
	v.AddPage(tasksPageGantt, v.gantt, true, false)
	return v
}

func (v *TasksView) setupTable() {
	v.table.SetBorder(true).SetTitle(" Task Instances ")
	v.table.SetSelectable(false, false)
	v.table.SetFixed(1, 0)
	v.table.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.DefaultDarkTheme.TableSelected).
		Foreground(theme.DefaultDarkTheme.PrimaryText).
		Attributes(tcell.AttrBold))
	v.table.SetFocusFunc(func() { v.table.SetBorderColor(theme.DefaultDarkTheme.BorderFocused) })
	v.table.SetBlurFunc(func() { v.table.SetBorderColor(theme.DefaultDarkTheme.BorderColor) })

	headers := []string{"Task ID", "State", "Operator", "Duration", "Try", "Start"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if i == 0 {
			cell.SetExpansion(1)
		}
		v.table.SetCell(0, i, cell)
	}

	v.table.SetSelectedFunc(func(row, column int) {
		if row > 0 && row <= len(v.tasks) {
			if v.onSelected != nil {
				v.onSelected(v.tasks[row-1].TaskId)
			}
		}
	})
}

func (v *TasksView) SetOnSelected(handler func(taskId string)) {
	v.onSelected = handler
}

// Update redraws the table view. The Gantt view is updated separately via
// UpdateGantt; switch which is visible with SetGanttMode.
func (v *TasksView) Update(tasks []models.TaskInstance) {
	v.tasks = tasks
	v.table.Clear()
	v.setupTable()
	if len(tasks) == 0 {
		return
	}
	v.table.SetSelectable(true, false)

	t := theme.DefaultDarkTheme
	for i, task := range tasks {
		row := i + 1
		bg := t.PrimaryBg
		if row%2 == 0 {
			bg = t.TableRowAlt
		}

		v.table.SetCell(row, 0, tview.NewTableCell(task.TaskId).
			SetTextColor(tcell.ColorWhite).SetExpansion(1).SetBackgroundColor(bg))

		symbol, color := t.StatusStyle(task.State)
		v.table.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%s %s", symbol, task.State)).
			SetTextColor(color).SetBackgroundColor(bg))

		v.table.SetCell(row, 2, tview.NewTableCell(task.Operator).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		v.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.1fs", task.Duration)).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		v.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", task.TryNumber)).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		startStr := ""
		if task.StartDate != nil && !task.StartDate.IsZero() {
			startStr = task.StartDate.Format("01-02 15:04:05")
		}
		v.table.SetCell(row, 5, tview.NewTableCell(startStr).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))
	}
}

// SetGanttMode switches which child page is visible.
func (v *TasksView) SetGanttMode(on bool) {
	if on {
		v.SwitchToPage(tasksPageGantt)
	} else {
		v.SwitchToPage(tasksPageTable)
	}
}

// UpdateGantt forwards a fresh render to the embedded GanttView.
func (v *TasksView) UpdateGantt(runId string, tis []models.TaskInstance, onCritical map[string]bool) {
	v.gantt.Update(runId, tis, onCritical)
}

// Root returns the Pages primitive (now interface, was *tview.Table).
func (v *TasksView) Root() tview.Primitive {
	return v.Pages
}
