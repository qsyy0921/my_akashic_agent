package command

import "time"

type ReportReceiverStatusCommand struct {
	ReceiverID  string
	Kind        string
	ChannelName string
	AccountID   string
	Endpoint    string
	Status      string
	Reason      string
	LastError   string
	Source      string
	Metadata    map[string]string
	Timestamp   time.Time
}

type CleanupStaleReceiverStatusesCommand struct {
	ReceiverID        string
	Timestamp         time.Time
	StaleAfterSeconds int
}
