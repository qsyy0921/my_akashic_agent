package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type stubAgentWorkerStatusRepository struct {
	items   []model.AgentWorkerStatus
	deleted []string
}

var _ outport.AgentWorkerStatusRepository = (*stubAgentWorkerStatusRepository)(nil)

func (s *stubAgentWorkerStatusRepository) SaveAgentWorkerStatus(_ context.Context, status model.AgentWorkerStatus) error {
	replaced := false
	for index, item := range s.items {
		if item.WorkerID == status.WorkerID {
			s.items[index] = status
			replaced = true
			break
		}
	}
	if !replaced {
		s.items = append(s.items, status)
	}
	return nil
}

func (s *stubAgentWorkerStatusRepository) DeleteAgentWorkerStatus(_ context.Context, workerID string) error {
	s.deleted = append(s.deleted, workerID)
	filtered := s.items[:0]
	for _, item := range s.items {
		if item.WorkerID == workerID {
			continue
		}
		filtered = append(filtered, item)
	}
	s.items = filtered
	return nil
}

func (s *stubAgentWorkerStatusRepository) ListAgentWorkerStatuses(_ context.Context) ([]model.AgentWorkerStatus, error) {
	return append([]model.AgentWorkerStatus(nil), s.items...), nil
}

func TestAgentWorkerStatusServiceReportsAndMarksStale(t *testing.T) {
	service := NewAgentWorkerStatusService()
	service.staleAfter = time.Minute
	service.clock = func() time.Time {
		return time.Date(2026, 5, 31, 10, 2, 0, 0, time.UTC)
	}

	item, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:       "akashic-python-worker",
		WorkerType:     "knowledge",
		Status:         "running",
		CurrentJobID:   "group-memory-1",
		ProcessedTotal: 2,
		FailedTotal:    1,
		Source:         "python",
		Timestamp:      time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("report worker status: %v", err)
	}
	if item.Status != "running" {
		t.Fatalf("status = %q, want running", item.Status)
	}

	view, err := service.ListAgentWorkerStatuses(context.Background(), query.AgentWorkerStatusFilter{})
	if err != nil {
		t.Fatalf("list worker statuses: %v", err)
	}
	if view.Totals["workers"] != 1 {
		t.Fatalf("workers total = %d, want 1", view.Totals["workers"])
	}
	if view.Totals["stale"] != 1 {
		t.Fatalf("stale total = %d, want 1", view.Totals["stale"])
	}
	if view.Workers[0].Status != "stopped" {
		t.Fatalf("stale status = %q, want stopped", view.Workers[0].Status)
	}
	if !view.Workers[0].Stale {
		t.Fatalf("worker should be marked stale")
	}
	if view.Workers[0].Metadata["last_status"] != "running" {
		t.Fatalf("last_status metadata = %q, want running", view.Workers[0].Metadata["last_status"])
	}
}

func TestAgentWorkerStatusServiceRejectsActiveLeaseConflict(t *testing.T) {
	service := NewAgentWorkerStatusService()
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	first, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:        "worker-a",
		InstanceID:      "instance-a",
		WorkerType:      "knowledge",
		Status:          "running",
		Source:          "python",
		Timestamp:       now,
		LeaseTTLSeconds: 120,
	})
	if err != nil {
		t.Fatalf("first report: %v", err)
	}
	if first.InstanceID != "instance-a" || first.LeaseUntil == "" {
		t.Fatalf("unexpected first view: %+v", first)
	}

	_, err = service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:        "worker-a",
		InstanceID:      "instance-b",
		WorkerType:      "knowledge",
		Status:          "running",
		Source:          "python",
		Timestamp:       now.Add(time.Second),
		LeaseTTLSeconds: 120,
	})
	if !errors.Is(err, ErrAgentWorkerLeaseConflict) {
		t.Fatalf("expected lease conflict, got %v", err)
	}

	released, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:   "worker-a",
		InstanceID: "instance-a",
		WorkerType: "knowledge",
		Status:     "stopped",
		Source:     "python",
		Timestamp:  now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("release report: %v", err)
	}
	if released.LeaseActive || released.LeaseUntil != "" {
		t.Fatalf("expected released lease, got %+v", released)
	}

	second, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:        "worker-a",
		InstanceID:      "instance-b",
		WorkerType:      "knowledge",
		Status:          "running",
		Source:          "python",
		Timestamp:       now.Add(3 * time.Second),
		LeaseTTLSeconds: 120,
	})
	if err != nil {
		t.Fatalf("second report after release: %v", err)
	}
	if second.InstanceID != "instance-b" {
		t.Fatalf("unexpected second instance: %+v", second)
	}
}

func TestAgentWorkerStatusServiceAllowsControlledTakeoverOfStaleInstance(t *testing.T) {
	service := NewAgentWorkerStatusService()
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	_, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:        "worker-a",
		InstanceID:      "instance-a",
		WorkerType:      "knowledge",
		Status:          "running",
		Source:          "python",
		Timestamp:       now,
		LeaseTTLSeconds: 120,
	})
	if err != nil {
		t.Fatalf("first report: %v", err)
	}

	replaced, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:                  "worker-a",
		InstanceID:                "instance-b",
		ReplaceExistingInstanceID: "instance-a",
		WorkerType:                "knowledge",
		Status:                    "running",
		Source:                    "python",
		Timestamp:                 now.Add(time.Second),
		LeaseTTLSeconds:           120,
	})
	if err != nil {
		t.Fatalf("takeover report: %v", err)
	}
	if replaced.InstanceID != "instance-b" {
		t.Fatalf("unexpected replacement instance: %+v", replaced)
	}
}

