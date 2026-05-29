package service

import (
	"context"
	"errors"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ShadowQueryService struct {
	reader outport.ShadowAuditReader
}

func NewShadowQueryService(reader outport.ShadowAuditReader) *ShadowQueryService {
	return &ShadowQueryService{reader: reader}
}

func (s *ShadowQueryService) ListObserved(ctx context.Context, limit int) ([]query.ShadowObservedEventView, error) {
	if s == nil || s.reader == nil {
		return nil, errors.New("shadow query service requires a reader")
	}
	records, err := s.reader.ListShadowObserved(ctx, limit)
	if err != nil {
		return nil, err
	}
	views := make([]query.ShadowObservedEventView, 0, len(records))
	for _, record := range records {
		envelope := record.Envelope
		views = append(views, query.ShadowObservedEventView{
			EventID:          envelope.EventID,
			Platform:         string(envelope.Channel.Kind),
			AccountID:        envelope.Channel.AccountID,
			ConversationID:   envelope.Channel.ConversationID,
			ConversationType: string(envelope.Channel.ConversationType),
			SenderID:         envelope.Sender.ID,
			Content:          envelope.Content,
			Timestamp:        envelope.Timestamp.Format(time.RFC3339Nano),
			AttachmentCount:  len(envelope.Attachments),
			Attachments:      toAttachmentViews(envelope.Attachments),
			DecisionAction:   string(record.Decision.Action),
			DecisionReason:   record.Decision.Reason,
			Metadata:         envelope.Metadata,
		})
	}
	return views, nil
}

func toAttachmentViews(attachments []model.Attachment) []query.ShadowAttachmentView {
	if len(attachments) == 0 {
		return nil
	}
	views := make([]query.ShadowAttachmentView, 0, len(attachments))
	for _, attachment := range attachments {
		views = append(views, query.ShadowAttachmentView{
			ID:        attachment.ID,
			Kind:      string(attachment.Kind),
			URL:       attachment.URL,
			MimeType:  attachment.MimeType,
			Name:      attachment.Name,
			SizeBytes: attachment.SizeBytes,
		})
	}
	return views
}

