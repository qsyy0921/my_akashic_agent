package query

type SchedulerJobView struct {
	ID              string `json:"id"`
	Trigger         string `json:"trigger"`
	Tier            string `json:"tier"`
	FireAt          string `json:"fire_at"`
	Channel         string `json:"channel"`
	ChatID          string `json:"chat_id"`
	IntervalSeconds *int   `json:"interval_seconds,omitempty"`
	CronExpr        string `json:"cron_expr,omitempty"`
	Message         string `json:"message,omitempty"`
	Prompt          string `json:"prompt,omitempty"`
	Name            string `json:"name,omitempty"`
	Timezone        string `json:"timezone"`
	CreatedAt       string `json:"created_at"`
	RunCount        int    `json:"run_count"`
	Enabled         bool   `json:"enabled"`
}

type SchedulerJobSnapshotView struct {
	Count      int    `json:"count"`
	Source     string `json:"source,omitempty"`
	SideEffect string `json:"side_effect"`
}

type SchedulerJobMutationView struct {
	Job        *SchedulerJobView `json:"job,omitempty"`
	JobID      string            `json:"job_id,omitempty"`
	Source     string            `json:"source,omitempty"`
	Created    *bool             `json:"created,omitempty"`
	Found      *bool             `json:"found,omitempty"`
	Deleted    bool              `json:"deleted"`
	SideEffect string            `json:"side_effect"`
}

type SchedulerJobDiagnosticsFilter struct {
	Limit          int
	Timestamp      string
	DueSoonSeconds int
}

type SchedulerJobDiagnosticSampleView struct {
	ID        string `json:"id"`
	Trigger   string `json:"trigger"`
	Tier      string `json:"tier"`
	Channel   string `json:"channel"`
	ChatID    string `json:"chat_id"`
	FireAt    string `json:"fire_at"`
	Status    string `json:"status"`
	RunCount  int    `json:"run_count"`
	Enabled   bool   `json:"enabled"`
	OverdueBy int    `json:"overdue_by_seconds,omitempty"`
	DueIn     int    `json:"due_in_seconds,omitempty"`
}

type SchedulerJobDiagnosticsView struct {
	SampledJobs    int                                `json:"sampled_jobs"`
	EnabledJobs    int                                `json:"enabled_jobs"`
	DisabledJobs   int                                `json:"disabled_jobs"`
	OverdueJobs    int                                `json:"overdue_jobs"`
	DueSoonJobs    int                                `json:"due_soon_jobs"`
	InstantJobs    int                                `json:"instant_jobs"`
	SoftJobs       int                                `json:"soft_jobs"`
	NextFireAt     string                             `json:"next_fire_at,omitempty"`
	JobsByTrigger  map[string]int                     `json:"jobs_by_trigger"`
	JobsByTier     map[string]int                     `json:"jobs_by_tier"`
	JobsByChannel  map[string]int                     `json:"jobs_by_channel"`
	JobsByStatus   map[string]int                     `json:"jobs_by_status"`
	Recent         []SchedulerJobDiagnosticSampleView `json:"recent"`
	DueSoonSeconds int                                `json:"due_soon_seconds"`
	SideEffect     string                             `json:"side_effect"`
}
