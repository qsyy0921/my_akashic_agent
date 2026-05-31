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
