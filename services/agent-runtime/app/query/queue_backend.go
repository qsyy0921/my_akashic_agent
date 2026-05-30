package query

type QueueBackendView struct {
	Provider                string   `json:"provider"`
	Mode                    string   `json:"mode"`
	MigrationPhase          string   `json:"migration_phase"`
	ExternalQueueConfigured bool     `json:"external_queue_configured"`
	ExternalQueueActive     bool     `json:"external_queue_active"`
	StateStoreAuthoritative bool     `json:"state_store_authoritative"`
	LeaseOwner              string   `json:"lease_owner"`
	ConsumerModel           string   `json:"consumer_model"`
	ConsumerConcurrency     int      `json:"consumer_concurrency"`
	MaxInFlight             int      `json:"max_in_flight"`
	OutboxQueueSource       string   `json:"outbox_queue_source"`
	AgentJobQueueSource     string   `json:"agent_job_queue_source"`
	DSNConfigured           bool     `json:"dsn_configured"`
	DSNRedacted             string   `json:"dsn_redacted,omitempty"`
	RecommendedFirstBackend string   `json:"recommended_first_backend"`
	SupportedProviders      []string `json:"supported_providers"`
	Notes                   []string `json:"notes,omitempty"`
}
