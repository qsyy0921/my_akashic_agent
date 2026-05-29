package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ShadowObservedRecord struct {
	Envelope model.MessageEnvelope
	Decision model.LoopDecision
}

type ShadowAuditReader interface {
	ListShadowObserved(ctx context.Context, limit int) ([]ShadowObservedRecord, error)
}

