package dto

type CreateAgentJobRequest struct {
	JobID          string            `json:"job_id"`
	JobType        string            `json:"job_type"`
	AgentID        string            `json:"agent_id"`
	Route          ChannelDTO        `json:"route"`
	SourceEventIDs []string          `json:"source_event_ids,omitempty"`
	SourceAssetIDs []string          `json:"source_asset_ids,omitempty"`
	Payload        map[string]string `json:"payload,omitempty"`
	MaxAttempts    int               `json:"max_attempts,omitempty"`
	Timestamp      string            `json:"timestamp,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type AgentJobLeaseRequest struct {
	WorkerID   string `json:"worker_id"`
	JobType    string `json:"job_type,omitempty"`
	LeaseToken string `json:"lease_token,omitempty"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

type AgentJobLeaseWorkRequest struct {
	WorkKind    string `json:"work_kind"`
	WorkID      string `json:"work_id"`
	AggregateID string `json:"aggregate_id,omitempty"`
	Subject     string `json:"subject,omitempty"`
	WorkerID    string `json:"worker_id"`
	LeaseToken  string `json:"lease_token,omitempty"`
	TTLSeconds  int    `json:"ttl_seconds,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type AgentJobStateRequest struct {
	Timestamp    string            `json:"timestamp,omitempty"`
	LeaseToken   string            `json:"lease_token,omitempty"`
	Result       map[string]string `json:"result,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}
