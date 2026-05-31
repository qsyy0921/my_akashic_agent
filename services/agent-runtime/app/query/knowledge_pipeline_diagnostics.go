package query

import "time"

type KnowledgePipelineDiagnosticsFilter struct {
	Limit             int
	StaleAfterSeconds int
	Now               time.Time
}

type KnowledgePipelineJobStageView struct {
	JobType        string        `json:"job_type"`
	Pending        int           `json:"pending"`
	Leased         int           `json:"leased"`
	Running        int           `json:"running"`
	Active         int           `json:"active"`
	SampledJobs    int           `json:"sampled_jobs"`
	HighPressure   bool          `json:"high_pressure"`
	PressureReason string        `json:"pressure_reason,omitempty"`
	LatestJob      *AgentJobView `json:"latest_job,omitempty"`
}

type KnowledgePipelineCheckpointLagView struct {
	CheckpointID    string `json:"checkpoint_id"`
	Cursor          int    `json:"cursor"`
	LatestSourceSeq int    `json:"latest_source_seq,omitempty"`
	Lag             int    `json:"lag,omitempty"`
	AgeSeconds      int    `json:"age_seconds,omitempty"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

type KnowledgePipelineView struct {
	TargetID            string                              `json:"target_id"`
	Channel             ObserveTargetChannelView            `json:"channel"`
	Enabled             bool                                `json:"enabled"`
	ObserveOnly         bool                                `json:"observe_only"`
	CaptureStatus       string                              `json:"capture_status"`
	ReceiverConnected   bool                                `json:"receiver_connected"`
	CaptureBlockers     []string                            `json:"capture_blockers,omitempty"`
	SequencedEvents     int                                 `json:"sequenced_events"`
	SourceSeqKnown      bool                                `json:"source_seq_known"`
	LatestSourceSeq     int                                 `json:"latest_source_seq,omitempty"`
	GroupMemory         KnowledgePipelineJobStageView       `json:"group_memory"`
	RagIngest           KnowledgePipelineJobStageView       `json:"rag_ingest"`
	MemoryCheckpoint    *KnowledgeCheckpointView            `json:"memory_checkpoint,omitempty"`
	RagCheckpoints      []KnowledgeCheckpointView           `json:"rag_checkpoints,omitempty"`
	MemoryCheckpointLag *KnowledgePipelineCheckpointLagView `json:"memory_checkpoint_lag,omitempty"`
	RagCheckpointLagMax *KnowledgePipelineCheckpointLagView `json:"rag_checkpoint_lag_max,omitempty"`
	WorkerCoverage      []AgentJobWorkerCoverageView        `json:"worker_coverage,omitempty"`
	Status              string                              `json:"status"`
	Reasons             []string                            `json:"reasons,omitempty"`
}

type KnowledgePipelineDiagnosticsView struct {
	GeneratedAt            string                  `json:"generated_at"`
	StaleAfterSeconds      int                     `json:"stale_after_seconds"`
	SampledJobLimit        int                     `json:"sampled_job_limit"`
	SampledCheckpointLimit int                     `json:"sampled_checkpoint_limit"`
	Totals                 map[string]int          `json:"totals"`
	Pipelines              []KnowledgePipelineView `json:"pipelines"`
	Notes                  []string                `json:"notes,omitempty"`
	SideEffect             string                  `json:"side_effect"`
}
