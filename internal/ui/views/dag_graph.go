package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// NodeState is a DAG-graph node's render state. Values match Airflow task
// states so theme.StatusStyle can map them to symbol and color directly.
type NodeState string

const (
	NodeSuccess NodeState = "success"
	NodeFailure NodeState = "failed"
	NodeRunning NodeState = "running"
	NodeSkipped NodeState = "skipped"
	NodePending NodeState = ""
)

const (
	graphStageWidth = 20
	graphNodeLabel  = 14
)

// NodeStateFromTI maps an Airflow TaskInstance.State to a NodeState.
func NodeStateFromTI(state string) NodeState {
	switch state {
	case "success":
		return NodeSuccess
	case "failed", "upstream_failed":
		return NodeFailure
	case "running":
		return NodeRunning
	case "skipped", "removed":
		return NodeSkipped
	default:
		return NodePending
	}
}

// topoLevels groups tasks into topological stages. Stage members are sorted by
// task_id for determinism. Cyclic leftovers are emitted as a final stage so the
// renderer never hangs or drops nodes.
func topoLevels(tasks []models.Task) [][]string {
	if len(tasks) == 0 {
		return nil
	}

	upstream := make(map[string][]string, len(tasks))
	known := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		known[t.TaskId] = true
	}
	for _, t := range tasks {
		ups := make([]string, 0, len(t.UpstreamTaskIds))
		for _, u := range t.UpstreamTaskIds {
			if known[u] {
				ups = append(ups, u)
			}
		}
		sort.Strings(ups)
		upstream[t.TaskId] = ups
	}

	placed := make(map[string]bool, len(tasks))
	var levels [][]string
	for len(placed) < len(tasks) {
		var stage []string
		for id := range known {
			if placed[id] {
				continue
			}
			ready := true
			for _, u := range upstream[id] {
				if !placed[u] {
					ready = false
					break
				}
			}
			if ready {
				stage = append(stage, id)
			}
		}
		if len(stage) == 0 {
			for id := range known {
				if !placed[id] {
					stage = append(stage, id)
				}
			}
		}
		sort.Strings(stage)
		for _, id := range stage {
			placed[id] = true
		}
		levels = append(levels, stage)
	}
	return levels
}

// renderGraph builds tview dynamic-color markup laying out tasks as
// topological stages from left to right. Stages beyond the available width are
// collapsed into an overflow marker.
func renderGraph(tasks []models.Task, stateOf func(taskId string) NodeState, width int) string {
	levels := topoLevels(tasks)
	if len(levels) == 0 {
		return "[gray]no tasks to graph"
	}
	if width < graphStageWidth {
		width = graphStageWidth
	}
	maxVisible := width / graphStageWidth
	if maxVisible < 1 {
		maxVisible = 1
	}
	visible := len(levels)
	truncated := 0
	if len(levels) > maxVisible {
		visible = maxVisible - 1
		if visible < 1 {
			visible = 1
		}
		truncated = len(levels) - visible
	}

	th := theme.DefaultDarkTheme
	var b strings.Builder
	for i := 0; i < visible; i++ {
		fmt.Fprintf(&b, "[yellow::b]%-*s[-:-:-]", graphStageWidth, fmt.Sprintf("Stage %d", i+1))
	}
	if truncated > 0 {
		fmt.Fprintf(&b, "[gray]+%d more[-]", truncated)
	}
	b.WriteByte('\n')

	maxRows := 0
	for i := 0; i < visible; i++ {
		if len(levels[i]) > maxRows {
			maxRows = len(levels[i])
		}
	}
	for row := 0; row < maxRows; row++ {
		for i := 0; i < visible; i++ {
			cell := ""
			if row < len(levels[i]) {
				id := levels[i][row]
				sym, color := th.StatusStyle(string(stateOf(id)))
				arrow := ""
				if i < visible-1 {
					arrow = " → "
				}
				cell = fmt.Sprintf("[%s]%s %s[-]%s", theme.MarkupHex(color), sym, truncate(id, graphNodeLabel), arrow)
			}
			b.WriteString(cell)
			if pad := graphStageWidth - displayLen(cell); pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
		b.WriteByte('\n')
	}
	if truncated > 0 {
		fmt.Fprintf(&b, "[gray]… %d more stage(s) - press g for tree view[-]\n", truncated)
	}
	return b.String()
}

func displayLen(s string) int {
	n := 0
	inTag := false
	for _, r := range s {
		switch {
		case r == '[':
			inTag = true
		case r == ']' && inTag:
			inTag = false
		case !inTag:
			n++
		}
	}
	return n
}
