package command

import "time"

type CreateImageJobCommand struct {
	RequestID   string
	Requester   ChannelCommand
	RequesterID string
	Prompt      string
	Provider    string
	Model       string
	Size        string
	Count       int
	MaxAttempts int
	Timestamp   time.Time
	Metadata    map[string]string
}

type MarkImageJobRunningCommand struct {
	JobID     string
	Timestamp time.Time
}

type CompleteImageJobCommand struct {
	JobID     string
	Results   []AttachmentCommand
	Timestamp time.Time
	Metadata  map[string]string
}

type FailImageJobCommand struct {
	JobID        string
	ErrorMessage string
	Timestamp    time.Time
}

