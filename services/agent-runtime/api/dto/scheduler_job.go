package dto

type SchedulerJobDTO struct {
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
	Timezone        string `json:"timezone,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	RunCount        int    `json:"run_count"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

type SchedulerJobSnapshotRequest struct {
	Jobs   []SchedulerJobDTO `json:"jobs"`
	Source string            `json:"source,omitempty"`
}
