package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// Bucket is one time slice on the X axis.
type Bucket struct {
	Start, End time.Time
}

// Cell is one rendered character.
type Cell struct {
	Char  rune
	State string // "queue" | "run" | "empty"
	Color string // theme color token; empty = no markup
}

// deref returns the value at t, or time zero if t is nil.
func deref(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// ComputeBuckets divides [tMin..tMax] into `width` equal slices.
// tMax is supplied explicitly so callers can pass `time.Now()` for running runs.
func ComputeBuckets(tis []models.TaskInstance, width int, tMax time.Time) ([]Bucket, time.Time, time.Time) {
	if width <= 0 || len(tis) == 0 {
		return nil, time.Time{}, time.Time{}
	}
	tMin := earliestQueued(tis)
	if tMin.IsZero() {
		return nil, time.Time{}, time.Time{}
	}
	if tMax.IsZero() || !tMax.After(tMin) {
		tMax = tMin.Add(time.Second)
	}
	step := tMax.Sub(tMin) / time.Duration(width)
	buckets := make([]Bucket, width)
	for i := range width {
		buckets[i].Start = tMin.Add(step * time.Duration(i))
		if i == width-1 {
			buckets[i].End = tMax
		} else {
			buckets[i].End = tMin.Add(step * time.Duration(i+1))
		}
	}
	return buckets, tMin, tMax
}

func earliestQueued(tis []models.TaskInstance) time.Time {
	var t time.Time
	for _, ti := range tis {
		q := deref(ti.QueuedDttm)
		if q.IsZero() {
			continue
		}
		if t.IsZero() || q.Before(t) {
			t = q
		}
	}
	return t
}

// RenderCells produces a Cell slice (queue/run/empty) aligned with buckets.
// `now` is used as the end time for tasks still running.
func RenderCells(ti models.TaskInstance, buckets []Bucket, now time.Time) []Cell {
	qStart := deref(ti.QueuedDttm)
	qEnd := deref(ti.StartDate)
	rStart := deref(ti.StartDate)
	rEnd := deref(ti.EndDate)
	if ti.StartDate != nil && rEnd.IsZero() && ti.State == "running" {
		rEnd = now
	}
	out := make([]Cell, len(buckets))
	for i, b := range buckets {
		switch {
		case overlap(b.Start, b.End, rStart, rEnd):
			out[i] = Cell{Char: '█', State: "run", Color: stateColor(ti.State)}
		case overlap(b.Start, b.End, qStart, qEnd):
			out[i] = Cell{Char: '▒', State: "queue", Color: "queued"}
		default:
			out[i] = Cell{Char: ' ', State: "empty"}
		}
	}
	return out
}

func overlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	if bStart.IsZero() || bEnd.IsZero() {
		return false
	}
	return !aEnd.Before(bStart) && !bEnd.Before(aStart)
}

// stateColor maps Airflow state → theme color token (consumed by EmitRLE).
func stateColor(state string) string {
	switch state {
	case "success":
		return "success"
	case "failed":
		return "failed"
	case "running":
		return "running"
	case "upstream_failed":
		return "upstream"
	case "skipped", "removed":
		return "skipped"
	default:
		return "running"
	}
}

// EmitRLE collapses adjacent same-color cells into one markup span.
// `bold` adds "::b" to every color span (used by critical-path highlighting).
func EmitRLE(cells []Cell, bold bool) string {
	var b strings.Builder
	i := 0
	for i < len(cells) {
		j := i
		for j < len(cells) && cells[j].Color == cells[i].Color {
			j++
		}
		if cells[i].Color != "" {
			hex := theme.GanttMarkupColor(cells[i].Color)
			tag := hex
			if bold {
				tag += "::b"
			}
			fmt.Fprintf(&b, "[%s]", tag)
			for k := i; k < j; k++ {
				b.WriteRune(cells[k].Char)
			}
			b.WriteString("[-]")
		} else {
			for k := i; k < j; k++ {
				b.WriteRune(cells[k].Char)
			}
		}
		i = j
	}
	return b.String()
}

// effectiveDuration returns the duration used by critical-path calc.
func effectiveDuration(ti models.TaskInstance, now time.Time) time.Duration {
	switch ti.State {
	case "success", "failed", "upstream_failed":
		if ti.EndDate == nil || ti.StartDate == nil {
			return 0
		}
		return ti.EndDate.Sub(*ti.StartDate)
	case "running":
		if ti.StartDate == nil {
			return 0
		}
		return now.Sub(*ti.StartDate)
	default:
		return 0
	}
}
