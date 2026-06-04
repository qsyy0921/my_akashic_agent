package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ReceiverStatusRepository interface {
	SaveReceiverStatus(ctx context.Context, status model.ReceiverStatus) error
	ListReceiverStatuses(ctx context.Context) ([]model.ReceiverStatus, error)
	DeleteReceiverStatus(ctx context.Context, receiverID string) error
}
