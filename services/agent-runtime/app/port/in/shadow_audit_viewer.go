package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ShadowAuditViewer interface {
	ListObserved(ctx context.Context, limit int) ([]query.ShadowObservedEventView, error)
}

