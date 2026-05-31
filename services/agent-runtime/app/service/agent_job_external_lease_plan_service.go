package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const (
	agentJobExternalLeaseNATSOwner       = "python_ai_worker_with_nats_result_ack"
	agentJobExternalLeaseStateStoreOwner = "python_ai_worker_state_store_lease"
)

type AgentJobExternalLeasePlanDeps struct {
	Readiness agentJobExternalLeasePlanReadinessChecker
}

type agentJobExternalLeasePlanReadinessChecker interface {
	CheckAgentJobExternalLeaseReadiness(ctx context.Context, cmd command.CheckAgentJobExternalLeaseReadinessCommand) (query.AgentJobExternalLeaseReadinessView, error)
}

type AgentJobExternalLeasePlanService struct {
	deps AgentJobExternalLeasePlanDeps
}

func NewAgentJobExternalLeasePlanService(deps AgentJobExternalLeasePlanDeps) *AgentJobExternalLeasePlanService {
	return &AgentJobExternalLeasePlanService{deps: deps}
}

func (s *AgentJobExternalLeasePlanService) PlanAgentJobExternalLease(
	ctx context.Context,
	cmd command.PlanAgentJobExternalLeaseCommand,
) (query.AgentJobExternalLeasePlanView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentJobExternalLeasePlanView{}, err
	}
	if s == nil {
		return query.AgentJobExternalLeasePlanView{}, errors.New("agent job external lease plan service is nil")
	}

	var blockers []string
	var readiness query.AgentJobExternalLeaseReadinessView
	if s.deps.Readiness == nil {
		blockers = append(blockers, "agent_job_external_lease_readiness_unavailable")
	} else {
		item, err := s.deps.Readiness.CheckAgentJobExternalLeaseReadiness(ctx, cmd.Readiness)
		if err != nil {
			return query.AgentJobExternalLeasePlanView{}, err
		}
		readiness = item
	}

	recommended := agentJobExternalLeaseNATSOwner
	desired, desiredOK := normalizeAgentJobExternalLeaseExecutionOwner(cmd.DesiredExecutionOwner)
	if desired == "auto" {
		desired = recommended
	}
	if !desiredOK {
		blockers = append(blockers, "invalid_desired_execution_owner")
		desired = recommended
	}

	current := strings.TrimSpace(readiness.ExecutionOwner)
	if current == "" {
		current = agentJobExternalLeaseStateStoreOwner
	}

	switch desired {
	case agentJobExternalLeaseNATSOwner:
		if !readiness.Ready {
			blockers = append(blockers, "agent_job_external_lease_readiness_not_ready")
			blockers = append(blockers, readiness.Blockers...)
		}
	case agentJobExternalLeaseStateStoreOwner:
		// State-store lease is the rollback/fallback owner. It does not require
		// result-ack readiness because Python keeps executing and confirming jobs
		// through the Go AgentJob state store.
	default:
		blockers = append(blockers, "desired_execution_owner_not_supported")
	}

	blockers = sortedUniqueSmokeStrings(blockers)
	ready := len(blockers) == 0 && desired == current
	decision := "blocked"
	if ready {
		decision = "ready"
	} else if len(blockers) == 0 && desired == agentJobExternalLeaseNATSOwner {
		decision = "ready_to_enable_result_ack"
	} else if len(blockers) == 0 && desired == agentJobExternalLeaseStateStoreOwner {
		decision = "ready_to_rollback_to_state_store"
	}

	return query.AgentJobExternalLeasePlanView{
		Ready:                     ready,
		Decision:                  decision,
		DesiredExecutionOwner:     desired,
		RecommendedExecutionOwner: recommended,
		CurrentExecutionOwner:     current,
		Readiness:                 readiness,
		RequiredChecks:            agentJobExternalLeasePlanRequiredChecks(desired),
		EnableSteps:               agentJobExternalLeasePlanEnableSteps(desired),
		VerificationSteps:         agentJobExternalLeasePlanVerificationSteps(desired),
		RollbackSteps:             agentJobExternalLeasePlanRollbackSteps(desired),
		Blockers:                  blockers,
		Attributes: map[string]string{
			"planned_by":  "agent_runtime_agent_job_external_lease_plan",
			"side_effect": "none",
		},
		Notes: []string{
			"read-only result-ack cutover plan; no queue acknowledgement, environment mutation, worker startup, or AI execution is performed",
			"Python remains responsible for model, memory, RAG, OCR, VLM, image generation, prompt and tool work; Go only plans deterministic AgentJob lifecycle ownership",
		},
		SideEffect: "none",
	}, nil
}

func normalizeAgentJobExternalLeaseExecutionOwner(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "auto", true
	}
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "auto":
		return "auto", true
	case "nats", "result_ack", "nats_result_ack", "external_lease", "python_ai_worker_with_nats_result_ack":
		return agentJobExternalLeaseNATSOwner, true
	case "state", "state_store", "state_store_lease", "python_ai_worker_state_store_lease":
		return agentJobExternalLeaseStateStoreOwner, true
	default:
		return value, false
	}
}

