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
	ErrorKind    string
	ErrorMessage string
	Timestamp    time.Time
}

type RetryOutboxCommand struct {
	EventID   string
	Timestamp time.Time
}

type LeaseNextOutboxCommand struct {
	WorkerID   string
	TTLSeconds int
	Timestamp  time.Time
}

type LeaseOutboxDeliveryCommand struct {
	EventID    string
	WorkerID   string
	TTLSeconds int
	Timestamp  time.Time
}
