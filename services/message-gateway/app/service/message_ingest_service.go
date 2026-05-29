package service

import (
	"context"
	"errors"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/service"
)

type MessageIngestService struct {
	eventBus    outport.MessageEventBus
	auditLog    outport.AuditLog
	sendLedger  outport.SendLedger
	nonceLedger outport.NonceLedger
	classifier  domainservice.ProvenanceClassifier
	loopGuard   domainservice.LoopGuard
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
