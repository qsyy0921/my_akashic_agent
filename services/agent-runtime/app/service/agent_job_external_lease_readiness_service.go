package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobExternalLeaseReadinessDeps struct {
	QueueBackend  agentJobExternalLeaseQueueBackendGetter
	RuntimeConfig runtimeConfigGetter
	AgentJobs     agentJobExternalLeaseMetricsGetter
	AgentWorkers  agentJobExternalLeaseWorkerStatusesGetter
}

type agentJobExternalLeaseQueueBackendGetter interface {
	Get(ctx context.Context) (query.QueueBackendView, error)
}

type agentJobExternalLeaseMetricsGetter interface {
	Get(ctx context.Context, filter query.AgentJobMetricsFilter) (query.AgentJobMetricsView, error)
}

type agentJobExternalLeaseWorkerStatusesGetter interface {
	ListAgentWorkerStatuses(ctx context.Context, filter query.AgentWorkerStatusFilter) (query.AgentWorkerStatusesView, error)
}

type AgentJobExternalLeaseReadinessService struct {
	deps AgentJobExternalLeaseReadinessDeps
}

func NewAgentJobExternalLeaseReadinessService(deps AgentJobExternalLeaseReadinessDeps) *AgentJobExternalLeaseReadinessService {
	return &AgentJobExternalLeaseReadinessService{deps: deps}
}

