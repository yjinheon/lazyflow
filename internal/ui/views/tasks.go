package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type TasksView struct {
	*tview.Table
	tasks      []models.TaskInstance
	onSelected func(taskId string)
}

func NewTasksView() *TasksView {
	v := &TasksView{
		Table: tview.NewTable(),
	}
	v.setup()
	return v
}

func (v *TasksView) setup() {
	v.SetBorder(true).SetTitle(" Task Instances ")
	v.SetSelectable(true, false)
	v.SetFixed(1, 0)
	v.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.DefaultDarkTheme.TableSelected).
		Foreground(theme.DefaultDarkTheme.PrimaryText).
		Attributes(tcell.AttrBold))
	v.SetFocusFunc(func() { v.SetBorderColor(theme.DefaultDarkTheme.BorderFocused) })
	v.SetBlurFunc(func() { v.SetBorderColor(theme.DefaultDarkTheme.BorderColor) })

	headers := []string{"Task ID", "State", "Operator", "Duration", "Try", "Start"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if i == 0 {
			cell.SetExpansion(1)
		}
		v.SetCell(0, i, cell)
	}

	v.SetSelectedFunc(func(row, column int) {
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

func (v *TasksView) Update(tasks []models.TaskInstance) {
	v.tasks = tasks
	v.Clear()
	v.setup()

	t := theme.DefaultDarkTheme
	for i, task := range tasks {
		row := i + 1
		bg := t.PrimaryBg
		if row%2 == 0 {
			bg = t.TableRowAlt
		}

		v.SetCell(row, 0, tview.NewTableCell(task.TaskId).
			SetTextColor(tcell.ColorWhite).SetExpansion(1).SetBackgroundColor(bg))

		symbol, color := t.StatusStyle(task.State)
		v.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%s %s", symbol, task.State)).
			SetTextColor(color).SetBackgroundColor(bg))

		v.SetCell(row, 2, tview.NewTableCell(task.Operator).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		v.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.1fs", task.Duration)).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		v.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", task.TryNumber)).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))

		startStr := ""
		if task.StartDate != nil && !task.StartDate.IsZero() {
			startStr = task.StartDate.Format("01-02 15:04:05")
		}
		v.SetCell(row, 5, tview.NewTableCell(startStr).
			SetTextColor(tcell.ColorWhite).SetBackgroundColor(bg))
	}
}

func (v *TasksView) Root() *tview.Table {
	return v.Table
}
