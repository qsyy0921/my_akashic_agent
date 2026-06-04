package command

import "time"

type ReportAgentWorkerStatusCommand struct {
	WorkerID                  string
	InstanceID                string
	ReplaceExistingInstanceID string
	WorkerType                string
	Status                    string
	CurrentJobID              string
	LastJobID                 string
	LastError                 string
	ProcessedTotal            int
	FailedTotal               int
	Source                    string
	Metadata                  map[string]string
	Timestamp                 time.Time
	LeaseTTLSeconds           int
}

type CleanupStaleAgentWorkerStatusesCommand struct {
	WorkerID          string
	InstanceID        string
	Timestamp         time.Time
	StaleAfterSeconds int
}
