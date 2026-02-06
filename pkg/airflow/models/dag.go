package models

import "time"

// DAG represents an Airflow 3 DAG from /api/v2/dags
type DAG struct {
	DagId                string    `json:"dag_id"`
	DagDisplayName       string    `json:"dag_display_name"`
	IsPaused             bool      `json:"is_paused"`
	IsStale              bool      `json:"is_stale"`
	Fileloc              string    `json:"fileloc"`
	RelativeFileloc      string    `json:"relative_fileloc"`
	Owners               []string  `json:"owners"`
	Description          *string   `json:"description"`
	TimetableDescription string    `json:"timetable_description"`
	Tags                 []DagTag  `json:"tags"`
	LastParsedTime       time.Time `json:"last_parsed_time"`
	MaxActiveTasks       int       `json:"max_active_tasks"`
	MaxActiveRuns        int       `json:"max_active_runs"`
	HasImportErrors      bool      `json:"has_import_errors"`
	BundleName           string    `json:"bundle_name"`

	// Populated by the UI layer, not from API directly
	LastRunState string `json:"-"`
}

// Schedule returns a display string for the DAG schedule.
func (d DAG) Schedule() string {
	if d.TimetableDescription != "" {
		return d.TimetableDescription
	}
	return "-"
}

// DisplayName returns the best available name.
func (d DAG) DisplayName() string {
	if d.DagDisplayName != "" {
		return d.DagDisplayName
	}
	return d.DagId
}

type DagTag struct {
	Name string `json:"name"`
}

// DAGCollection represents the response from /api/v2/dags
type DAGCollection struct {
	DAGs         []DAG `json:"dags"`
	TotalEntries int   `json:"total_entries"`
}
