package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ProactiveStateRepository interface {
	SaveProactiveDelivery(ctx context.Context, record model.ProactiveDeliveryRecord) error
	FindProactiveDelivery(ctx context.Context, sessionKey string, deliveryKey string) (model.ProactiveDeliveryRecord, bool, error)
	CountProactiveDeliveriesSince(ctx context.Context, sessionKey string, since time.Time) (int, error)
	ListProactiveDeliveries(ctx context.Context, filter query.ProactiveDeliveryFilter) ([]model.ProactiveDeliveryRecord, error)

	SaveProactiveContextOnly(ctx context.Context, record model.ProactiveContextOnlyRecord) error
	CountProactiveContextOnlySince(ctx context.Context, sessionKey string, since time.Time) (int, error)

	SaveProactiveSessionMark(ctx context.Context, mark model.ProactiveSessionMark) error
	FindProactiveSessionMark(ctx context.Context, sessionKey string, key string) (model.ProactiveSessionMark, bool, error)
}
