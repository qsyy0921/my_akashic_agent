package service

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestWorkQueueExternalLeaseServiceDispatchesOutboxAndAcks(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 0, 20, 0, 0, time.UTC)
	if err := saveExternalLeaseDelivery(ctx, store, "outbox:lease:success", 3, now); err != nil {
		t.Fatal(err)
	}
	adapter := &recordingLeaseDeliveryAdapter{}
	service := NewWorkQueueExternalLeaseService(
		NewOutboxServiceWithEvents(store, store, store),
		NewDeliveryDispatchServiceWithAdapters(store, adapter),
		WithExternalLeaseChannelByAccount(map[string]string{"1049511700": "qq_1049511700"}),
	)

	result, err := service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
		WorkKind:        "outbox_delivery",
		WorkID:          "outbox:lease:success",
		WorkerID:        "nats-worker-1",
		LeaseTTLSeconds: 60,
		Timestamp:       now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("execute lease: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionAck || result.Reason != "delivery_succeeded" {
		t.Fatalf("expected ack success, got %+v", result)
	}
	if result.StateStatus != string(model.DeliverySucceeded) || result.Attempts != 1 {
		t.Fatalf("expected succeeded delivery state, got %+v", result)
	}
	if len(adapter.steps) != 1 || adapter.steps[0].Channel != "qq_1049511700" {
		t.Fatalf("expected mapped delivery adapter step, got %#v", adapter.steps)
	}
	stored, ok, err := store.FindOutboxDelivery(ctx, "outbox:lease:success")
	if err != nil || !ok {
		t.Fatalf("find stored delivery: ok=%t err=%v", ok, err)
	}
	if stored.Status != model.DeliverySucceeded || stored.LeaseOwner != "" {
		t.Fatalf("expected terminal stored delivery, got %#v", stored)
	}
}

func TestWorkQueueExternalLeaseServiceRetriesRetryableDispatchFailure(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 0, 25, 0, 0, time.UTC)
	if err := saveExternalLeaseDelivery(ctx, store, "outbox:lease:retry", 3, now); err != nil {
		t.Fatal(err)
	}
	adapter := &recordingLeaseDeliveryAdapter{
		err: leaseDeliveryError{kind: string(model.DeliveryErrorPlatformTimeout), message: "platform timeout"},
	}
	service := NewWorkQueueExternalLeaseService(
		NewOutboxServiceWithEvents(store, store, store),
		NewDeliveryDispatchServiceWithAdapters(store, adapter),
	)

	result, err := service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
		WorkKind:  "outbox_delivery",
		WorkID:    "outbox:lease:retry",
		Timestamp: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("execute lease: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionNack || result.Reason != "delivery_retry_scheduled" {
		t.Fatalf("expected nack retry, got %+v", result)
	}
	if result.StateStatus != string(model.DeliveryQueued) || result.Attempts != 1 {
		t.Fatalf("expected queued retry state, got %+v", result)
	}
	stored, ok, err := store.FindOutboxDelivery(ctx, "outbox:lease:retry")
	if err != nil || !ok {
		t.Fatalf("find stored delivery: ok=%t err=%v", ok, err)
	}
	if stored.Status != model.DeliveryQueued || stored.Attempts != 1 {
		t.Fatalf("expected queued retry state, got %#v", stored)
	}
}

func TestWorkQueueExternalLeaseServiceAcksTerminalFailure(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 0, 30, 0, 0, time.UTC)
	if err := saveExternalLeaseDelivery(ctx, store, "outbox:lease:dead", 1, now); err != nil {
		t.Fatal(err)
	}
	adapter := &recordingLeaseDeliveryAdapter{
		err: leaseDeliveryError{kind: string(model.DeliveryErrorPlatformTimeout), message: "platform timeout"},
	}
	service := NewWorkQueueExternalLeaseService(
		NewOutboxServiceWithEvents(store, store, store),
		NewDeliveryDispatchServiceWithAdapters(store, adapter),
	)

	result, err := service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
		WorkKind:  "outbox_delivery",
		WorkID:    "outbox:lease:dead",
		Timestamp: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("execute lease: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionAck || result.Reason != "delivery_terminal_failure" {
		t.Fatalf("expected ack terminal failure, got %+v", result)
	}
	if result.StateStatus != string(model.DeliveryDeadLettered) || result.Attempts != 1 {
		t.Fatalf("expected dead-lettered state, got %+v", result)
	}
}

