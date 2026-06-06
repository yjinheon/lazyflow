package models

// Pool represents an Airflow worker pool (/api/v2/pools).
type Pool struct {
	Name           string `json:"name"`
	Slots          int    `json:"slots"`
	OccupiedSlots  int    `json:"occupied_slots"`
	RunningSlots   int    `json:"running_slots"`
	QueuedSlots    int    `json:"queued_slots"`
	OpenSlots      int    `json:"open_slots"`
	ScheduledSlots int    `json:"scheduled_slots"`
	DeferredSlots  int    `json:"deferred_slots"`
}

// PoolCollection is the list response from /api/v2/pools.
type PoolCollection struct {
	Pools        []Pool `json:"pools"`
	TotalEntries int    `json:"total_entries"`
}
