package model

import (
	"errors"
	"strings"
	"time"
)

type ImageJobStatus string

const (
	ImageJobStatusPending   ImageJobStatus = "pending"
	ImageJobStatusRunning   ImageJobStatus = "running"
	ImageJobStatusSucceeded ImageJobStatus = "succeeded"
	ImageJobStatusFailed    ImageJobStatus = "failed"
)

type ImageJobOptions struct {
	Provider string
	Model    string
	Size     string
	Count    int
}

type ImageJob struct {
	JobID        string
	RequestID    string
	Requester    ChannelRef
	RequesterID  string
	Prompt       string
	Options      ImageJobOptions
	Status       ImageJobStatus
	Attempts     int
	MaxAttempts  int
	Results      []Attachment
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Metadata     map[string]string
}

func NewImageJob(
	jobID string,
	requestID string,
	requester ChannelRef,
	requesterID string,
	prompt string,
	options ImageJobOptions,
	maxAttempts int,
	now time.Time,
	metadata map[string]string,
) (ImageJob, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	job := ImageJob{
		JobID:       strings.TrimSpace(jobID),
		RequestID:   strings.TrimSpace(requestID),
		Requester:   requester,
		RequesterID: strings.TrimSpace(requesterID),
		Prompt:      strings.TrimSpace(prompt),
		Options: ImageJobOptions{
			Provider: strings.TrimSpace(options.Provider),
			Model:    strings.TrimSpace(options.Model),
			Size:     strings.TrimSpace(options.Size),
			Count:    options.Count,
		},
		Status:      ImageJobStatusPending,
		MaxAttempts: maxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    copyMetadata(metadata),
	}
	if job.Options.Count <= 0 {
		job.Options.Count = 1
	}
	if err := job.Validate(); err != nil {
		return ImageJob{}, err
	}
	return job, nil
}

func (j ImageJob) Validate() error {
	if strings.TrimSpace(j.JobID) == "" {
		return errors.New("image job requires job id")
	}
	if strings.TrimSpace(string(j.Requester.Kind)) == "" {
		return errors.New("image job requires requester channel kind")
	}
	if strings.TrimSpace(j.Requester.AccountID) == "" {
		return errors.New("image job requires requester account id")
	}
	if strings.TrimSpace(j.Requester.ConversationID) == "" {
		return errors.New("image job requires requester conversation id")
	}
	if strings.TrimSpace(j.Prompt) == "" {
		return errors.New("image job requires prompt")
	}
	if j.Options.Count <= 0 {
		return errors.New("image job count must be positive")
	}
	if j.MaxAttempts <= 0 {
		return errors.New("image job max attempts must be positive")
	}
	if j.Status == "" {
		return errors.New("image job requires status")
	}
	if j.CreatedAt.IsZero() || j.UpdatedAt.IsZero() {
		return errors.New("image job requires timestamps")
	}
	return nil
}

func (j *ImageJob) MarkRunning(now time.Time) error {
	if j.Status == ImageJobStatusSucceeded {
		return errors.New("succeeded image job cannot run again")
	}
	if j.Attempts >= j.MaxAttempts {
		return errors.New("image job exceeded max attempts")
	}
	j.Status = ImageJobStatusRunning
	j.Attempts++
	j.ErrorMessage = ""
	j.touch(now)
	return j.Validate()
}

func (j *ImageJob) MarkSucceeded(results []Attachment, now time.Time) error {
	if len(results) == 0 {
		return errors.New("succeeded image job requires result attachments")
	}
	j.Status = ImageJobStatusSucceeded
	j.Results = append([]Attachment(nil), results...)
	j.ErrorMessage = ""
	j.touch(now)
	return j.Validate()
}

func (j *ImageJob) MarkFailed(message string, now time.Time) error {
	j.Status = ImageJobStatusFailed
	j.ErrorMessage = strings.TrimSpace(message)
	j.touch(now)
	return j.Validate()
}

func (j *ImageJob) CanRetry() bool {
	return j.Status == ImageJobStatusFailed && j.Attempts < j.MaxAttempts
}

func (j *ImageJob) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	j.UpdatedAt = now
}

func copyMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	copied := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copied[key] = value
	}
	return copied
}
