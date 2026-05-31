package query

type AgentWorkerStatusFilter struct {
	StaleAfterSeconds int
}

type AgentWorkerStatusView struct {
	WorkerID       string            `json:"worker_id"`
	WorkerType     string            `json:"worker_type"`
	Status         string            `json:"status"`
	CurrentJobID   string            `json:"current_job_id,omitempty"`
	LastJobID      string            `json:"last_job_id,omitempty"`
	LastError      string            `json:"last_error,omitempty"`
	ProcessedTotal int               `json:"processed_total"`
	FailedTotal    int               `json:"failed_total"`
	Source         string            `json:"source,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	UpdatedAt      string            `json:"updated_at"`
	Stale          bool              `json:"stale"`
}

type AgentWorkerStatusesView struct {
	Workers    []AgentWorkerStatusView `json:"workers"`
	Totals     map[string]int          `json:"totals"`
	Notes      []string                `json:"notes,omitempty"`
	SideEffect string                  `json:"side_effect"`
}
