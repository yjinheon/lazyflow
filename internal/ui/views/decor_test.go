package views

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func dagsFixture() []models.DAG {
	return []models.DAG{{DagId: "alpha"}, {DagId: "beta"}, {DagId: "gamma"}}
}

// NewTableCell seeds a non-default Style, so SetTextColor lands in Style and
// the legacy Color field stays zero.
func cellFg(c *tview.TableCell) tcell.Color {
	fg, _, _ := c.Style.Decompose()
	return fg
}

func markedRows(v *DagListView, n int) []string {
	out := []string{}
	for i := 1; i <= n; i++ {
		if c := v.GetCell(i, 0); c != nil && strings.HasPrefix(c.Text, activeMarker) {
			out = append(out, strings.TrimPrefix(c.Text, activeMarker))
		}
	}
	return out
}

// The cursor and the committed selection are different things: moving the
// cursor must not mark a row, only Enter (setActiveDag) may.
func TestCursorMovementDoesNotMarkRow(t *testing.T) {
	theme.ApplyTheme(theme.TokyoNightStorm)
	v := NewDagListView()
	v.Update(dagsFixture())

	if got := markedRows(v, 3); len(got) != 0 {
		t.Fatalf("rows marked before any commit: %v", got)
	}

	v.Select(3, 0) // cursor onto "gamma"
	if got := markedRows(v, 3); len(got) != 0 {
		t.Fatalf("moving the cursor marked rows: %v", got)
	}

	v.setActiveDag("beta")
	got := markedRows(v, 3)
	if len(got) != 1 || got[0] != "beta" {
		t.Fatalf("marked rows = %v, want [beta]", got)
	}
	if c := v.GetCell(2, 0); cellFg(c) != theme.ActiveTheme().Accent {
		t.Error("committed row is not accent-coloured")
	}
	if c := v.GetCell(1, 0); cellFg(c) != theme.ActiveTheme().PrimaryText {
		t.Error("uncommitted row should keep primary text colour")
	}
}

// A refresh must not lose the committed marker.
func TestCommittedMarkerSurvivesRerender(t *testing.T) {
	theme.ApplyTheme(theme.TokyoNightStorm)
	v := NewDagListView()
	v.Update(dagsFixture())
	v.setActiveDag("gamma")

	v.Update(dagsFixture()) // poller refresh
	if got := markedRows(v, 3); len(got) != 1 || got[0] != "gamma" {
		t.Fatalf("after re-render marked rows = %v, want [gamma]", got)
	}
}

// Empty tables show guidance and stay non-selectable (freeze guard).
func TestEmptyHintsAreNonSelectable(t *testing.T) {
	theme.ApplyTheme(theme.TokyoNightStorm)

	dl := NewDagListView()
	dl.Update(nil)
	hint := dl.GetCell(1, 0)
	if hint == nil || hint.Text == "" {
		t.Fatal("DagList shows no empty-state hint")
	}
	if hint.NotSelectable != true {
		t.Error("empty hint cell must not be selectable")
	}
	if r, _ := dl.GetSelectable(); r {
		t.Error("DagList must stay non-selectable while empty")
	}

	rv := NewRunsView()
	rv.Update(nil)
	if c := rv.GetCell(1, 0); c == nil || c.Text == "" {
		t.Error("Runs shows no empty-state hint")
	}
	if r, _ := rv.GetSelectable(); r {
		t.Error("Runs must stay non-selectable while empty")
	}
}

// The hint should tell the user what to do next, and it changes with context.
func TestDagListHintIsContextual(t *testing.T) {
	v := NewDagListView()
	v.Update(nil)
	base := v.emptyHint()

	v.SetFilter("failed")
	if v.emptyHint() == base {
		t.Error("filtered empty state should differ from the default hint")
	}

	v.SetFilter("all")
	v.Search("zzz")
	if v.emptyHint() == base {
		t.Error("searched empty state should differ from the default hint")
	}
}

func TestDagListOrdersActiveFirstThenDagID(t *testing.T) {
	v := NewDagListView()
	v.Update([]models.DAG{
		{DagId: "z_paused", IsPaused: true},
		{DagId: "beta"},
		{DagId: "a_paused", IsPaused: true},
		{DagId: "Alpha"},
	})

	want := []string{"Alpha", "beta", "a_paused", "z_paused"}
	for i, id := range want {
		if got := v.dags[i].DagId; got != id {
			t.Fatalf("row %d = %q, want %q (all=%v)", i, got, id, v.dags)
		}
	}
}

func TestDagListStatusFilters(t *testing.T) {
	v := NewDagListView()
	v.Update([]models.DAG{
		{DagId: "paused", IsPaused: true},
		{DagId: "run", LastRunState: "running"},
		{DagId: "ok", LastRunState: "success"},
		{DagId: "bad", LastRunState: "failed"},
	})

	for filter, want := range map[string]string{
		"paused": "paused", "running": "run", "success": "ok", "failed": "bad",
	} {
		v.SetFilter(filter)
		if len(v.dags) != 1 || v.dags[0].DagId != want {
			t.Errorf("filter %q = %v, want only %q", filter, v.dags, want)
		}
	}
}
