package service

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestQueueBackendServiceReconcilesShadowPublishAgainstStateAndEvents(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 23, 45, 0, 0, time.UTC)

	sender := NewMessageSendServiceWithOutboxEventsAndWorkQueue(store, store, store, store, store, nil)
	if err := sender.Send(ctx, command.SendMessageCommand{
		EventID: "outbox:diagnostics:1",
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:   "hello",
		Timestamp: now,
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	jobs := NewAgentJobServiceWithEvents(store, store)
	if _, err := jobs.Create(ctx, command.CreateAgentJobCommand{
		JobID:   "agent-job:diagnostics:1",
		JobType: "rag_ingest",
		AgentID: "akashic",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "3219982",
			ConversationType: "group",
		},
		MaxAttempts: 3,
		Timestamp:   now,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	service := NewQueueBackendServiceWithDiagnostics(query.QueueBackendView{
		Provider:       "nats_jetstream",
		Mode:           "shadow_publish",
		MigrationPhase: "shadow_ready",
	}, QueueBackendDiagnosticsDeps{
		Diagnostics: fakeQueueDiagnosticsReader{
			snapshot: query.QueueShadowPublishDiagnostics{
				Enabled:        true,
				AttemptTotal:   2,
				SucceededTotal: 1,
				FailedTotal:    1,
				WorkKinds: []query.QueuePublishWorkKindStats{
					{WorkKind: "outbox_delivery", AttemptCount: 1, SucceededCount: 1},
					{WorkKind: "agent_job", AttemptCount: 1, FailedCount: 1, LastError: "nats unavailable"},
				},
				Subjects: []query.QueuePublishSubjectStats{
					{Subject: "akashic.work.outbox.qq.1049511700", WorkKind: "outbox_delivery", AttemptCount: 1, SucceededCount: 1},
				},
			},
		},
		OutboxRepo:     store,
		OutboxEvents:   store,
		AgentJobRepo:   store,
		AgentJobEvents: store,
	})

	view, err := service.Get(ctx)
	if err != nil {
		t.Fatalf("get queue backend: %v", err)
	}
	if view.ShadowPublish == nil {
		t.Fatalf("expected shadow publish diagnostics")
	}
	diagnostics := view.ShadowPublish
	if !diagnostics.Enabled || diagnostics.StateStoreTotal != 2 || diagnostics.EventStreamTotal != 2 {
		t.Fatalf("unexpected reconciliation totals: %+v", diagnostics)
	}
	if diagnostics.StateMinusPublishedTotal != 1 || diagnostics.EventMinusPublishedTotal != 1 {
		t.Fatalf("unexpected publish deltas: %+v", diagnostics)
	}
	outbox := findQueueWorkKind(t, diagnostics.WorkKinds, "outbox_delivery")
	if outbox.StateStoreCount != 1 || outbox.EventStreamCount != 1 || outbox.StateMinusPublished != 0 {
		t.Fatalf("unexpected outbox reconciliation: %+v", outbox)
	}
	agentJob := findQueueWorkKind(t, diagnostics.WorkKinds, "agent_job")
	if agentJob.StateStoreCount != 1 || agentJob.EventStreamCount != 1 || agentJob.StateMinusPublished != 1 || agentJob.LastError != "nats unavailable" {
		t.Fatalf("unexpected agent job reconciliation: %+v", agentJob)
	}
}

func TestQueueBackendServiceExposesDualReadDiagnostics(t *testing.T) {
	ctx := context.Background()
	compare := NewWorkQueueCompareService(memory.NewStore(), memory.NewStore())
	if _, err := compare.CompareWorkQueueCandidate(ctx, command.CompareWorkQueueCandidateCommand{
		WorkKind:   "agent_job",
		WorkID:     "missing-job",
		Subject:    "akashic.work.agent_job.rag_ingest",
		ObservedAt: time.Date(2026, 5, 30, 23, 46, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("compare candidate: %v", err)
	}
	service := NewQueueBackendServiceWithDiagnostics(query.QueueBackendView{
		Provider:       "nats_jetstream",
		Mode:           "dual_read_compare",
		MigrationPhase: "dual_read_compare",
	}, QueueBackendDiagnosticsDeps{
		Compare: compare,
	})

	view, err := service.Get(ctx)
	if err != nil {
		t.Fatalf("get queue backend: %v", err)
	}
	if view.DualReadCompare == nil || !view.DualReadCompare.Enabled {
		t.Fatalf("expected dual read diagnostics: %+v", view.DualReadCompare)
	}
	if view.DualReadCompare.ComparedTotal != 1 || view.DualReadCompare.MismatchedTotal != 1 {
		t.Fatalf("unexpected dual read totals: %+v", view.DualReadCompare)
	}
}

func TestQueueBackendServiceExposesExternalLeaseExecutionDiagnostics(t *testing.T) {
	service := NewQueueBackendServiceWithDiagnostics(query.QueueBackendView{
		Provider:       "nats_jetstream",
		Mode:           "external_lease",
		MigrationPhase: "external_lease",
		ExternalLease:  &query.QueueExternalLeaseGate{Enabled: true, AllowExecution: true},
	}, QueueBackendDiagnosticsDeps{
		ExternalLease: fakeExternalLeaseDiagnosticsReader{
			snapshot: query.QueueExternalLeaseDiagnostics{
				Enabled:       true,
				SampleLimit:   100,
				ExecutedTotal: 2,
				Dispositions: []query.QueueExternalLeaseCounter{
					{Name: "ack", Count: 1},
					{Name: "nack", Count: 1},
				},
				RecentExecutions: []query.QueueExternalLeaseExecutionView{
					{WorkKind: "agent_job", WorkID: "job-1", Disposition: "nack", Reason: "agent_job_waiting_for_result"},
				},
			},
		},
	})

	view, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("get queue backend: %v", err)
	}
	if view.ExternalLease == nil || view.ExternalLease.Diagnostics == nil {
		t.Fatalf("expected external lease diagnostics: %+v", view.ExternalLease)
	}
	if !view.ExternalLease.Diagnostics.Enabled || view.ExternalLease.Diagnostics.ExecutedTotal != 2 {
		t.Fatalf("unexpected external lease diagnostics: %+v", view.ExternalLease.Diagnostics)
	}
	if len(view.ExternalLease.Diagnostics.RecentExecutions) != 1 ||
		view.ExternalLease.Diagnostics.RecentExecutions[0].Reason != "agent_job_waiting_for_result" {
		t.Fatalf("unexpected recent executions: %+v", view.ExternalLease.Diagnostics.RecentExecutions)
	}
}

func TestQueueBackendServiceReportsMissingExternalLeaseDiagnosticsRecorder(t *testing.T) {
	service := NewQueueBackendServiceWithDiagnostics(query.QueueBackendView{
		Provider:       "nats_jetstream",
		Mode:           "external_lease",
		MigrationPhase: "external_lease_gate",
		ExternalLease:  &query.QueueExternalLeaseGate{Enabled: true, AllowExecution: false},
	}, QueueBackendDiagnosticsDeps{})

	view, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("get queue backend: %v", err)
	}
	if view.ExternalLease == nil || view.ExternalLease.Diagnostics == nil {
		t.Fatalf("expected external lease diagnostics placeholder: %+v", view.ExternalLease)
	}
	if view.ExternalLease.Diagnostics.Enabled {
		t.Fatalf("diagnostics should be disabled without active recorder: %+v", view.ExternalLease.Diagnostics)
	}
	if len(view.ExternalLease.Diagnostics.Notes) == 0 {
		t.Fatalf("expected diagnostic note when recorder is absent")
	}
}

type fakeQueueDiagnosticsReader struct {
	snapshot query.QueueShadowPublishDiagnostics
}

func (r fakeQueueDiagnosticsReader) SnapshotWorkQueuePublishDiagnostics(context.Context) (query.QueueShadowPublishDiagnostics, error) {
	return r.snapshot, nil
}

type fakeExternalLeaseDiagnosticsReader struct {
	snapshot query.QueueExternalLeaseDiagnostics
}

func (r fakeExternalLeaseDiagnosticsReader) SnapshotExternalLeaseDiagnostics(context.Context) (query.QueueExternalLeaseDiagnostics, error) {
	return r.snapshot, nil
}

func findQueueWorkKind(t *testing.T, items []query.QueuePublishWorkKindStats, workKind string) query.QueuePublishWorkKindStats {
	t.Helper()
	for _, item := range items {
		if item.WorkKind == workKind {
			return item
		}
	}
	t.Fatalf("missing work kind %s in %+v", workKind, items)
	return query.QueuePublishWorkKindStats{}
}
