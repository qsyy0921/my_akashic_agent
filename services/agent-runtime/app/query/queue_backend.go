package query

type QueueBackendView struct {
	Provider                string                         `json:"provider"`
	Mode                    string                         `json:"mode"`
	MigrationPhase          string                         `json:"migration_phase"`
	Stream                  string                         `json:"stream,omitempty"`
	SubjectPrefix           string                         `json:"subject_prefix,omitempty"`
	ExternalQueueConfigured bool                           `json:"external_queue_configured"`
	ExternalQueueActive     bool                           `json:"external_queue_active"`
	StateStoreAuthoritative bool                           `json:"state_store_authoritative"`
	LeaseOwner              string                         `json:"lease_owner"`
	ConsumerModel           string                         `json:"consumer_model"`
	ConsumerConcurrency     int                            `json:"consumer_concurrency"`
	MaxInFlight             int                            `json:"max_in_flight"`
	OutboxQueueSource       string                         `json:"outbox_queue_source"`
	AgentJobQueueSource     string                         `json:"agent_job_queue_source"`
	DSNConfigured           bool                           `json:"dsn_configured"`
	DSNRedacted             string                         `json:"dsn_redacted,omitempty"`
	RecommendedFirstBackend string                         `json:"recommended_first_backend"`
	SupportedProviders      []string                       `json:"supported_providers"`
	Notes                   []string                       `json:"notes,omitempty"`
	ShadowPublish           *QueueShadowPublishDiagnostics `json:"shadow_publish,omitempty"`
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
