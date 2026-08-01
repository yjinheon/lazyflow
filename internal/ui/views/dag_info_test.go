package views

import (
	"strings"
	"testing"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestDagInfoRecentRuns(t *testing.T) {
	v := NewDagInfoView()
	v.Update(models.DAG{DagId: "etl_daily"})
	v.UpdateRecentRuns("✓✓✗")
	if got := v.Meta().GetText(true); !strings.Contains(got, "✓✓✗") {
		t.Fatalf("metadata = %q, want recent-run sparkline", got)
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
