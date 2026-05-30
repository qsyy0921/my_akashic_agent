package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ReceiverStatusManager interface {
	ReportReceiverStatus(ctx context.Context, cmd command.ReportReceiverStatusCommand) (query.ReceiverStatusesView, error)
	ListReceiverStatuses(ctx context.Context) (query.ReceiverStatusesView, error)
}
