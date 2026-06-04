package service

import (
	"context"
	"errors"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const agentJobExternalLeaseLauncherScriptPath = ".\\scripts\\start-agent-runtime.ps1"

type agentJobExternalLeaseLauncherBundlePlanner interface {
	PlanAgentJobExternalLease(ctx context.Context, cmd command.PlanAgentJobExternalLeaseCommand) (query.AgentJobExternalLeasePlanView, error)
}

type AgentJobExternalLeaseLauncherBundleService struct {
	planner agentJobExternalLeaseLauncherBundlePlanner
}

func NewAgentJobExternalLeaseLauncherBundleService(
	planner agentJobExternalLeaseLauncherBundlePlanner,
) *AgentJobExternalLeaseLauncherBundleService {
	return &AgentJobExternalLeaseLauncherBundleService{planner: planner}
}

func (s *AgentJobExternalLeaseLauncherBundleService) GetAgentJobExternalLeaseLauncherBundle(
	ctx context.Context,
	filter query.AgentJobExternalLeaseLauncherBundleFilter,
) (query.AgentJobExternalLeaseLauncherBundleView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentJobExternalLeaseLauncherBundleView{}, err
	}
	if s == nil || s.planner == nil {
		return query.AgentJobExternalLeaseLauncherBundleView{}, errors.New("agent job external lease launcher bundle requires planner")
	}

	plan, err := s.planner.PlanAgentJobExternalLease(ctx, command.PlanAgentJobExternalLeaseCommand{
		Readiness: command.CheckAgentJobExternalLeaseReadinessCommand{
			JobLimit:          filter.JobLimit,
			EventLimit:        filter.EventLimit,
			StaleAfterSeconds: filter.StaleAfterSeconds,
		},
		DesiredExecutionOwner: filter.DesiredExecutionOwner,
	})
	if err != nil {
		return query.AgentJobExternalLeaseLauncherBundleView{}, err
	}

	launcherParameters := agentJobExternalLeaseLauncherParameters(plan.DesiredExecutionOwner)
	environmentOverrides := agentJobExternalLeaseLauncherEnvironmentOverrides(plan.DesiredExecutionOwner)
	requiredExternalInputs := agentJobExternalLeaseLauncherRequiredExternalInputs(plan.DesiredExecutionOwner)
	blockers := append([]string(nil), plan.Blockers...)
	reason := "agent_job_external_lease_launcher_bundle_ready"
	ready := len(blockers) == 0 && len(launcherParameters) > 0
	if len(launcherParameters) == 0 {
		ready = false
		blockers = append(blockers, "desired_execution_owner_not_supported")
	}
	if !ready {
		reason = "agent_job_external_lease_launcher_bundle_blocked"
	}
	blockers = sortedUniqueSmokeStrings(blockers)

	return query.AgentJobExternalLeaseLauncherBundleView{
		Ready:                     ready,
		Reason:                    reason,
		Blockers:                  blockers,
		DesiredExecutionOwner:     plan.DesiredExecutionOwner,
		RecommendedExecutionOwner: plan.RecommendedExecutionOwner,
		CurrentExecutionOwner:     plan.CurrentExecutionOwner,
		Plan:                      plan,
		ScriptPath:                agentJobExternalLeaseLauncherScriptPath,
		LauncherParameters:        launcherParameters,
		EnvironmentOverrides:      environmentOverrides,
		RequiredExternalInputs:    requiredExternalInputs,
		VerificationSteps:         agentJobExternalLeaseLauncherVerificationSteps(plan.DesiredExecutionOwner),
		Notes: []string{
			"read-only launcher bundle for repo-local agent-runtime startup; it does not mutate the current runtime, approvals, queue state, or worker ownership",
			"QueueDSN remains an operator-supplied external input and is intentionally not echoed by this bundle",
		},
		SideEffect: "none",
	}, nil
}

