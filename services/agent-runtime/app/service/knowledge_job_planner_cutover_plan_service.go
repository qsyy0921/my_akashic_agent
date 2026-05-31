package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const (
	knowledgePlannerGoAdmissionOwner     = "go_runtime_knowledge_job_planner"
	knowledgePlannerLegacyAdmissionOwner = "python_legacy_knowledge_enqueue"
)

type KnowledgeJobPlannerCutoverPlanDeps struct {
	Readiness knowledgeJobPlannerCutoverReadinessChecker
}

type knowledgeJobPlannerCutoverReadinessChecker interface {
	CheckKnowledgeJobPlannerReadiness(ctx context.Context, cmd command.CheckKnowledgeJobPlannerReadinessCommand) (query.KnowledgeJobPlannerReadinessView, error)
}

type KnowledgeJobPlannerCutoverPlanService struct {
	deps KnowledgeJobPlannerCutoverPlanDeps
}

func NewKnowledgeJobPlannerCutoverPlanService(deps KnowledgeJobPlannerCutoverPlanDeps) *KnowledgeJobPlannerCutoverPlanService {
	return &KnowledgeJobPlannerCutoverPlanService{deps: deps}
}

func (s *KnowledgeJobPlannerCutoverPlanService) PlanKnowledgeJobPlannerCutover(
	ctx context.Context,
	cmd command.PlanKnowledgeJobPlannerCutoverCommand,
) (query.KnowledgeJobPlannerCutoverPlanView, error) {
	if err := ctx.Err(); err != nil {
		return query.KnowledgeJobPlannerCutoverPlanView{}, err
	}
	if s == nil {
		return query.KnowledgeJobPlannerCutoverPlanView{}, errors.New("knowledge job planner cutover plan service is nil")
	}

	var blockers []string
	var readiness query.KnowledgeJobPlannerReadinessView
	if s.deps.Readiness == nil {
		blockers = append(blockers, "knowledge_job_planner_readiness_unavailable")
	} else {
		item, err := s.deps.Readiness.CheckKnowledgeJobPlannerReadiness(ctx, cmd.Readiness)
		if err != nil {
			return query.KnowledgeJobPlannerCutoverPlanView{}, err
		}
		readiness = item
	}

	recommended := knowledgePlannerGoAdmissionOwner
	desired, desiredOK := normalizeKnowledgePlannerAdmissionOwner(cmd.DesiredAdmissionOwner)
	if desired == "auto" {
		desired = recommended
	}
	if !desiredOK {
		blockers = append(blockers, "invalid_desired_admission_owner")
		desired = recommended
	}

	current := currentKnowledgePlannerAdmissionOwner(readiness)
	switch desired {
	case knowledgePlannerGoAdmissionOwner:
		if !readiness.Ready {
			blockers = append(blockers, "knowledge_job_planner_readiness_not_ready")
			blockers = append(blockers, readiness.Blockers...)
		}
	case knowledgePlannerLegacyAdmissionOwner:
		// Legacy Python enqueue is the rollback/fallback owner. It does not need
		// Go planner readiness because Python continues to create compatible jobs.
	default:
		blockers = append(blockers, "desired_admission_owner_not_supported")
	}

	blockers = sortedUniqueKnowledgePlannerStrings(blockers)
	ready := len(blockers) == 0 && desired == current
	decision := "blocked"
	if ready {
		decision = "ready"
	} else if len(blockers) == 0 && desired == knowledgePlannerGoAdmissionOwner {
		decision = "ready_to_enable_go_planner"
	} else if len(blockers) == 0 && desired == knowledgePlannerLegacyAdmissionOwner {
		decision = "ready_to_rollback_to_python_legacy"
	}

	return query.KnowledgeJobPlannerCutoverPlanView{
		Ready:                     ready,
		Decision:                  decision,
		DesiredAdmissionOwner:     desired,
		RecommendedAdmissionOwner: recommended,
		CurrentAdmissionOwner:     current,
		Readiness:                 readiness,
		RequiredChecks:            knowledgePlannerCutoverRequiredChecks(desired),
		EnableSteps:               knowledgePlannerCutoverEnableSteps(desired),
		VerificationSteps:         knowledgePlannerCutoverVerificationSteps(desired),
		RollbackSteps:             knowledgePlannerCutoverRollbackSteps(desired),
		Blockers:                  blockers,
		Attributes: map[string]string{
			"planned_by":  "agent_runtime_knowledge_job_planner_cutover_plan",
			"side_effect": "none",
		},
		Notes: []string{
			"read-only knowledge planner cutover plan; no environment variables are changed, no AgentJob records are created, and no workers are started",
			"Python remains responsible for group memory extraction, RAGFlow ingestion, LLM/VLM/OCR/file understanding, prompt orchestration and provider strategy",
		},
		SideEffect: "none",
	}, nil
}

