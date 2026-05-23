package views

import (
	"strings"
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}

func ptr(t time.Time) *time.Time { return &t }

func TestEffectiveDuration_states(t *testing.T) {
	now := mustParse(t, "2026-05-23T10:10:00Z")
	tStart := mustParse(t, "2026-05-23T10:00:00Z")
	tEnd := mustParse(t, "2026-05-23T10:05:00Z")
	cases := []struct {
		name string
		ti   models.TaskInstance
		want time.Duration
	}{
		{"success", models.TaskInstance{State: "success", StartDate: &tStart, EndDate: &tEnd}, 5 * time.Minute},
		{"running uses now-start", models.TaskInstance{State: "running", StartDate: &tStart}, 10 * time.Minute},
		{"queued is zero", models.TaskInstance{State: "queued", StartDate: &tStart}, 0},
		{"skipped is zero", models.TaskInstance{State: "skipped"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := effectiveDuration(tc.ti, now)
			if got != tc.want {
				t.Fatalf("got=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestComputeBuckets_basic(t *testing.T) {
	tis := []models.TaskInstance{
		{
			QueuedDttm: ptr(mustParse(t, "2026-05-23T10:00:00Z")),
			StartDate:  ptr(mustParse(t, "2026-05-23T10:00:30Z")),
			EndDate:    ptr(mustParse(t, "2026-05-23T10:05:00Z")),
			State:      "success",
		},
	}
	buckets, tMin, tMax := ComputeBuckets(tis, 10, mustParse(t, "2026-05-23T10:05:00Z"))
	if len(buckets) != 10 {
		t.Fatalf("buckets len=%d", len(buckets))
	}
	if tMin.IsZero() || tMax.IsZero() {
		t.Fatal("tMin/tMax zero")
	}
	if buckets[0].Start != tMin || buckets[9].End != tMax {
		t.Fatal("bucket boundaries off")
	}
}

func TestComputeBuckets_emptyInput(t *testing.T) {
	buckets, _, _ := ComputeBuckets(nil, 10, time.Now())
	if buckets != nil {
		t.Fatal("expected nil")
	}
}

func TestRenderCells_queueAndRun(t *testing.T) {
	q := mustParse(t, "2026-05-23T10:00:00Z")
	s := mustParse(t, "2026-05-23T10:00:30Z")
	e := mustParse(t, "2026-05-23T10:01:00Z")
	ti := models.TaskInstance{QueuedDttm: &q, StartDate: &s, EndDate: &e, State: "success"}
	tMax := e
	buckets, _, _ := ComputeBuckets([]models.TaskInstance{ti}, 4, tMax)
	cells := RenderCells(ti, buckets, tMax)
	if len(cells) != 4 {
		t.Fatalf("cells len=%d", len(cells))
	}
	// First bucket = queue, last bucket = run
	if cells[0].Char != '▒' {
		t.Fatalf("cells[0]=%+v want queue", cells[0])
	}
	if cells[3].Char != '█' {
		t.Fatalf("cells[3]=%+v want run", cells[3])
	}
}

func TestEmitRLE_collapsesAdjacent(t *testing.T) {
	cells := []Cell{
		{Char: '▒', Color: "queued"},
		{Char: '▒', Color: "queued"},
		{Char: '█', Color: "success"},
		{Char: '█', Color: "success"},
		{Char: ' ', Color: ""},
	}
	out := EmitRLE(cells, false)
	// Two color spans (queued + success).
	if got := strings.Count(out, "[queued]") + strings.Count(out, "[success]"); got != 2 {
		t.Fatalf("expected 2 color tags, got %d in %q", got, out)
	}
}

func TestEmitRLE_boldWrapsAllSpans(t *testing.T) {
	cells := []Cell{{Char: '█', Color: "success"}}
	out := EmitRLE(cells, true)
	if !strings.Contains(out, "::b") {
		t.Fatalf("bold not applied: %q", out)
	}
}
