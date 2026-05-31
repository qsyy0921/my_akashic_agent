package query

type KnowledgeJobPlannerRunView struct {
	Timestamp          string   `json:"timestamp"`
	Bucket             int64    `json:"bucket"`
	Targets            int      `json:"targets"`
	CreatedOrExisting  int      `json:"created_or_existing"`
	SuppressedByDedupe int      `json:"suppressed_by_dedupe"`
	GroupMemoryJobs    int      `json:"group_memory_jobs"`
	RagIngestJobs      int      `json:"rag_ingest_jobs"`
	SkippedTargets     int      `json:"skipped_targets"`
	Groups             []string `json:"groups,omitempty"`
	SideEffect         string   `json:"side_effect"`
}

type KnowledgeJobPlannerPreviewView struct {
	Timestamp       string                              `json:"timestamp"`
	Bucket          int64                               `json:"bucket"`
	IntervalSeconds int                                 `json:"interval_seconds"`
	Targets         int                                 `json:"targets"`
	SkippedTargets  int                                 `json:"skipped_targets"`
	TotalJobs       int                                 `json:"total_jobs"`
	GroupMemoryJobs int                                 `json:"group_memory_jobs"`
	RagIngestJobs   int                                 `json:"rag_ingest_jobs"`
	Groups          []string                            `json:"groups,omitempty"`
	Plans           []KnowledgeJobPlannerTargetPlanView `json:"plans"`
	Skipped         []KnowledgeJobPlannerSkippedView    `json:"skipped,omitempty"`
	SideEffect      string                              `json:"side_effect"`
}

type KnowledgeJobPlannerTargetPlanView struct {
	TargetID string                           `json:"target_id"`
	Channel  ObserveTargetChannelView         `json:"channel"`
	Datasets []string                         `json:"datasets,omitempty"`
	Jobs     []KnowledgeJobPlannerJobPlanView `json:"jobs"`
	Metadata map[string]string                `json:"metadata,omitempty"`
}

type KnowledgeJobPlannerJobPlanView struct {
	JobID       string            `json:"job_id"`
	JobType     string            `json:"job_type"`
	AgentID     string            `json:"agent_id"`
	Route       AgentJobRouteView `json:"route"`
	Payload     map[string]string `json:"payload,omitempty"`
	DedupeKey   string            `json:"dedupe_key"`
	MaxAttempts int               `json:"max_attempts"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type KnowledgeJobPlannerSkippedView struct {
	TargetID string                   `json:"target_id"`
	Channel  ObserveTargetChannelView `json:"channel"`
	Reason   string                   `json:"reason"`
}

type KnowledgeJobPlannerReadinessView struct {
	Ready                  bool                           `json:"ready"`
	Reason                 string                         `json:"reason"`
	PlannerEnabled         bool                           `json:"planner_enabled"`
	PlannerRunning         bool                           `json:"planner_running"`
	KnowledgeWorkerReady   bool                           `json:"knowledge_worker_ready"`
	KnowledgeWorkerActive  int                            `json:"knowledge_worker_active"`
	KnowledgeWorkerStale   int                            `json:"knowledge_worker_stale"`
	KnowledgeWorkerFailed  int                            `json:"knowledge_worker_failed"`
	KnowledgeWorkerStopped int                            `json:"knowledge_worker_stopped"`
	Blockers               []string                       `json:"blockers,omitempty"`
	Preview                KnowledgeJobPlannerPreviewView `json:"preview"`
	Attributes             map[string]string              `json:"attributes,omitempty"`
	Notes                  []string                       `json:"notes,omitempty"`
	SideEffect             string                         `json:"side_effect"`
}