func normalizeKnowledgePlannerAdmissionOwner(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "auto", true
	}
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "auto":
		return "auto", true
	case "go", "go_runtime", "go_planner", "go_runtime_planner", "knowledge_job_planner", "go_runtime_knowledge_job_planner":
		return knowledgePlannerGoAdmissionOwner, true
	case "python", "legacy", "python_legacy", "legacy_enqueue", "python_legacy_enqueue", "python_legacy_knowledge_enqueue":
		return knowledgePlannerLegacyAdmissionOwner, true
	default:
		return value, false
	}
}

func currentKnowledgePlannerAdmissionOwner(readiness query.KnowledgeJobPlannerReadinessView) string {
	if readiness.PlannerEnabled && readiness.PlannerRunning {
		return knowledgePlannerGoAdmissionOwner
	}
	return knowledgePlannerLegacyAdmissionOwner
}

func knowledgePlannerCutoverRequiredChecks(desired string) []query.KnowledgeJobPlannerCutoverStep {
	steps := []query.KnowledgeJobPlannerCutoverStep{
		{
			Phase:    "precheck",
			Action:   "check_knowledge_job_planner_preview",
			Method:   "GET",
			Endpoint: "/v1/knowledge-job-planner/preview",
			Detail:   "must show observe-only group_memory_extract/rag_ingest plans without creating AgentJob records",
		},
		{
			Phase:    "precheck",
			Action:   "check_knowledge_job_planner_readiness",
			Method:   "GET",
			Endpoint: "/v1/knowledge-job-planner/readiness",
			Detail:   "must be ready before enabling Go-owned recurring knowledge admission",
		},
		{
			Phase:    "precheck",
			Action:   "check_python_knowledge_worker",
			Method:   "GET",
			Endpoint: "/v1/agent-worker-statuses",
			Detail:   "Python knowledge workers must be active because Go only admits jobs and Python executes memory/RAG work",
		},
	}
	if desired == knowledgePlannerGoAdmissionOwner {
		steps = append(steps, query.KnowledgeJobPlannerCutoverStep{
			Phase:  "precheck",
			Action: "confirm_observe_only_scope",
			Detail: "operator should confirm targets are observe-only and reply behavior is unchanged before enabling recurring admission",
		})
	}
	return indexedKnowledgePlannerCutoverSteps(steps)
}

