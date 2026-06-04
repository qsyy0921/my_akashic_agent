package query

type QueueBackendView struct {
	Provider                                    string                                    `json:"provider"`
	Mode                                        string                                    `json:"mode"`
	MigrationPhase                              string                                    `json:"migration_phase"`
	Stream                                      string                                    `json:"stream,omitempty"`
	SubjectPrefix                               string                                    `json:"subject_prefix,omitempty"`
	ExternalQueueConfigured                     bool                                      `json:"external_queue_configured"`
	ExternalQueueActive                         bool                                      `json:"external_queue_active"`
	StateStoreAuthoritative                     bool                                      `json:"state_store_authoritative"`
	LeaseOwner                                  string                                    `json:"lease_owner"`
	ConsumerModel                               string                                    `json:"consumer_model"`
	ConsumerConcurrency                         int                                       `json:"consumer_concurrency"`
	MaxInFlight                                 int                                       `json:"max_in_flight"`
	OutboxQueueSource                           string                                    `json:"outbox_queue_source"`
	AgentJobQueueSource                         string                                    `json:"agent_job_queue_source"`
	OutboxExecutionOwner                        string                                    `json:"outbox_execution_owner"`
	OutboxExecutionScope                        string                                    `json:"outbox_execution_scope,omitempty"`
	OutboxAllowedKinds                          []string                                  `json:"outbox_allowed_kinds,omitempty"`
	OutboxAllowedKindsByAccount                 map[string][]string                       `json:"outbox_allowed_kinds_by_account,omitempty"`
	OutboxAllowedKindsByAccountConversationType map[string]map[string][]string            `json:"outbox_allowed_kinds_by_account_conversation_type,omitempty"`
	OutboxAllowedKindsByAccountConversationID   map[string]map[string]map[string][]string `json:"outbox_allowed_kinds_by_account_conversation_id,omitempty"`
	AgentJobExecutionOwner                      string                                    `json:"agent_job_execution_owner"`
	DSNConfigured                               bool                                      `json:"dsn_configured"`
	DSNRedacted                                 string                                    `json:"dsn_redacted,omitempty"`
	RecommendedFirstBackend                     string                                    `json:"recommended_first_backend"`
	SupportedProviders                          []string                                  `json:"supported_providers"`
	ProviderCapabilities                        []QueueProviderCapabilityView             `json:"provider_capabilities,omitempty"`
	SelectedProviderCapability                  *QueueProviderCapabilityView              `json:"selected_provider_capability,omitempty"`
	Notes                                       []string                                  `json:"notes,omitempty"`
	ShadowPublish                               *QueueShadowPublishDiagnostics            `json:"shadow_publish,omitempty"`
	DualReadCompare                             *QueueDualReadDiagnostics                 `json:"dual_read_compare,omitempty"`
	ExternalLease                               *QueueExternalLeaseGate                   `json:"external_lease,omitempty"`
}

type QueueProviderCapabilityView struct {
	Provider                    string   `json:"provider"`
	Label                       string   `json:"label,omitempty"`
	Status                      string   `json:"status"`
	Recommended                 bool     `json:"recommended"`
	RecommendedPhase            string   `json:"recommended_phase,omitempty"`
	Implemented                 bool     `json:"implemented"`
	SupportsStateStoreLease     bool     `json:"supports_state_store_lease"`
	SupportsExternalQueue       bool     `json:"supports_external_queue"`
	SupportsShadowPublish       bool     `json:"supports_shadow_publish"`
	SupportsDualReadCompare     bool     `json:"supports_dual_read_compare"`
	SupportsExternalLease       bool     `json:"supports_external_lease"`
	SupportsAgentJobResultAck   bool     `json:"supports_agent_job_result_ack"`
	SupportsConcurrentConsumers bool     `json:"supports_concurrent_consumers"`
	SupportsDelayedNack         bool     `json:"supports_delayed_nack"`
	ConsumerModel               string   `json:"consumer_model,omitempty"`
	AdapterBoundary             string   `json:"adapter_boundary,omitempty"`
	Notes                       []string `json:"notes,omitempty"`
	Blockers                    []string `json:"blockers,omitempty"`
}

