package views

import (
	"strings"
	"testing"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestRenderPoolBar(t *testing.T) {
	cases := []struct {
		name                   string
		occupied, slots, width int
		wantFilled             int
		wantColor              string
	}{
		{"empty", 0, 16, 8, 0, "green"},
		{"half", 4, 8, 8, 4, "green"},
		{"high", 7, 8, 8, 7, "yellow"},
		{"full", 8, 8, 8, 8, "red"},
		{"saturated", 12, 8, 8, 8, "red"},
		{"zero-slots", 0, 0, 8, 0, "green"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := renderPoolBar(c.occupied, c.slots, c.width)
			if n := strings.Count(out, "█"); n != c.wantFilled {
				t.Errorf("filled cells = %d, want %d (out=%q)", n, c.wantFilled, out)
			}
			if total := strings.Count(out, "█") + strings.Count(out, "░"); total != c.width {
				t.Errorf("total cells = %d, want %d", total, c.width)
			}
			if !strings.Contains(out, c.wantColor) {
				t.Errorf("color = missing %q in %q", c.wantColor, out)
			}
		})
	}
}

func TestRenderPoolBar_zeroWidth(t *testing.T) {
	if out := renderPoolBar(4, 8, 0); strings.Contains(out, "█") {
		t.Errorf("zero width should produce no fill, got %q", out)
	}
}

func TestClusterToggleView(t *testing.T) {
	v := NewClusterInfoView()
	v.UpdatePools([]models.Pool{
		{Name: "default", Slots: 8, OccupiedSlots: 6, QueuedSlots: 1},
		{Name: "spark", Slots: 4, OccupiedSlots: 4, QueuedSlots: 2},
	})

	// Default mode is compact: shows a bar and a queued warning.
	compact := v.GetText(false)
	if !strings.Contains(compact, "█") {
		t.Errorf("compact mode missing bar: %q", compact)
	}
	if !strings.Contains(compact, "⚠") {
		t.Errorf("compact mode missing queued warning: %q", compact)
	}

	// Toggle -> table mode: shows the NAME header.
	v.ToggleView()
	table := v.GetText(false)
	if !strings.Contains(table, "NAME") {
		t.Errorf("table mode missing NAME header: %q", table)
	}

	// Toggle back -> compact again.
	v.ToggleView()
	if back := v.GetText(false); !strings.Contains(back, "█") {
		t.Errorf("toggle back to compact failed: %q", back)
	}
}

func TestClusterPoolsRenderFromCacheAfterHealth(t *testing.T) {
	v := NewClusterInfoView()
	v.UpdatePools([]models.Pool{{Name: "default", Slots: 8, OccupiedSlots: 2}})
	// A later health update must not wipe the pools section.
	v.Update(nil)
	if out := v.GetText(false); !strings.Contains(out, "Pools") {
		t.Errorf("pools section lost after health update: %q", out)
	}
}
