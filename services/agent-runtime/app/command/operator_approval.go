package command

import "time"

type RecordOperatorApprovalCommand struct {
	ApprovalID string
	TargetKind string
	TargetID   string
	Decision   string
	OperatorID string
	Reason     string
	ExpiresAt  time.Time
	Timestamp  time.Time
	Metadata   map[string]string
}
