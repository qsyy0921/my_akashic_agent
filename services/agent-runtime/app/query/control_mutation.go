package query

type ControlMutationAuditFilter struct {
	TargetKind string
	TargetID   string
	Status     string
	ApprovalID string
	Limit      int
}

type ControlMutationAuditView struct {
	MutationID  string            `json:"mutation_id"`
	TargetKind  string            `json:"target_kind"`
	TargetID    string            `json:"target_id"`
	Action      string            `json:"action"`
	Status      string            `json:"status"`
	OperatorID  string            `json:"operator_id"`
	ApprovalID  string            `json:"approval_id"`
	Reason      string            `json:"reason,omitempty"`
	RollbackOf  string            `json:"rollback_of,omitempty"`
	RollbackRef string            `json:"rollback_ref,omitempty"`
	CreatedAt   string            `json:"created_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ControlMutationAuditsView struct {
	Mutations  []ControlMutationAuditView `json:"mutations"`
	Totals     map[string]int             `json:"totals"`
	Notes      []string                   `json:"notes,omitempty"`
	SideEffect string                     `json:"side_effect"`
}

type ControlMutationPreflight struct {
	TargetKind string
	TargetID   string
	Action     string
	OperatorID string
	ApprovalID string
}

type ControlMutationSuggestedAuditView struct {
	TargetKind string            `json:"target_kind"`
	TargetID   string            `json:"target_id"`
	Action     string            `json:"action"`
	Status     string            `json:"status"`
	OperatorID string            `json:"operator_id"`
	ApprovalID string            `json:"approval_id"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type ControlMutationPreflightView struct {
	Ready          bool                               `json:"ready"`
	Reason         string                             `json:"reason"`
	Blockers       []string                           `json:"blockers,omitempty"`
	TargetKind     string                             `json:"target_kind"`
	TargetID       string                             `json:"target_id"`
	Action         string                             `json:"action"`
	OperatorID     string                             `json:"operator_id"`
	ApprovalID     string                             `json:"approval_id"`
	ApprovalCheck  OperatorApprovalCheckView          `json:"approval_check"`
	SuggestedAudit *ControlMutationSuggestedAuditView `json:"suggested_audit,omitempty"`
	Notes          []string                           `json:"notes,omitempty"`
	SideEffect     string                             `json:"side_effect"`
}
