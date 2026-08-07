package views

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestTasksViewUpdateDefinitions_rendersDAGTasks(t *testing.T) {
	v := NewTasksView()

	v.UpdateDefinitions("etl", []models.Task{
		{
			TaskId:            "extract",
			Operator:          "BashOperator",
			Owner:             "data",
			Retries:           2,
			TriggerRule:       "all_success",
			Pool:              "default_pool",
			Queue:             "default",
			DownstreamTaskIds: []string{"load"},
		},
	})

	if got := v.table.GetCell(0, 0).Text; got != "Task ID" {
		t.Fatalf("header=%q", got)
	}
	if got := v.table.GetCell(1, 0).Text; got != rowLabel("extract", false) {
		t.Fatalf("task id=%q", got)
	}
	if got := v.table.GetCell(1, 1).Text; got != "BashOperator" {
		t.Fatalf("operator=%q", got)
	}
	if name, _ := v.GetFrontPage(); name != tasksPageDefs {
		t.Fatalf("page=%q, want %q", name, tasksPageDefs)
	}
}

// TestTasksViewModes covers the three faces of the tab: definitions when no run
// is selected, the run dashboard once one is, and gantt on top of either.
func TestTasksViewModes(t *testing.T) {
	v := NewTasksView()
	v.UpdateDefinitions("etl", []models.Task{{TaskId: "extract"}})

	v.UpdateRun(models.DAGRun{RunId: "run-1"},
		[]models.TaskInstance{{TaskId: "extract", State: "success"}}, nil, nil)
	if name, _ := v.GetFrontPage(); name != tasksPageRun {
		t.Fatalf("after UpdateRun page=%q, want %q", name, tasksPageRun)
	}

	v.SetGanttMode(true)
	if name, _ := v.GetFrontPage(); name != tasksPageGantt {
		t.Fatalf("gantt page=%q, want %q", name, tasksPageGantt)
	}

	v.SetGanttMode(false)
	if name, _ := v.GetFrontPage(); name != tasksPageRun {
		t.Fatalf("back from gantt page=%q, want %q", name, tasksPageRun)
	}

	// A new DAG drops the run, so the tab falls back to definitions.
	v.UpdateDefinitions("other", []models.Task{{TaskId: "load"}})
	if name, _ := v.GetFrontPage(); name != tasksPageDefs {
		t.Fatalf("after UpdateDefinitions page=%q, want %q", name, tasksPageDefs)
	}
}

// ActiveTabPrimitive hands the whole tab to SetFocus, so focus has to reach the
// selectable widget through Pages → Flex on its own.
func TestTasksViewFocusReachesActiveWidget(t *testing.T) {
	app := tview.NewApplication()
	v := NewTasksView()

	v.UpdateRun(models.DAGRun{RunId: "run-1"},
		[]models.TaskInstance{{TaskId: "extract", State: "success"}}, nil, nil)
	app.SetFocus(v)
	if app.GetFocus() != tview.Primitive(v.run.taskList) {
		t.Fatalf("run mode focus = %T, want the dashboard task list", app.GetFocus())
	}

	v.UpdateDefinitions("etl", []models.Task{{TaskId: "extract"}})
	app.SetFocus(v)
	if app.GetFocus() != tview.Primitive(v.table) {
		t.Fatalf("definition mode focus = %T, want the definitions table", app.GetFocus())
	}
}

func TestTasksViewOnSelected_firesFromRunDashboard(t *testing.T) {
	v := NewTasksView()
	var got string
	v.SetOnSelected(func(taskId string) { got = taskId })

	v.UpdateRun(models.DAGRun{RunId: "run-1"},
		[]models.TaskInstance{{TaskId: "extract", State: "success"}}, nil, nil)
	// The dashboard's own task callback must route out through the tab.
	v.run.onTaskSel("extract")

	if got != "extract" {
		t.Fatalf("onSelected got %q, want %q", got, "extract")
	}
}
