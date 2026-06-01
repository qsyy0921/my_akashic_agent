package query

type QueueTopologyView struct {
	Provider                string                  `json:"provider"`
	Mode                    string                  `json:"mode"`
	MigrationPhase          string                  `json:"migration_phase"`
	SelectedProvider        string                  `json:"selected_provider,omitempty"`
	RecommendedProvider     string                  `json:"recommended_provider,omitempty"`
	StateStoreAuthoritative bool                    `json:"state_store_authoritative"`
	ExternalQueueActive     bool                    `json:"external_queue_active"`
	ExternalLeaseReady      bool                    `json:"external_lease_ready"`
	ExecutionScope          string                  `json:"execution_scope,omitempty"`
	Nodes                   []QueueTopologyNodeView `json:"nodes"`
	Edges                   []QueueTopologyEdgeView `json:"edges"`
	WorkKinds               []QueueTopologyWorkKind `json:"work_kinds"`
	Blockers                []string                `json:"blockers,omitempty"`
	Notes                   []string                `json:"notes,omitempty"`
	SideEffect              string                  `json:"side_effect"`
}

type QueueTopologyNodeView struct {
	ID         string            `json:"id"`
	Label      string            `json:"label"`
	Kind       string            `json:"kind"`
	Status     string            `json:"status"`
	Owner      string            `json:"owner,omitempty"`
	Active     bool              `json:"active"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type QueueTopologyEdgeView struct {
	From           string   `json:"from"`
	To             string   `json:"to"`
	WorkKind       string   `json:"work_kind"`
	QueueSource    string   `json:"queue_source"`
	ExecutionOwner string   `json:"execution_owner"`
	AckOwner       string   `json:"ack_owner"`
	Status         string   `json:"status"`
	GateState      string   `json:"gate_state,omitempty"`
	Blockers       []string `json:"blockers,omitempty"`
}

type QueueTopologyWorkKind struct {
	WorkKind       string   `json:"work_kind"`
	QueueSource    string   `json:"queue_source"`
	ExecutionOwner string   `json:"execution_owner"`
	AckOwner       string   `json:"ack_owner"`
	GateState      string   `json:"gate_state,omitempty"`
	Allowed        bool     `json:"allowed"`
	Blockers       []string `json:"blockers,omitempty"`
	Notes          []string `json:"notes,omitempty"`
}
