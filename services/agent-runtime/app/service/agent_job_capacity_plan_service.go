package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobCapacityPlanDeps struct {
	AgentJobs    agentJobCapacityMetricsGetter
	AgentWorkers agentJobCapacityWorkerStatusesGetter
}

type agentJobCapacityMetricsGetter interface {
	Get(ctx context.Context, filter query.AgentJobMetricsFilter) (query.AgentJobMetricsView, error)
}

type agentJobCapacityWorkerStatusesGetter interface {
	ListAgentWorkerStatuses(ctx context.Context, filter query.AgentWorkerStatusFilter) (query.AgentWorkerStatusesView, error)
}

type AgentJobCapacityPlanService struct {
	deps AgentJobCapacityPlanDeps
}

func NewAgentJobCapacityPlanService(deps AgentJobCapacityPlanDeps) *AgentJobCapacityPlanService {
	return &AgentJobCapacityPlanService{deps: deps}
}

func (s *AgentJobCapacityPlanService) PlanAgentJobCapacity(
	ctx context.Context,
	cmd command.PlanAgentJobCapacityCommand,
) (query.AgentJobCapacityPlanView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentJobCapacityPlanView{}, err
	}
	if s == nil {
		return query.AgentJobCapacityPlanView{}, errors.New("agent job capacity plan service is nil")
	}

	var (
		blockers []string
		notes    []string
		jobs     query.AgentJobMetricsView
		workers  query.AgentWorkerStatusesView
	)
	if s.deps.AgentJobs == nil {
		blockers = append(blockers, "agent_job_metrics_unavailable")
	} else if item, err := s.deps.AgentJobs.Get(ctx, query.AgentJobMetricsFilter{
		JobLimit:   boundedAgentJobCapacityLimit(cmd.JobLimit, 200),
		EventLimit: boundedAgentJobCapacityLimit(cmd.EventLimit, 50),
	}); err != nil {
		blockers = append(blockers, "agent_job_metrics_unavailable")
		notes = append(notes, fmt.Sprintf("agent job metrics unavailable: %v", err))
	} else {
		jobs = item
	}

	if s.deps.AgentWorkers == nil {
		blockers = append(blockers, "agent_worker_statuses_unavailable")
	} else if item, err := s.deps.AgentWorkers.ListAgentWorkerStatuses(ctx, query.AgentWorkerStatusFilter{
		StaleAfterSeconds: boundedAgentJobCapacityLimit(cmd.StaleAfterSeconds, defaultRuntimeOverviewStaleAfterSeconds),
	}); err != nil {
		blockers = append(blockers, "agent_worker_statuses_unavailable")
		notes = append(notes, fmt.Sprintf("agent worker statuses unavailable: %v", err))
	} else {
		workers = item
	}

	coverage := agentJobWorkerCoverageFromPressure(jobs.Pressure.ByType, workers)
	pressureByType := agentJobCapacityPressureByType(jobs.Pressure.ByType)
	items := make([]query.AgentJobCapacityPlanItemView, 0, len(coverage))
	summary := query.AgentJobCapacitySummaryView{
		JobTypes:                jobs.Pressure.JobTypes,
		HighPressureJobTypes:    jobs.Pressure.HighPressureJobTypes,
		MaxPending:              jobs.Pressure.MaxPending,
		MaxActive:               jobs.Pressure.MaxActive,
		OldestPendingAgeSeconds: jobs.Pressure.OldestPendingAgeSeconds,
	}

	for _, coverageItem := range coverage {
		pressure := pressureByType[coverageItem.JobType]
		item := agentJobCapacityPlanItem(coverageItem, pressure)
		items = append(items, item)
		if len(coverageItem.ExpectedWorkerTypes) == 0 {
			summary.UnmappedJobTypes++
		} else {
			summary.MappedJobTypes++
		}
		if coverageItem.ActiveWorkers > 0 {
			summary.ActiveWorkerJobTypes++
		}
		if coverageItem.StaleWorkers > 0 {
			summary.StaleWorkerJobTypes++
		}
		if coverageItem.FailedWorkers > 0 {
			summary.FailedWorkerJobTypes++
		}
		switch item.Severity {
		case "danger":
			summary.CapacityBlockedJobTypes++
		case "warn":
			summary.WorkerWarningJobTypes++
		}
	}

	if summary.CapacityBlockedJobTypes > 0 {
		blockers = append(blockers, "agent_job_capacity_blocked")
	}
	if summary.WorkerWarningJobTypes > 0 {
		blockers = append(blockers, "agent_job_worker_warning")
	}
	if summary.HighPressureJobTypes > 0 {
		blockers = append(blockers, "agent_job_high_pressure")
	}
	blockers = sortedUniqueSmokeStrings(blockers)

	ready := len(blockers) == 0
	reason := "agent_job_capacity_ready"
	if !ready {
		reason = "agent_job_capacity_attention_required"
	}

	return query.AgentJobCapacityPlanView{
		Ready:             ready,
		Reason:            reason,
		Summary:           summary,
		Items:             items,
		VerificationSteps: agentJobCapacityVerificationSteps(),
		Blockers:          blockers,
		Attributes: map[string]string{
			"planned_by":  "agent_runtime_agent_job_capacity_plan",
			"side_effect": "none",
		},
		Notes: append(notes,
			"read-only capacity plan; no job lease, worker startup, queue acknowledgement, retry, config mutation, or AI execution is performed",
			"Python remains responsible for worker execution, model calls, memory extraction, RAG, OCR/VLM, image generation, prompts and tool work",
		),
		SideEffect: "none",
	}, nil
}

