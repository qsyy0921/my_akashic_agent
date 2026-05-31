package schedulerjobstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/schedulerjobstore"
)

func TestSchedulerJobStorePersistsSnapshotOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scheduler-jobs.json")
	store, err := schedulerjobstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	fireAt := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	jobs := []model.SchedulerJob{
		validJob("job-a", fireAt),
		validJob("job-b", fireAt.Add(time.Hour)),
	}
	if err := store.ReplaceSchedulerJobs(context.Background(), jobs); err != nil {
		t.Fatal(err)
	}

	reopened, err := schedulerjobstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListSchedulerJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "job-a" || items[1].ID != "job-b" {
		t.Fatalf("unexpected persisted order: %+v", items)
	}

	if err := reopened.ReplaceSchedulerJobs(context.Background(), []model.SchedulerJob{jobs[1]}); err != nil {
		t.Fatal(err)
	}
	items, err = reopened.ListSchedulerJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "job-b" {
		t.Fatalf("snapshot replace should remove old jobs: %+v", items)
	}
}

func validJob(id string, fireAt time.Time) model.SchedulerJob {
	return model.SchedulerJob{
		ID:        id,
		Trigger:   "after",
		Tier:      "instant",
		FireAt:    fireAt,
		Channel:   "qq",
		ChatID:    "1049511700",
		Message:   "提醒",
		Timezone:  "Asia/Shanghai",
		CreatedAt: fireAt.Add(-time.Hour),
		Enabled:   true,
	}
}
