package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultKnowledgeDiagnosticsLimit             = 50
	defaultKnowledgeDiagnosticsStaleAfterSeconds = 15 * 60
)

type KnowledgeWorkerDiagnosticsService struct {
	agentJobs   outport.AgentJobRepository
	checkpoints outport.KnowledgeCheckpointRepository
}

func NewKnowledgeWorkerDiagnosticsService(
	agentJobs outport.AgentJobRepository,
	checkpoints outport.KnowledgeCheckpointRepository,
) *KnowledgeWorkerDiagnosticsService {
	return &KnowledgeWorkerDiagnosticsService{agentJobs: agentJobs, checkpoints: checkpoints}
}

func (s *KnowledgeWorkerDiagnosticsService) Get(
	ctx context.Context,
	filter query.KnowledgeWorkerDiagnosticsFilter,
) (query.KnowledgeWorkerDiagnosticsView, error) {
	if s == nil || s.agentJobs == nil || s.checkpoints == nil {
		return query.KnowledgeWorkerDiagnosticsView{}, errors.New("knowledge worker diagnostics service requires repositories")
	}
	now := filter.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	limit := boundedKnowledgeDiagnosticsLimit(filter.Limit)
	staleAfterSeconds := filter.StaleAfterSeconds
	if staleAfterSeconds <= 0 {
		staleAfterSeconds = defaultKnowledgeDiagnosticsStaleAfterSeconds
	}
	staleAfter := time.Duration(staleAfterSeconds) * time.Second

	workers := make([]query.KnowledgeWorkerDiagnosticView, 0, 2)
	totals := map[string]int{
		"jobs":           0,
		"checkpoints":    0,
		"stale_leases":   0,
		"leaseable_jobs": 0,
	}
	for _, spec := range knowledgeWorkerSpecs() {
		worker, err := s.workerDiagnostics(ctx, spec, limit, staleAfter, now)
		if err != nil {
			return query.KnowledgeWorkerDiagnosticsView{}, err
		}
		totals["jobs"] += len(worker.RecentJobs)
		totals["checkpoints"] += len(worker.Checkpoints)
		totals["stale_leases"] += worker.StaleLeaseCount
		totals["leaseable_jobs"] += worker.LeaseableCount
		workers = append(workers, worker)
	}

	return query.KnowledgeWorkerDiagnosticsView{
		GeneratedAt:            formatKnowledgeDiagnosticsTime(now),
		StaleAfterSeconds:      staleAfterSeconds,
		SampledJobLimit:        limit,
		SampledCheckpointLimit: limit,
		Totals:                 totals,
		Workers:                workers,
	}, nil
}

func (s *KnowledgeWorkerDiagnosticsService) workerDiagnostics(
	ctx context.Context,
	spec knowledgeWorkerSpec,
	limit int,
	staleAfter time.Duration,
	now time.Time,
) (query.KnowledgeWorkerDiagnosticView, error) {
	jobs, err := s.agentJobs.ListAgentJobs(ctx, query.AgentJobFilter{
		JobType: string(spec.jobType),
		Limit:   limit,
	})
	if err != nil {
		return query.KnowledgeWorkerDiagnosticView{}, err
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		if jobs[i].UpdatedAt.Equal(jobs[j].UpdatedAt) {
			return jobs[i].JobID > jobs[j].JobID
		}
		return jobs[i].UpdatedAt.After(jobs[j].UpdatedAt)
	})

	statusCounts := make(map[string]int)
	staleLeaseCount := 0
	leaseableCount := 0
	recentJobs := make([]query.AgentJobView, 0, len(jobs))
	for _, job := range jobs {
		statusCounts[string(job.Status)]++
		if job.CanLease(now) {
			leaseableCount++
		}
		if isStaleKnowledgeJob(job, staleAfter, now) {
			staleLeaseCount++
		}
		recentJobs = append(recentJobs, assembler.ToAgentJobView(job))
	}

	checkpoints, err := s.checkpoints.ListKnowledgeCheckpoints(ctx, query.KnowledgeCheckpointFilter{
		Limit:  limit,
		Prefix: spec.checkpointPrefix,
	})
	if err != nil {
		return query.KnowledgeWorkerDiagnosticView{}, err
	}
	sort.SliceStable(checkpoints, func(i, j int) bool {
		if checkpoints[i].UpdatedAt.Equal(checkpoints[j].UpdatedAt) {
			return checkpoints[i].CheckpointID > checkpoints[j].CheckpointID
		}
		return checkpoints[i].UpdatedAt.After(checkpoints[j].UpdatedAt)
	})
	checkpointViews := make([]query.KnowledgeCheckpointView, 0, len(checkpoints))
	for _, checkpoint := range checkpoints {
		checkpointViews = append(checkpointViews, assembler.ToKnowledgeCheckpointView(checkpoint))
	}

	var latestJob *query.AgentJobView
	if len(recentJobs) > 0 {
		item := recentJobs[0]
		latestJob = &item
	}
	var latestCheckpoint *query.KnowledgeCheckpointView
	if len(checkpointViews) > 0 {
		item := checkpointViews[0]
		latestCheckpoint = &item
	}

	return query.KnowledgeWorkerDiagnosticView{
		JobType:          string(spec.jobType),
		CheckpointPrefix: spec.checkpointPrefix,
		StatusCounts:     statusCounts,
		StaleLeaseCount:  staleLeaseCount,
		LeaseableCount:   leaseableCount,
		RecentJobs:       recentJobs,
		Checkpoints:      checkpointViews,
		LatestJob:        latestJob,
		LatestCheckpoint: latestCheckpoint,
	}, nil
}

type knowledgeWorkerSpec struct {
	jobType          model.AgentJobType
	checkpointPrefix string
}

func knowledgeWorkerSpecs() []knowledgeWorkerSpec {
	return []knowledgeWorkerSpec{
		{jobType: model.AgentJobGroupMemoryExtract, checkpointPrefix: "memory:"},
		{jobType: model.AgentJobRagIngest, checkpointPrefix: "ragflow:"},
	}
}

func boundedKnowledgeDiagnosticsLimit(limit int) int {
	if limit <= 0 {
		return defaultKnowledgeDiagnosticsLimit
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func isStaleKnowledgeJob(job model.AgentJob, staleAfter time.Duration, now time.Time) bool {
	if job.Status != model.AgentJobLeased && job.Status != model.AgentJobRunning {
		return false
	}
	if !job.LeaseExpiresAt.IsZero() && now.After(job.LeaseExpiresAt) {
		return true
	}
	if staleAfter <= 0 || job.UpdatedAt.IsZero() {
		return false
	}
	return now.Sub(job.UpdatedAt) > staleAfter
}

func formatKnowledgeDiagnosticsTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
