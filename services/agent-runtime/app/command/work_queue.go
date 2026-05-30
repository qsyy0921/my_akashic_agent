package command

import "time"

type CompareWorkQueueCandidateCommand struct {
	WorkKind    string
	WorkID      string
	AggregateID string
	Subject     string
	Status      string
	ObservedAt  time.Time
	Metadata    map[string]string
}