func TestAgentWorkerStatusServiceRejectsTakeoverWithMismatchedReplaceID(t *testing.T) {
	service := NewAgentWorkerStatusService()
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	_, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:        "worker-a",
		InstanceID:      "instance-a",
		WorkerType:      "knowledge",
		Status:          "running",
		Source:          "python",
		Timestamp:       now,
		LeaseTTLSeconds: 120,
	})
	if err != nil {
		t.Fatalf("first report: %v", err)
	}

	_, err = service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:                  "worker-a",
		InstanceID:                "instance-b",
		ReplaceExistingInstanceID: "instance-x",
		WorkerType:                "knowledge",
		Status:                    "running",
		Source:                    "python",
		Timestamp:                 now.Add(time.Second),
		LeaseTTLSeconds:           120,
	})
	if !errors.Is(err, ErrAgentWorkerLeaseConflict) {
		t.Fatalf("expected lease conflict, got %v", err)
	}
}

func TestAgentWorkerStatusServiceCleanupStaleWorkerStatuses(t *testing.T) {
	service := NewAgentWorkerStatusService()
	service.staleAfter = time.Minute
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)

	_, err := service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:        "worker-stale",
		InstanceID:      "instance-stale",
		WorkerType:      "knowledge",
		Status:          "running",
		Source:          "python",
		Timestamp:       now.Add(-2 * time.Minute),
		LeaseTTLSeconds: 120,
	})
	if err != nil {
		t.Fatalf("report stale worker: %v", err)
	}
	_, err = service.ReportAgentWorkerStatus(context.Background(), command.ReportAgentWorkerStatusCommand{
		WorkerID:        "worker-active",
		InstanceID:      "instance-active",
		WorkerType:      "knowledge",
		Status:          "running",
		Source:          "python",
		Timestamp:       now,
		LeaseTTLSeconds: 120,
	})
	if err != nil {
		t.Fatalf("report active worker: %v", err)
	}

	view, err := service.CleanupStaleAgentWorkerStatuses(context.Background(), command.CleanupStaleAgentWorkerStatusesCommand{
		Timestamp:         now,
		StaleAfterSeconds: 60,
	})
	if err != nil {
		t.Fatalf("cleanup stale workers: %v", err)
	}
	if view.Totals["deleted"] != 1 || view.Totals["remaining"] != 1 || view.Totals["remaining_stale"] != 0 {
		t.Fatalf("unexpected cleanup totals: %#v", view.Totals)
	}
	if len(view.Deleted) != 1 || view.Deleted[0].WorkerID != "worker-stale" || !view.Deleted[0].Stale {
		t.Fatalf("unexpected deleted view: %#v", view.Deleted)
	}
	if len(view.Remaining) != 1 || view.Remaining[0].WorkerID != "worker-active" || view.Remaining[0].Stale {
		t.Fatalf("unexpected remaining view: %#v", view.Remaining)
	}

	listed, err := service.ListAgentWorkerStatuses(context.Background(), query.AgentWorkerStatusFilter{StaleAfterSeconds: 60})
	if err != nil {
		t.Fatalf("list after cleanup: %v", err)
	}
	if listed.Totals["workers"] != 1 || listed.Workers[0].WorkerID != "worker-active" {
		t.Fatalf("unexpected list after cleanup: %#v", listed)
	}
}

func TestAgentWorkerStatusServicePrunesStaleStatusesOnRepositoryLoad(t *testing.T) {
	now := time.Now().UTC()
	staleStatus, err := model.NewAgentWorkerStatus(model.AgentWorkerStatusSpec{
		WorkerID:   "worker-stale",
		InstanceID: "instance-stale",
		WorkerType: "knowledge",
		Status:     "running",
		Source:     "python",
		LeaseUntil: now.Add(-time.Minute),
	}, now.Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("create stale status: %v", err)
	}
	activeStatus, err := model.NewAgentWorkerStatus(model.AgentWorkerStatusSpec{
		WorkerID:   "worker-active",
		InstanceID: "instance-active",
		WorkerType: "knowledge",
		Status:     "running",
		Source:     "python",
		LeaseUntil: now.Add(2 * time.Minute),
	}, now.Add(-30*time.Second))
	if err != nil {
		t.Fatalf("create active status: %v", err)
	}
	repository := &stubAgentWorkerStatusRepository{
		items: []model.AgentWorkerStatus{staleStatus, activeStatus},
	}

	service, err := NewAgentWorkerStatusServiceWithRepository(context.Background(), repository, time.Minute)
	if err != nil {
		t.Fatalf("load service from repository: %v", err)
	}
	service.clock = func() time.Time { return now }

	view, err := service.ListAgentWorkerStatuses(context.Background(), query.AgentWorkerStatusFilter{StaleAfterSeconds: 60})
	if err != nil {
		t.Fatalf("list statuses after load: %v", err)
	}
	if view.Totals["workers"] != 1 || view.Totals["stale"] != 0 {
		t.Fatalf("unexpected totals after load prune: %#v", view.Totals)
	}
	if len(view.Workers) != 1 || view.Workers[0].WorkerID != "worker-active" {
		t.Fatalf("unexpected workers after load prune: %#v", view.Workers)
	}
	if len(repository.deleted) != 1 || repository.deleted[0] != "worker-stale" {
		t.Fatalf("unexpected deleted worker ids: %#v", repository.deleted)
	}
	if len(repository.items) != 1 || repository.items[0].WorkerID != "worker-active" {
		t.Fatalf("unexpected repository items after prune: %#v", repository.items)
	}
}