func knowledgePlannerCutoverEnableSteps(desired string) []query.KnowledgeJobPlannerCutoverStep {
	switch desired {
	case knowledgePlannerGoAdmissionOwner:
		return indexedKnowledgePlannerCutoverSteps([]query.KnowledgeJobPlannerCutoverStep{
			{
				Phase:  "enable",
				Action: "enable_go_knowledge_job_planner",
				Env: map[string]string{
					"AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED":          "true",
					"AKASHIC_KNOWLEDGE_JOB_PLANNER_INTERVAL_SECONDS": "300",
					"AKASHIC_KNOWLEDGE_JOB_PLANNER_MAX_ATTEMPTS":     "3",
					"AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_MAX_MESSAGES": "200",
				},
				Detail: "restart agent-runtime after setting these flags; Python knowledge workers should keep running and only execute leased jobs",
			},
			{
				Phase:    "enable",
				Action:   "confirm_go_planner_running",
				Method:   "GET",
				Endpoint: "/v1/runtime-workers",
				Detail:   "knowledge_job_planner should be enabled and running",
			},
		})
	case knowledgePlannerLegacyAdmissionOwner:
		return indexedKnowledgePlannerCutoverSteps([]query.KnowledgeJobPlannerCutoverStep{
			{
				Phase:  "enable",
				Action: "disable_go_knowledge_job_planner",
				Env: map[string]string{
					"AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED": "false",
				},
				Detail: "restart agent-runtime; Python compatibility enqueue can resume while Go AgentJob lifecycle remains authoritative for leased jobs",
			},
		})
	default:
		return nil
	}
}

func knowledgePlannerCutoverVerificationSteps(desired string) []query.KnowledgeJobPlannerCutoverStep {
	steps := []query.KnowledgeJobPlannerCutoverStep{
		{Phase: "verify", Action: "read_runtime_config", Method: "GET", Endpoint: "/v1/runtime-config"},
		{Phase: "verify", Action: "read_runtime_workers", Method: "GET", Endpoint: "/v1/runtime-workers"},
		{Phase: "verify", Action: "recheck_knowledge_job_planner_readiness", Method: "GET", Endpoint: "/v1/knowledge-job-planner/readiness"},
		{Phase: "verify", Action: "watch_agent_jobs", Method: "GET", Endpoint: "/v1/jobs?type=group_memory_extract"},
		{Phase: "verify", Action: "watch_knowledge_pipelines", Method: "GET", Endpoint: "/v1/knowledge-pipeline-diagnostics"},
		{Phase: "verify", Action: "watch_runtime_overview", Method: "GET", Endpoint: "/v1/runtime-overview"},
	}
	if desired == knowledgePlannerGoAdmissionOwner {
		steps = append(steps, query.KnowledgeJobPlannerCutoverStep{
			Phase:    "verify",
			Action:   "confirm_python_legacy_enqueue_suppressed",
			Method:   "GET",
			Endpoint: "/v1/runtime-config",
			Detail:   "Python knowledge worker should see Go planner enabled and skip legacy enqueue while still executing leased jobs",
		})
	}
	return indexedKnowledgePlannerCutoverSteps(steps)
}

func knowledgePlannerCutoverRollbackSteps(desired string) []query.KnowledgeJobPlannerCutoverStep {
	switch desired {
	case knowledgePlannerGoAdmissionOwner:
		return indexedKnowledgePlannerCutoverSteps([]query.KnowledgeJobPlannerCutoverStep{
			{
				Phase:  "rollback",
				Action: "disable_go_knowledge_job_planner",
				Env: map[string]string{
					"AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED": "false",
				},
				Detail: "restart agent-runtime; pending Go-created AgentJobs remain recoverable in the state store",
			},
			{
				Phase:    "rollback",
				Action:   "confirm_python_legacy_admission_owner",
				Method:   "GET",
				Endpoint: "/v1/knowledge-job-planner/cutover-plan?desired_admission_owner=python_legacy_knowledge_enqueue",
				Detail:   "current_admission_owner should become python_legacy_knowledge_enqueue after rollback",
			},
		})
	case knowledgePlannerLegacyAdmissionOwner:
		return nil
	default:
		return nil
	}
}

func indexedKnowledgePlannerCutoverSteps(steps []query.KnowledgeJobPlannerCutoverStep) []query.KnowledgeJobPlannerCutoverStep {
	for index := range steps {
		steps[index].StepIndex = index + 1
	}
	return steps
}
