package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/service"
)

type ImageJobService struct {
	repository outport.ImageJobRepository
	queue      outport.ImageJobQueue
}

func NewImageJobService(repository outport.ImageJobRepository, queue outport.ImageJobQueue) *ImageJobService {
	return &ImageJobService{repository: repository, queue: queue}
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
