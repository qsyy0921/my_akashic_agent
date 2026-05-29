package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type MessageIngestService struct {
	eventBus    outport.MessageEventBus
	auditLog    outport.AuditLog
	sendLedger  outport.SendLedger
	nonceLedger outport.NonceLedger
	classifier  domainservice.ProvenanceClassifier
	loopGuard   domainservice.LoopGuard
	mediaAssets outport.MediaAssetRepository
}

func NewMessageIngestService(
	eventBus outport.MessageEventBus,
	auditLog outport.AuditLog,
	sendLedger outport.SendLedger,
	nonceLedger outport.NonceLedger,
	classifier domainservice.ProvenanceClassifier,
	loopGuard domainservice.LoopGuard,
) *MessageIngestService {
	return &MessageIngestService{
		eventBus:    eventBus,
		auditLog:    auditLog,
		sendLedger:  sendLedger,
		nonceLedger: nonceLedger,
		classifier:  classifier,
		loopGuard:   loopGuard,
	}
}

func NewMessageIngestServiceWithMediaAssets(
	eventBus outport.MessageEventBus,
	auditLog outport.AuditLog,
	sendLedger outport.SendLedger,
	nonceLedger outport.NonceLedger,
	classifier domainservice.ProvenanceClassifier,
	loopGuard domainservice.LoopGuard,
	mediaAssets outport.MediaAssetRepository,
) *MessageIngestService {
	service := NewMessageIngestService(
		eventBus,
		auditLog,
		sendLedger,
		nonceLedger,
		classifier,
		loopGuard,
	)
	service.mediaAssets = mediaAssets
	return service
}

func (s *MessageIngestService) Ingest(ctx context.Context, cmd command.IngestMessageCommand) error {
	if s == nil || s.eventBus == nil {
		return errors.New("message ingest service requires an event bus")
	}

	envelope := assembler.ToEnvelope(cmd)
	if err := envelope.Validate(); err != nil {
		return err
	}

	envelope.Provenance = s.classifier.Classify(envelope)
	decision := s.loopGuard.Decide(envelope, s.sendLedger, s.nonceLedger)

	if err := s.eventBus.PublishObserved(ctx, envelope, decision); err != nil {
		return err
	}
	if err := s.registerEnvelopeAttachments(ctx, envelope); err != nil {
		return err
	}
	if s.auditLog != nil {
		if err := s.auditLog.RecordMessageDecision(ctx, envelope, decision); err != nil {
			return err
		}
	}
	if envelope.Provenance.Nonce != "" && s.nonceLedger != nil {
		if err := s.nonceLedger.RecordNonce(ctx, envelope.Provenance.Nonce, envelope.Timestamp); err != nil {
			return err
		}
	}
	if decision.Action != model.LoopActionAllow {
		return nil
	}

	return s.eventBus.PublishAgentInbound(ctx, envelope)
}

func (s *MessageIngestService) ShadowIngest(ctx context.Context, cmd command.IngestMessageCommand) (model.LoopDecision, error) {
	if s == nil || s.eventBus == nil {
		return model.LoopDecision{}, errors.New("message ingest service requires an event bus")
	}

	envelope := assembler.ToEnvelope(cmd)
	if err := envelope.Validate(); err != nil {
		return model.LoopDecision{}, err
	}

	envelope.Provenance = s.classifier.Classify(envelope)
	decision := s.loopGuard.Decide(envelope, s.sendLedger, s.nonceLedger)

	if err := s.eventBus.PublishObserved(ctx, envelope, decision); err != nil {
		return model.LoopDecision{}, err
	}
	if err := s.registerEnvelopeAttachments(ctx, envelope); err != nil {
		return model.LoopDecision{}, err
	}
	if s.auditLog != nil {
		if err := s.auditLog.RecordMessageDecision(ctx, envelope, decision); err != nil {
			return model.LoopDecision{}, err
		}
	}
	if envelope.Provenance.Nonce != "" && s.nonceLedger != nil {
		if err := s.nonceLedger.RecordNonce(ctx, envelope.Provenance.Nonce, envelope.Timestamp); err != nil {
			return model.LoopDecision{}, err
		}
	}

	return decision, nil
}

func (s *MessageIngestService) registerEnvelopeAttachments(ctx context.Context, envelope model.MessageEnvelope) error {
	if s == nil || s.mediaAssets == nil || len(envelope.Attachments) == 0 {
		return nil
	}
	for index, attachment := range envelope.Attachments {
		if strings.TrimSpace(attachment.URL) == "" && strings.TrimSpace(attachment.Name) == "" {
			continue
		}
		asset, err := mediaAssetFromAttachment(envelope, attachment, index+1)
		if err != nil {
			return err
		}
		if existing, ok, err := s.mediaAssets.FindMediaAsset(ctx, asset.AssetID); err != nil {
			return err
		} else if ok {
			_ = existing
			continue
		}
		if err := s.mediaAssets.SaveMediaAsset(ctx, asset); err != nil {
			return err
		}
	}
	return nil
}

func mediaAssetFromAttachment(envelope model.MessageEnvelope, attachment model.Attachment, index int) (model.MediaAsset, error) {
	assetID := strings.TrimSpace(attachment.ID)
	if assetID == "" {
		assetID = fmt.Sprintf(
			"asset:%s:%s:%s:%s:%s:%d",
			safeMediaPart(string(envelope.Channel.Kind)),
			safeMediaPart(envelope.Channel.AccountID),
			safeMediaPart(string(envelope.Channel.ConversationType)),
			safeMediaPart(envelope.Channel.ConversationID),
			safeMediaPart(envelope.EventID),
			index,
		)
	}
	metadata := map[string]string{
		"registered_from": "message_ingest",
		"source_event_id": envelope.EventID,
	}
	for key, value := range envelope.Metadata {
		if key == "shadow_mode" ||
			key == "observe_only" ||
			key == "session_key" ||
			key == "session_message_id" ||
			key == "platform_message_id" {
			metadata[key] = value
		}
	}
	return model.NewMediaAsset(envelope.Channel, model.MediaAssetSpec{
		AssetID:         assetID,
		SourceMessageID: envelope.EventID,
		SenderID:        envelope.Sender.ID,
		Kind:            mediaAssetKindFromAttachment(attachment.Kind),
		URL:             attachment.URL,
		MimeType:        attachment.MimeType,
		Name:            attachment.Name,
		SizeBytes:       attachment.SizeBytes,
		Metadata:        metadata,
	}, envelope.Timestamp)
}

func mediaAssetKindFromAttachment(kind model.AttachmentKind) model.MediaAssetKind {
	switch kind {
	case model.AttachmentKindImage:
		return model.MediaAssetImage
	case model.AttachmentKindFile:
		return model.MediaAssetFile
	default:
		return model.MediaAssetUnknown
	}
}

func safeMediaPart(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "\t", "_", "\n", "_", "\r", "_")
	value = replacer.Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
