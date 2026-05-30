package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestOutboxDeliveryLifecycleDeadLettersAfterMaxAttempts(t *testing.T) {
	now := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutbound(now), 2, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if delivery.Status != model.DeliveryQueued {
		t.Fatalf("expected queued, got %s", delivery.Status)
	}

	if err := delivery.MarkDispatching(now.Add(time.Second)); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if delivery.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", delivery.Attempts)
	}
	if err := delivery.MarkFailedWithKind(model.DeliveryErrorPlatformTimeout, "first failure", now.Add(2*time.Second)); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if delivery.Status != model.DeliveryFailed {
		t.Fatalf("expected failed, got %s", delivery.Status)
	}
	if delivery.ErrorKind != model.DeliveryErrorPlatformTimeout {
		t.Fatalf("expected timeout failure kind, got %s", delivery.ErrorKind)
	}

	if err := delivery.Retry(now.Add(3 * time.Second)); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if err := delivery.MarkDispatching(now.Add(4 * time.Second)); err != nil {
		t.Fatalf("second dispatch: %v", err)
	}
	if delivery.ErrorKind != "" {
		t.Fatalf("expected dispatching to clear failure kind, got %s", delivery.ErrorKind)
	}
	if err := delivery.MarkFailed("second failure", now.Add(5*time.Second)); err != nil {
		t.Fatalf("second failure: %v", err)
	}
	if delivery.Status != model.DeliveryDeadLettered {
		t.Fatalf("expected dead letter, got %s", delivery.Status)
	}
	if delivery.ErrorKind != model.DeliveryErrorUnknown {
		t.Fatalf("expected fallback failure kind, got %s", delivery.ErrorKind)
	}
}

func TestSucceededDeliveryCannotBeRetried(t *testing.T) {
	now := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutbound(now), 3, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := delivery.MarkDispatching(now.Add(time.Second)); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := delivery.MarkSucceeded(now.Add(2 * time.Second)); err != nil {
		t.Fatalf("succeed: %v", err)
	}
	if err := delivery.Retry(now.Add(3 * time.Second)); err == nil {
		t.Fatal("expected retry to fail for succeeded delivery")
	}
}

func TestOutboxDeliveryLeaseExpiresAndCanBeReLeased(t *testing.T) {
	now := time.Date(2026, 5, 30, 2, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutbound(now), 3, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if !delivery.CanLease(now) {
		t.Fatal("expected queued delivery to be leaseable")
	}
	if err := delivery.Lease("worker-a", time.Minute, now.Add(time.Second)); err != nil {
		t.Fatalf("lease delivery: %v", err)
	}
	if delivery.Status != model.DeliveryDispatching {
		t.Fatalf("expected dispatching, got %s", delivery.Status)
	}
	if delivery.Attempts != 1 {
		t.Fatalf("expected one attempt, got %d", delivery.Attempts)
	}
	if delivery.LeaseOwner != "worker-a" {
		t.Fatalf("expected lease owner, got %s", delivery.LeaseOwner)
	}
	if delivery.CanLease(now.Add(30 * time.Second)) {
		t.Fatal("did not expect delivery to be leaseable before lease expiry")
	}
	if !delivery.CanLease(now.Add(2 * time.Minute)) {
		t.Fatal("expected expired dispatching lease to be leaseable")
	}
	if err := delivery.Lease("worker-b", time.Minute, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("re-lease delivery: %v", err)
	}
	if delivery.LeaseOwner != "worker-b" {
		t.Fatalf("expected new lease owner, got %s", delivery.LeaseOwner)
	}
	if delivery.Attempts != 2 {
		t.Fatalf("expected second attempt after re-lease, got %d", delivery.Attempts)
	}
}

func TestOutboxDeliveryNormalizesUnknownFailureKind(t *testing.T) {
	now := time.Date(2026, 5, 30, 3, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutbound(now), 3, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := delivery.MarkDispatching(now.Add(time.Second)); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := delivery.MarkFailedWithKind("strange-kind", "bad adapter", now.Add(2*time.Second)); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if delivery.ErrorKind != model.DeliveryErrorUnknown {
		t.Fatalf("expected unknown kind, got %s", delivery.ErrorKind)
	}
	if err := delivery.Retry(now.Add(3 * time.Second)); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if delivery.ErrorKind != "" || delivery.ErrorMessage != "" {
		t.Fatalf("expected retry to clear failure details, got kind=%q message=%q", delivery.ErrorKind, delivery.ErrorMessage)
	}
}

func sampleOutbound(timestamp time.Time) model.OutboundMessage {
	return model.OutboundMessage{
		EventID: "out-1",
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
