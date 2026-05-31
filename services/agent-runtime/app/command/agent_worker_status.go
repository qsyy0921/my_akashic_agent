package command

import "time"

type ReportAgentWorkerStatusCommand struct {
	WorkerID       string
	WorkerType     string
	Status         string
	CurrentJobID   string
	LastJobID      string
	LastError      string
	ProcessedTotal int
	FailedTotal    int
	Source         string
	Metadata       map[string]string
	Timestamp      time.Time
}
