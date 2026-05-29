package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/query"
)

type ShadowAuditViewer interface {
	ListObserved(ctx context.Context, limit int) ([]query.ShadowObservedEventView, error)
}