func agentJobExternalLeasePlanRequiredChecks(desired string) []query.AgentJobExternalLeasePlanStep {
	steps := []query.AgentJobExternalLeasePlanStep{
		{
			Phase:    "precheck",
			Action:   "check_agent_job_external_lease_readiness",
			Method:   "GET",
			Endpoint: "/v1/agent-job-external-lease/readiness",
			Detail:   "must be ready before moving generic agent_job result acknowledgement into NATS external lease",
		},
		{
			Phase:    "precheck",
			Action:   "check_queue_backend",
			Method:   "GET",
			Endpoint: "/v1/queue-backend",
			Detail:   "external_lease.allow_execution must be true and allowed_work_kinds must include agent_job",
		},
		{
			Phase:    "precheck",
			Action:   "check_python_worker_coverage",
			Method:   "GET",
			Endpoint: "/v1/agent-worker-statuses",
			Detail:   "Python AI workers must be active for pressured job types before result-ack cutover",
		},
	}
	if desired == agentJobExternalLeaseNATSOwner {
		steps = append(steps, query.AgentJobExternalLeasePlanStep{
			Phase:  "precheck",
			Action: "run_agent_job_duplicate_and_flow_smoke",
			Detail: "operator should verify duplicate admission and Python worker result writeback before setting the smoke flags",
		})
	}
	return indexedAgentJobExternalLeasePlanSteps(steps)
}

func agentJobExternalLeasePlanEnableSteps(desired string) []query.AgentJobExternalLeasePlanStep {
	switch desired {
	case agentJobExternalLeaseNATSOwner:
		return indexedAgentJobExternalLeasePlanSteps([]query.AgentJobExternalLeasePlanStep{
			{
				Phase:  "enable",
				Action: "configure_agent_job_nats_result_ack",
				Env: map[string]string{
					"AKASHIC_QUEUE_BACKEND":                          "nats_jetstream",
					"AKASHIC_QUEUE_MODE":                             "external_lease",
					"AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER":           "true",
					"AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED":           "true",
					"AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED":     "true",
					"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "true",
					"AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED": "true",
					"AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED":      "true",
					"AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN":           "true",
				},
				Detail: "restart agent-runtime and Python AI workers after setting these flags; Go still does not execute model/RAG/memory work",
			},
			{
				Phase:    "enable",
				Action:   "confirm_agent_job_result_ack_owner",
				Method:   "GET",
				Endpoint: "/v1/queue-backend",
				Detail:   "agent_job_execution_owner should become python_ai_worker_with_nats_result_ack",
			},
		})
	case agentJobExternalLeaseStateStoreOwner:
		return indexedAgentJobExternalLeasePlanSteps([]query.AgentJobExternalLeasePlanStep{
			{
				Phase:  "enable",
				Action: "disable_agent_job_nats_result_ack",
				Env: map[string]string{
					"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "false",
					"AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED": "false",
					"AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED":      "false",
				},
				Detail: "restart agent-runtime; Python workers return to Go state-store AgentJob lease confirmation while outbox external lease may remain independently configured",
			},
		})
	default:
		return nil
	}
}

func agentJobExternalLeasePlanVerificationSteps(desired string) []query.AgentJobExternalLeasePlanStep {
	steps := []query.AgentJobExternalLeasePlanStep{
		{Phase: "verify", Action: "read_runtime_config", Method: "GET", Endpoint: "/v1/runtime-config"},
		{Phase: "verify", Action: "read_queue_backend", Method: "GET", Endpoint: "/v1/queue-backend"},
		{Phase: "verify", Action: "recheck_agent_job_external_lease_readiness", Method: "GET", Endpoint: "/v1/agent-job-external-lease/readiness"},
		{Phase: "verify", Action: "watch_agent_job_metrics", Method: "GET", Endpoint: "/v1/job-metrics"},
		{Phase: "verify", Action: "watch_runtime_overview", Method: "GET", Endpoint: "/v1/runtime-overview"},
	}
	if desired == agentJobExternalLeaseNATSOwner {
		steps = append(steps, query.AgentJobExternalLeasePlanStep{
			Phase:    "verify",
			Action:   "watch_external_lease_result_ack_diagnostics",
			Method:   "GET",
			Endpoint: "/v1/queue-backend",
			Detail:   "external_lease diagnostics should show agent_job result-ack work without Go executing AI side effects",
		})
	}
	return indexedAgentJobExternalLeasePlanSteps(steps)
}

func agentJobExternalLeasePlanRollbackSteps(desired string) []query.AgentJobExternalLeasePlanStep {
	switch desired {
	case agentJobExternalLeaseNATSOwner:
		return indexedAgentJobExternalLeasePlanSteps([]query.AgentJobExternalLeasePlanStep{
			{
				Phase:  "rollback",
				Action: "disable_agent_job_result_ack_scope",
				Env: map[string]string{
					"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "false",
					"AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED": "false",
					"AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED":      "false",
				},
				Detail: "restart agent-runtime; this keeps base external_lease/outbox settings untouched and only removes agent_job from the NATS result-ack scope",
			},
			{
				Phase:    "rollback",
				Action:   "confirm_agent_job_state_store_owner",
				Method:   "GET",
				Endpoint: "/v1/queue-backend",
				Detail:   "agent_job_execution_owner should return to python_ai_worker_state_store_lease",
			},
		})
	case agentJobExternalLeaseStateStoreOwner:
		return nil
	default:
		return nil
	}
}

func indexedAgentJobExternalLeasePlanSteps(steps []query.AgentJobExternalLeasePlanStep) []query.AgentJobExternalLeasePlanStep {
	for index := range steps {
		steps[index].StepIndex = index + 1
	}
	return steps
}
