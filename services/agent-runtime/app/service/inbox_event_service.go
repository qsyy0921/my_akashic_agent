package service

import (
	"context"
	"errors"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type InboxEventService struct {
	repository outport.InboxEventRepository
}

func NewInboxEventService(repository outport.InboxEventRepository) *InboxEventService {
	return &InboxEventService{repository: repository}
}

func (s *InboxEventService) GetInboxEvent(ctx context.Context, eventID string) (query.InboxEventView, error) {
	if s == nil || s.repository == nil {
		return query.InboxEventView{}, errors.New("inbox event service requires a repository")
	}
	event, ok, err := s.repository.FindInboxEvent(ctx, eventID)
	if err != nil {
		return query.InboxEventView{}, err
	}
	if !ok {
		return query.InboxEventView{}, errors.New("inbox event not found")
	}
	return toInboxEventView(event), nil
}

func (s *InboxEventService) ListInboxEvents(ctx context.Context, filter query.InboxEventFilter) ([]query.InboxEventView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("inbox event service requires a repository")
	}
	events, err := s.repository.ListInboxEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	views := make([]query.InboxEventView, 0, len(events))
	for _, event := range events {
		views = append(views, toInboxEventView(event))
	}
	return views, nil
}

func toInboxEventView(event model.InboxEvent) query.InboxEventView {
	envelope := event.Envelope
	return query.InboxEventView{
		EventID: envelope.EventID,
		Channel: query.InboxChannelView{
			Kind:             string(envelope.Channel.Kind),
			AccountID:        envelope.Channel.AccountID,
			ConversationID:   envelope.Channel.ConversationID,
			ConversationType: string(envelope.Channel.ConversationType),
		},
		Sender: query.InboxSenderView{
			ID:          envelope.Sender.ID,
			DisplayName: envelope.Sender.DisplayName,
			Kind:        string(envelope.Sender.Kind),
		},
		Content:         envelope.Content,
		Timestamp:       envelope.Timestamp.Format(time.RFC3339Nano),
		ReceivedAt:      event.ReceivedAt.Format(time.RFC3339Nano),
		AttachmentCount: len(envelope.Attachments),
		Attachments:     toAttachmentViews(envelope.Attachments),
		DecisionAction:  string(event.Decision.Action),
		DecisionReason:  event.Decision.Reason,
		ObserveOnly:     event.ObserveOnly(),
		Metadata:        envelope.Metadata,
		Provenance: query.InboxProvenanceView{
			Type:           string(envelope.Provenance.Type),
			FromBotID:      envelope.Provenance.FromBotID,
			ContentHash:    envelope.Provenance.ContentHash,
			Nonce:          envelope.Provenance.Nonce,
			Hop:            envelope.Provenance.Hop,
			HasProtocolTag: envelope.Provenance.HasProtocolTag,
		},
	}
}
