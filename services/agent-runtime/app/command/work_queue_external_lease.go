package command

import "time"

type ExecuteWorkQueueLeaseCommand struct {
	WorkKind         string
	WorkID           string
	AggregateID      string
	Subject          string
	WorkerID         string
	LeaseTTLSeconds  int
	ObservedAt       time.Time
	Timestamp        time.Time
	ChannelByAccount map[string]string
	Metadata         map[string]string
}
