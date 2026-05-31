package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
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
