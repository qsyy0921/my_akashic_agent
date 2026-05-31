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

type QueueExternalLeaseDiagnostics struct {
	Enabled          bool                              `json:"enabled"`
	SampleLimit      int                               `json:"sample_limit"`
	ExecutedTotal    int                               `json:"executed_total"`
	ErrorTotal       int                               `json:"error_total"`
	Dispositions     []QueueExternalLeaseCounter       `json:"dispositions,omitempty"`
	Reasons          []QueueExternalLeaseCounter       `json:"reasons,omitempty"`
	WorkKinds        []QueueExternalLeaseCounter       `json:"work_kinds,omitempty"`
	RecentExecutions []QueueExternalLeaseExecutionView `json:"recent_executions,omitempty"`
	Notes            []string                          `json:"notes,omitempty"`
}

type QueueExternalLeaseCounter struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
