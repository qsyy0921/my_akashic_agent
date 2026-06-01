package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobPriorityPlanDeps struct {
	AgentJobs    agentJobCapacityMetricsGetter
	AgentWorkers agentJobCapacityWorkerStatusesGetter
}

type AgentJobPriorityPlanService struct {
	deps AgentJobPriorityPlanDeps
}

func NewAgentJobPriorityPlanService(deps AgentJobPriorityPlanDeps) *AgentJobPriorityPlanService {
	return &AgentJobPriorityPlanService{deps: deps}
}

func (s *AgentJobPriorityPlanService) PlanAgentJobPriority(
	ctx context.Context,
	cmd command.PlanAgentJobPriorityCommand,
) (query.AgentJobPriorityPlanView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentJobPriorityPlanView{}, err
	}
	if s == nil {
		return query.AgentJobPriorityPlanView{}, errors.New("agent job priority plan service is nil")
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
	items := make([]query.AgentJobPriorityPlanItemView, 0, len(coverage))
	summary := query.AgentJobPrioritySummaryView{JobTypes: jobs.Pressure.JobTypes}

	for _, coverageItem := range coverage {
		pressure := pressureByType[coverageItem.JobType]
		item := agentJobPriorityPlanItem(coverageItem, pressure)
		items = append(items, item)
		if item.PriorityScore > summary.MaxPriorityScore {
			summary.MaxPriorityScore = item.PriorityScore
		}
		switch item.PriorityClass {
		case "critical":
			summary.HighPriorityJobTypes++
			summary.BlockedJobTypes++
		case "high":
			summary.HighPriorityJobTypes++
			summary.WarningJobTypes++
		case "medium":
			summary.WarningJobTypes++
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.PriorityScore != right.PriorityScore {
			return left.PriorityScore > right.PriorityScore
		}
		if left.Pending != right.Pending {
			return left.Pending > right.Pending
		}
		if left.OldestPendingAgeSeconds != right.OldestPendingAgeSeconds {
			return left.OldestPendingAgeSeconds > right.OldestPendingAgeSeconds
		}
		if left.Active != right.Active {
			return left.Active < right.Active
		}
		return left.JobType < right.JobType
	})
	for index := range items {
		items[index].Rank = index + 1
	}

	if summary.BlockedJobTypes > 0 {
		blockers = append(blockers, "agent_job_priority_blocked")
	}
	if summary.HighPriorityJobTypes > 0 {
		blockers = append(blockers, "agent_job_priority_attention_required")
	}
	blockers = sortedUniqueSmokeStrings(blockers)

	ready := len(blockers) == 0
	reason := "agent_job_priority_ready"
	if !ready {
		reason = "agent_job_priority_attention_required"
	}

	return query.AgentJobPriorityPlanView{
		Ready:             ready,
		Reason:            reason,
		Summary:           summary,
		Items:             items,
		VerificationSteps: agentJobPriorityVerificationSteps(),
		Blockers:          blockers,
		Attributes: map[string]string{
			"planned_by":           "agent_runtime_agent_job_priority_plan",
			"side_effect":          "none",
			"priority_control":     "manual_only",
			"worker_control_owner": "python",
		},
		Notes: append(notes,
			"read-only priority plan; no job creation, lease, queue acknowledgement, retry, config mutation, worker startup, or AI execution is performed",
			"Python remains responsible for worker execution, model calls, memory extraction, RAG, OCR/VLM, image generation, prompts and tool work",
		),
		SideEffect: "none",
	}, nil
}

func agentJobPriorityPlanItem(
	coverage query.AgentJobWorkerCoverageView,
	pressure query.AgentJobTypePressureView,
) query.AgentJobPriorityPlanItemView {
	priorityClass, score, action, recommendation := agentJobPriorityRecommendation(coverage)
	return query.AgentJobPriorityPlanItemView{
		JobType:                 coverage.JobType,
		PriorityClass:           priorityClass,
		PriorityScore:           score,
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

func agentJobPriorityRecommendation(item query.AgentJobWorkerCoverageView) (string, int, string, string) {
	if len(item.ExpectedWorkerTypes) == 0 {
		return "advisory", 10, "register_worker_mapping", "No Python worker mapping is registered; keep this job type advisory until ownership is explicit."
	}
	if item.HighPressure && item.ActiveWorkers == 0 {
		return "critical", 100, "recover_worker_before_priority_tuning", fmt.Sprintf("Recover a Python %s worker before changing priority or queue acknowledgement.", agentJobCapacityWorkerTypesText(item.ExpectedWorkerTypes))
	}
	if item.FailedWorkers > 0 {
		return "high", 90, "inspect_failed_worker_before_requeue", "Inspect failed Python worker status and logs before retrying or increasing priority."
	}
	if item.StaleWorkers > 0 {
		return "high", 80, "renew_or_restart_stale_worker_before_requeue", "Renew or restart stale Python workers before trusting queue priority decisions."
	}
	if item.HighPressure {
		return "medium", 70, "prioritize_after_worker_health_verified", "Active Python workers exist under high pressure; prioritize this job type only after checking worker throughput."
	}
	if item.ActiveWorkers == 0 {
		return "medium", 60, "start_idle_worker_before_admission", "Mapped job type has no active Python worker; start one before expecting timely execution."
	}
	return "low", 20, "monitor", "Worker coverage is healthy; keep monitoring pressure before changing priority."
}

func agentJobPriorityVerificationSteps() []query.AgentJobPriorityPlanStep {
	steps := []query.AgentJobPriorityPlanStep{
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
			Action:   "compare_capacity_plan",
			Method:   "GET",
			Endpoint: "/v1/agent-job-capacity/plan",
			Detail:   "confirm priority advice agrees with capacity blockers before any future operator cutover",
		},
	}
	for index := range steps {
		steps[index].StepIndex = index + 1
	}
	return steps
}
