package query

import "time"

type KnowledgePipelineDiagnosticsFilter struct {
	Limit             int
	StaleAfterSeconds int
	Now               time.Time
}

type KnowledgePipelineJobStageView struct {
	JobType                 string        `json:"job_type"`
	Pending                 int           `json:"pending"`
	Leased                  int           `json:"leased"`
	Running                 int           `json:"running"`
	Active                  int           `json:"active"`
	OldestPendingAgeSeconds int           `json:"oldest_pending_age_seconds,omitempty"`
	OldestActiveAgeSeconds  int           `json:"oldest_active_age_seconds,omitempty"`
	StaleActiveLeases       int           `json:"stale_active_leases,omitempty"`
	ExpiredActiveLeases     int           `json:"expired_active_leases,omitempty"`
	SampledJobs             int           `json:"sampled_jobs"`
	HighPressure            bool          `json:"high_pressure"`
	PressureReason          string        `json:"pressure_reason,omitempty"`
	FreshnessStatus         string        `json:"freshness_status,omitempty"`
	FreshnessReason         string        `json:"freshness_reason,omitempty"`
	LatestJob               *AgentJobView `json:"latest_job,omitempty"`
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

type KnowledgePipelineRagIngestSnapshotView struct {
	MessageCount   int    `json:"message_count"`
	DocumentCount  int    `json:"document_count"`
	StartSeq       int    `json:"start_seq"`
	EndSeq         int    `json:"end_seq"`
	ParseRequested bool   `json:"parse_requested"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	DisplayName    string `json:"display_name,omitempty"`
}

type KnowledgePipelineRagIndexStateView struct {
	Ready           bool   `json:"ready"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
	MessageCount    int    `json:"message_count,omitempty"`
	DocumentCount   int    `json:"document_count"`
	StartSeq        int    `json:"start_seq,omitempty"`
	EndSeq          int    `json:"end_seq,omitempty"`
	LatestSourceSeq int    `json:"latest_source_seq,omitempty"`
	SourceLag       int    `json:"source_lag,omitempty"`
	ParseRequested  bool   `json:"parse_requested"`
	LastIngestAt    string `json:"last_ingest_at,omitempty"`
	AgeSeconds      int    `json:"age_seconds,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
}

type KnowledgePipelineRagDatasetView struct {
	DatasetID       string                                  `json:"dataset_id"`
	DisplayName     string                                  `json:"display_name,omitempty"`
	Configured      bool                                    `json:"configured"`
	RuntimeObserved bool                                    `json:"runtime_observed"`
	JobStage        KnowledgePipelineJobStageView           `json:"job_stage"`
	Checkpoint      *KnowledgeCheckpointView                `json:"checkpoint,omitempty"`
	CheckpointLag   *KnowledgePipelineCheckpointLagView     `json:"checkpoint_lag,omitempty"`
	IngestSnapshot  *KnowledgePipelineRagIngestSnapshotView `json:"ingest_snapshot,omitempty"`
	RagIndexState   KnowledgePipelineRagIndexStateView      `json:"rag_index_state"`
	Status          string                                  `json:"status"`
	Reasons         []string                                `json:"reasons,omitempty"`
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
	RagDatasets         []KnowledgePipelineRagDatasetView   `json:"rag_datasets,omitempty"`
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