func TestWorkQueueExternalLeaseServiceTermsUnsupportedWork(t *testing.T) {
	service := NewWorkQueueExternalLeaseService(
		NewOutboxService(memory.NewStore(), memory.NewStore()),
		NewDeliveryDispatchService(memory.NewStore()),
	)

	result, err := service.ExecuteWorkQueueLease(context.Background(), command.ExecuteWorkQueueLeaseCommand{
		WorkKind: "unknown",
		WorkID:   "job:1",
	})
	if err != nil {
		t.Fatalf("execute lease: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionTerm || result.Reason != "unsupported_work_kind" {
		t.Fatalf("expected term unsupported, got %+v", result)
	}
}

func TestWorkQueueExternalLeaseServiceNacksPendingAgentJobForPythonWorker(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := NewAgentJobServiceWithEvents(store, store)
	now := time.Date(2026, 5, 31, 0, 35, 0, 0, time.UTC)
	if _, err := jobs.Create(ctx, sampleExternalLeaseAgentJob("job:agent:pending", 2, now)); err != nil {
		t.Fatalf("create agent job: %v", err)
	}
	service := NewWorkQueueExternalLeaseService(nil, nil, WithExternalLeaseAgentJobs(jobs))

	result, err := service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
		WorkKind:  "agent_job",
		WorkID:    "job:agent:pending",
		Timestamp: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("execute agent job result ack: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionNack || result.Reason != "agent_job_pending_for_python_worker" {
		t.Fatalf("expected nack while Python worker owns execution, got %+v", result)
	}
	if result.StateStatus != string(model.AgentJobPending) || result.Attempts != 0 {
		t.Fatalf("expected untouched pending job state, got %+v", result)
	}
}

func TestWorkQueueExternalLeaseServiceAcksTerminalAgentJobDuplicateDeliveries(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := NewAgentJobServiceWithEvents(store, store)
	now := time.Date(2026, 5, 31, 0, 40, 0, 0, time.UTC)
	if _, err := jobs.Create(ctx, sampleExternalLeaseAgentJob("job:agent:done", 2, now)); err != nil {
		t.Fatalf("create agent job: %v", err)
	}
	leased, err := jobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      "job:agent:done",
		WorkerID:   "python-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease job: %v", err)
	}
	if _, err := jobs.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      "job:agent:done",
		LeaseToken: leased.LeaseToken,
		Timestamp:  now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if _, err := jobs.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:      "job:agent:done",
		LeaseToken: leased.LeaseToken,
		Result:     map[string]string{"ok": "true"},
		Timestamp:  now.Add(3 * time.Second),
	}); err != nil {
		t.Fatalf("complete job: %v", err)
	}
	service := NewWorkQueueExternalLeaseService(nil, nil, WithExternalLeaseAgentJobs(jobs))

	for i := 0; i < 2; i++ {
		result, err := service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
			WorkKind:  "agent_job",
			WorkID:    "job:agent:done",
			Timestamp: now.Add(time.Duration(10+i) * time.Second),
		})
		if err != nil {
			t.Fatalf("execute terminal result ack #%d: %v", i+1, err)
		}
		if result.Disposition != QueueLeaseDispositionAck || result.Reason != "agent_job_terminal" {
			t.Fatalf("expected duplicate terminal delivery to ack, got %+v", result)
		}
		if result.StateStatus != string(model.AgentJobSucceeded) || result.Attempts != 1 {
			t.Fatalf("expected succeeded state in ack view, got %+v", result)
		}
	}
}

func TestWorkQueueExternalLeaseServiceRecoversExpiredAgentJobBeforeNackOrAck(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := NewAgentJobServiceWithEvents(store, store)
	now := time.Date(2026, 5, 31, 0, 45, 0, 0, time.UTC)
	if _, err := jobs.Create(ctx, sampleExternalLeaseAgentJob("job:agent:recover", 2, now)); err != nil {
		t.Fatalf("create recoverable job: %v", err)
	}
	if _, err := jobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      "job:agent:recover",
		WorkerID:   "python-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease recoverable job: %v", err)
	}
	if _, err := jobs.Create(ctx, sampleExternalLeaseAgentJob("job:agent:dead", 1, now)); err != nil {
		t.Fatalf("create exhausted job: %v", err)
	}
	if _, err := jobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      "job:agent:dead",
		WorkerID:   "python-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("lease exhausted job: %v", err)
	}
	service := NewWorkQueueExternalLeaseService(nil, nil, WithExternalLeaseAgentJobs(jobs))

	result, err := service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
		WorkKind:  "agent_job",
		WorkID:    "job:agent:recover",
		Timestamp: now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("recover expired job: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionNack || result.Reason != "agent_job_expired_lease_recovered" {
		t.Fatalf("expected recovered job to nack for replay, got %+v", result)
	}
	if result.StateStatus != string(model.AgentJobPending) || result.Attempts != 1 {
		t.Fatalf("expected pending recovered job state, got %+v", result)
	}

	result, err = service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
		WorkKind:  "agent_job",
		WorkID:    "job:agent:dead",
		Timestamp: now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("recover exhausted job: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionAck || result.Reason != "agent_job_terminal_after_recovery" {
		t.Fatalf("expected exhausted job recovery to ack terminal state, got %+v", result)
	}
	if result.StateStatus != string(model.AgentJobDeadLettered) || result.Attempts != 1 {
		t.Fatalf("expected dead-lettered recovered job state, got %+v", result)
	}
}

