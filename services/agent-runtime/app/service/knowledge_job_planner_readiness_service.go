package service

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeJobPlannerReadinessDeps struct {
	Previewer      runtimeKnowledgeJobPlannerPreviewer
	RuntimeConfig  runtimeConfigGetter
	RuntimeWorkers runtimeWorkerDiagnosticsGetter
	AgentWorkers   runtimeAgentWorkerStatusesGetter
	DefaultPlan    command.PlanKnowledgeJobsCommand
}

type KnowledgeJobPlannerReadinessService struct {
	deps KnowledgeJobPlannerReadinessDeps
}

func NewKnowledgeJobPlannerReadinessService(deps KnowledgeJobPlannerReadinessDeps) *KnowledgeJobPlannerReadinessService {
	return &KnowledgeJobPlannerReadinessService{deps: deps}
}

func (s *KnowledgeJobPlannerReadinessService) CheckKnowledgeJobPlannerReadiness(
	ctx context.Context,
	cmd command.CheckKnowledgeJobPlannerReadinessCommand,
) (query.KnowledgeJobPlannerReadinessView, error) {
	if err := ctx.Err(); err != nil {
		return query.KnowledgeJobPlannerReadinessView{}, err
	}
	if s == nil || s.deps.Previewer == nil {
		return query.KnowledgeJobPlannerReadinessView{}, errors.New("knowledge job planner readiness requires previewer")
	}
	plan := s.deps.DefaultPlan
	if !planCommandEmpty(cmd.Plan) {
		plan = mergePlanKnowledgeJobsCommand(plan, cmd.Plan)
	}
	staleAfterSeconds := cmd.StaleAfterSeconds
	if staleAfterSeconds <= 0 {
		staleAfterSeconds = defaultRuntimeOverviewStaleAfterSeconds
	}

	preview, err := s.deps.Previewer.PreviewKnowledgeJobs(ctx, plan)
	if err != nil {
		return query.KnowledgeJobPlannerReadinessView{}, err
	}

	var runtimeConfig query.RuntimeConfigView
	if s.deps.RuntimeConfig != nil {
		runtimeConfig, _ = s.deps.RuntimeConfig.GetRuntimeConfig(ctx)
	}
	var runtimeWorkers query.RuntimeWorkerDiagnosticsView
	if s.deps.RuntimeWorkers != nil {
		runtimeWorkers, _ = s.deps.RuntimeWorkers.GetRuntimeWorkers(ctx)
	}
	var agentWorkers query.AgentWorkerStatusesView
	if s.deps.AgentWorkers != nil {
		agentWorkers, _ = s.deps.AgentWorkers.ListAgentWorkerStatuses(ctx, query.AgentWorkerStatusFilter{
			StaleAfterSeconds: staleAfterSeconds,
		})
	}

	plannerEnabled := runtimeConfig.Workers.KnowledgeJobPlannerEnabled
	plannerRunning := knowledgePlannerRuntimeWorkerRunning(runtimeWorkers)
	workerCounts := knowledgeWorkerReadinessCounts(agentWorkers)
	blockers := make([]string, 0)
	if preview.TotalJobs == 0 {
		blockers = append(blockers, "knowledge_job_planner_no_planned_jobs")
	}
	if workerCounts.active == 0 {
		blockers = append(blockers, "knowledge_worker_unavailable")
	}
	if plannerEnabled && !plannerRunning {
		blockers = append(blockers, "knowledge_job_planner_worker_not_running")
	}
	blockers = sortedUniqueKnowledgePlannerStrings(blockers)
	ready := len(blockers) == 0
	reason := "knowledge_job_planner_ready"
	if !ready {
		reason = "knowledge_job_planner_not_ready"
	}
	return query.KnowledgeJobPlannerReadinessView{
		Ready:                  ready,
		Reason:                 reason,
		PlannerEnabled:         plannerEnabled,
		PlannerRunning:         plannerRunning,
		KnowledgeWorkerReady:   workerCounts.active > 0,
		KnowledgeWorkerActive:  workerCounts.active,
		KnowledgeWorkerStale:   workerCounts.stale,
		KnowledgeWorkerFailed:  workerCounts.failed,
		KnowledgeWorkerStopped: workerCounts.stopped,
		Blockers:               blockers,
		Preview:                preview,
		Attributes: map[string]string{
			"checked_by":  "agent_runtime_knowledge_job_planner_readiness",
			"side_effect": "none",
		},
		Notes: []string{
			"read-only planner admission preflight; no AgentJob records are created",
			"disabled planner is treated as preflight state; enabled but non-running planner is blocked",
		},
		SideEffect: "none",
	}, nil
}

type knowledgeWorkerCounts struct {
	active  int
	stale   int
	failed  int
	stopped int
}

func knowledgePlannerRuntimeWorkerRunning(view query.RuntimeWorkerDiagnosticsView) bool {
	for _, worker := range view.Workers {
		if worker.Name == "knowledge_job_planner" {
			return worker.Running
		}
	}
	return false
}

func knowledgeWorkerReadinessCounts(view query.AgentWorkerStatusesView) knowledgeWorkerCounts {
	var counts knowledgeWorkerCounts
	for _, worker := range view.Workers {
		if strings.TrimSpace(worker.WorkerType) != "knowledge" {
			continue
		}
		if worker.Stale {
			counts.stale++
		}
		switch worker.Status {
		case "starting", "idle", "running":
			if !worker.Stale {
				counts.active++
			}
		case "failed":
			counts.failed++
		case "stopped":
			counts.stopped++
		}
	}
	return counts
}

func mergePlanKnowledgeJobsCommand(base command.PlanKnowledgeJobsCommand, override command.PlanKnowledgeJobsCommand) command.PlanKnowledgeJobsCommand {
	if strings.TrimSpace(override.PlannerID) != "" {
		base.PlannerID = override.PlannerID
	}
	if strings.TrimSpace(override.AgentID) != "" {
		base.AgentID = override.AgentID
	}
	if override.IntervalSeconds > 0 {
		base.IntervalSeconds = override.IntervalSeconds
	}
	if override.MaxAttempts > 0 {
		base.MaxAttempts = override.MaxAttempts
	}
	if override.RagMaxMessages > 0 {
		base.RagMaxMessages = override.RagMaxMessages
	}
	if !override.Timestamp.IsZero() {
		base.Timestamp = override.Timestamp
	}
	base.RagParse = override.RagParse
	return base
}

func planCommandEmpty(cmd command.PlanKnowledgeJobsCommand) bool {
	return strings.TrimSpace(cmd.PlannerID) == "" &&
		strings.TrimSpace(cmd.AgentID) == "" &&
		cmd.IntervalSeconds <= 0 &&
		cmd.MaxAttempts <= 0 &&
		cmd.RagMaxMessages <= 0 &&
		cmd.Timestamp.IsZero() &&
		!cmd.RagParse
}

func sortedUniqueKnowledgePlannerStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}
