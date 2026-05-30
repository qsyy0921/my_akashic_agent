package query

type QueueExternalLeaseExecutionView struct {
	WorkKind    string `json:"work_kind"`
	WorkID      string `json:"work_id"`
	AggregateID string `json:"aggregate_id,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
	StateStatus string `json:"state_status,omitempty"`
	Attempts    int    `json:"attempts,omitempty"`
	ExecutedAt  string `json:"executed_at"`
}
