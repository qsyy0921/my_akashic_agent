package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestOutboxServiceTransitionsAndRetry(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 4, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 2, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	running, err := service.MarkDispatching(ctx, command.MarkOutboxDispatchingCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if running.Status != string(model.DeliveryDispatching) || running.Attempts != 1 {
		t.Fatalf("unexpected running state: %+v", running)
	}

	failed, err := service.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-1",
		ErrorKind:    string(model.DeliveryErrorPlatformTimeout),
		ErrorMessage: "platform timeout",
		Timestamp:    now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if failed.Status != string(model.DeliveryFailed) {
		t.Fatalf("expected failed, got %s", failed.Status)
	}
	if failed.ErrorKind != string(model.DeliveryErrorPlatformTimeout) {
		t.Fatalf("expected timeout failure kind, got %s", failed.ErrorKind)
	}

	retried, err := service.Retry(ctx, command.RetryOutboxCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if retried.Status != string(model.DeliveryQueued) {
		t.Fatalf("expected queued, got %s", retried.Status)
	}
	if retried.ErrorKind != "" || retried.ErrorMessage != "" {
		t.Fatalf("expected retry to clear failure details, got kind=%q message=%q", retried.ErrorKind, retried.ErrorMessage)
	}
	if len(store.OutboxQueue()) != 1 {
		t.Fatalf("expected retried delivery to be enqueued, got %d", len(store.OutboxQueue()))
	}
}

func TestOutboxServiceLeaseNextMarksDeliveryDispatching(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 5, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 2, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "qq-dispatcher",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.Status != string(model.DeliveryDispatching) {
		t.Fatalf("expected dispatching, got %s", leased.Status)
	}
	if leased.Attempts != 1 {
		t.Fatalf("expected attempts to increment, got %d", leased.Attempts)
	}
	if leased.LeaseOwner != "qq-dispatcher" {
		t.Fatalf("expected lease owner, got %q", leased.LeaseOwner)
	}
	if leased.LeaseExpiresAt == "" {
		t.Fatalf("expected lease expiry")
	}

	_, err = service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "qq-dispatcher-2",
		TTLSeconds: 60,
		Timestamp:  now.Add(2 * time.Second),
	})
	if err == nil {
		t.Fatalf("expected no leaseable delivery while lease is active")
	}
}

func TestOutboxServiceDeadLettersNonRetryableFailureKind(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 5, 30, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 3, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	if _, err := service.MarkDispatching(ctx, command.MarkOutboxDispatchingCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	dead, err := service.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-1",
		ErrorKind:    string(model.DeliveryErrorRoute),
		ErrorMessage: "route missing",
		Timestamp:    now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if dead.Status != string(model.DeliveryDeadLettered) {
		t.Fatalf("expected dead-lettered, got %s", dead.Status)
	}
	if dead.ErrorKind != string(model.DeliveryErrorRoute) {
		t.Fatalf("expected route error kind, got %s", dead.ErrorKind)
	}
	if _, err := service.Retry(ctx, command.RetryOutboxCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(3 * time.Second),
	}); err == nil {
		t.Fatalf("expected retry to fail for dead-lettered delivery")
	}
}

func TestOutboxServiceRecordsLifecycleEvents(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 3, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}

	service := appservice.NewOutboxServiceWithEvents(store, store, store)
	if _, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if _, err := service.MarkSucceeded(ctx, command.MarkOutboxSucceededCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}

	events, err := store.ListOutboxDeliveryEvents(ctx, query.OutboxDeliveryEventFilter{
		DeliveryID: "outbox-1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("list outbox events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 outbox events, got %d", len(events))
	}
	if events[0].EventType != model.OutboxDeliveryEventSucceeded || events[1].EventType != model.OutboxDeliveryEventLeased {
		t.Fatalf("unexpected event order/types: %#v", events)
	}
}

func TestOutboxMetricsServiceSummarizesThroughputAndDeadLetters(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	outbox := appservice.NewOutboxServiceWithEvents(store, store, store)
	metrics := appservice.NewOutboxMetricsService(store, store)
	now := time.Date(2026, 5, 30, 6, 30, 0, 0, time.UTC)

	succeededMessage := sampleOutboxMessage(now)
	succeededMessage.EventID = "outbox-metrics-succeeded"
	succeededMessage.Channel.Kind = "telegram"
	succeeded, err := model.NewOutboxDelivery(succeededMessage, 2, now)
	if err != nil {
		t.Fatalf("new succeeded delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, succeeded); err != nil {
		t.Fatalf("save succeeded delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    "outbox-metrics-succeeded",
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease succeeded delivery: %v", err)
	}
	if _, err := outbox.MarkSucceeded(ctx, command.MarkOutboxSucceededCommand{
		EventID:   "outbox-metrics-succeeded",
		Timestamp: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}

	deadMessage := sampleOutboxMessage(now.Add(3 * time.Second))
	deadMessage.EventID = "outbox-metrics-dead"
	deadMessage.Channel.Kind = "qq"
	dead, err := model.NewOutboxDelivery(deadMessage, 1, now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("new dead-letter delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, dead); err != nil {
		t.Fatalf("save dead-letter delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    "outbox-metrics-dead",
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(4 * time.Second),
	}); err != nil {
		t.Fatalf("lease dead-letter delivery: %v", err)
	}
	if _, err := outbox.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-metrics-dead",
		ErrorKind:    string(model.DeliveryErrorRoute),
		ErrorMessage: "missing adapter",
		Timestamp:    now.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("fail dead-letter delivery: %v", err)
	}

	view, err := metrics.Get(ctx, query.OutboxMetricsFilter{
		DeliveryLimit: 10,
		EventLimit:    20,
	})
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if view.SampledDeliveries != 2 || view.SampledEvents != 4 {
		t.Fatalf("unexpected sample counts: %+v", view)
	}
	if view.DeliveriesByStatus[string(model.DeliverySucceeded)] != 1 ||
		view.DeliveriesByStatus[string(model.DeliveryDeadLettered)] != 1 {
		t.Fatalf("unexpected deliveries by status: %+v", view.DeliveriesByStatus)
	}
	if view.DeliveriesByChannelKind["telegram"].ByStatus[string(model.DeliverySucceeded)] != 1 ||
		view.DeliveriesByChannelKind["qq"].ByStatus[string(model.DeliveryDeadLettered)] != 1 {
		t.Fatalf("unexpected deliveries by channel kind: %+v", view.DeliveriesByChannelKind)
	}
	if view.Throughput.Leased != 2 || view.Throughput.Succeeded != 1 || view.Throughput.Failed != 1 ||
		view.Throughput.DeadLettered != 1 || view.Throughput.TerminalEvents != 2 {
		t.Fatalf("unexpected throughput metrics: %+v", view.Throughput)
	}
	if view.DeadLetters.CurrentTotal != 1 || view.DeadLetters.ByChannelKind["qq"] != 1 {
		t.Fatalf("unexpected dead-letter metrics: %+v", view.DeadLetters)
	}
	if len(view.DeadLetters.Recent) != 1 ||
		view.DeadLetters.Recent[0].DeliveryID != "outbox-metrics-dead" ||
		view.DeadLetters.Recent[0].EventType != string(model.OutboxDeliveryEventFailed) {
		t.Fatalf("unexpected recent dead letters: %+v", view.DeadLetters.Recent)
	}
}

func sampleOutboxMessage(timestamp time.Time) model.OutboundMessage {
	return model.OutboundMessage{
		EventID: "outbox-1",
		Channel: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:   "hello",
		Timestamp: timestamp,
	}
}
