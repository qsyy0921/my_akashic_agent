package query

type AgentJobFilter struct {
	JobType string
	Status  string
	Limit   int
}

type AgentJobRouteView struct {
	Kind             string `json:"kind"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
}

type AgentJobView struct {
	JobID          string            `json:"job_id"`
	JobType        string            `json:"job_type"`
	AgentID        string            `json:"agent_id"`
	Route          AgentJobRouteView `json:"route"`
	SourceEventIDs []string          `json:"source_event_ids"`
	SourceAssetIDs []string          `json:"source_asset_ids"`
	Payload        map[string]string `json:"payload,omitempty"`
	Status         string            `json:"status"`
	Attempts       int               `json:"attempts"`
	MaxAttempts    int               `json:"max_attempts"`
	LeaseOwner     string            `json:"lease_owner,omitempty"`
	LeaseExpiresAt string            `json:"lease_expires_at,omitempty"`
	Result         map[string]string `json:"result,omitempty"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

