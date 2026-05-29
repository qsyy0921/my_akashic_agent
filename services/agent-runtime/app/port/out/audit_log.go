package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type AuditLog interface {
	RecordMessageDecision(ctx context.Context, envelope model.MessageEnvelope, decision model.LoopDecision) error
}

