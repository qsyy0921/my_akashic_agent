package command

import "time"

type RecordControlMutationAuditCommand struct {
	MutationID  string
	TargetKind  string
	TargetID    string
	Action      string
	Status      string
	OperatorID  string
	ApprovalID  string
	Reason      string
	RollbackOf  string
	RollbackRef string
	Metadata    map[string]string
	Timestamp   time.Time
}
