package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// runSummary holds aggregate counts for a run's task instances.
type runSummary struct {
	Total, Done, Failed, Queued, Running int
}

// summarize counts task instances by coarse state. Done means success-like
// terminal states that do not need more work.
func summarize(tis []models.TaskInstance) runSummary {
	var s runSummary
	s.Total = len(tis)
	for _, ti := range tis {
		switch ti.State {
		case "success", "skipped", "removed":
			s.Done++
		case "failed", "upstream_failed":
			s.Failed++
		case "queued", "scheduled":
			s.Queued++
		case "running":
			s.Running++
		}
	}
	return s
}

// ExecutionView is the full-screen live run drill-in.
type ExecutionView struct {
	*tview.Flex

	summary  *tview.TextView
	taskList *tview.Table
	detail   *tview.TextView
	logs     *tview.TextView
	miniDAG  *tview.TextView
	gantt    *GanttView

	tasks     []models.TaskInstance
	defs      []models.Task
	runId     string
	onTaskSel func(taskId string)
}

func NewExecutionView() *ExecutionView {
	v := &ExecutionView{
		Flex:     tview.NewFlex(),
		summary:  tview.NewTextView(),
		taskList: tview.NewTable(),
		detail:   tview.NewTextView(),
		logs:     tview.NewTextView(),
		miniDAG:  tview.NewTextView(),
		gantt:    NewGanttView(),
	}
	v.setup()
	return v
}

func (v *ExecutionView) setup() {
	th := theme.DefaultDarkTheme

	v.summary.SetDynamicColors(true)
	v.summary.SetBorder(true).SetTitle(" Run ")

	v.taskList.SetBorder(true).SetTitle(" Tasks ")
	v.taskList.SetSelectable(true, false)
	v.taskList.SetFixed(1, 0)
	v.taskList.SetSelectedStyle(tcell.StyleDefault.
		Background(th.TableSelected).Foreground(th.PrimaryText).Attributes(tcell.AttrBold))

	v.detail.SetDynamicColors(true)
	v.detail.SetBorder(true).SetTitle(" Task Detail ")

	v.logs.SetDynamicColors(true).SetScrollable(true)
	v.logs.SetBorder(true).SetTitle(" Logs ")

	v.miniDAG.SetDynamicColors(true).SetWrap(false)
	v.miniDAG.SetBorder(true).SetTitle(" DAG ")

	centre := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.detail, 0, 1, false).
		AddItem(v.logs, 0, 1, false)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.miniDAG, 0, 3, false).
		AddItem(v.gantt, 0, 2, false)

	body := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(v.taskList, 0, 3, true).
		AddItem(centre, 0, 5, false).
		AddItem(right, 0, 4, false)

	v.SetDirection(tview.FlexRow).
		AddItem(v.summary, 3, 0, false).
		AddItem(body, 0, 1, true)

	v.taskList.SetSelectedFunc(func(row, _ int) {
		if row > 0 && row <= len(v.tasks) && v.onTaskSel != nil {
			v.onTaskSel(v.tasks[row-1].TaskId)
		}
	})
	v.taskList.SetSelectionChangedFunc(func(row, _ int) {
		if row > 0 && row <= len(v.tasks) {
			v.renderDetail(v.tasks[row-1])
		}
	})
}

func (v *ExecutionView) SetOnTaskSelected(fn func(taskId string)) { v.onTaskSel = fn }
func (v *ExecutionView) Root() tview.Primitive                    { return v.Flex }
func (v *ExecutionView) TaskList() *tview.Table                   { return v.taskList }

