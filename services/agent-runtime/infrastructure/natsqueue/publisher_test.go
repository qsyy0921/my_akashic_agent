package natsqueue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestOutboxSubjectAndNotificationAreWorkHints(t *testing.T) {
	delivery := sampleOutboxDelivery(t)
	subject := OutboxSubject("akashic.work", delivery)

	if subject != "akashic.work.outbox.qq.1049511700" {
		t.Fatalf("unexpected outbox subject: %s", subject)
	}
	notification := OutboxNotification(subject, delivery, time.Date(2026, 5, 30, 23, 0, 0, 0, time.UTC))
	if notification.WorkKind != "outbox_delivery" || notification.WorkID != "outbox-queue-1" {
		t.Fatalf("unexpected outbox notification: %+v", notification)
	}
	if !notification.StateAuthoritative || notification.ConsumerModel != "goroutine_worker_pool" {
		t.Fatalf("notification must keep Go state authoritative: %+v", notification)
	}
	if notification.Route.AccountID != "1049511700" || notification.Route.ConversationID != "2365524513" {
		t.Fatalf("unexpected route hint: %+v", notification.Route)
	}
	if _, err := json.Marshal(notification); err != nil {
		t.Fatalf("notification should marshal: %v", err)
	}
}

func TestAgentJobSubjectAndNotificationAreWorkHints(t *testing.T) {
	job := sampleAgentJob(t)
	subject := AgentJobSubject("akashic.work", job)

	if subject != "akashic.work.agent_job.rag_ingest" {
		t.Fatalf("unexpected agent job subject: %s", subject)
	}
	notification := AgentJobNotification(subject, job, time.Date(2026, 5, 30, 23, 1, 0, 0, time.UTC))
	if notification.WorkKind != "agent_job" || notification.WorkID != "job-queue-1" {
		t.Fatalf("unexpected agent job notification: %+v", notification)
	}
	if len(notification.SourceEventIDs) != 1 || notification.SourceEventIDs[0] != "qq:msg:1" {
		t.Fatalf("missing source event ids: %+v", notification)
	}
	if notification.Route.Kind != "qq" || notification.Route.ConversationType != "group" {
		t.Fatalf("unexpected route hint: %+v", notification.Route)
	}
	if _, err := json.Marshal(notification); err != nil {
		t.Fatalf("notification should marshal: %v", err)
	}
}

func TestSubjectTokensAreSanitized(t *testing.T) {
	delivery := sampleOutboxDelivery(t)
	delivery.Message.Channel.AccountID = "qq.104/951 1700"

	subject := OutboxSubject(".akashic work.", delivery)

	if subject != "akashic_work.outbox.qq.qq_104_951_1700" {
		t.Fatalf("unexpected sanitized subject: %s", subject)
	}
}

func sampleOutboxDelivery(t *testing.T) model.OutboxDelivery {
	t.Helper()
	now := time.Date(2026, 5, 30, 23, 0, 0, 0, time.UTC)
	message := model.OutboundMessage{
		EventID: "outbox-queue-1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   "hello",
		Timestamp: now,
		Metadata:  map[string]string{"trace_id": "trace-1"},
	}
	delivery, err := model.NewOutboxDelivery(message, 3, now)
	if err != nil {
		t.Fatalf("new outbox delivery: %v", err)
	}
	return delivery
}

func sampleAgentJob(t *testing.T) model.AgentJob {
	t.Helper()
	now := time.Date(2026, 5, 30, 23, 0, 0, 0, time.UTC)
	job, err := model.NewAgentJob(model.AgentJobSpec{
		JobID:   "job-queue-1",
		JobType: model.AgentJobRagIngest,
		AgentID: "akashic",
		Route: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "3219982",
			ConversationType: model.ConversationTypeGroup,
		},
		SourceEventIDs: []string{"qq:msg:1"},
		SourceAssetIDs: []string{"asset:1"},
		Payload:        map[string]string{"dataset_id": "ds1"},
		Metadata:       map[string]string{"observe_only": "true"},
	}, now)
	if err != nil {
		t.Fatalf("new agent job: %v", err)
	}
	return job
}
