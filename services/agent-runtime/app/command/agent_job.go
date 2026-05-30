package command

import "time"

type CreateAgentJobCommand struct {
	JobID          string
	JobType        string
	AgentID        string
	Route          ChannelCommand
	SourceEventIDs []string
	SourceAssetIDs []string
	Payload        map[string]string
	MaxAttempts    int
	Timestamp      time.Time
	Metadata       map[string]string
}

type AgentJobLeaseCommand struct {
	JobID      string
	WorkerID   string
	LeaseToken string
	TTLSeconds int
	Timestamp  time.Time
}

type AgentJobLeaseNextCommand struct {
	WorkerID   string
	JobType    string
	LeaseToken string
	TTLSeconds int
	Timestamp  time.Time
}

type RenewAgentJobLeaseCommand struct {
	JobID      string
	LeaseToken string
	TTLSeconds int
	Timestamp  time.Time
}

type MarkAgentJobRunningCommand struct {
	JobID      string
	LeaseToken string
	Timestamp  time.Time
}

type CompleteAgentJobCommand struct {
	JobID      string
	LeaseToken string
	Result     map[string]string
	Timestamp  time.Time
}

type FailAgentJobCommand struct {
	JobID        string
	LeaseToken   string
	ErrorMessage string
	Timestamp    time.Time
}

type RetryAgentJobCommand struct {
	JobID     string
	Timestamp time.Time
}

type CancelAgentJobCommand struct {
	JobID     string
	Timestamp time.Time
}
