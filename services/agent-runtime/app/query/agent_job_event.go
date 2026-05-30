package query

type AgentJobEventFilter struct {
	JobID     string
	JobType   string
	EventType string
	Limit     int
}

type AgentJobEventView struct {
	EventID        string            `json:"event_id"`
	JobID          string            `json:"job_id"`
	JobType        string            `json:"job_type"`
	EventType      string            `json:"event_type"`
	Status         string            `json:"status"`
	Attempt        int               `json:"attempt"`
	MaxAttempts    int               `json:"max_attempts"`
	LeaseOwner     string            `json:"lease_owner,omitempty"`
	LeaseExpiresAt string            `json:"lease_expires_at,omitempty"`
	OccurredAt     string            `json:"occurred_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}