func agentJobExternalLeaseLauncherParameters(desired string) map[string]string {
	parameters := map[string]string{}
	switch desired {
	case agentJobExternalLeaseNATSOwner:
		parameters["QueueBackend"] = "nats_jetstream"
		parameters["QueueMode"] = "external_lease"
		parameters["QueueExternalLeaseCutover"] = "true"
		parameters["QueueDualReadSmokePassed"] = "true"
		parameters["QueueStateLeaseWorkersDisabled"] = "true"
		parameters["QueueExternalLeaseAgentJobEnabled"] = "true"
		parameters["QueueAgentJobDuplicateSmokePassed"] = "true"
		parameters["QueueAgentJobFlowSmokePassed"] = "true"
		parameters["AgentJobStrictLeaseToken"] = "true"
	case agentJobExternalLeaseStateStoreOwner:
		parameters["QueueExternalLeaseAgentJobEnabled"] = "false"
		parameters["QueueAgentJobDuplicateSmokePassed"] = "false"
		parameters["QueueAgentJobFlowSmokePassed"] = "false"
	default:
		return nil
	}
	return parameters
}

func agentJobExternalLeaseLauncherEnvironmentOverrides(desired string) map[string]string {
	parameters := agentJobExternalLeaseLauncherParameters(desired)
	if len(parameters) == 0 {
		return nil
	}
	env := make(map[string]string, len(parameters))
	for parameter, value := range parameters {
		switch parameter {
		case "QueueBackend":
			env["AKASHIC_QUEUE_BACKEND"] = value
		case "QueueMode":
			env["AKASHIC_QUEUE_MODE"] = value
		case "QueueExternalLeaseCutover":
			env["AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER"] = value
		case "QueueDualReadSmokePassed":
			env["AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED"] = value
		case "QueueStateLeaseWorkersDisabled":
			env["AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED"] = value
		case "QueueExternalLeaseAgentJobEnabled":
			env["AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"] = value
		case "QueueAgentJobDuplicateSmokePassed":
			env["AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED"] = value
		case "QueueAgentJobFlowSmokePassed":
			env["AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED"] = value
		case "AgentJobStrictLeaseToken":
			env["AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN"] = value
		}
	}
	return env
}

func agentJobExternalLeaseLauncherRequiredExternalInputs(desired string) []query.AgentJobExternalLeaseLauncherBundleInputView {
	if desired != agentJobExternalLeaseNATSOwner {
		return nil
	}
	return []query.AgentJobExternalLeaseLauncherBundleInputView{
		{
			Name:           "NATS DSN",
			Parameter:      "QueueDSN",
			EnvironmentKey: "AKASHIC_QUEUE_DSN",
			Required:       true,
			Secret:         true,
			Detail:         "operator-supplied external queue connection string required before enabling agent_job result-ack through NATS external lease",
		},
	}
}

func agentJobExternalLeaseLauncherVerificationSteps(desired string) []query.AgentJobExternalLeasePlanStep {
	steps := []query.AgentJobExternalLeasePlanStep{
		{Phase: "verify", Action: "read_runtime_config", Method: "GET", Endpoint: "/v1/runtime-config"},
		{Phase: "verify", Action: "read_queue_backend", Method: "GET", Endpoint: "/v1/queue-backend"},
		{Phase: "verify", Action: "read_queue_topology", Method: "GET", Endpoint: "/v1/queue-topology"},
	}
	if desired == agentJobExternalLeaseNATSOwner {
		steps = append(steps,
			query.AgentJobExternalLeasePlanStep{Phase: "verify", Action: "recheck_agent_job_external_lease_preflight_without_approval", Method: "GET", Endpoint: "/v1/agent-job-external-lease/preflight"},
			query.AgentJobExternalLeasePlanStep{Phase: "verify", Action: "recheck_agent_job_external_lease_plan", Method: "GET", Endpoint: "/v1/agent-job-external-lease/plan"},
		)
	}
	return indexedAgentJobExternalLeasePlanSteps(steps)
}
