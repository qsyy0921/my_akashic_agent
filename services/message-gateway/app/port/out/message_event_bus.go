package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
)

type MessageEventBus interface {
	PublishObserved(ctx context.Context, envelope model.MessageEnvelope, decision model.LoopDecision) error
	PublishAgentInbound(ctx context.Context, envelope model.MessageEnvelope) error
	PublishOutbound(ctx context.Context, message model.OutboundMessage) error
}