func (v *ExecutionView) UpdateRun(run models.DAGRun, tis []models.TaskInstance, defs []models.Task, onCritical map[string]bool) {
	// Preserve the user's current selection across poll-driven refreshes.
	// Only reset to the first task when the run itself changes; otherwise the
	// periodic "tasks" poll would snap the selection back to row 1 every tick.
	sameRun := v.runId == run.RunId
	prevTaskId := ""
	if sameRun {
		if r, _ := v.taskList.GetSelection(); r > 0 && r <= len(v.tasks) {
			prevTaskId = v.tasks[r-1].TaskId
		}
	}

	v.runId = run.RunId
	v.tasks = tis
	v.defs = defs
	v.renderSummary(run, tis)
	v.renderTaskList(tis)
	v.renderMiniDAG()
	v.gantt.Update(run.RunId, tis, onCritical)

	if len(tis) == 0 {
		v.detail.SetText("[gray]No task instances loaded.")
		return
	}

	// Re-select the previously selected task by id so the choice survives
	// reordering; fall back to the first row for a new run or if it vanished.
	row := 1
	if prevTaskId != "" {
		for i, ti := range tis {
			if ti.TaskId == prevTaskId {
				row = i + 1
				break
			}
		}
	}
	v.taskList.Select(row, 0)
	v.renderDetail(tis[row-1])
}

func (v *ExecutionView) renderSummary(run models.DAGRun, tis []models.TaskInstance) {
	th := theme.DefaultDarkTheme
	sym, color := th.StatusStyle(run.State)
	s := summarize(tis)
	start := ""
	if !run.StartDate.IsZero() {
		start = run.StartDate.Format("01-02 15:04:05")
	}
	v.summary.SetText(fmt.Sprintf(
		" [%s]%s %s[-]  -  %s  -  [green]%d/%d done[-] - [red]%d failed[-] - [gray]%d queued[-]  -  [gray]Esc back / Tab focus[-]",
		theme.MarkupHex(color), sym, run.RunId, start, s.Done, s.Total, s.Failed, s.Queued))
}

func (v *ExecutionView) renderTaskList(tis []models.TaskInstance) {
	th := theme.DefaultDarkTheme
	v.taskList.Clear()
	hdr := []string{"Task", "State", "Try"}
	for i, h := range hdr {
		c := tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false)
		if i == 0 {
			c.SetExpansion(1)
		}
		v.taskList.SetCell(0, i, c)
	}
	for i, ti := range tis {
		row := i + 1
		sym, color := th.StatusStyle(ti.State)
		v.taskList.SetCell(row, 0, tview.NewTableCell(truncate(ti.TaskId, 22)).SetExpansion(1))
		v.taskList.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%s %s", sym, ti.State)).SetTextColor(color))
		v.taskList.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", ti.TryNumber)))
	}
}

func (v *ExecutionView) renderDetail(ti models.TaskInstance) {
	start, end := "-", "-"
	if ti.StartDate != nil && !ti.StartDate.IsZero() {
		start = ti.StartDate.Format("01-02 15:04:05")
	}
	if ti.EndDate != nil && !ti.EndDate.IsZero() {
		end = ti.EndDate.Format("01-02 15:04:05")
	}
	v.detail.SetText(fmt.Sprintf(
		"[yellow]Task:[-] %s\n[yellow]State:[-] %s\n[yellow]Operator:[-] %s\n[yellow]Try:[-] %d\n[yellow]Duration:[-] %.1fs\n[yellow]Pool:[-] %s\n[yellow]Queue:[-] %s\n[yellow]Start:[-] %s\n[yellow]End:[-] %s\n[yellow]Host:[-] %s",
		ti.TaskId, ti.State, ti.Operator, ti.TryNumber, ti.Duration, ti.Pool, ti.Queue, start, end, ti.Hostname))
}

func (v *ExecutionView) renderMiniDAG() {
	_, _, w, _ := v.miniDAG.GetInnerRect()
	if w <= 0 {
		w = 30
	}
	stateByTask := map[string]string{}
	for _, ti := range v.tasks {
		stateByTask[ti.TaskId] = ti.State
	}
	v.miniDAG.SetText(renderGraph(v.defs, func(id string) NodeState {
		return NodeStateFromTI(stateByTask[id])
	}, w))
}

func (v *ExecutionView) SetLogs(text string) {
	v.logs.SetText(tview.Escape(text))
	v.logs.ScrollToEnd()
}

func (v *ExecutionView) SetLogMessage(msg string) {
	v.logs.SetText("[gray]" + tview.Escape(msg))
}

func (v *ExecutionView) SetLogError(msg string) {
	v.logs.SetText("[red]" + tview.Escape(msg))
}
