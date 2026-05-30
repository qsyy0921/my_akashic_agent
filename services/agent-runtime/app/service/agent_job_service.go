package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type AgentJobService struct {
	repository       outport.AgentJobRepository
	events           outport.AgentJobEventSink
	workQueue        outport.WorkQueuePublisher
	strictLeaseToken bool
}

type AgentJobServiceOption func(*AgentJobService)

func WithStrictAgentJobLeaseToken(enabled bool) AgentJobServiceOption {
	return func(service *AgentJobService) {
		if service != nil {
			service.strictLeaseToken = enabled
		}
	}
}

func NewAgentJobService(repository outport.AgentJobRepository, options ...AgentJobServiceOption) *AgentJobService {
	return newAgentJobService(repository, nil, nil, options...)
}

func NewAgentJobServiceWithEvents(
	repository outport.AgentJobRepository,
	events outport.AgentJobEventSink,
	options ...AgentJobServiceOption,
) *AgentJobService {
	return newAgentJobService(repository, events, nil, options...)
}

func NewAgentJobServiceWithEventsAndWorkQueue(
	repository outport.AgentJobRepository,
	events outport.AgentJobEventSink,
	workQueue outport.WorkQueuePublisher,
	options ...AgentJobServiceOption,
) *AgentJobService {
	return newAgentJobService(repository, events, workQueue, options...)
}

func newAgentJobService(
	repository outport.AgentJobRepository,
	events outport.AgentJobEventSink,
	workQueue outport.WorkQueuePublisher,
	options ...AgentJobServiceOption,
) *AgentJobService {
	service := &AgentJobService{repository: repository, events: events, workQueue: workQueue}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
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
	if err := s.recordEvent(ctx, job, model.AgentJobEventCreated, cmd.Timestamp); err != nil {
		return query.AgentJobView{}, err
	}
	s.publishAgentJobWork(ctx, job)
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
	return s.update(ctx, cmd.JobID, model.AgentJobEventLeased, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.Lease(cmd.WorkerID, time.Duration(cmd.TTLSeconds)*time.Second, agentJobLeaseToken(cmd.LeaseToken), now)
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
	if err := job.Lease(cmd.WorkerID, time.Duration(cmd.TTLSeconds)*time.Second, agentJobLeaseToken(cmd.LeaseToken), cmd.Timestamp); err != nil {
		return query.AgentJobView{}, err
	}
	if err := s.repository.SaveAgentJob(ctx, job); err != nil {
		return query.AgentJobView{}, err
	}
	if err := s.recordEvent(ctx, job, model.AgentJobEventLeased, cmd.Timestamp); err != nil {
		return query.AgentJobView{}, err
	}
	return assembler.ToAgentJobView(job), nil
}

func (s *AgentJobService) LeaseWork(ctx context.Context, cmd command.AgentJobLeaseWorkCommand) (query.AgentJobView, error) {
	workKind := normalizeWorkKind(cmd.WorkKind)
	if workKind != "agent_job" {
		return query.AgentJobView{}, errors.New("agent job lease work requires agent_job work kind")
	}
	workID := firstNonBlank(cmd.WorkID, cmd.AggregateID)
	if workID == "" {
		return query.AgentJobView{}, errors.New("agent job lease work requires work id")
	}
	if aggregateID := strings.TrimSpace(cmd.AggregateID); aggregateID != "" && aggregateID != workID {
		return query.AgentJobView{}, errors.New("agent job lease work aggregate id mismatch")
	}
	return s.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      workID,
		WorkerID:   cmd.WorkerID,
		LeaseToken: cmd.LeaseToken,
		TTLSeconds: cmd.TTLSeconds,
		Timestamp:  cmd.Timestamp,
	})
}

func (s *AgentJobService) RenewLease(ctx context.Context, cmd command.RenewAgentJobLeaseCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, model.AgentJobEventRenewed, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.RenewLease(cmd.LeaseToken, time.Duration(cmd.TTLSeconds)*time.Second, now)
	})
}

func (s *AgentJobService) MarkRunning(ctx context.Context, cmd command.MarkAgentJobRunningCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, model.AgentJobEventRunning, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		if err := s.requireLeaseToken(cmd.LeaseToken); err != nil {
			return err
		}
		if err := job.ValidateLeaseToken(cmd.LeaseToken); err != nil {
			return err
		}
		return job.MarkRunning(now)
	})
}

func (s *AgentJobService) Complete(ctx context.Context, cmd command.CompleteAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, model.AgentJobEventSucceeded, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		if err := s.requireLeaseToken(cmd.LeaseToken); err != nil {
			return err
		}
		if err := job.ValidateLeaseToken(cmd.LeaseToken); err != nil {
			return err
		}
		return job.MarkSucceeded(cmd.Result, now)
	})
}

func (s *AgentJobService) Fail(ctx context.Context, cmd command.FailAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, model.AgentJobEventFailed, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		if err := s.requireLeaseToken(cmd.LeaseToken); err != nil {
			return err
		}
		if err := job.ValidateLeaseToken(cmd.LeaseToken); err != nil {
			return err
		}
		return job.MarkFailed(cmd.ErrorMessage, now)
	})
}

func (s *AgentJobService) Retry(ctx context.Context, cmd command.RetryAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, model.AgentJobEventRetry, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.Retry(now)
	})
}

func (s *AgentJobService) Cancel(ctx context.Context, cmd command.CancelAgentJobCommand) (query.AgentJobView, error) {
	return s.update(ctx, cmd.JobID, model.AgentJobEventCancelled, cmd.Timestamp, func(job *model.AgentJob, now time.Time) error {
		return job.Cancel(now)
	})
}

func (s *AgentJobService) update(
	ctx context.Context,
	jobID string,
	eventType model.AgentJobEventType,
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
	if err := s.recordEvent(ctx, job, eventType, timestamp); err != nil {
		return query.AgentJobView{}, err
	}
	return assembler.ToAgentJobView(job), nil
}

func (s *AgentJobService) recordEvent(
	ctx context.Context,
	job model.AgentJob,
	eventType model.AgentJobEventType,
	timestamp time.Time,
) error {
	if s == nil || s.events == nil {
		return nil
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	eventID := fmt.Sprintf(
		"agent-job-event:%s:%s:%d",
		job.JobID,
		eventType,
		timestamp.UTC().UnixNano(),
	)
	event, err := model.NewAgentJobEventFromJob(eventID, eventType, job, timestamp)
	if err != nil {
		return err
	}
	return s.events.AppendAgentJobEvent(ctx, event)
}

func (s *AgentJobService) publishAgentJobWork(ctx context.Context, job model.AgentJob) {
	if s == nil || s.workQueue == nil {
		return
	}
	_ = s.workQueue.PublishAgentJob(ctx, job)
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

func (s *AgentJobService) requireLeaseToken(leaseToken string) error {
	if s == nil || !s.strictLeaseToken {
		return nil
	}
	if strings.TrimSpace(leaseToken) == "" {
		return errors.New("agent job lease token required")
	}
	return nil
}

func agentJobLeaseToken(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err == nil {
		return hex.EncodeToString(raw)
	}
	return fmt.Sprintf("agent-job-lease:%d", time.Now().UTC().UnixNano())
}
