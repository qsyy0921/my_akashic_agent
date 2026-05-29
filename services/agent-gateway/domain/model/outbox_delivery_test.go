package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
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
	if err := delivery.MarkFailed("first failure", now.Add(2*time.Second)); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if delivery.Status != model.DeliveryFailed {
		t.Fatalf("expected failed, got %s", delivery.Status)
	}

	if err := delivery.Retry(now.Add(3 * time.Second)); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if err := delivery.MarkDispatching(now.Add(4 * time.Second)); err != nil {
		t.Fatalf("second dispatch: %v", err)
	}
	if err := delivery.MarkFailed("second failure", now.Add(5*time.Second)); err != nil {
		t.Fatalf("second failure: %v", err)
	}
	if delivery.Status != model.DeliveryDeadLettered {
		t.Fatalf("expected dead letter, got %s", delivery.Status)
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
