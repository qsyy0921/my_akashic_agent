package command

import "time"

type MarkOutboxDispatchingCommand struct {
	EventID   string
	Timestamp time.Time
}

type MarkOutboxSucceededCommand struct {
	EventID   string
	Timestamp time.Time
}

type MarkOutboxFailedCommand struct {
	EventID      string
	ErrorMessage string
	Timestamp    time.Time
}

type RetryOutboxCommand struct {
	EventID   string
	Timestamp time.Time
}

