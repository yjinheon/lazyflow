package models

import "time"

type BackfillResponse struct {
	ID                int       `json:"id"`
	DagId             string    `json:"dag_id"`
	FromDate          time.Time `json:"from_date"`
	ToDate            time.Time `json:"to_date"`
	DagRunConf        any       `json:"dag_run_conf"`
	IsPaused          bool      `json:"is_paused"`
	ReprocessBehavior string    `json:"reprocess_behavior"`
	MaxActiveRuns     int       `json:"max_active_runs"`
	CreatedAt         time.Time `json:"created_at"`
	CompletedAt       time.Time `json:"completed_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DagDisplayName    string    `json:"dag_display_name"`
}
