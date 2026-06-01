package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ReceiverStatusManager interface {
	ReportReceiverStatus(ctx context.Context, cmd command.ReportReceiverStatusCommand) (query.ReceiverStatusesView, error)
	ListReceiverStatuses(ctx context.Context) (query.ReceiverStatusesView, error)
	AcquireReceiverLease(ctx context.Context, cmd command.AcquireReceiverLeaseCommand) (query.ReceiverLeaseView, error)
	RenewReceiverLease(ctx context.Context, cmd command.RenewReceiverLeaseCommand) (query.ReceiverLeaseView, error)
	ReleaseReceiverLease(ctx context.Context, cmd command.ReleaseReceiverLeaseCommand) (query.ReceiverLeaseView, error)
	CleanupExpiredReceiverLeases(ctx context.Context, cmd command.CleanupExpiredReceiverLeasesCommand) (query.ReceiverLeaseCleanupView, error)
	ListReceiverLeases(ctx context.Context) (query.ReceiverLeasesView, error)
}
