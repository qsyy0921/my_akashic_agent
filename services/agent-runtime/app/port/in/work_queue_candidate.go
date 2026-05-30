package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type WorkQueueCandidateComparer interface {
	CompareWorkQueueCandidate(ctx context.Context, cmd command.CompareWorkQueueCandidateCommand) (query.QueueCandidateComparisonView, error)
}

type WorkQueueCompareDiagnosticReader interface {
	SnapshotDualReadDiagnostics(ctx context.Context) (query.QueueDualReadDiagnostics, error)
}
