package natsqueue

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCompareCommandFromNATSMessageUsesWorkNotificationIDs(t *testing.T) {
	notification := WorkNotification{
		WorkKind:    "agent_job",
		WorkID:      "job:1",
		AggregateID: "job:1",
		Status:      "pending",
		Subject:     "akashic.work.agent_job.rag_ingest",
		Metadata:    map[string]string{"trace_id": "trace-1"},
	}
	raw, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}
	observedAt := time.Date(2026, 5, 30, 23, 55, 0, 0, time.UTC)

	cmd := CompareCommandFromNATSMessage("akashic.work.agent_job.rag_ingest", raw, observedAt)

	if cmd.WorkKind != "agent_job" || cmd.WorkID != "job:1" || cmd.AggregateID != "job:1" {
		t.Fatalf("unexpected compare command ids: %+v", cmd)
	}
	if cmd.Subject != "akashic.work.agent_job.rag_ingest" || cmd.Metadata["trace_id"] != "trace-1" {
		t.Fatalf("unexpected compare command route: %+v", cmd)
	}
	if !cmd.ObservedAt.Equal(observedAt) {
		t.Fatalf("unexpected observed time: %s", cmd.ObservedAt)
	}
}

func TestCompareCommandFromNATSMessageHandlesInvalidPayload(t *testing.T) {
	cmd := CompareCommandFromNATSMessage("akashic.work.bad", []byte("{"), time.Date(2026, 5, 30, 23, 56, 0, 0, time.UTC))

	if cmd.WorkKind != "invalid_payload" || cmd.WorkID != "akashic.work.bad" || cmd.Metadata["decode_error"] == "" {
		t.Fatalf("unexpected invalid payload command: %+v", cmd)
	}
}

func TestNormalizeCompareConsumerConfigUsesDedicatedDefaultDurable(t *testing.T) {
	config := normalizeCompareConsumerConfig(CompareConsumerConfig{
		URL:                 "nats://127.0.0.1:4222",
		ConsumerConcurrency: 3,
		MaxInFlight:         1,
	})

	if config.Durable != defaultCompareDurable {
		t.Fatalf("unexpected default durable: %q", config.Durable)
	}
	if config.MaxInFlight != 3 {
		t.Fatalf("max in-flight should be raised to concurrency, got %d", config.MaxInFlight)
	}
}
