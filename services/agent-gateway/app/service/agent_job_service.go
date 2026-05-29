package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

type AgentJobService struct {
	repository outport.AgentJobRepository
}

func NewAgentJobService(repository outport.AgentJobRepository) *AgentJobService {
	return &AgentJobService{repository: repository}
}

func (s *AgentJobService) Create(ctx context.Context, cmd command.CreateAgentJobCommand) (query.AgentJobView, error) {
	if s == nil || s.repository == nil {
		return query.AgentJobView{}, errors.New("agent job service requires repository")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}
	if existing, ok, err := s.repository.FindAgentJob(ctx, cmd.JobID); err != nil {
		return query.AgentJobView{}, err
	} else if ok {
		return assembler.ToAgentJobView(existing), nil
	}
	job, err := model.NewAgentJob(model.AgentJobSpec{
		JobID:          cmd.JobID,
		JobType:        model.AgentJobType(cmd.JobType),
		AgentID:        cmd.AgentID,
		Route:          assembler.ToChannelRef(cmd.Route),
		SourceEventIDs: cmd.SourceEventIDs,
		SourceAssetIDs: cmd.SourceAssetIDs,
		Payload:        cmd.Payload,
		MaxAttempts:    cmd.MaxAttempts,
		Metadata:       cmd.Metadata,
	}, cmd.Timestamp)
	if err != nil {
		return query.AgentJobView{}, err
	}
	if err := s.repository.SaveAgentJob(ctx, job); err != nil {
		return query.AgentJobView{}, err
	}
	return assembler.ToAgentJobView(job), nil
}

func (s *AgentJobService) List(ctx context.Context, filter query.AgentJobFilter) ([]query.AgentJobView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("agent job service requires repository")
	}
	items, err := s.repository.ListAgentJobs(ctx, filter)
	if err != nil {
		return nil, err
	}
	return assembler.ToAgentJobViews(items), nil
}

func (s *AgentJobService) Get(ctx context.Context, jobID string) (query.AgentJobView, error) {
	job, err := s.getModel(ctx, jobID)
	if err != nil {
		return query.AgentJobView{}, err
	}
	return assembler.ToAgentJobView(job), nil
}

func (s *AgentJobService) Lease(ctx context.Context, cmd command.AgentJobLeaseCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.Lease(cmd.WorkerID, time.Duration(cmd.TTLSeconds)*time.Second, now)
	})
}

func (s *AgentJobService) LeaseNext(ctx context.Context, cmd command.AgentJobLeaseNextCommand) (query.AgentJobView, error) {
	if s == nil || s.repository == nil {
		return query.AgentJobView{}, errors.New("agent job service requires repository")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}
	job, ok, err := s.repository.FindLeaseableAgentJob(ctx, cmd.JobType, cmd.Timestamp)
	if err != nil {
		return query.AgentJobView{}, err
	}
	if !ok {
		return query.AgentJobView{}, errors.New("no leaseable agent job found")
	}
	if err := job.Lease(cmd.WorkerID, time.Duration(cmd.TTLSeconds)*time.Second, cmd.Timestamp); err != nil {
		return query.AgentJobView{}, err
	}
	if err := s.repository.SaveAgentJob(ctx, job); err != nil {
		return query.AgentJobView{}, err
	}
	return assembler.ToAgentJobView(job), nil
}

func (s *AgentJobService) MarkRunning(ctx context.Context, cmd command.MarkAgentJobRunningCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.MarkRunning(now)
	})
}

func (s *AgentJobService) Complete(ctx context.Context, cmd command.CompleteAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.MarkSucceeded(cmd.Result, now)
	})
}

func (s *AgentJobService) Fail(ctx context.Context, cmd command.FailAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.MarkFailed(cmd.ErrorMessage, now)
	})
}

func (s *AgentJobService) Retry(ctx context.Context, cmd command.RetryAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.Retry(now)
	})
}

func (s *AgentJobService) Cancel(ctx context.Context, cmd command.CancelAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.Cancel(now)
	})
}

func (s *AgentJobService) update(
	ctx context.Context,
	jobID string,
	timestamp time.Time,
	mutate func(job *model.AgentJob, now time.Time) error,
) (query.AgentJobView, error) {
	job, err := s.getModel(ctx, jobID)
	if err != nil {
		return query.AgentJobView{}, err
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	if err := mutate(&job, timestamp); err != nil {
		return query.AgentJobView{}, err
	}
	if err := s.repository.SaveAgentJob(ctx, job); err != nil {
		return query.AgentJobView{}, err
	}
	return assembler.ToAgentJobView(job), nil
}

func (s *AgentJobService) getModel(ctx context.Context, jobID string) (model.AgentJob, error) {
	if s == nil || s.repository == nil {
		return model.AgentJob{}, errors.New("agent job service requires repository")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return model.AgentJob{}, errors.New("agent job id required")
	}
	job, ok, err := s.repository.FindAgentJob(ctx, jobID)
	if err != nil {
		return model.AgentJob{}, err
	}
	if !ok {
		return model.AgentJob{}, errors.New("agent job not found")
	}
	return job, nil
}
