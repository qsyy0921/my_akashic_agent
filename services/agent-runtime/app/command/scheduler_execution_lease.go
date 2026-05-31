package command

import "time"

type AcquireSchedulerExecutionLeaseCommand struct {
	JobID      string
	HolderID   string
	TTLSeconds int
	Metadata   map[string]string
	Timestamp  time.Time
}

type RenewSchedulerExecutionLeaseCommand struct {
	JobID      string
	HolderID   string
	LeaseToken string
	TTLSeconds int
	Timestamp  time.Time
}

type ReleaseSchedulerExecutionLeaseCommand struct {
	JobID      string
	HolderID   string
	LeaseToken string
	Timestamp  time.Time
}
