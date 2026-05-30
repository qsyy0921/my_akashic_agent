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

func TestWorkLeaseCommandFromNATSMessageUsesWorkNotificationIDs(t *testing.T) {
	notification := WorkNotification{
		WorkKind:    "outbox_delivery",
		WorkID:      "outbox:1",
		AggregateID: "outbox:1",
		Status:      "queued",
		Subject:     "akashic.work.outbox.qq.1049511700",
		Metadata:    map[string]string{"trace_id": "trace-lease-1"},
	}
	raw, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}
	observedAt := time.Date(2026, 5, 31, 0, 10, 0, 0, time.UTC)

	cmd := WorkLeaseCommandFromNATSMessage("akashic.work.outbox.qq.1049511700", raw, observedAt)

	if cmd.WorkKind != "outbox_delivery" || cmd.WorkID != "outbox:1" || cmd.AggregateID != "outbox:1" {
		t.Fatalf("unexpected lease command ids: %+v", cmd)
	}
	if cmd.Subject != "akashic.work.outbox.qq.1049511700" || cmd.Metadata["trace_id"] != "trace-lease-1" {
		t.Fatalf("unexpected lease command route: %+v", cmd)
	}
}

func TestNormalizeExternalLeaseConsumerConfigUsesDedicatedDefaultDurable(t *testing.T) {
	config := normalizeExternalLeaseConsumerConfig(ExternalLeaseConsumerConfig{
		URL:                 "nats://127.0.0.1:4222",
		ConsumerConcurrency: 4,
		MaxInFlight:         1,
	})

	if config.Durable != defaultExternalLeaseDurable {
		t.Fatalf("unexpected default durable: %q", config.Durable)
	}
	if config.MaxInFlight != 4 {
		t.Fatalf("max in-flight should be raised to concurrency, got %d", config.MaxInFlight)
	}
	if config.WorkerID == "" || config.LeaseTTLSeconds <= 0 {
		t.Fatalf("expected worker defaults, got %#v", config)
	}
	if config.NackDelay != defaultExternalLeaseNackDelay {
		t.Fatalf("unexpected default nack delay: %s", config.NackDelay)
	}
	if got := externalLeaseSubscriptionSubject(config.SubjectPrefix, config.IncludeAgentJobs); got != "akashic.work.outbox.>" {
		t.Fatalf("unexpected outbox-only subscription subject: %s", got)
	}
}

func TestNormalizeExternalLeaseConsumerConfigCanIncludeAgentJobSubjects(t *testing.T) {
	config := normalizeExternalLeaseConsumerConfig(ExternalLeaseConsumerConfig{
		URL:              "nats://127.0.0.1:4222",
		SubjectPrefix:    "akashic.work",
		IncludeAgentJobs: true,
	})

	if config.Durable != defaultExternalLeaseAllDurable {
		t.Fatalf("unexpected default all-work durable: %q", config.Durable)
	}
	if got := externalLeaseSubscriptionSubject(config.SubjectPrefix, config.IncludeAgentJobs); got != "akashic.work.>" {
		t.Fatalf("unexpected all-work subscription subject: %s", got)
	}
}
