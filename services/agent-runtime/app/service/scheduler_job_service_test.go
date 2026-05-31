package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestSchedulerJobServiceReplacesAndListsJobs(t *testing.T) {
	ctx := context.Background()
	service := appservice.NewSchedulerJobService(memory.NewStore())
	fireAt := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	view, err := service.ReplaceSchedulerJobs(ctx, command.ReplaceSchedulerJobsCommand{
		Source: "python_scheduler",
		Jobs: []command.SchedulerJobCommand{{
			ID:        "job-1",
			Trigger:   "after",
			Tier:      "instant",
			FireAt:    fireAt,
			Channel:   "qq",
			ChatID:    "1049511700",
			Message:   "提醒",
			Timezone:  "Asia/Shanghai",
			CreatedAt: fireAt.Add(-time.Hour),
			Enabled:   true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Count != 1 || view.SideEffect != "runtime_state_write" {
		t.Fatalf("unexpected snapshot view: %+v", view)
	}

	items, err := service.ListSchedulerJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "job-1" || items[0].FireAt != fireAt.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected jobs: %+v", items)
	}
}

func TestSchedulerJobServiceRejectsInvalidSnapshot(t *testing.T) {
	service := appservice.NewSchedulerJobService(memory.NewStore())
	_, err := service.ReplaceSchedulerJobs(context.Background(), command.ReplaceSchedulerJobsCommand{
		Jobs: []command.SchedulerJobCommand{{
			ID:      "job-1",
			Trigger: "after",
			Tier:    "instant",
			Channel: "qq",
			ChatID:  "1049511700",
			Message: "missing fire_at",
			Enabled: true,
		}},
	})
	if err == nil {
		t.Fatal("expected missing fire_at to fail")
	}
}

func TestSchedulerJobServiceDiagnosticsSummarizesTiming(t *testing.T) {
	ctx := context.Background()
	service := appservice.NewSchedulerJobService(memory.NewStore())
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	_, err := service.ReplaceSchedulerJobs(ctx, command.ReplaceSchedulerJobsCommand{
		Jobs: []command.SchedulerJobCommand{
			{
				ID:        "overdue",
				Trigger:   "after",
				Tier:      "instant",
				FireAt:    now.Add(-time.Minute),
				Channel:   "qq",
				ChatID:    "1049511700",
				Message:   "已过期",
				CreatedAt: now.Add(-time.Hour),
				Enabled:   true,
			},
			{
				ID:        "soon",
				Trigger:   "every",
				Tier:      "soft",
				FireAt:    now.Add(2 * time.Minute),
				Channel:   "telegram",
				ChatID:    "100",
				Prompt:    "总结",
				CreatedAt: now.Add(-time.Hour),
				Enabled:   true,
			},
			{
				ID:        "disabled",
				Trigger:   "at",
				Tier:      "instant",
				FireAt:    now.Add(time.Hour),
				Channel:   "qq",
				ChatID:    "2365524513",
				Message:   "禁用",
				CreatedAt: now.Add(-time.Hour),
				Enabled:   false,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	view, err := service.GetSchedulerJobDiagnostics(ctx, query.SchedulerJobDiagnosticsFilter{
		Timestamp:      now.Format(time.RFC3339Nano),
		DueSoonSeconds: 300,
		Limit:          10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.SampledJobs != 3 || view.EnabledJobs != 2 || view.DisabledJobs != 1 {
		t.Fatalf("unexpected totals: %+v", view)
	}
	if view.OverdueJobs != 1 || view.DueSoonJobs != 1 {
		t.Fatalf("unexpected timing totals: %+v", view)
	}
	if view.JobsByTrigger["every"] != 1 || view.JobsByTier["soft"] != 1 || view.JobsByChannel["qq"] != 2 {
		t.Fatalf("unexpected buckets: %+v", view)
	}
	if len(view.Recent) != 3 || view.Recent[0].ID != "overdue" || view.Recent[0].Status != "overdue" {
		t.Fatalf("unexpected samples: %+v", view.Recent)
	}
	if view.SideEffect != "none" {
		t.Fatalf("diagnostics must be read-only: %+v", view)
	}
}
