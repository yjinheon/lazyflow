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
		{"active", "12"},
		{"inactive", "3"},
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
