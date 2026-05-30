package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type AgentJobEventType string

const (
	AgentJobEventCreated   AgentJobEventType = "created"
	AgentJobEventLeased    AgentJobEventType = "leased"
	AgentJobEventRunning   AgentJobEventType = "running"
	AgentJobEventSucceeded AgentJobEventType = "succeeded"
	AgentJobEventFailed    AgentJobEventType = "failed"
	AgentJobEventRetry     AgentJobEventType = "retry"
	AgentJobEventCancelled AgentJobEventType = "cancelled"
)

type AgentJobEvent struct {
	EventID        string
	JobID          string
	JobType        AgentJobType
	EventType      AgentJobEventType
	Status         AgentJobStatus
	Attempt        int
	MaxAttempts    int
	LeaseOwner     string
	LeaseExpiresAt time.Time
	OccurredAt     time.Time
	Metadata       map[string]string
}

type AgentJobEventSpec struct {
	EventID        string
	JobID          string
	JobType        AgentJobType
	EventType      AgentJobEventType
	Status         AgentJobStatus
	Attempt        int
	MaxAttempts    int
	LeaseOwner     string
	LeaseExpiresAt time.Time
	Metadata       map[string]string
}

func NewAgentJobEvent(spec AgentJobEventSpec, now time.Time) (AgentJobEvent, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	event := AgentJobEvent{
		EventID:        strings.TrimSpace(spec.EventID),
		JobID:          strings.TrimSpace(spec.JobID),
		JobType:        spec.JobType,
		EventType:      spec.EventType,
		Status:         spec.Status,
		Attempt:        spec.Attempt,
		MaxAttempts:    spec.MaxAttempts,
		LeaseOwner:     strings.TrimSpace(spec.LeaseOwner),
		LeaseExpiresAt: spec.LeaseExpiresAt,
		OccurredAt:     now,
		Metadata:       copyStringMap(spec.Metadata),
	}
	if err := event.Validate(); err != nil {
		return AgentJobEvent{}, err
	}
	return event, nil
}

func NewAgentJobEventFromJob(eventID string, eventType AgentJobEventType, job AgentJob, now time.Time) (AgentJobEvent, error) {
	metadata := map[string]string{}
	if job.ErrorMessage != "" {
		metadata["error_message"] = job.ErrorMessage
	}
	if strings.TrimSpace(job.LeaseToken) != "" {
		metadata["lease_token_hash"] = hashAgentJobLeaseToken(job.LeaseToken)
	}
	return NewAgentJobEvent(AgentJobEventSpec{
		EventID:        eventID,
		JobID:          job.JobID,
		JobType:        job.JobType,
		EventType:      eventType,
		Status:         job.Status,
		Attempt:        job.Attempts,
		MaxAttempts:    job.MaxAttempts,
		LeaseOwner:     job.LeaseOwner,
		LeaseExpiresAt: job.LeaseExpiresAt,
		Metadata:       metadata,
	}, now)
}

func (e AgentJobEvent) Validate() error {
	if strings.TrimSpace(e.EventID) == "" {
		return errors.New("agent job event requires event id")
	}
	if strings.TrimSpace(e.JobID) == "" {
		return errors.New("agent job event requires job id")
	}
	switch e.JobType {
	case AgentJobImageGeneration, AgentJobMediaVision, AgentJobMediaOCR, AgentJobGroupMemoryExtract, AgentJobRagIngest, AgentJobRagEval:
	default:
		return errors.New("agent job event has invalid job type")
	}
	switch e.EventType {
	case AgentJobEventCreated, AgentJobEventLeased, AgentJobEventRunning, AgentJobEventSucceeded, AgentJobEventFailed, AgentJobEventRetry, AgentJobEventCancelled:
	default:
		return errors.New("agent job event has invalid event type")
	}
	switch e.Status {
	case AgentJobPending, AgentJobLeased, AgentJobRunning, AgentJobSucceeded, AgentJobFailed, AgentJobDeadLettered, AgentJobCancelled:
	default:
		return errors.New("agent job event has invalid status")
	}
	if e.Attempt < 0 {
		return errors.New("agent job event attempt cannot be negative")
	}
	if e.MaxAttempts <= 0 {
		return errors.New("agent job event requires positive max attempts")
	}
	if e.OccurredAt.IsZero() {
		return errors.New("agent job event requires occurred_at")
	}
	return nil
}

func hashAgentJobLeaseToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
