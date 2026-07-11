package metrics

import (
	"math"
	"sort"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// isTerminal reports whether a run reached a state usable for reliability and
// latency statistics. running/queued/other are excluded.
func isTerminal(state string) bool {
	return state == "success" || state == "failed"
}

// SuccessFailed counts terminal runs by outcome; running/queued are ignored.
func SuccessFailed(runs []models.DAGRun) (success, failed int) {
	for _, r := range runs {
		switch r.State {
		case "success":
			success++
		case "failed":
			failed++
		}
	}
	return success, failed
}

// FailureStreak counts consecutive failed runs from the most recent terminal
// run backwards. running/queued runs are skipped and do not break the streak.
func FailureStreak(runs []models.DAGRun) int {
	ordered := make([]models.DAGRun, len(runs))
	copy(ordered, runs)
	sort.Slice(ordered, func(i, j int) bool {
		return recency(ordered[i]).After(recency(ordered[j]))
	})
	streak := 0
	for _, r := range ordered {
		switch r.State {
		case "failed":
			streak++
		case "success":
			return streak
		default:
			continue // running/queued: skip
		}
	}
	return streak
}

// FlakyTasks counts task instances that succeeded only after a retry
// (try_number > 1 and final state success).
func FlakyTasks(tasks []models.TaskInstance) int {
	n := 0
	for _, ti := range tasks {
		if ti.TryNumber > 1 && ti.State == "success" {
			n++
		}
	}
	return n
}

// Percentiles returns nearest-rank p50/p90/p99 of terminal-run durations.
// Runs without a positive duration (e.g. still running) are excluded.
// Returns zeros when there are no samples.
func Percentiles(runs []models.DAGRun) (p50, p90, p99 time.Duration) {
	durs := make([]time.Duration, 0, len(runs))
	for _, r := range runs {
		if !isTerminal(r.State) {
			continue
		}
		if d := r.Duration(); d > 0 {
			durs = append(durs, d)
		}
	}
	if len(durs) == 0 {
		return 0, 0, 0
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	return nearestRank(durs, 50), nearestRank(durs, 90), nearestRank(durs, 99)
}

// nearestRank returns the p-th percentile (0<p<=100) of a sorted, non-empty
// slice using the nearest-rank method.
func nearestRank(sorted []time.Duration, p int) time.Duration {
	rank := int(math.Ceil(float64(p) / 100 * float64(len(sorted))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

// AvgQueueTime returns the mean queue wait (start - queued) across tasks that
// have both timestamps. Returns 0 when no task qualifies.
func AvgQueueTime(tasks []models.TaskInstance) time.Duration {
	var total time.Duration
	n := 0
	for _, ti := range tasks {
		if ti.QueuedDttm == nil || ti.StartDate == nil {
			continue
		}
		if ti.StartDate.Before(*ti.QueuedDttm) {
			continue
		}
		total += ti.StartDate.Sub(*ti.QueuedDttm)
		n++
	}
	if n == 0 {
		return 0
	}
	return total / time.Duration(n)
}

// TrendDirection classifies success-rate movement across a window.
type TrendDirection int

const (
	TrendNA TrendDirection = iota
	TrendImproving
	TrendFlat
	TrendDegrading
)

func (t TrendDirection) String() string {
	switch t {
	case TrendImproving:
		return "improving"
	case TrendFlat:
		return "flat"
	case TrendDegrading:
		return "degrading"
	default:
		return "n/a"
	}
}

// Trend compares the success rate of the most recent half of terminal runs
// against the older half. A swing beyond ±10 percentage points is
// improving/degrading; within the band is flat. Fewer than 4 terminal runs in
// either half yields TrendNA.
func Trend(runs []models.DAGRun) TrendDirection {
	terminal := make([]models.DAGRun, 0, len(runs))
	for _, r := range runs {
		if isTerminal(r.State) {
			terminal = append(terminal, r)
		}
	}
	sort.Slice(terminal, func(i, j int) bool {
		return recency(terminal[i]).Before(recency(terminal[j]))
	})
	n := len(terminal)
	half := n / 2
	if half < 4 {
		return TrendNA
	}
	older := terminal[:half]
	recent := terminal[n-half:]
	delta := successRate(recent) - successRate(older)
	switch {
	case delta > 0.10:
		return TrendImproving
	case delta < -0.10:
		return TrendDegrading
	default:
		return TrendFlat
	}
}

func successRate(runs []models.DAGRun) float64 {
	s, f := SuccessFailed(runs)
	total := s + f
	if total == 0 {
		return 0
	}
	return float64(s) / float64(total)
}