type QueueShadowPublishDiagnostics struct {
	Enabled                  bool                        `json:"enabled"`
	SampleLimit              int                         `json:"sample_limit"`
	AttemptTotal             int                         `json:"attempt_total"`
	SucceededTotal           int                         `json:"succeeded_total"`
	FailedTotal              int                         `json:"failed_total"`
	StateStoreTotal          int                         `json:"state_store_total"`
	EventStreamTotal         int                         `json:"event_stream_total"`
	StateMinusPublishedTotal int                         `json:"state_minus_published_total"`
	EventMinusPublishedTotal int                         `json:"event_minus_published_total"`
	LastPublishedAt          string                      `json:"last_published_at,omitempty"`
	LastFailedAt             string                      `json:"last_failed_at,omitempty"`
	LastError                string                      `json:"last_error,omitempty"`
	WorkKinds                []QueuePublishWorkKindStats `json:"work_kinds,omitempty"`
	Subjects                 []QueuePublishSubjectStats  `json:"subjects,omitempty"`
	Notes                    []string                    `json:"notes,omitempty"`
}

type QueuePublishWorkKindStats struct {
	WorkKind            string `json:"work_kind"`
	AttemptCount        int    `json:"attempt_count"`
	SucceededCount      int    `json:"succeeded_count"`
	FailedCount         int    `json:"failed_count"`
	StateStoreCount     int    `json:"state_store_count"`
	EventStreamCount    int    `json:"event_stream_count"`
	StateMinusPublished int    `json:"state_minus_published"`
	EventMinusPublished int    `json:"event_minus_published"`
	LastPublishedAt     string `json:"last_published_at,omitempty"`
	LastFailedAt        string `json:"last_failed_at,omitempty"`
	LastError           string `json:"last_error,omitempty"`
}

type QueuePublishSubjectStats struct {
	Subject         string `json:"subject"`
	WorkKind        string `json:"work_kind"`
	AttemptCount    int    `json:"attempt_count"`
	SucceededCount  int    `json:"succeeded_count"`
	FailedCount     int    `json:"failed_count"`
	LastPublishedAt string `json:"last_published_at,omitempty"`
	LastFailedAt    string `json:"last_failed_at,omitempty"`
	LastError       string `json:"last_error,omitempty"`
}

type QueueDualReadDiagnostics struct {
	Enabled           bool                           `json:"enabled"`
	SampleLimit       int                            `json:"sample_limit"`
	ComparedTotal     int                            `json:"compared_total"`
	MatchedTotal      int                            `json:"matched_total"`
	MismatchedTotal   int                            `json:"mismatched_total"`
	LastComparedAt    string                         `json:"last_compared_at,omitempty"`
	LastMismatchAt    string                         `json:"last_mismatch_at,omitempty"`
	Reasons           []QueueCompareReasonStats      `json:"reasons,omitempty"`
	WorkKinds         []QueueCompareWorkKindStats    `json:"work_kinds,omitempty"`
	RecentComparisons []QueueCandidateComparisonView `json:"recent_comparisons,omitempty"`
	Notes             []string                       `json:"notes,omitempty"`
}

type QueueCompareReasonStats struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type QueueCompareWorkKindStats struct {
	WorkKind      string `json:"work_kind"`
	ComparedCount int    `json:"compared_count"`
	MatchedCount  int    `json:"matched_count"`
	MismatchCount int    `json:"mismatch_count"`
}

type QueueCandidateComparisonView struct {
	WorkKind    string `json:"work_kind"`
	WorkID      string `json:"work_id"`
	AggregateID string `json:"aggregate_id,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Matched     bool   `json:"matched"`
	Reason      string `json:"reason"`
	StateStatus string `json:"state_status,omitempty"`
	ObservedAt  string `json:"observed_at"`
	ComparedAt  string `json:"compared_at"`
}

type QueueExternalLeaseGate struct {
	Enabled          bool                           `json:"enabled"`
	CutoverRequested bool                           `json:"cutover_requested"`
	AllowExecution   bool                           `json:"allow_execution"`
	GateState        string                         `json:"gate_state"`
	ExecutionScope   string                         `json:"execution_scope"`
	AckPolicy        string                         `json:"ack_policy"`
	NackPolicy       string                         `json:"nack_policy"`
	RetryPolicy      string                         `json:"retry_policy"`
	DeadLetterPolicy string                         `json:"dead_letter_policy"`
	RollbackPolicy   string                         `json:"rollback_policy"`
	AllowedWorkKinds []string                       `json:"allowed_work_kinds,omitempty"`
	BlockedWorkKinds []QueueExternalLeaseBlock      `json:"blocked_work_kinds,omitempty"`
	RequiredChecks   []QueueExternalLeaseCheck      `json:"required_checks"`
	Blockers         []string                       `json:"blockers,omitempty"`
	Notes            []string                       `json:"notes,omitempty"`
	Diagnostics      *QueueExternalLeaseDiagnostics `json:"diagnostics,omitempty"`
}

type QueueExternalLeaseCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type QueueExternalLeaseBlock struct {
	WorkKind       string `json:"work_kind"`
	Reason         string `json:"reason"`
	RequiredChange string `json:"required_change,omitempty"`
}
