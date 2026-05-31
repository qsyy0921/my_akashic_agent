package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type InboundDedupeManager interface {
	Check(ctx context.Context, cmd command.CheckInboundDedupeCommand) (query.InboundDedupeView, error)
	List(ctx context.Context, filter query.InboundDedupeFilter) (query.InboundDedupeRecordsView, error)
	Metrics(ctx context.Context, filter query.InboundDedupeMetricsFilter) (query.InboundDedupeMetricsView, error)
}
