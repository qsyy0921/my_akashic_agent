package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type InboxEventRepository interface {
	SaveInboxEvent(ctx context.Context, event model.InboxEvent) error
	FindInboxEvent(ctx context.Context, eventID string) (model.InboxEvent, bool, error)
	ListInboxEvents(ctx context.Context, filter query.InboxEventFilter) ([]model.InboxEvent, error)
}