func TestWorkQueueExternalLeaseServiceRetriesFailedAgentJobBeforeNack(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	jobs := NewAgentJobServiceWithEvents(store, store)
	now := time.Date(2026, 5, 31, 0, 50, 0, 0, time.UTC)
	if _, err := jobs.Create(ctx, sampleExternalLeaseAgentJob("job:agent:failed", 2, now)); err != nil {
		t.Fatalf("create failed job: %v", err)
	}
	leased, err := jobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      "job:agent:failed",
		WorkerID:   "python-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease failed job: %v", err)
	}
	if _, err := jobs.Fail(ctx, command.FailAgentJobCommand{
		JobID:        "job:agent:failed",
		LeaseToken:   leased.LeaseToken,
		ErrorMessage: "temporary model error",
		Timestamp:    now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("fail job: %v", err)
	}
	service := NewWorkQueueExternalLeaseService(nil, nil, WithExternalLeaseAgentJobs(jobs))

	result, err := service.ExecuteWorkQueueLease(ctx, command.ExecuteWorkQueueLeaseCommand{
		WorkKind:  "agent_job",
		WorkID:    "job:agent:failed",
		Timestamp: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("retry failed job: %v", err)
	}
	if result.Disposition != QueueLeaseDispositionNack || result.Reason != "agent_job_retry_scheduled" {
		t.Fatalf("expected retryable failure to nack after retry, got %+v", result)
	}
	if result.StateStatus != string(model.AgentJobPending) || result.Attempts != 1 {
		t.Fatalf("expected retryable job back to pending, got %+v", result)
	}
}

func saveExternalLeaseDelivery(ctx context.Context, store *memory.Store, eventID string, maxAttempts int, now time.Time) error {
	delivery, err := model.NewOutboxDelivery(model.OutboundMessage{
		EventID: eventID,
		Channel: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   "hello",
		Timestamp: now,
	}, maxAttempts, now)
	if err != nil {
		return err
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		return err
	}
	return store.EnqueueOutboxDelivery(ctx, delivery)
}

func sampleExternalLeaseAgentJob(jobID string, maxAttempts int, timestamp time.Time) command.CreateAgentJobCommand {
	return command.CreateAgentJobCommand{
		JobID:   jobID,
		JobType: string(model.AgentJobRagIngest),
		AgentID: "knowledge-worker",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		Payload:     map[string]string{"dataset_id": "qq-27234224"},
		MaxAttempts: maxAttempts,
		Timestamp:   timestamp,
	}
}

type recordingLeaseDeliveryAdapter struct {
	steps []model.DeliveryDispatchStep
	err   error
}

func (a *recordingLeaseDeliveryAdapter) SupportsDeliveryChannel(channel string) bool {
	return channel == "qq" || channel == "qq_1049511700"
}

func (a *recordingLeaseDeliveryAdapter) DispatchDeliveryStep(_ context.Context, step model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error) {
	a.steps = append(a.steps, step)
	if a.err != nil {
		return model.DeliveryDispatchResult{}, a.err
	}
	return model.DeliveryDispatchResult{
		StepIndex:         step.StepIndex,
		Kind:              step.Kind,
		Channel:           step.Channel,
		ChatID:            step.ChatID,
		Status:            model.DeliveryDispatchSent,
		Provider:          "fake",
		ProviderMessageID: "fake-message-id",
	}, nil
}

type leaseDeliveryError struct {
	kind    string
	message string
}

func (e leaseDeliveryError) Error() string {
	if e.message == "" {
		return "delivery failed"
	}
	return e.message
}

func (e leaseDeliveryError) DeliveryErrorKind() string {
	return e.kind
}
