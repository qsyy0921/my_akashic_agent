package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobExternalLeaseReadinessChecker interface {
	CheckAgentJobExternalLeaseReadiness(ctx context.Context, cmd command.CheckAgentJobExternalLeaseReadinessCommand) (query.AgentJobExternalLeaseReadinessView, error)
}

type AgentJobExternalLeasePlanner interface {
	PlanAgentJobExternalLease(ctx context.Context, cmd command.PlanAgentJobExternalLeaseCommand) (query.AgentJobExternalLeasePlanView, error)
}

type AgentJobExternalLeasePreflightChecker interface {
	CheckAgentJobExternalLeasePreflight(ctx context.Context, filter query.AgentJobExternalLeasePreflightFilter) (query.AgentJobExternalLeasePreflightView, error)
}

type AgentJobExternalLeaseLauncherBundleViewer interface {
	GetAgentJobExternalLeaseLauncherBundle(ctx context.Context, filter query.AgentJobExternalLeaseLauncherBundleFilter) (query.AgentJobExternalLeaseLauncherBundleView, error)
}

type AgentJobExternalLeaseCutoverDiffViewer interface {
	GetAgentJobExternalLeaseCutoverDiff(ctx context.Context, filter query.AgentJobExternalLeaseCutoverDiffFilter) (query.AgentJobExternalLeaseCutoverDiffView, error)
}
