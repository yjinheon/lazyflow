package cache

import (
	"sort"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func mergeDAGRuns(existing, incoming []models.DAGRun) []models.DAGRun {
	byKey := make(map[string]models.DAGRun, len(existing)+len(incoming))
	for _, r := range existing {
		byKey[r.DagId+"/"+r.RunId] = r
	}
	for _, r := range incoming {
		byKey[r.DagId+"/"+r.RunId] = r
	}
	out := make([]models.DAGRun, 0, len(byKey))
	for _, r := range byKey {
		out = append(out, r)
	}
	sortDAGRuns(out)
	return out
}

func mergeTaskInstances(existing, incoming []models.TaskInstance) []models.TaskInstance {
	byKey := make(map[string]models.TaskInstance, len(existing)+len(incoming))
	for _, ti := range existing {
		byKey[ti.DagId+"/"+ti.RunId+"/"+ti.TaskId] = ti
	}
	for _, ti := range incoming {
		byKey[ti.DagId+"/"+ti.RunId+"/"+ti.TaskId] = ti
	}
	out := make([]models.TaskInstance, 0, len(byKey))
	for _, ti := range byKey {
		out = append(out, ti)
	}
	sortTaskInstances(out)
	return out
}

func filterDAGRuns(runs []models.DAGRun, since time.Time, limit int) ([]models.DAGRun, bool) {
	out := make([]models.DAGRun, 0, len(runs))
	for _, r := range runs {
		if !since.IsZero() && runTime(r).Before(since) {
			continue
		}
		out = append(out, r)
	}
	sortDAGRuns(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, len(out) > 0
}

func filterTaskInstances(tasks []models.TaskInstance, since time.Time, limit int) ([]models.TaskInstance, bool) {
	out := make([]models.TaskInstance, 0, len(tasks))
	for _, ti := range tasks {
		if !since.IsZero() && taskTime(ti).Before(since) {
			continue
		}
		out = append(out, ti)
	}
	sortTaskInstances(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, len(out) > 0
}

func dashboardRows(runs []models.DAGRun, tasks []models.TaskInstance, since time.Time, limit int) []DagDashboardRow {
	type agg struct {
		row             DagDashboardRow
		durationSamples int
		durationTotal   time.Duration
		queueSamples    int
		queueTotal      time.Duration
	}

	rows := make(map[string]*agg)
	for _, r := range runs {
		ts := runTime(r)
		if !since.IsZero() && ts.Before(since) {
			continue
		}
		a := rows[r.DagId]
		if a == nil {
			a = &agg{row: DagDashboardRow{DagId: r.DagId}}
			rows[r.DagId] = a
		}
		a.row.Runs++
		switch r.State {
		case "success":
			a.row.Success++
		case "failed":
			a.row.Failed++
		case "running":
			a.row.Running++
		case "queued":
			a.row.Queued++
		}
		if ts.After(a.row.LastRunAfter) {
			a.row.LastRunAfter = ts
			a.row.LastState = r.State
		}
		if d := r.Duration(); d > 0 {
			a.durationSamples++
			a.durationTotal += d
			if d > a.row.MaxDuration {
				a.row.MaxDuration = d
			}
		}
	}

	for _, ti := range tasks {
		ts := taskTime(ti)
		if !since.IsZero() && ts.Before(since) {
			continue
		}
		a := rows[ti.DagId]
		if a == nil {
			a = &agg{row: DagDashboardRow{DagId: ti.DagId}}
			rows[ti.DagId] = a
		}
		if ti.State == "failed" {
			a.row.FailedTasks++
		}
		if ti.TryNumber > 1 {
			a.row.RetriedTasks++
		}
		if q := queueDuration(ti); q > 0 {
			a.queueSamples++
			a.queueTotal += q
		}
	}

	out := make([]DagDashboardRow, 0, len(rows))
	for _, a := range rows {
		if a.durationSamples > 0 {
			a.row.AvgDuration = a.durationTotal / time.Duration(a.durationSamples)
		}
		if a.queueSamples > 0 {
			a.row.AvgQueueTime = a.queueTotal / time.Duration(a.queueSamples)
		}
		out = append(out, a.row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Failed != out[j].Failed {
			return out[i].Failed > out[j].Failed
		}
		return out[i].DagId < out[j].DagId
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func sortDAGRuns(runs []models.DAGRun) {
	sort.Slice(runs, func(i, j int) bool {
		return runTime(runs[i]).After(runTime(runs[j]))
	})
}

func sortTaskInstances(tasks []models.TaskInstance) {
	sort.Slice(tasks, func(i, j int) bool {
		return taskTime(tasks[i]).After(taskTime(tasks[j]))
	})
}

func runTime(r models.DAGRun) time.Time {
	if !r.RunAfter.IsZero() {
		return r.RunAfter
	}
	if !r.LogicalDate.IsZero() {
		return r.LogicalDate
	}
	if !r.StartDate.IsZero() {
		return r.StartDate
	}
	if !r.EndDate.IsZero() {
		return r.EndDate
	}
	return time.Time{}
}

func taskTime(ti models.TaskInstance) time.Time {
	if ti.StartDate != nil && !ti.StartDate.IsZero() {
		return *ti.StartDate
	}
	if ti.QueuedDttm != nil && !ti.QueuedDttm.IsZero() {
		return *ti.QueuedDttm
	}
	if ti.EndDate != nil && !ti.EndDate.IsZero() {
		return *ti.EndDate
	}
	return time.Time{}
}

func queueDuration(ti models.TaskInstance) time.Duration {
	if ti.QueuedDttm == nil || ti.StartDate == nil {
		return 0
	}
	if ti.StartDate.Before(*ti.QueuedDttm) {
		return 0
	}
	return ti.StartDate.Sub(*ti.QueuedDttm)
}
