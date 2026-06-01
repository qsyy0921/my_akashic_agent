package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ControlMutationPreflightChecker interface {
	CheckControlMutationPreflight(ctx context.Context, preflight query.ControlMutationPreflight) (query.ControlMutationPreflightView, error)
}
