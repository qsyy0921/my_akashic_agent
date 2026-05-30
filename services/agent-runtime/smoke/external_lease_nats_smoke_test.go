package smoke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	inport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/in"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/natsqueue"
)

func TestExternalLeaseNATSSmokeOutboxDispositions(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("AKASHIC_NATS_SMOKE_DSN"))
	if dsn == "" {
		t.Skip("set AKASHIC_NATS_SMOKE_DSN to run the external lease NATS smoke")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	stream := "AKASHIC_SMOKE_LEASE_" + suffix
	subjectPrefix := "akashic.smoke.lease." + suffix
	durable := "AKASHIC_SMOKE_LEASE_" + suffix

	publisher, err := natsqueue.NewPublisher(natsqueue.Config{
		URL:           dsn,
		Stream:        stream,
		SubjectPrefix: subjectPrefix,
		Timeout:       3 * time.Second,
	})
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	defer publisher.Close()
	defer deleteSmokeStream(t, dsn, stream)

	store := memory.NewStore()
	adapter := &smokeDeliveryAdapter{}
	outbox := appservice.NewOutboxServiceWithEvents(store, store, store)
	dispatch := appservice.NewDeliveryDispatchServiceWithAdapters(store, adapter)
	executor := &recordingLeaseExecutor{
		inner: appservice.NewWorkQueueExternalLeaseService(
			outbox,
			dispatch,
			appservice.WithExternalLeaseChannelByAccount(map[string]string{"1049511700": "qq_smoke"}),
			appservice.WithExternalLeaseWorker("smoke-worker", 60),
		),
		results: make(chan query.QueueExternalLeaseExecutionView, 8),
	}
	consumer, err := natsqueue.NewExternalLeaseConsumer(natsqueue.ExternalLeaseConsumerConfig{
		URL:                 dsn,
		Stream:              stream,
		SubjectPrefix:       subjectPrefix,
		Durable:             durable,
		WorkerID:            "smoke-worker",
		LeaseTTLSeconds:     60,
		NackDelay:           10 * time.Second,
		Timeout:             3 * time.Second,
		ConsumerConcurrency: 1,
		MaxInFlight:         4,
		ChannelByAccount:    map[string]string{"1049511700": "qq_smoke"},
	})
	if err != nil {
		t.Fatalf("new external lease consumer: %v", err)
	}
	defer consumer.Close()

	runCtx, stopConsumer := context.WithCancel(ctx)
	defer stopConsumer()
	errs := make(chan error, 1)
	go func() {
		errs <- consumer.Run(runCtx, executor)
	}()

	now := time.Date(2026, 5, 31, 1, 0, 0, 0, time.UTC)
	success := saveSmokeDelivery(t, ctx, store, "smoke:external:success:"+suffix, "success", 3, now)
	retry := saveSmokeDelivery(t, ctx, store, "smoke:external:retry:"+suffix, "retry", 3, now)
	dead := saveSmokeDelivery(t, ctx, store, "smoke:external:dead:"+suffix, "dead", 1, now)
	for _, delivery := range []model.OutboxDelivery{success, retry, dead} {
		if err := publisher.PublishOutboxDelivery(ctx, delivery); err != nil {
			t.Fatalf("publish delivery %s: %v", delivery.Message.EventID, err)
		}
	}
	if err := publishUnsupportedOutboxWork(dsn, stream, subjectPrefix, suffix); err != nil {
		t.Fatalf("publish unsupported work: %v", err)
	}

	results := collectLeaseResults(t, ctx, executor.results, 4)
	assertLeaseDisposition(t, results, success.Message.EventID, appservice.QueueLeaseDispositionAck, "delivery_succeeded")
	assertLeaseDisposition(t, results, retry.Message.EventID, appservice.QueueLeaseDispositionNack, "delivery_retry_scheduled")
	assertLeaseDisposition(t, results, dead.Message.EventID, appservice.QueueLeaseDispositionAck, "delivery_terminal_failure")
	assertLeaseDisposition(t, results, "smoke:unsupported:"+suffix, appservice.QueueLeaseDispositionTerm, "unsupported_work_kind")

	assertSmokeDeliveryState(t, ctx, store, success.Message.EventID, model.DeliverySucceeded, 1)
	assertSmokeDeliveryState(t, ctx, store, retry.Message.EventID, model.DeliveryQueued, 1)
	assertSmokeDeliveryState(t, ctx, store, dead.Message.EventID, model.DeliveryDeadLettered, 1)
	steps := adapter.dispatchedSteps()
	if len(steps) != 3 {
		t.Fatalf("expected only three real delivery adapter steps, got %d", len(steps))
	}
	for _, step := range steps {
		if step.Channel != "qq_smoke" {
			t.Fatalf("expected smoke-only channel, got %#v", step)
		}
	}

	stopConsumer()
	if err := <-errs; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("consumer stopped unexpectedly: %v", err)
	}
}

type recordingLeaseExecutor struct {
	inner   inport.WorkQueueLeaseExecutor
	results chan query.QueueExternalLeaseExecutionView
}

func (e *recordingLeaseExecutor) ExecuteWorkQueueLease(
	ctx context.Context,
	cmd command.ExecuteWorkQueueLeaseCommand,
) (query.QueueExternalLeaseExecutionView, error) {
	result, err := e.inner.ExecuteWorkQueueLease(ctx, cmd)
	if err == nil {
		select {
		case e.results <- result:
		case <-ctx.Done():
		}
	}
	return result, err
}

