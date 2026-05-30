package query

import "time"

type KnowledgeWorkerDiagnosticsFilter struct {
	Limit             int
	StaleAfterSeconds int
	Now               time.Time
}

type KnowledgeWorkerDiagnosticsView struct {
	GeneratedAt            string                          `json:"generated_at"`
	StaleAfterSeconds      int                             `json:"stale_after_seconds"`
	SampledJobLimit        int                             `json:"sampled_job_limit"`
	SampledCheckpointLimit int                             `json:"sampled_checkpoint_limit"`
	Totals                 map[string]int                  `json:"totals"`
	Workers                []KnowledgeWorkerDiagnosticView `json:"workers"`
}

type KnowledgeWorkerDiagnosticView struct {
	JobType          string                    `json:"job_type"`
	CheckpointPrefix string                    `json:"checkpoint_prefix,omitempty"`
	StatusCounts     map[string]int            `json:"status_counts"`
	StaleLeaseCount  int                       `json:"stale_lease_count"`
	LeaseableCount   int                       `json:"leaseable_count"`
	RecentJobs       []AgentJobView            `json:"recent_jobs"`
	Checkpoints      []KnowledgeCheckpointView `json:"checkpoints,omitempty"`
	LatestJob        *AgentJobView             `json:"latest_job,omitempty"`
	LatestCheckpoint *KnowledgeCheckpointView  `json:"latest_checkpoint,omitempty"`
}