func boundedAgentJobCapacityLimit(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value > 200 {
		return 200
	}
	return value
}

func agentJobCapacityPressureByType(items []query.AgentJobTypePressureView) map[string]query.AgentJobTypePressureView {
	result := make(map[string]query.AgentJobTypePressureView, len(items))
	for _, item := range items {
		result[item.JobType] = item
	}
	return result
}

func agentJobCapacityPlanItem(
	coverage query.AgentJobWorkerCoverageView,
	pressure query.AgentJobTypePressureView,
) query.AgentJobCapacityPlanItemView {
	severity, action, recommendation := agentJobCapacityRecommendation(coverage)
	return query.AgentJobCapacityPlanItemView{
		JobType:                 coverage.JobType,
		Severity:                severity,
		Action:                  action,
		Recommendation:          recommendation,
		Pending:                 pressure.Pending,
		Leased:                  pressure.Leased,
		Running:                 pressure.Running,
		Active:                  pressure.Active,
		OldestPendingAgeSeconds: pressure.OldestPendingAgeSeconds,
		HighPressure:            pressure.HighPressure,
		PressureReason:          pressure.PressureReason,
		Coverage:                coverage,
	}
}

func agentJobCapacityRecommendation(item query.AgentJobWorkerCoverageView) (string, string, string) {
	if len(item.ExpectedWorkerTypes) == 0 {
		return "muted", "no_worker_mapping", "No Python worker mapping is registered for this job type; keep it advisory until the job type has an explicit owner."
	}
	if item.HighPressure && item.ActiveWorkers == 0 {
		return "danger", "start_or_recover_python_worker", fmt.Sprintf("Start or recover a Python %s worker before increasing admission or queue acknowledgement scope.", agentJobCapacityWorkerTypesText(item.ExpectedWorkerTypes))
	}
	if item.FailedWorkers > 0 {
		return "warn", "inspect_failed_worker", "Inspect failed Python worker status, last_error and logs before changing concurrency."
	}
	if item.StaleWorkers > 0 {
		return "warn", "renew_or_restart_stale_worker", "Renew or restart stale Python workers so Go can trust worker coverage before tuning throughput."
	}
	if item.HighPressure {
		return "warn", "increase_worker_concurrency_or_prioritize_queue", "Active Python workers exist, but pressure is high; consider worker concurrency, priority or admission tuning outside this read-only plan."
	}
	if item.ActiveWorkers == 0 {
		return "warn", "start_idle_worker", "No active Python worker is available for this mapped job type; start one before expecting timely execution."
	}
	return "ok", "monitor", "Capacity coverage is available; continue monitoring pressure and worker heartbeat."
}

func agentJobCapacityWorkerTypesText(items []string) string {
	if len(items) == 0 {
		return "mapped"
	}
	if len(items) == 1 {
		return items[0]
	}
	return fmt.Sprintf("%v", items)
}

func agentJobCapacityVerificationSteps() []query.AgentJobCapacityPlanStep {
	steps := []query.AgentJobCapacityPlanStep{
		{
			Phase:    "verify",
			Action:   "watch_agent_job_metrics",
			Method:   "GET",
			Endpoint: "/v1/job-metrics",
			Detail:   "confirm pending, active and oldest pending age by job type",
		},
		{
			Phase:    "verify",
			Action:   "watch_python_worker_status",
			Method:   "GET",
			Endpoint: "/v1/agent-worker-statuses",
			Detail:   "confirm Python worker heartbeat, stale state, failed state and current job",
		},
		{
			Phase:    "verify",
			Action:   "watch_runtime_overview",
			Method:   "GET",
			Endpoint: "/v1/runtime-overview",
			Detail:   "confirm the same pressure and worker coverage remains visible from the aggregate dashboard API",
		},
	}
	for index := range steps {
		steps[index].StepIndex = index + 1
	}
	return steps
}