func (s *AgentJobExternalLeaseReadinessService) CheckAgentJobExternalLeaseReadiness(
	ctx context.Context,
	cmd command.CheckAgentJobExternalLeaseReadinessCommand,
) (query.AgentJobExternalLeaseReadinessView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentJobExternalLeaseReadinessView{}, err
	}
	if s == nil {
		return query.AgentJobExternalLeaseReadinessView{}, errors.New("agent job external lease readiness service is nil")
	}

	var (
		blockers      []string
		queueBackend  query.QueueBackendView
		runtimeConfig query.RuntimeConfigView
		agentJobs     query.AgentJobMetricsView
		agentWorkers  query.AgentWorkerStatusesView
	)

	if s.deps.QueueBackend == nil {
		blockers = append(blockers, "queue_backend_unavailable")
	} else if item, err := s.deps.QueueBackend.Get(ctx); err != nil {
		blockers = append(blockers, "queue_backend_unavailable")
	} else {
		queueBackend = item
	}

	if s.deps.RuntimeConfig == nil {
		blockers = append(blockers, "runtime_config_unavailable")
	} else if item, err := s.deps.RuntimeConfig.GetRuntimeConfig(ctx); err != nil {
		blockers = append(blockers, "runtime_config_unavailable")
	} else {
		runtimeConfig = item
	}

	if s.deps.AgentJobs == nil {
		blockers = append(blockers, "agent_job_metrics_unavailable")
	} else if item, err := s.deps.AgentJobs.Get(ctx, query.AgentJobMetricsFilter{
		JobLimit:   boundedAgentJobExternalLeaseLimit(cmd.JobLimit, 200),
		EventLimit: boundedAgentJobExternalLeaseLimit(cmd.EventLimit, 50),
	}); err != nil {
		blockers = append(blockers, "agent_job_metrics_unavailable")
	} else {
		agentJobs = item
	}

	if s.deps.AgentWorkers == nil {
		blockers = append(blockers, "agent_worker_statuses_unavailable")
	} else if item, err := s.deps.AgentWorkers.ListAgentWorkerStatuses(ctx, query.AgentWorkerStatusFilter{
		StaleAfterSeconds: boundedAgentJobExternalLeaseLimit(cmd.StaleAfterSeconds, defaultRuntimeOverviewStaleAfterSeconds),
	}); err != nil {
		blockers = append(blockers, "agent_worker_statuses_unavailable")
	} else {
		agentWorkers = item
	}

	externalLeaseReady := queueBackend.ExternalLease != nil && queueBackend.ExternalLease.AllowExecution
	if !externalLeaseReady {
		blockers = append(blockers, "external_lease_execution_not_allowed")
		if queueBackend.ExternalLease == nil {
			blockers = append(blockers, "external_lease_not_configured")
		} else {
			blockers = append(blockers, queueBackend.ExternalLease.Blockers...)
		}
	}

	agentJobAllowed := queueBackend.ExternalLease != nil && stringSliceContains(queueBackend.ExternalLease.AllowedWorkKinds, "agent_job")
	resultAckReady := agentJobAllowed && strings.TrimSpace(queueBackend.AgentJobExecutionOwner) == "python_ai_worker_with_nats_result_ack"
	if !agentJobAllowed {
		blockers = append(blockers, "agent_job_not_allowed_in_external_lease")
	}
	if strings.TrimSpace(queueBackend.AgentJobExecutionOwner) != "python_ai_worker_with_nats_result_ack" {
		blockers = append(blockers, "agent_job_result_ack_owner_not_enabled")
	}

	strictLeaseToken := runtimeConfig.Workers.AgentJobStrictLeaseToken
	if !strictLeaseToken {
		blockers = append(blockers, "strict_lease_token_disabled")
	}
	if !runtimeConfig.Workers.QueueExternalLeaseAgentJobEnable {
		blockers = append(blockers, "queue_external_lease_agent_job_flag_disabled")
	}

	coverage := agentJobWorkerCoverageFromPressure(agentJobs.Pressure.ByType, agentWorkers)
	workerReady := true
	for _, item := range coverage {
		if item.CoverageStatus == "danger" {
			workerReady = false
			blockers = append(blockers, "agent_job_worker_coverage_blocked")
			break
		}
	}

	blockers = sortedUniqueSmokeStrings(blockers)
	ready := externalLeaseReady && resultAckReady && strictLeaseToken && runtimeConfig.Workers.QueueExternalLeaseAgentJobEnable && workerReady && len(blockers) == 0
	reason := "agent_job_external_lease_not_ready"
	if ready {
		reason = "agent_job_external_lease_ready"
	}

	return query.AgentJobExternalLeaseReadinessView{
		Ready:                   ready,
		Reason:                  reason,
		ExternalLeaseReady:      externalLeaseReady,
		AgentJobResultAckReady:  resultAckReady,
		StrictLeaseTokenEnabled: strictLeaseToken,
		AgentJobWorkerReady:     workerReady,
		ExecutionOwner:          strings.TrimSpace(queueBackend.AgentJobExecutionOwner),
		QueueProvider:           queueBackend.Provider,
		QueueMode:               queueBackend.Mode,
		ExecutionScope:          agentJobExternalLeaseExecutionScope(queueBackend),
		AllowedWorkKinds:        agentJobExternalLeaseAllowedWorkKinds(queueBackend),
		BlockedWorkKinds:        agentJobExternalLeaseBlockedWorkKinds(queueBackend),
		RequiredChecks:          agentJobExternalLeaseRequiredChecks(queueBackend),
		WorkerCoverage:          coverage,
		Blockers:                blockers,
		Attributes: map[string]string{
			"checked_by":  "agent_runtime_agent_job_external_lease_readiness",
			"side_effect": "none",
		},
		Notes: []string{
			"read-only result-ack readiness; Go does not execute model, RAG, memory, OCR, VLM, or image jobs",
			"Python workers remain responsible for AI side effects and report status/lease progress back to Go",
		},
		SideEffect: "none",
	}, nil
}

func boundedAgentJobExternalLeaseLimit(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value > 200 {
		return 200
	}
	return value
}

func stringSliceContains(items []string, value string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}

func agentJobExternalLeaseExecutionScope(queueBackend query.QueueBackendView) string {
	if queueBackend.ExternalLease == nil {
		return ""
	}
	return queueBackend.ExternalLease.ExecutionScope
}

func agentJobExternalLeaseAllowedWorkKinds(queueBackend query.QueueBackendView) []string {
	if queueBackend.ExternalLease == nil {
		return nil
	}
	return append([]string(nil), queueBackend.ExternalLease.AllowedWorkKinds...)
}

func agentJobExternalLeaseBlockedWorkKinds(queueBackend query.QueueBackendView) []query.QueueExternalLeaseBlock {
	if queueBackend.ExternalLease == nil {
		return nil
	}
	return append([]query.QueueExternalLeaseBlock(nil), queueBackend.ExternalLease.BlockedWorkKinds...)
}

func agentJobExternalLeaseRequiredChecks(queueBackend query.QueueBackendView) []query.QueueExternalLeaseCheck {
	if queueBackend.ExternalLease == nil {
		return nil
	}
	return append([]query.QueueExternalLeaseCheck(nil), queueBackend.ExternalLease.RequiredChecks...)
}
