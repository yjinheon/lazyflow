package models

// HealthInfo represents the response from /api/v2/monitor/health
type HealthInfo struct {
	Metadatabase *HealthStatus `json:"metadatabase"`
	Scheduler    *HealthStatus `json:"scheduler"`
	Triggerer    *HealthStatus `json:"triggerer"`
	DagProcessor *HealthStatus `json:"dag_processor"`
}

type HealthStatus struct {
	Status                       string `json:"status"`
	LatestSchedulerHeartbeat     string `json:"latest_scheduler_heartbeat,omitempty"`
	LatestTriggererHeartbeat     string `json:"latest_triggerer_heartbeat,omitempty"`
	LatestDagProcessorHeartbeat  string `json:"latest_dag_processor_heartbeat,omitempty"`
}
