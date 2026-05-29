package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type MessageSendService struct {
	eventBus          outport.MessageEventBus
	sendLedger        outport.SendLedger
	outboxRepository  outport.OutboxRepository
	outboxQueue       outport.OutboxQueue
	defaultMaxAttempt int
}

func NewMessageSendService(
	eventBus outport.MessageEventBus,
	sendLedger outport.SendLedger,
	outboxRepository outport.OutboxRepository,
	outboxQueue outport.OutboxQueue,
) *MessageSendService {
	return &MessageSendService{
		eventBus:          eventBus,
		sendLedger:        sendLedger,
		outboxRepository:  outboxRepository,
		outboxQueue:       outboxQueue,
		defaultMaxAttempt: 3,
	}
}

func (s *MessageSendService) Send(ctx context.Context, cmd command.SendMessageCommand) error {
	if s == nil || s.eventBus == nil {
		return errors.New("message send service requires an event bus")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}
	if cmd.WithBotProtocol {
		nonce := cmd.ProtocolNonce
		if nonce == "" {
			nonce = domainservice.NewNonce()
		}
		fromBot := cmd.ProtocolFromBot
		if fromBot == "" {
			fromBot = cmd.Channel.AccountID
		}
		hop := cmd.ProtocolNextHop
		if hop <= 0 {
			hop = 1
		}
		cmd.Content = domainservice.PrependBotProtocol(cmd.Content, model.BotProtocol{
			FromBotID: fromBot,
			Nonce:     nonce,
			Hop:       hop,
		})
	}

	outbound := assembler.ToOutbound(cmd)
	if err := outbound.Validate(); err != nil {
		return err
	}

	if s.sendLedger != nil {
		if err := s.sendLedger.RecordSent(ctx, model.SendRecord{
			FromBotID:      outbound.Channel.AccountID,
			ConversationID: outbound.Channel.ConversationID,
			ContentHash:    domainservice.ContentHash(outbound.Content),
			Timestamp:      outbound.Timestamp,
		}); err != nil {
			return err
		}
	}
	if s.outboxRepository != nil {
		delivery, err := model.NewOutboxDelivery(outbound, maxAttempts(outbound.Metadata, s.defaultMaxAttempt), cmd.Timestamp)
		if err != nil {
			return err
		}
		if err := s.outboxRepository.SaveOutboxDelivery(ctx, delivery); err != nil {
			return err
		}
		if s.outboxQueue != nil {
			if err := s.outboxQueue.EnqueueOutboxDelivery(ctx, delivery); err != nil {
				return err
			}
		}
	}

	return s.eventBus.PublishOutbound(ctx, outbound)
}

func maxAttempts(metadata map[string]string, fallback int) int {
	if fallback <= 0 {
		fallback = 3
	}
	if metadata == nil {
		return fallback
	}
	value := metadata["max_attempts"]
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

