package service

import (
	"context"
	"errors"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/service"
)

type MessageSendService struct {
	eventBus   outport.MessageEventBus
	sendLedger outport.SendLedger
}

func NewMessageSendService(eventBus outport.MessageEventBus, sendLedger outport.SendLedger) *MessageSendService {
	return &MessageSendService{eventBus: eventBus, sendLedger: sendLedger}
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

	return s.eventBus.PublishOutbound(ctx, outbound)
}
