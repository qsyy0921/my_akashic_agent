package service_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

func TestDeliveryPlannerMapsAccountAndSplitsMediaSteps(t *testing.T) {
	planner := domainservice.NewDeliveryPlanner()
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	delivery := testOutboxDelivery(t, model.OutboundMessage{
		EventID: "qq:private:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: model.ConversationTypePrivate,
		},
		Content: "hello",
		Attachments: []model.Attachment{
			{Kind: model.AttachmentKindImage, URL: "file:///E:/agent/akashic/.tmp/a.png"},
			{Kind: model.AttachmentKindFile, URL: "file:///E:/agent/akashic/.tmp/a.pdf"},
		},
		Timestamp: now,
	})

	plan, err := planner.Plan(delivery, domainservice.DeliveryPlannerConfig{
		ChannelByAccount: map[string]string{"2365524513": "qq_2365524513"},
	})
	if err != nil {
		t.Fatalf("plan dispatch: %v", err)
	}

	if plan.Channel != "qq_2365524513" || plan.ChatID != "1049511700" {
		t.Fatalf("unexpected route: %+v", plan)
	}
	if plan.StepCount != 2 || len(plan.Steps) != 2 {
		t.Fatalf("expected two steps, got %+v", plan.Steps)
	}
	if plan.Steps[0].Kind != model.DeliveryDispatchStepImage ||
		plan.Steps[0].Message != "hello" ||
		plan.Steps[0].Image != "E:/agent/akashic/.tmp/a.png" {
		t.Fatalf("unexpected image step: %+v", plan.Steps[0])
	}
	if plan.Steps[1].Kind != model.DeliveryDispatchStepFile ||
		plan.Steps[1].Message != "" ||
		plan.Steps[1].File != "E:/agent/akashic/.tmp/a.pdf" {
		t.Fatalf("unexpected file step: %+v", plan.Steps[1])
	}
}

func TestDeliveryPlannerCreatesTextStepWhenNoMedia(t *testing.T) {
	planner := domainservice.NewDeliveryPlanner()
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	delivery := testOutboxDelivery(t, model.OutboundMessage{
		EventID: "telegram:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindTelegram,
			AccountID:        "telegram-bot",
			ConversationID:   "8655199155",
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   "hello telegram",
		Timestamp: now,
	})

	plan, err := planner.Plan(delivery, domainservice.DeliveryPlannerConfig{})
	if err != nil {
		t.Fatalf("plan dispatch: %v", err)
	}

	if len(plan.Steps) != 1 || plan.Steps[0].Kind != model.DeliveryDispatchStepText {
		t.Fatalf("unexpected steps: %+v", plan.Steps)
	}
	if plan.Steps[0].Channel != "telegram" || plan.Steps[0].Message != "hello telegram" {
		t.Fatalf("unexpected text step: %+v", plan.Steps[0])
	}
}

func TestDeliveryPlannerReturnsRouteErrorForMissingChannel(t *testing.T) {
	planner := domainservice.NewDeliveryPlanner()
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	delivery := testOutboxDelivery(t, model.OutboundMessage{
		EventID: "bad:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   "hello",
		Timestamp: now,
	})
	delivery.Message.Channel.Kind = ""

	_, err := planner.Plan(delivery, domainservice.DeliveryPlannerConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
	kinded, ok := err.(interface{ DeliveryErrorKind() string })
	if !ok || kinded.DeliveryErrorKind() != string(model.DeliveryErrorRoute) {
		t.Fatalf("expected route error, got %T %v", err, err)
	}
}

func testOutboxDelivery(t *testing.T, message model.OutboundMessage) model.OutboxDelivery {
	t.Helper()
	delivery, err := model.NewOutboxDelivery(message, 3, message.Timestamp)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	return delivery
}
