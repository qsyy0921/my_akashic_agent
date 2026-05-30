package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToOutboxDeliveryView(delivery model.OutboxDelivery) query.OutboxDeliveryView {
	attachments := make([]query.OutboxAttachmentView, 0, len(delivery.Message.Attachments))
	for _, item := range delivery.Message.Attachments {
		attachments = append(attachments, query.OutboxAttachmentView{
			ID:        item.ID,
			Kind:      string(item.Kind),
			URL:       item.URL,
			MimeType:  item.MimeType,
			Name:      item.Name,
			SizeBytes: item.SizeBytes,
		})
	}

	return query.OutboxDeliveryView{
		EventID: delivery.Message.EventID,
		Channel: query.OutboxChannelView{
			Kind:             string(delivery.Message.Channel.Kind),
			AccountID:        delivery.Message.Channel.AccountID,
			ConversationID:   delivery.Message.Channel.ConversationID,
			ConversationType: string(delivery.Message.Channel.ConversationType),
		},
		Content:        delivery.Message.Content,
		Attachments:    attachments,
		Status:         string(delivery.Status),
		Attempts:       delivery.Attempts,
		MaxAttempts:    delivery.MaxAttempts,
		LeaseOwner:     delivery.LeaseOwner,
		LeaseExpiresAt: formatTime(delivery.LeaseExpiresAt),
		ErrorMessage:   delivery.ErrorMessage,
		CreatedAt:      formatTime(delivery.CreatedAt),
		UpdatedAt:      formatTime(delivery.UpdatedAt),
		Metadata:       delivery.Message.Metadata,
	}
}

func ToOutboxDeliveryViews(items []model.OutboxDelivery) []query.OutboxDeliveryView {
	views := make([]query.OutboxDeliveryView, 0, len(items))
	for _, item := range items {
		views = append(views, ToOutboxDeliveryView(item))
	}
	return views
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
