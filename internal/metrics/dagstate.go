// Package metrics holds pure, side-effect-free aggregation helpers over Airflow
// domain models. Keeping these out of the UI and API layers makes them trivially
package metrics

import (
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func recency(r models.DAGRun) time.Time {
	if !r.RunAfter.IsZero() {
		return r.RunAfter
	}
	return r.LogicalDate
}

func laterThan(a, b models.DAGRun) bool {
	ar, br := recency(a), recency(b)
	if ar.Equal(br) {
		return a.StartDate.After(b.StartDate)
	}
	return ar.After(br)
}

func RollupLatestState(runs []models.DAGRun) map[string]string {
	latest := make(map[string]models.DAGRun)
	for _, r := range runs {
		cur, ok := latest[r.DagId]
		if !ok || laterThan(r, cur) {
			latest[r.DagId] = r
		}
	}
	out := make(map[string]string, len(latest))
	for id, r := range latest {
		out[id] = r.State
	}
	return out
}

func CountByState(rollup map[string]string) (running, success, failed int) {
	for _, st := range rollup {
		switch st {
		case "running":
			running++
		case "success":
			success++
		case "failed":
			failed++
		}
	}
	return running, success, failed
}

func CountWindowStates(runs []models.DAGRun, since time.Time) (running, success, failed int) {
	for _, r := range runs {
		if r.State == "running" {
			running++
			continue
		}
		if !since.IsZero() && recency(r).Before(since) {
			continue
		}
		switch r.State {
		case "success":
			success++
		case "failed":
			failed++
		}
	}
	return running, success, failed
}
