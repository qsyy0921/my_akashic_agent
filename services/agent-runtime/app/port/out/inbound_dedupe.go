package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type InboundDedupeRepository interface {
	FindInboundDedupeRecord(ctx context.Context, scope string, messageKey string) (model.InboundDedupeRecord, bool, error)
	SaveInboundDedupeRecord(ctx context.Context, record model.InboundDedupeRecord) error
	DeleteExpiredInboundDedupeRecords(ctx context.Context, now time.Time) (int, error)
	ListInboundDedupeRecords(ctx context.Context, filter query.InboundDedupeFilter) ([]model.InboundDedupeRecord, error)
}
