package models

import "time"

type DAGRun struct {
	DagId       string    `json:"dag_id"`
	RunId       string    `json:"dag_run_id"`
	State       string    `json:"state"`
	LogicalDate time.Time `json:"logical_date"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	RunType     string    `json:"run_type"`
	Conf        any       `json:"conf"`
	Note        string    `json:"note"`
}

// Duration returns the elapsed time of the run.
func (r DAGRun) Duration() time.Duration {
	end := r.EndDate
	if end.IsZero() {
		end = time.Now()
	}
	if r.StartDate.IsZero() {
		return 0
	}
	return end.Sub(r.StartDate)
}

type DAGRunCollection struct {
	DAGRuns      []DAGRun `json:"dag_runs"`
	TotalEntries int      `json:"total_entries"`
}
