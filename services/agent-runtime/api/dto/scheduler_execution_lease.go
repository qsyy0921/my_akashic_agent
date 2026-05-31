package dto

type AcquireSchedulerExecutionLeaseRequest struct {
	JobID      string            `json:"job_id"`
	HolderID   string            `json:"holder_id"`
	TTLSeconds int               `json:"ttl_seconds,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
}

type RenewSchedulerExecutionLeaseRequest struct {
	JobID      string `json:"job_id"`
	HolderID   string `json:"holder_id"`
	LeaseToken string `json:"lease_token"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

type ReleaseSchedulerExecutionLeaseRequest struct {
	JobID      string `json:"job_id"`
	HolderID   string `json:"holder_id"`
	LeaseToken string `json:"lease_token"`
	Timestamp  string `json:"timestamp,omitempty"`
}
