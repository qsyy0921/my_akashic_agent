package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type RuntimeConfigViewer interface {
	GetRuntimeConfig(ctx context.Context) (query.RuntimeConfigView, error)
}
