package command

import "time"

type AcquireReceiverLeaseCommand struct {
	ReceiverID  string
	Kind        string
	ChannelName string
	AccountID   string
	HolderID    string
	TTLSeconds  int
	Metadata    map[string]string
	Timestamp   time.Time
}

type RenewReceiverLeaseCommand struct {
	ReceiverID string
	HolderID   string
	LeaseToken string
	TTLSeconds int
	Timestamp  time.Time
}

type ReleaseReceiverLeaseCommand struct {
	ReceiverID string
	HolderID   string
	LeaseToken string
	Timestamp  time.Time
}

type CleanupExpiredReceiverLeasesCommand struct {
	Timestamp time.Time
}
