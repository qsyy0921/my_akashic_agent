package command

import "time"

type SchedulerJobCommand struct {
	ID              string
	Trigger         string
	Tier            string
	FireAt          time.Time
	Channel         string
	ChatID          string
	IntervalSeconds *int
	CronExpr        string
	Message         string
	Prompt          string
	Name            string
	Timezone        string
	CreatedAt       time.Time
	RunCount        int
	Enabled         bool
}

type ReplaceSchedulerJobsCommand struct {
	Jobs   []SchedulerJobCommand
	Source string
}

type UpsertSchedulerJobCommand struct {
	Job    SchedulerJobCommand
	Source string
}

type DeleteSchedulerJobCommand struct {
	ID     string
	Source string
}
