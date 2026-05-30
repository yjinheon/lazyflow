package views

import (
	"fmt"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

const (
	lineagePageTree  = "tree"
	lineagePageGraph = "graph"
)

type LineageView struct {
	*tview.Pages
	treeFlex *tview.Flex
	tree     *tview.TreeView
	details  *tview.TextView
	graph    *tview.TextView
	tasks    []models.Task
}

func NewLineageView() *LineageView {
	v := &LineageView{
		Pages:    tview.NewPages(),
		treeFlex: tview.NewFlex(),
		tree:     tview.NewTreeView(),
		details:  tview.NewTextView(),
		graph:    tview.NewTextView(),
	}
	v.setup()
	return v
}

func (v *LineageView) setup() {
	v.tree.SetBorder(true).
		SetTitle(" Task Lineage ").
		SetBorderColor(theme.DefaultDarkTheme.BorderColor)

	v.details.SetBorder(true).
		SetTitle(" Task Details ").
		SetBorderColor(theme.DefaultDarkTheme.BorderColor)

	v.details.SetDynamicColors(true)

	v.treeFlex.SetDirection(tview.FlexColumn).
		AddItem(v.tree, 0, 60, true).
		AddItem(v.details, 0, 40, false)

	v.graph.SetBorder(true).
		SetTitle(" DAG Graph (g: tree) ").
		SetBorderColor(theme.DefaultDarkTheme.BorderColor)
	v.graph.SetDynamicColors(true).SetWrap(false)

	v.AddPage(lineagePageTree, v.treeFlex, true, true)
	v.AddPage(lineagePageGraph, v.graph, true, false)
}

func (v *LineageView) SetTasks(dagId string, tasks []models.Task) {
	v.tasks = tasks
	v.details.SetText("")

	root := tview.NewTreeNode(dagId).
		SetColor(theme.DefaultDarkTheme.Accent)

	v.tree.SetRoot(root).SetCurrentNode(root)
	if len(tasks) == 0 {
		v.details.SetText("[gray]No task lineage loaded.")
		return
	}

	taskMap := make(map[string]models.Task)
	for _, t := range tasks {
		taskMap[t.TaskId] = t
	}

	roots := []models.Task{}
	for _, t := range tasks {
		if len(t.UpstreamTaskIds) == 0 {
			roots = append(roots, t)
		}
	}

	if len(roots) == 0 && len(tasks) > 0 {
		roots = append(roots, tasks[0])
	}

	for _, t := range roots {
		node := v.buildNode(t, taskMap, 0)
		root.AddChild(node)
	}
}

func (v *LineageView) buildNode(task models.Task, taskMap map[string]models.Task, depth int) *tview.TreeNode {
	if depth > 10 {
		return tview.NewTreeNode("... (max depth)")
	}

	node := tview.NewTreeNode(task.TaskId).
		SetReference(task.TaskId).
		SetSelectable(true)

	node.SetSelectedFunc(func() {
		v.showDetails(task)
	})

	for _, downId := range task.DownstreamTaskIds {
		if downstreamTask, exists := taskMap[downId]; exists {
			childNode := v.buildNode(downstreamTask, taskMap, depth+1)
			node.AddChild(childNode)
		}
	}

	return node
}

func (v *LineageView) showDetails(task models.Task) {
	text := fmt.Sprintf("[yellow]Task ID:[white] %s\n", task.TaskId)
	text += fmt.Sprintf("[yellow]Owner:[white] %s\n", task.Owner)
	text += fmt.Sprintf("\n[yellow]Upstream:[white] %v\n", task.UpstreamTaskIds)
	text += fmt.Sprintf("[yellow]Downstream:[white] %v\n", task.DownstreamTaskIds)

	v.details.SetText(text)
}

func (v *LineageView) SetGraphMode(on bool) {
	if on {
		v.SwitchToPage(lineagePageGraph)
		return
	}
	v.SwitchToPage(lineagePageTree)
}

func (v *LineageView) IsGraphMode() bool {
	name, _ := v.GetFrontPage()
	return name == lineagePageGraph
}

func (v *LineageView) UpdateGraph(stateByTask map[string]string) {
	_, _, w, _ := v.graph.GetInnerRect()
	if w <= 0 {
		w = 80
	}
	stateOf := func(id string) NodeState {
		if stateByTask == nil {
			return NodePending
		}
		return NodeStateFromTI(stateByTask[id])
	}
	body := renderGraph(v.tasks, stateOf, w)
	legend := fmt.Sprintf("\n[gray]legend:[-] %s%s %s%s %s%s %s%s %s%s\n",
		mk(theme.DefaultDarkTheme, "success"), " success ",
		mk(theme.DefaultDarkTheme, "running"), " running ",
		mk(theme.DefaultDarkTheme, "failed"), " failed ",
		mk(theme.DefaultDarkTheme, "skipped"), " skipped ",
		mk(theme.DefaultDarkTheme, ""), " pending")
	v.graph.SetText(body + legend)
}

func mk(t theme.Theme, state string) string {
	sym, color := t.StatusStyle(state)
	return fmt.Sprintf("[%s]%s[-]", theme.MarkupHex(color), sym)
}

func (v *LineageView) Root() tview.Primitive {
	return v.Pages
}
