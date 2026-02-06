package models

import "time"

type TaskInstance struct {
	TaskId          string     `json:"task_id"`
	TaskDisplayName string     `json:"task_display_name"`
	DagId           string     `json:"dag_id"`
	RunId           string     `json:"dag_run_id"`
	State           string     `json:"state"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	Duration        float64    `json:"duration"`
	TryNumber       int        `json:"try_number"`
	Operator        string     `json:"operator_name"`
	Pool            string     `json:"pool"`
	Queue           string     `json:"queue"`
	Hostname        string     `json:"hostname"`
}

type TaskInstanceCollection struct {
	TaskInstances []TaskInstance `json:"task_instances"`
	TotalEntries  int           `json:"total_entries"`
}
