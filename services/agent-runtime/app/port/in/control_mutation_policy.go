package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ControlMutationPolicyViewer interface {
	GetControlMutationPolicy(ctx context.Context, filter query.ControlMutationPolicyFilter) (query.ControlMutationPolicyView, error)
}
