package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type InboxEventViewer interface {
	GetInboxEvent(ctx context.Context, eventID string) (query.InboxEventView, error)
	ListInboxEvents(ctx context.Context, filter query.InboxEventFilter) ([]query.InboxEventView, error)
}
