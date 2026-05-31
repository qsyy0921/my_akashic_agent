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
