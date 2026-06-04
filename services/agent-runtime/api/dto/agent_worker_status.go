package dto

type AgentWorkerStatusRequest struct {
	WorkerID                  string            `json:"worker_id"`
	InstanceID                string            `json:"instance_id"`
	ReplaceExistingInstanceID string            `json:"replace_existing_instance_id"`
	WorkerType                string            `json:"worker_type"`
	Status                    string            `json:"status"`
	CurrentJobID              string            `json:"current_job_id"`
	LastJobID                 string            `json:"last_job_id"`
	LastError                 string            `json:"last_error"`
	ProcessedTotal            int               `json:"processed_total"`
	FailedTotal               int               `json:"failed_total"`
	Source                    string            `json:"source"`
	Metadata                  map[string]string `json:"metadata"`
	Timestamp                 string            `json:"timestamp"`
	LeaseTTLSeconds           int               `json:"lease_ttl_seconds"`
}

type CleanupStaleAgentWorkerStatusesRequest struct {
	WorkerID          string `json:"worker_id"`
	InstanceID        string `json:"instance_id"`
	Timestamp         string `json:"timestamp"`
	StaleAfterSeconds int    `json:"stale_after_seconds"`
}
