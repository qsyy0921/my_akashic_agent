package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type SchedulerJobService struct {
	repository outport.SchedulerJobRepository
}

func NewSchedulerJobService(repository outport.SchedulerJobRepository) *SchedulerJobService {
	return &SchedulerJobService{repository: repository}
}

func (s *SchedulerJobService) ReplaceSchedulerJobs(ctx context.Context, cmd command.ReplaceSchedulerJobsCommand) (query.SchedulerJobSnapshotView, error) {
	if s == nil || s.repository == nil {
		return query.SchedulerJobSnapshotView{}, errors.New("scheduler job service requires repository")
	}
	jobs := make([]model.SchedulerJob, 0, len(cmd.Jobs))
	for _, item := range cmd.Jobs {
		job := model.SchedulerJob{
			ID:              strings.TrimSpace(item.ID),
			Trigger:         strings.TrimSpace(item.Trigger),
			Tier:            strings.TrimSpace(item.Tier),
			FireAt:          utcOrZero(item.FireAt),
			Channel:         strings.TrimSpace(item.Channel),
			ChatID:          strings.TrimSpace(item.ChatID),
			IntervalSeconds: item.IntervalSeconds,
			CronExpr:        strings.TrimSpace(item.CronExpr),
			Message:         item.Message,
			Prompt:          item.Prompt,
			Name:            strings.TrimSpace(item.Name),
			Timezone:        timezoneOrUTC(item.Timezone),
			CreatedAt:       createdAtOrNow(item.CreatedAt),
			RunCount:        item.RunCount,
			Enabled:         item.Enabled,
		}
		if err := job.Validate(); err != nil {
			return query.SchedulerJobSnapshotView{}, err
		}
		jobs = append(jobs, job)
	}
	if err := s.repository.ReplaceSchedulerJobs(ctx, jobs); err != nil {
		return query.SchedulerJobSnapshotView{}, err
	}
	return query.SchedulerJobSnapshotView{
		Count:      len(jobs),
		Source:     strings.TrimSpace(cmd.Source),
		SideEffect: "runtime_state_write",
	}, nil
}

func (s *SchedulerJobService) ListSchedulerJobs(ctx context.Context) ([]query.SchedulerJobView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("scheduler job service requires repository")
	}
	jobs, err := s.repository.ListSchedulerJobs(ctx)
	if err != nil {
		return nil, err
	}
	return assembler.ToSchedulerJobViews(jobs), nil
}

func createdAtOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func utcOrZero(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC()
}

func timezoneOrUTC(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "UTC"
	}
	return value
}
