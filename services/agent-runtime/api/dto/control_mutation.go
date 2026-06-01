package dto

type RecordControlMutationAuditRequest struct {
	MutationID  string            `json:"mutation_id"`
	TargetKind  string            `json:"target_kind"`
	TargetID    string            `json:"target_id"`
	Action      string            `json:"action"`
	Status      string            `json:"status"`
	OperatorID  string            `json:"operator_id"`
	ApprovalID  string            `json:"approval_id"`
	Reason      string            `json:"reason"`
	RollbackOf  string            `json:"rollback_of"`
	RollbackRef string            `json:"rollback_ref"`
	Metadata    map[string]string `json:"metadata"`
	Timestamp   string            `json:"timestamp"`
}
