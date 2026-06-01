package query

type OperatorApprovalFilter struct {
	TargetKind string
	TargetID   string
	Decision   string
	Limit      int
}

type OperatorApprovalView struct {
	ApprovalID string            `json:"approval_id"`
	TargetKind string            `json:"target_kind"`
	TargetID   string            `json:"target_id"`
	Decision   string            `json:"decision"`
	OperatorID string            `json:"operator_id"`
	Reason     string            `json:"reason,omitempty"`
	Active     bool              `json:"active"`
	ExpiresAt  string            `json:"expires_at,omitempty"`
	CreatedAt  string            `json:"created_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type OperatorApprovalsView struct {
	Approvals  []OperatorApprovalView `json:"approvals"`
	Totals     map[string]int         `json:"totals"`
	Notes      []string               `json:"notes,omitempty"`
	SideEffect string                 `json:"side_effect"`
}
