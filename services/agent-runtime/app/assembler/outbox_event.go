package assembler

import (
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToOutboxDeliveryEventView(event model.OutboxDeliveryEvent) query.OutboxDeliveryEventView {
	return query.OutboxDeliveryEventView{
		EventID:    event.EventID,
		DeliveryID: event.DeliveryID,
		Channel: query.OutboxChannelView{
			Kind:             string(event.Channel.Kind),
			AccountID:        event.Channel.AccountID,
			ConversationID:   event.Channel.ConversationID,
			ConversationType: string(event.Channel.ConversationType),
		},
		EventType:      string(event.EventType),
		Status:         string(event.Status),
		Attempt:        event.Attempt,
		MaxAttempts:    event.MaxAttempts,
		LeaseOwner:     event.LeaseOwner,
		LeaseExpiresAt: formatTime(event.LeaseExpiresAt),
		ErrorKind:      string(event.ErrorKind),
		ErrorMessage:   event.ErrorMessage,
		OccurredAt:     formatTime(event.OccurredAt),
		Metadata:       event.Metadata,
	}
}

func ToOutboxDeliveryEventViews(items []model.OutboxDeliveryEvent) []query.OutboxDeliveryEventView {
	views := make([]query.OutboxDeliveryEventView, 0, len(items))
	for _, item := range items {
		views = append(views, ToOutboxDeliveryEventView(item))
	}
	return views
}
