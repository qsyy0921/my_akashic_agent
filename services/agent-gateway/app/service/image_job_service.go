package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/service"
)

type ImageJobService struct {
	repository outport.ImageJobRepository
	queue      outport.ImageJobQueue
	agentJobs  outport.AgentJobRepository
}

func NewImageJobService(repository outport.ImageJobRepository, queue outport.ImageJobQueue) *ImageJobService {
	return &ImageJobService{repository: repository, queue: queue}
}

func NewImageJobServiceWithAgentJobs(
	repository outport.ImageJobRepository,
	queue outport.ImageJobQueue,
	agentJobs outport.AgentJobRepository,
) *ImageJobService {
	return &ImageJobService{repository: repository, queue: queue, agentJobs: agentJobs}
}

func (s *ImageJobService) Create(ctx context.Context, cmd command.CreateImageJobCommand) (query.ImageJobView, error) {
	if s == nil || s.repository == nil || s.queue == nil {
		return query.ImageJobView{}, errors.New("image job service requires repository and queue")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}

	jobID := domainservice.NewImageJobID(cmd.RequestID)
	if existing, ok, err := s.repository.FindImageJob(ctx, jobID); err != nil {
		return query.ImageJobView{}, err
	} else if ok {
		if err := s.ensureGenericImageJob(ctx, existing); err != nil {
			return query.ImageJobView{}, err
		}
		return assembler.ToImageJobView(existing), nil
	}

	job, err := model.NewImageJob(
		jobID,
		cmd.RequestID,
		assembler.ToChannelRef(cmd.Requester),
		cmd.RequesterID,
		cmd.Prompt,
		model.ImageJobOptions{
			Provider: cmd.Provider,
			Model:    cmd.Model,
			Size:     cmd.Size,
			Count:    cmd.Count,
		},
		cmd.MaxAttempts,
		cmd.Timestamp,
		cmd.Metadata,
	)
	if err != nil {
		return query.ImageJobView{}, err
	}
	if err := s.repository.SaveImageJob(ctx, job); err != nil {
		return query.ImageJobView{}, err
	}
	if err := s.queue.EnqueueImageJob(ctx, job); err != nil {
		return query.ImageJobView{}, err
	}
	if err := s.ensureGenericImageJob(ctx, job); err != nil {
		return query.ImageJobView{}, err
	}
	return assembler.ToImageJobView(job), nil
}

func (s *ImageJobService) MarkRunning(ctx context.Context, cmd command.MarkImageJobRunningCommand) (query.ImageJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.ImageJob, now time.Time) error {
		return job.MarkRunning(now)
	})
}

func (s *ImageJobService) Complete(ctx context.Context, cmd command.CompleteImageJobCommand) (query.ImageJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.ImageJob, now time.Time) error {
		results := assembler.ToAttachments(cmd.Results)
		if len(cmd.Metadata) > 0 {
			if job.Metadata == nil {
				job.Metadata = map[string]string{}
			}
			for key, value := range cmd.Metadata {
				job.Metadata[key] = value
			}
		}
		return job.MarkSucceeded(results, now)
	})
}

func (s *ImageJobService) Fail(ctx context.Context, cmd command.FailImageJobCommand) (query.ImageJobView, error) {
	return s.update(ctx, cmd.JobID, cmd.Timestamp, func(job *model.ImageJob, now time.Time) error {
		return job.MarkFailed(cmd.ErrorMessage, now)
	})
}

func (s *ImageJobService) Get(ctx context.Context, jobID string) (query.ImageJobView, error) {
	if s == nil || s.repository == nil {
		return query.ImageJobView{}, errors.New("image job service requires repository")
	}
	job, err := s.getModel(ctx, jobID)
	if err != nil {
		return query.ImageJobView{}, err
	}
	return assembler.ToImageJobView(job), nil
}

func (s *ImageJobService) update(
	ctx context.Context,
	jobID string,
	timestamp time.Time,
	mutate func(job *model.ImageJob, now time.Time) error,
) (query.ImageJobView, error) {
	if s == nil || s.repository == nil {
		return query.ImageJobView{}, errors.New("image job service requires repository")
	}
	job, err := s.getModel(ctx, jobID)
	if err != nil {
		return query.ImageJobView{}, err
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	if err := mutate(&job, timestamp); err != nil {
		return query.ImageJobView{}, err
	}
	if err := s.repository.SaveImageJob(ctx, job); err != nil {
		return query.ImageJobView{}, err
	}
	return assembler.ToImageJobView(job), nil
}

func (s *ImageJobService) getModel(ctx context.Context, jobID string) (model.ImageJob, error) {
	jobID = strings.TrimSpace(jobID)
	job, ok, err := s.repository.FindImageJob(ctx, jobID)
	if err != nil {
		return model.ImageJob{}, err
	}
	if !ok {
		return model.ImageJob{}, errors.New("image job not found")
	}
	return job, nil
}

func (s *ImageJobService) ensureGenericImageJob(ctx context.Context, job model.ImageJob) error {
	if s == nil || s.agentJobs == nil {
		return nil
	}
	if _, ok, err := s.agentJobs.FindAgentJob(ctx, job.JobID); err != nil {
		return err
	} else if ok {
		return nil
	}
	metadata := copyStringMap(job.Metadata)
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["compat_api"] = "image-jobs"
	metadata["legacy_image_job_id"] = job.JobID

	generic, err := model.NewAgentJob(model.AgentJobSpec{
		JobID:          job.JobID,
		JobType:        model.AgentJobImageGeneration,
		AgentID:        imageJobAgentID(job.Metadata),
		Route:          imageJobRoute(job),
		SourceEventIDs: imageJobSourceEventIDs(job),
		Payload: map[string]string{
			"legacy_image_job_id": job.JobID,
			"request_id":          job.RequestID,
			"requester_id":        job.RequesterID,
			"prompt":              job.Prompt,
			"provider":            job.Options.Provider,
			"model":               job.Options.Model,
			"size":                job.Options.Size,
			"count":               strconv.Itoa(job.Options.Count),
		},
		MaxAttempts: job.MaxAttempts,
		Metadata:    metadata,
	}, job.CreatedAt)
	if err != nil {
		return err
	}
	return s.agentJobs.SaveAgentJob(ctx, generic)
}

func imageJobRoute(job model.ImageJob) model.ChannelRef {
	route := job.Requester
	if route.ConversationType == "" {
		route.ConversationType = model.ConversationTypePrivate
	}
	return route
}

func imageJobAgentID(metadata map[string]string) string {
	for _, key := range []string{"agent_id", "worker_agent_id"} {
		value := strings.TrimSpace(metadata[key])
		if value != "" {
			return value
		}
	}
	return "akashic-python-worker"
}

func imageJobSourceEventIDs(job model.ImageJob) []string {
	ids := make([]string, 0, 2)
	for _, key := range []string{"source_event_id", "source_message_id"} {
		value := strings.TrimSpace(job.Metadata[key])
		if value != "" {
			ids = append(ids, value)
		}
	}
	return ids
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
