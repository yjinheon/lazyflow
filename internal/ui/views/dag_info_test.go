package views

import (
	"strings"
	"testing"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestDagInfoFilterLabels(t *testing.T) {
	v := NewDagInfoView()
	v.SetWindowLabel("7d")
	v.Update(models.DAG{DagId: "etl_daily"})

	// Before stats: counts shown as "-".
	if got, _ := v.FilterList().GetItemText(1); !strings.Contains(got, "Success") || !strings.Contains(got, "-") {
		t.Fatalf("pre-stats success item = %q, want 'Success  -'", got)
	}

	v.UpdateRunStats(1, 12, 2, "✓✓✗")
	list := v.FilterList()
	if list.GetItemCount() != 4 {
		t.Fatalf("filter item count = %d, want 4", list.GetItemCount())
	}
	cases := map[int]string{1: "12", 2: "2", 3: "1"} // success/failed/running counts
	for idx, want := range cases {
		if got, _ := list.GetItemText(idx); !strings.Contains(got, want) {
			t.Errorf("item %d = %q, want it to contain %q", idx, got, want)
		}
	}
}

func TestDagInfoFilterDefsContract(t *testing.T) {
	// The Enter handler maps list index → filterDefs[idx].state, so the order
	// and states are load-bearing.
	want := []string{"", "success", "failed", "running"}
	if len(filterDefs) != len(want) {
		t.Fatalf("filterDefs len = %d, want %d", len(filterDefs), len(want))
	}
	for i, w := range want {
		if filterDefs[i].state != w {
			t.Errorf("filterDefs[%d].state = %q, want %q", i, filterDefs[i].state, w)
		}
	}
}

func TestDagInfoUpdateResetsSelection(t *testing.T) {
	v := NewDagInfoView()
	v.FilterList().SetCurrentItem(2)
	v.Update(models.DAG{DagId: "d"})
	if got := v.FilterList().GetCurrentItem(); got != 0 {
		t.Fatalf("Update should reset selection to All (0), got %d", got)
	}
}

func TestRunSparkline(t *testing.T) {
	// newest-first input; output is oldest→newest.
	runs := []models.DAGRun{
		{State: "failed"}, // newest
		{State: "success"},
		{State: "running"},
	}
	got := RunSparkline(runs, 10)
	// oldest→newest: running, success, failed
	want := "[blue]⟳[-][green]✓[-][red]✗[-]"
	if got != want {
		t.Fatalf("RunSparkline = %q, want %q", got, want)
	}

	if RunSparkline(nil, 10) != "" {
		t.Errorf("RunSparkline(nil) should be empty")
	}
	if got := RunSparkline(runs, 1); got != "[red]✗[-]" {
		t.Errorf("RunSparkline cap=1 = %q, want newest only", got)
	}
}
