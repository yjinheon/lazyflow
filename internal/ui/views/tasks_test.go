package views

import (
	"testing"

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
	if got := v.table.GetCell(1, 0).Text; got != "extract" {
		t.Fatalf("task id=%q", got)
	}
	if got := v.table.GetCell(1, 1).Text; got != "BashOperator" {
		t.Fatalf("operator=%q", got)
	}
	if !v.showingDefinitions {
		t.Fatal("expected definition mode")
	}
}

func TestTasksViewUpdateInstances_returnsToRunMode(t *testing.T) {
	v := NewTasksView()
	v.UpdateDefinitions("etl", []models.Task{{TaskId: "extract"}})

	v.Update([]models.TaskInstance{{TaskId: "extract", State: "success", Operator: "BashOperator"}})

	if v.showingDefinitions {
		t.Fatal("expected task instance mode")
	}
	if got := v.table.GetCell(0, 1).Text; got != "State" {
		t.Fatalf("header=%q", got)
	}
	if got := v.table.GetCell(1, 0).Text; got != "extract" {
		t.Fatalf("task id=%q", got)
	}
}
