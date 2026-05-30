package views

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestTopoLevels(t *testing.T) {
	tasks := []models.Task{
		{TaskId: "load", UpstreamTaskIds: []string{"transform"}},
		{TaskId: "extract"},
		{TaskId: "transform", UpstreamTaskIds: []string{"extract"}},
	}
	got := topoLevels(tasks)
	want := [][]string{{"extract"}, {"transform"}, {"load"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topoLevels = %v, want %v", got, want)
	}
}

func TestTopoLevelsParallel(t *testing.T) {
	tasks := []models.Task{
		{TaskId: "root"},
		{TaskId: "b", UpstreamTaskIds: []string{"root"}},
		{TaskId: "a", UpstreamTaskIds: []string{"root"}},
		{TaskId: "sink", UpstreamTaskIds: []string{"a", "b"}},
	}
	got := topoLevels(tasks)
	want := [][]string{{"root"}, {"a", "b"}, {"sink"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topoLevels = %v, want %v", got, want)
	}
}

func TestTopoLevelsCycleGraceful(t *testing.T) {
	tasks := []models.Task{
		{TaskId: "a", UpstreamTaskIds: []string{"b"}},
		{TaskId: "b", UpstreamTaskIds: []string{"a"}},
	}
	got := topoLevels(tasks)
	if len(got) == 0 {
		t.Fatal("topoLevels returned no stages for cyclic graph")
	}
	var count int
	for _, lvl := range got {
		count += len(lvl)
	}
	if count != 2 {
		t.Fatalf("topoLevels dropped nodes: got %d, want 2", count)
	}
}

func TestNodeStateFromTI(t *testing.T) {
	cases := map[string]NodeState{
		"success":         NodeSuccess,
		"failed":          NodeFailure,
		"upstream_failed": NodeFailure,
		"running":         NodeRunning,
		"skipped":         NodeSkipped,
		"removed":         NodeSkipped,
		"queued":          NodePending,
		"":                NodePending,
	}
	for in, want := range cases {
		if got := NodeStateFromTI(in); got != want {
			t.Errorf("NodeStateFromTI(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderGraphContainsTasks(t *testing.T) {
	tasks := []models.Task{
		{TaskId: "extract"},
		{TaskId: "transform", UpstreamTaskIds: []string{"extract"}},
		{TaskId: "load", UpstreamTaskIds: []string{"transform"}},
	}
	stateOf := func(id string) NodeState {
		if id == "extract" {
			return NodeSuccess
		}
		return NodePending
	}
	out := renderGraph(tasks, stateOf, 120)
	for _, id := range []string{"extract", "transform", "load"} {
		if !strings.Contains(out, id) {
			t.Errorf("renderGraph output missing task %q\n%s", id, out)
		}
	}
	if !strings.Contains(out, "Stage 1") {
		t.Errorf("renderGraph missing stage label\n%s", out)
	}
}

func TestRenderGraphOverflowMarker(t *testing.T) {
	tasks := []models.Task{
		{TaskId: "s1"},
		{TaskId: "s2", UpstreamTaskIds: []string{"s1"}},
		{TaskId: "s3", UpstreamTaskIds: []string{"s2"}},
		{TaskId: "s4", UpstreamTaskIds: []string{"s3"}},
		{TaskId: "s5", UpstreamTaskIds: []string{"s4"}},
		{TaskId: "s6", UpstreamTaskIds: []string{"s5"}},
	}
	out := renderGraph(tasks, func(string) NodeState { return NodePending }, 24)
	if !strings.Contains(out, "more stage") {
		t.Errorf("expected overflow marker for narrow width\n%s", out)
	}
}

func TestRenderGraphEmpty(t *testing.T) {
	out := renderGraph(nil, func(string) NodeState { return NodePending }, 80)
	if !strings.Contains(out, "no tasks") {
		t.Errorf("expected empty message, got %q", out)
	}
}