type smokeDeliveryAdapter struct {
	mu    sync.Mutex
	steps []model.DeliveryDispatchStep
}

func (a *smokeDeliveryAdapter) SupportsDeliveryChannel(channel string) bool {
	return channel == "qq_smoke"
}

func (a *smokeDeliveryAdapter) DispatchDeliveryStep(_ context.Context, step model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error) {
	a.mu.Lock()
	a.steps = append(a.steps, step)
	a.mu.Unlock()
	switch step.Message {
	case "retry", "dead":
		return model.DeliveryDispatchResult{}, smokeDeliveryError{kind: string(model.DeliveryErrorPlatformTimeout), message: "smoke platform timeout"}
	default:
		return model.DeliveryDispatchResult{
			StepIndex:         step.StepIndex,
			Kind:              step.Kind,
			Channel:           step.Channel,
			ChatID:            step.ChatID,
			Status:            model.DeliveryDispatchSent,
			Provider:          "fake-smoke",
			ProviderMessageID: "fake-smoke-message-id",
		}, nil
	}
}

func (a *smokeDeliveryAdapter) dispatchedSteps() []model.DeliveryDispatchStep {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]model.DeliveryDispatchStep(nil), a.steps...)
}

type smokeDeliveryError struct {
	kind    string
	message string
}

func (e smokeDeliveryError) Error() string {
	return e.message
}

func (e smokeDeliveryError) DeliveryErrorKind() string {
	return e.kind
}

func saveSmokeDelivery(
	t *testing.T,
	ctx context.Context,
	store *memory.Store,
	eventID string,
	content string,
	maxAttempts int,
	now time.Time,
) model.OutboxDelivery {
	t.Helper()
	delivery, err := model.NewOutboxDelivery(model.OutboundMessage{
		EventID: eventID,
		Channel: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   content,
		Timestamp: now,
	}, maxAttempts, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}
	return delivery
}

func publishUnsupportedOutboxWork(dsn string, stream string, subjectPrefix string, suffix string) error {
	conn, err := nats.Connect(dsn, nats.Timeout(3*time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()
	js, err := conn.JetStream(nats.MaxWait(3 * time.Second))
	if err != nil {
		return err
	}
	subject := subjectPrefix + ".outbox.qq.1049511700"
	notification := natsqueue.WorkNotification{
		SchemaVersion:      "1",
		WorkKind:           "agent_job",
		WorkID:             "smoke:unsupported:" + suffix,
		AggregateID:        "smoke:unsupported:" + suffix,
		Status:             "pending",
		Subject:            subject,
		NotificationTime:   time.Now().UTC().Format(time.RFC3339Nano),
		ConsumerModel:      "goroutine_worker_pool",
		StateAuthoritative: true,
	}
	raw, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	_, err = js.Publish(subject, raw, nats.MsgId(notification.WorkID), nats.ExpectStream(stream), nats.AckWait(3*time.Second))
	return err
}

func collectLeaseResults(
	t *testing.T,
	ctx context.Context,
	results <-chan query.QueueExternalLeaseExecutionView,
	count int,
) []query.QueueExternalLeaseExecutionView {
	t.Helper()
	items := make([]query.QueueExternalLeaseExecutionView, 0, count)
	for len(items) < count {
		select {
		case item := <-results:
			items = append(items, item)
		case <-ctx.Done():
			t.Fatalf("timed out waiting for lease results; got %d/%d", len(items), count)
		}
	}
	return items
}

func assertLeaseDisposition(
	t *testing.T,
	results []query.QueueExternalLeaseExecutionView,
	workID string,
	disposition string,
	reason string,
) {
	t.Helper()
	for _, result := range results {
		if result.WorkID != workID {
			continue
		}
		if result.Disposition != disposition || result.Reason != reason {
			t.Fatalf("unexpected disposition for %s: %+v", workID, result)
		}
		return
	}
	t.Fatalf("missing lease result for %s in %+v", workID, results)
}

func assertSmokeDeliveryState(
	t *testing.T,
	ctx context.Context,
	store *memory.Store,
	eventID string,
	status model.DeliveryStatus,
	attempts int,
) {
	t.Helper()
	delivery, ok, err := store.FindOutboxDelivery(ctx, eventID)
	if err != nil || !ok {
		t.Fatalf("find delivery %s: ok=%t err=%v", eventID, ok, err)
	}
	if delivery.Status != status || delivery.Attempts != attempts {
		t.Fatalf("unexpected delivery state for %s: %#v", eventID, delivery)
	}
}

func deleteSmokeStream(t *testing.T, dsn string, stream string) {
	t.Helper()
	conn, err := nats.Connect(dsn, nats.Timeout(3*time.Second))
	if err != nil {
		t.Logf("cleanup nats connect failed: %v", err)
		return
	}
	defer conn.Close()
	js, err := conn.JetStream(nats.MaxWait(3 * time.Second))
	if err != nil {
		t.Logf("cleanup jetstream failed: %v", err)
		return
	}
	if err := js.DeleteStream(stream); err != nil && !strings.Contains(err.Error(), "stream not found") {
		t.Logf("cleanup stream %s failed: %v", stream, err)
	}
}
