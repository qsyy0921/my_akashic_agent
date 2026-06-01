package dto

type RecordOperatorApprovalRequest struct {
	ApprovalID string            `json:"approval_id,omitempty"`
	TargetKind string            `json:"target_kind"`
	TargetID   string            `json:"target_id"`
	Decision   string            `json:"decision"`
	OperatorID string            `json:"operator_id"`
	Reason     string            `json:"reason,omitempty"`
	ExpiresAt  string            `json:"expires_at,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
