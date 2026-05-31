package query

type AgentJobExternalLeaseReadinessView struct {
	Ready                   bool                         `json:"ready"`
	Reason                  string                       `json:"reason"`
	ExternalLeaseReady      bool                         `json:"external_lease_ready"`
	AgentJobResultAckReady  bool                         `json:"agent_job_result_ack_ready"`
	StrictLeaseTokenEnabled bool                         `json:"strict_lease_token_enabled"`
	AgentJobWorkerReady     bool                         `json:"agent_job_worker_ready"`
	ExecutionOwner          string                       `json:"execution_owner,omitempty"`
	QueueProvider           string                       `json:"queue_provider,omitempty"`
	QueueMode               string                       `json:"queue_mode,omitempty"`
	ExecutionScope          string                       `json:"execution_scope,omitempty"`
	AllowedWorkKinds        []string                     `json:"allowed_work_kinds,omitempty"`
	BlockedWorkKinds        []QueueExternalLeaseBlock    `json:"blocked_work_kinds,omitempty"`
	RequiredChecks          []QueueExternalLeaseCheck    `json:"required_checks,omitempty"`
	WorkerCoverage          []AgentJobWorkerCoverageView `json:"worker_coverage,omitempty"`
	Blockers                []string                     `json:"blockers,omitempty"`
	Attributes              map[string]string            `json:"attributes,omitempty"`
	Notes                   []string                     `json:"notes,omitempty"`
	SideEffect              string                       `json:"side_effect"`
}
