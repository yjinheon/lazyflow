package models

type Task struct {
	TaskId            string   `json:"task_id"`
	Owner             string   `json:"owner"`
	Operator          string   `json:"operator_name"`
	Pool              string   `json:"pool"`
	Queue             string   `json:"queue"`
	DownstreamTaskIds []string `json:"downstream_task_ids"`
	UpstreamTaskIds   []string `json:"upstream_task_ids"`
	TriggerRule       string   `json:"trigger_rule"`
	Retries           float64  `json:"retries"`
}

type TaskCollection struct {
	Tasks        []Task `json:"tasks"`
	TotalEntries int    `json:"total_entries"`
}
