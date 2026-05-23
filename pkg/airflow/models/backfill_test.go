package models

import (
	"testing"
	"time"
)

func TestBackfill_State(t *testing.T) {
	cases := []struct {
		name string
		bf   Backfill
		want string
	}{
		{"paused", Backfill{IsPaused: true}, "paused"},
		{"completed", Backfill{CompletedAt: time.Now()}, "completed"},
		{"running", Backfill{}, "running"},
		{"paused beats completed", Backfill{IsPaused: true, CompletedAt: time.Now()}, "paused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.bf.State(); got != tc.want {
				t.Fatalf("State()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestBackfillCollection_zeroValue(t *testing.T) {
	var c BackfillCollection
	if c.Backfills != nil {
		t.Fatal("Backfills should be nil zero value")
	}
}
