package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestDeliveryDispatchServiceBlocksObserveOnlyGroupReplies(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 6, 2, 15, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(model.OutboundMessage{
		EventID: "qq:group:observe-only:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: model.ConversationTypeGroup,
		},
		Content:   "should not send",
		Timestamp: now,
	}, 1, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	observeTargets := appservice.NewObserveTargetService()
	if _, err := observeTargets.SyncObserveTargets(ctx, command.SyncObserveTargetsCommand{
		Source: "python_config",
		Targets: []command.ObserveTargetCommand{{
			TargetID: "qq:1049511700:group:27234224",
			Channel: command.ChannelCommand{
				Kind:             "qq",
				AccountID:        "1049511700",
				ConversationID:   "27234224",
				ConversationType: "group",
			},
			ObserveOnly:  true,
			ReplyAllowed: false,
			Enabled:      true,
		}},
		Timestamp: now,
	}); err != nil {
		t.Fatalf("sync observe target: %v", err)
	}

	service := appservice.NewDeliveryDispatchServiceWithObserveTargetsAndAdapters(store, observeTargets)
	_, err = service.Plan(ctx, command.PlanDeliveryDispatchCommand{
		EventID: "qq:group:observe-only:1",
	})
	if err == nil {
		t.Fatal("expected observe-only group reply to be blocked")
	}
	kinded, ok := err.(interface{ DeliveryErrorKind() string })
	if !ok || kinded.DeliveryErrorKind() != string(model.DeliveryErrorRoute) {
		t.Fatalf("expected route error, got %T %v", err, err)
	}
	if err.Error() != "observe-only target does not allow replies" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestDeliveryDispatchServiceBlocksAllQQGroupReplies(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 6, 2, 15, 2, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(model.OutboundMessage{
		EventID: "qq:group:disabled:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "2365524513",
			ConversationID:   "284331268",
			ConversationType: model.ConversationTypeGroup,
		},
		Content:   "should never send to qq groups",
		Timestamp: now,
	}, 1, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}

	service := appservice.NewDeliveryDispatchServiceWithObserveTargetsAdaptersAndPolicy(store, nil, false)
	_, err = service.Plan(ctx, command.PlanDeliveryDispatchCommand{
		EventID: "qq:group:disabled:1",
	})
	if err == nil {
		t.Fatal("expected qq group reply to be blocked")
	}
	kinded, ok := err.(interface{ DeliveryErrorKind() string })
	if !ok || kinded.DeliveryErrorKind() != string(model.DeliveryErrorRoute) {
		t.Fatalf("expected route error, got %T %v", err, err)
	}
	if err.Error() != "qq group sends are disabled" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestDeliveryDispatchServiceAllowsPrivateRepliesOutsideObserveOnlyTargets(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 6, 2, 15, 5, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(model.OutboundMessage{
		EventID: "qq:private:allowed:1",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: model.ConversationTypePrivate,
		},
		Content:   "allowed private reply",
		Timestamp: now,
	}, 1, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	observeTargets := appservice.NewObserveTargetService()
	if _, err := observeTargets.SyncObserveTargets(ctx, command.SyncObserveTargetsCommand{
		Source: "python_config",
		Targets: []command.ObserveTargetCommand{{
			TargetID: "qq:1049511700:group:27234224",
			Channel: command.ChannelCommand{
				Kind:             "qq",
				AccountID:        "1049511700",
				ConversationID:   "27234224",
				ConversationType: "group",
			},
			ObserveOnly:  true,
			ReplyAllowed: false,
			Enabled:      true,
		}},
		Timestamp: now,
	}); err != nil {
		t.Fatalf("sync observe target: %v", err)
	}

	service := appservice.NewDeliveryDispatchServiceWithObserveTargetsAndAdapters(store, observeTargets)
	plan, err := service.Plan(ctx, command.PlanDeliveryDispatchCommand{
		EventID:          "qq:private:allowed:1",
		ChannelByAccount: map[string]string{"1049511700": "qq_1049511700"},
	})
	if err != nil {
		t.Fatalf("plan private delivery: %v", err)
	}
	if plan.Channel != "qq_1049511700" || plan.ChatID != "2365524513" || plan.StepCount != 1 {
		t.Fatalf("unexpected private plan: %+v", plan)
	}
}
