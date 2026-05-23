package views

import (
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func tiOf(id, state string, start, end time.Time) models.TaskInstance {
	return models.TaskInstance{TaskId: id, State: state, StartDate: &start, EndDate: &end}
}

func taskOf(id string, ups ...string) models.Task {
	return models.Task{TaskId: id, UpstreamTaskIds: ups}
}

func TestComputeCriticalPath_diamond(t *testing.T) {
	// a → b → d, a → c → d. b=10m, c=2m. Critical path: a, b, d.
	t0 := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	tasks := []models.Task{
		taskOf("a"),
		taskOf("b", "a"),
		taskOf("c", "a"),
		taskOf("d", "b", "c"),
	}
	tis := []models.TaskInstance{
		tiOf("a", "success", t0, t0.Add(time.Minute)),
		tiOf("b", "success", t0.Add(time.Minute), t0.Add(11*time.Minute)),
		tiOf("c", "success", t0.Add(time.Minute), t0.Add(3*time.Minute)),
		tiOf("d", "success", t0.Add(11*time.Minute), t0.Add(12*time.Minute)),
	}
	got := ComputeCriticalPath(tasks, tis, t0.Add(12*time.Minute))
	want := map[string]bool{"a": true, "b": true, "d": true}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("missing %q in critical path: %+v", k, got)
		}
	}
	if got["c"] {
		t.Fatalf("c should not be critical")
	}
}

func TestComputeCriticalPath_singleNode(t *testing.T) {
	t0 := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	tasks := []models.Task{taskOf("only")}
	tis := []models.TaskInstance{tiOf("only", "success", t0, t0.Add(time.Minute))}
	got := ComputeCriticalPath(tasks, tis, t0.Add(time.Minute))
	if !got["only"] {
		t.Fatalf("single node should be critical: %+v", got)
	}
}

func TestComputeCriticalPath_emptyTasks(t *testing.T) {
	got := ComputeCriticalPath(nil, nil, time.Now())
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

func TestComputeCriticalPath_tieBreakDeterministic(t *testing.T) {
	// a → b, a → c. b duration == c duration. Tie-break picks smaller id (b).
	t0 := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	tasks := []models.Task{taskOf("a"), taskOf("b", "a"), taskOf("c", "a")}
	tis := []models.TaskInstance{
		tiOf("a", "success", t0, t0.Add(time.Minute)),
		tiOf("b", "success", t0.Add(time.Minute), t0.Add(2*time.Minute)),
		tiOf("c", "success", t0.Add(time.Minute), t0.Add(2*time.Minute)),
	}
	got := ComputeCriticalPath(tasks, tis, t0.Add(2*time.Minute))
	if !got["b"] || got["c"] {
		t.Fatalf("tie-break should pick 'b', got %+v", got)
	}
}
