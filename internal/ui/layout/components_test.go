package layout

import (
	"strings"
	"testing"
)

func TestKpiBarDAGStateCounts(t *testing.T) {
	k := NewKpiBar()
	k.SetDAGCounts(12, 3)
	k.SetDAGStateCounts(2, 9, 1)

	cases := []struct {
		key  string
		want string
	}{
		{"all", "15"},
		{"active", "12"},
		{"paused", "3"},
		{"running", "2"},
		{"success", "9"},
		{"failed", "1"},
	}
	for _, c := range cases {
		card, ok := k.cards[c.key]
		if !ok {
			t.Fatalf("card %q missing", c.key)
		}
		got := card.GetText(true)
		if !strings.Contains(got, c.want) {
			t.Errorf("card %q = %q, want it to contain %q", c.key, got, c.want)
		}
		if !strings.Contains(got, "DAGs") {
			t.Errorf("card %q = %q, want subtitle 'DAGs'", c.key, got)
		}
	}
}

func TestKpiBarSelectsDAGFilter(t *testing.T) {
	k := NewKpiBar()
	selected := ""
	k.SetOnSelected(func(filter string) { selected = filter })

	k.SelectFilter("paused")
	if selected != "paused" || k.ActiveFilter() != "paused" {
		t.Fatalf("selection = callback %q, active %q; want paused", selected, k.ActiveFilter())
	}

	k.selectRelative(1)
	if selected != "running" || k.ActiveFilter() != "running" {
		t.Fatalf("next selection = callback %q, active %q; want running", selected, k.ActiveFilter())
	}
}
