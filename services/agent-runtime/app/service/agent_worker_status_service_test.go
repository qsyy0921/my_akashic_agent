package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

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
