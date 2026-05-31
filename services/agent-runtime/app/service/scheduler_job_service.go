package service

import (
	"context"
	"errors"
	"sort"
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

func (s *SchedulerJobService) GetSchedulerJobDiagnostics(ctx context.Context, filter query.SchedulerJobDiagnosticsFilter) (query.SchedulerJobDiagnosticsView, error) {
	if s == nil || s.repository == nil {
		return query.SchedulerJobDiagnosticsView{}, errors.New("scheduler job service requires repository")
	}
	jobs, err := s.repository.ListSchedulerJobs(ctx)
	if err != nil {
		return query.SchedulerJobDiagnosticsView{}, err
	}
	return schedulerJobDiagnostics(jobs, filter), nil
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

func schedulerJobDiagnostics(jobs []model.SchedulerJob, filter query.SchedulerJobDiagnosticsFilter) query.SchedulerJobDiagnosticsView {
	now := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(filter.Timestamp)); err == nil {
		now = parsed.UTC()
	}
	dueSoonSeconds := filter.DueSoonSeconds
	if dueSoonSeconds <= 0 {
		dueSoonSeconds = 5 * 60
	}
	if dueSoonSeconds > 24*60*60 {
		dueSoonSeconds = 24 * 60 * 60
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	items := append([]model.SchedulerJob(nil), jobs...)
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.FireAt.Equal(right.FireAt) {
			return left.ID < right.ID
		}
		if left.FireAt.IsZero() {
			return false
		}
		if right.FireAt.IsZero() {
			return true
		}
		return left.FireAt.Before(right.FireAt)
	})

	view := query.SchedulerJobDiagnosticsView{
		JobsByTrigger:  map[string]int{"at": 0, "after": 0, "every": 0},
		JobsByTier:     map[string]int{"instant": 0, "soft": 0},
		JobsByChannel:  make(map[string]int),
		JobsByStatus:   map[string]int{"disabled": 0, "overdue": 0, "due_soon": 0, "future": 0},
		Recent:         make([]query.SchedulerJobDiagnosticSampleView, 0, minSchedulerJobDiagnosticLimit(limit, len(items))),
		DueSoonSeconds: dueSoonSeconds,
		SideEffect:     "none",
	}

	for _, job := range items {
		status, dueIn, overdueBy := schedulerJobTimingStatus(job, now, time.Duration(dueSoonSeconds)*time.Second)
		view.SampledJobs++
		if job.Enabled {
			view.EnabledJobs++
		} else {
			view.DisabledJobs++
		}
		if status == "overdue" {
			view.OverdueJobs++
		}
		if status == "due_soon" {
			view.DueSoonJobs++
		}
		if job.Tier == "instant" {
			view.InstantJobs++
		}
		if job.Tier == "soft" {
			view.SoftJobs++
		}
		view.JobsByTrigger[job.Trigger]++
		view.JobsByTier[job.Tier]++
		if strings.TrimSpace(job.Channel) != "" {
			view.JobsByChannel[job.Channel]++
		}
		view.JobsByStatus[status]++
		if view.NextFireAt == "" && job.Enabled && !job.FireAt.IsZero() {
			view.NextFireAt = job.FireAt.UTC().Format(time.RFC3339Nano)
		}
		if len(view.Recent) < limit {
			sample := query.SchedulerJobDiagnosticSampleView{
				ID:       job.ID,
				Trigger:  job.Trigger,
				Tier:     job.Tier,
				Channel:  job.Channel,
				ChatID:   job.ChatID,
				FireAt:   job.FireAt.UTC().Format(time.RFC3339Nano),
				Status:   status,
				RunCount: job.RunCount,
				Enabled:  job.Enabled,
			}
			if dueIn > 0 {
				sample.DueIn = dueIn
			}
			if overdueBy > 0 {
				sample.OverdueBy = overdueBy
			}
			view.Recent = append(view.Recent, sample)
		}
	}
	return view
}

func schedulerJobTimingStatus(job model.SchedulerJob, now time.Time, dueSoonWindow time.Duration) (string, int, int) {
	if !job.Enabled {
		return "disabled", 0, 0
	}
	if job.FireAt.IsZero() {
		return "future", 0, 0
	}
	fireAt := job.FireAt.UTC()
	if fireAt.Before(now) || fireAt.Equal(now) {
		return "overdue", 0, int(now.Sub(fireAt).Seconds())
	}
	dueIn := int(fireAt.Sub(now).Seconds())
	if fireAt.Before(now.Add(dueSoonWindow)) || fireAt.Equal(now.Add(dueSoonWindow)) {
		return "due_soon", dueIn, 0
	}
	return "future", dueIn, 0
}

func minSchedulerJobDiagnosticLimit(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
