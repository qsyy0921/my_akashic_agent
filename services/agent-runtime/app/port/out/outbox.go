package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type OutboxRepository interface {
	SaveOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
	FindOutboxDelivery(ctx context.Context, eventID string) (model.OutboxDelivery, bool, error)
	ListOutboxDeliveries(ctx context.Context, limit int) ([]model.OutboxDelivery, error)
	FindLeaseableOutboxDelivery(ctx context.Context, filter OutboxLeaseFilter) (model.OutboxDelivery, bool, error)
}

type OutboxLeaseFilter struct {
	Now                                       time.Time
	BlockedAccountKeys                        map[string]struct{}
	AllowedStepKinds                          map[model.DeliveryDispatchStepKind]struct{}
	AllowedStepKindsByAccount                 map[string]map[model.DeliveryDispatchStepKind]struct{}
	AllowedStepKindsByAccountConversationType map[string]map[model.ConversationType]map[model.DeliveryDispatchStepKind]struct{}
	AllowedStepKindsByAccountConversationID   map[string]map[model.ConversationType]map[string]map[model.DeliveryDispatchStepKind]struct{}
}

func (f OutboxLeaseFilter) Allows(delivery model.OutboxDelivery) bool {
	if len(f.BlockedAccountKeys) > 0 {
		if _, blocked := f.BlockedAccountKeys[delivery.Message.Channel.AccountKey()]; blocked {
			return false
		}
	}
	allowedStepKinds := f.AllowedStepKinds
	accountConversationScoped := false
	if len(f.AllowedStepKindsByAccountConversationID) > 0 {
		if scopedByType, ok := f.AllowedStepKindsByAccountConversationID[delivery.Message.Channel.AccountID]; ok && len(scopedByType) > 0 {
			if scopedByID, ok := scopedByType[delivery.Message.Channel.ConversationType]; ok && len(scopedByID) > 0 {
				if scoped, ok := scopedByID[delivery.Message.Channel.ConversationID]; ok && len(scoped) > 0 {
					allowedStepKinds = scoped
					accountConversationScoped = true
				}
			}
		}
	}
	if len(f.AllowedStepKindsByAccountConversationType) > 0 {
		if scopedByType, ok := f.AllowedStepKindsByAccountConversationType[delivery.Message.Channel.AccountID]; ok && len(scopedByType) > 0 {
			if scoped, ok := scopedByType[delivery.Message.Channel.ConversationType]; ok && len(scoped) > 0 {
				allowedStepKinds = scoped
				accountConversationScoped = true
			}
		}
	}
	if !accountConversationScoped && len(f.AllowedStepKindsByAccount) > 0 {
		if scoped, ok := f.AllowedStepKindsByAccount[delivery.Message.Channel.AccountID]; ok && len(scoped) > 0 {
			allowedStepKinds = scoped
		}
	}
	if len(allowedStepKinds) == 0 {
		return true
	}
	for _, kind := range requiredDeliveryStepKinds(delivery) {
		if _, ok := allowedStepKinds[kind]; !ok {
			return false
		}
	}
	return true
}

func requiredDeliveryStepKinds(delivery model.OutboxDelivery) []model.DeliveryDispatchStepKind {
	kinds := make([]model.DeliveryDispatchStepKind, 0, len(delivery.Message.Attachments)+1)
	seen := make(map[model.DeliveryDispatchStepKind]struct{})
	add := func(kind model.DeliveryDispatchStepKind) {
		if _, ok := seen[kind]; ok {
			return
		}
		seen[kind] = struct{}{}
		kinds = append(kinds, kind)
	}
	for _, attachment := range delivery.Message.Attachments {
		if attachment.Kind == model.AttachmentKindImage {
			add(model.DeliveryDispatchStepImage)
			continue
		}
		add(model.DeliveryDispatchStepFile)
	}
	if delivery.Message.Content != "" || len(kinds) == 0 {
		add(model.DeliveryDispatchStepText)
	}
	return kinds
}

type OutboxQueue interface {
	EnqueueOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
}

type OutboxDeliveryEventSink interface {
	AppendOutboxDeliveryEvent(ctx context.Context, event model.OutboxDeliveryEvent) error
}

type OutboxDeliveryEventStore interface {
	OutboxDeliveryEventSink
	ListOutboxDeliveryEvents(ctx context.Context, filter query.OutboxDeliveryEventFilter) ([]model.OutboxDeliveryEvent, error)
}
