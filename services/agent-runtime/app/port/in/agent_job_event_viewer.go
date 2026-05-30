package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobEventViewer interface {
	List(ctx context.Context, filter query.AgentJobEventFilter) ([]query.AgentJobEventView, error)
}
