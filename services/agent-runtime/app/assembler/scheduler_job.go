package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToSchedulerJobView(job model.SchedulerJob) query.SchedulerJobView {
	return query.SchedulerJobView{
		ID:              job.ID,
		Trigger:         job.Trigger,
		Tier:            job.Tier,
		FireAt:          formatSchedulerTime(job.FireAt),
		Channel:         job.Channel,
		ChatID:          job.ChatID,
		IntervalSeconds: job.IntervalSeconds,
		CronExpr:        job.CronExpr,
		Message:         job.Message,
		Prompt:          job.Prompt,
		Name:            job.Name,
		Timezone:        job.Timezone,
		CreatedAt:       formatSchedulerTime(job.CreatedAt),
		RunCount:        job.RunCount,
		Enabled:         job.Enabled,
	}
}

func ToSchedulerJobViews(jobs []model.SchedulerJob) []query.SchedulerJobView {
	views := make([]query.SchedulerJobView, 0, len(jobs))
	for _, job := range jobs {
		views = append(views, ToSchedulerJobView(job))
	}
	return views
}

func formatSchedulerTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
