package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ReceiverLeaseRepository interface {
	SaveReceiverLease(ctx context.Context, lease model.ReceiverLease) error
	DeleteReceiverLease(ctx context.Context, receiverID string) error
	ListReceiverLeases(ctx context.Context) ([]model.ReceiverLease, error)
}
