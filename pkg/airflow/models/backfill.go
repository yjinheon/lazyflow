package models

import "time"

// Backfill is the canonical backfill record used across the app.
// (BackfillResponse is kept as an alias for compatibility with existing create code paths.)
type Backfill struct {
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

	// Derived (not in API response, computed by ListBackfills caller).
	TotalRuns     int `json:"-"`
	CompletedRuns int `json:"-"`
	FailedRuns    int `json:"-"`
	RunningRuns   int `json:"-"`
}

// BackfillResponse remains the API response type for POST /backfills (create).
// We keep it as an alias so existing call sites keep compiling.
type BackfillResponse = Backfill

type BackfillCollection struct {
	Backfills    []Backfill `json:"backfills"`
	TotalEntries int        `json:"total_entries"`
}

type DryRunResponse struct {
	LogicalDates []string `json:"logical_dates"`
}

// State derives a display state. Order matters: paused > completed > running.
func (b *Backfill) State() string {
	if b.IsPaused {
		return "paused"
	}
	if !b.CompletedAt.IsZero() {
		return "completed"
	}
	return "running"
}
