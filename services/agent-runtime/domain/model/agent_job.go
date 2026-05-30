package model

import (
	"errors"
	"strings"
	"time"
)

type AgentJobType string

const (
	AgentJobImageGeneration    AgentJobType = "image_generation"
	AgentJobMediaVision        AgentJobType = "media_vision"
	AgentJobMediaOCR           AgentJobType = "media_ocr"
	AgentJobGroupMemoryExtract AgentJobType = "group_memory_extract"
	AgentJobRagIngest          AgentJobType = "rag_ingest"
	AgentJobRagEval            AgentJobType = "rag_eval"
)

type AgentJobStatus string

const (
	AgentJobPending      AgentJobStatus = "pending"
	AgentJobLeased       AgentJobStatus = "leased"
	AgentJobRunning      AgentJobStatus = "running"
	AgentJobSucceeded    AgentJobStatus = "succeeded"
	AgentJobFailed       AgentJobStatus = "failed"
	AgentJobDeadLettered AgentJobStatus = "dead_lettered"
	AgentJobCancelled    AgentJobStatus = "cancelled"
)

type AgentJob struct {
	JobID          string
	JobType        AgentJobType
	AgentID        string
	Route          ChannelRef
	SourceEventIDs []string
	SourceAssetIDs []string
	Payload        map[string]string
	Status         AgentJobStatus
	Attempts       int
	MaxAttempts    int
	LeaseOwner     string
	LeaseToken     string
	LeaseExpiresAt time.Time
	Result         map[string]string
	ErrorMessage   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Metadata       map[string]string
}

type AgentJobSpec struct {
	JobID          string
	JobType        AgentJobType
	AgentID        string
	Route          ChannelRef
	SourceEventIDs []string
	SourceAssetIDs []string
	Payload        map[string]string
	MaxAttempts    int
	Metadata       map[string]string
}

func NewAgentJob(spec AgentJobSpec, now time.Time) (AgentJob, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if spec.MaxAttempts <= 0 {
		spec.MaxAttempts = 3
	}
	job := AgentJob{
		JobID:          strings.TrimSpace(spec.JobID),
		JobType:        spec.JobType,
		AgentID:        strings.TrimSpace(spec.AgentID),
		Route:          spec.Route,
		SourceEventIDs: cleanList(spec.SourceEventIDs),
		SourceAssetIDs: cleanList(spec.SourceAssetIDs),
		Payload:        spec.Payload,
		Status:         AgentJobPending,
		MaxAttempts:    spec.MaxAttempts,
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       spec.Metadata,
	}
	if err := job.Validate(); err != nil {
		return AgentJob{}, err
	}
	return job, nil
}

func (j AgentJob) Validate() error {
	if strings.TrimSpace(j.JobID) == "" {
		return errors.New("agent job requires job id")
	}
	if j.JobType == "" {
		return errors.New("agent job requires type")
	}
	if strings.TrimSpace(j.AgentID) == "" {
		return errors.New("agent job requires agent id")
	}
	if strings.TrimSpace(string(j.Route.Kind)) == "" {
		return errors.New("agent job requires route kind")
	}
	if strings.TrimSpace(j.Route.AccountID) == "" {
		return errors.New("agent job requires route account id")
	}
	if strings.TrimSpace(j.Route.ConversationID) == "" {
		return errors.New("agent job requires route conversation id")
	}
	if strings.TrimSpace(string(j.Route.ConversationType)) == "" {
		return errors.New("agent job requires route conversation type")
	}
	if j.Attempts < 0 {
		return errors.New("agent job attempts cannot be negative")
	}
	if j.MaxAttempts <= 0 {
		return errors.New("agent job requires positive max attempts")
	}
	if j.Attempts > j.MaxAttempts {
		return errors.New("agent job attempts exceed max attempts")
	}
	if j.CreatedAt.IsZero() {
		return errors.New("agent job requires created_at")
	}
	if j.UpdatedAt.IsZero() {
		return errors.New("agent job requires updated_at")
	}
	switch j.JobType {
	case AgentJobImageGeneration, AgentJobMediaVision, AgentJobMediaOCR, AgentJobGroupMemoryExtract, AgentJobRagIngest, AgentJobRagEval:
	default:
		return errors.New("agent job has invalid type")
	}
	switch j.Status {
	case AgentJobPending, AgentJobLeased, AgentJobRunning, AgentJobSucceeded, AgentJobFailed, AgentJobDeadLettered, AgentJobCancelled:
		return nil
	default:
		return errors.New("agent job has invalid status")
	}
}

func (j AgentJob) CanLease(now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if j.Status == AgentJobPending {
		return true
	}
	if (j.Status == AgentJobLeased || j.Status == AgentJobRunning) && !j.LeaseExpiresAt.IsZero() && now.After(j.LeaseExpiresAt) {
		return true
	}
	return false
}

func (j *AgentJob) Lease(owner string, ttl time.Duration, leaseToken string, now time.Time) error {
	if j == nil {
		return errors.New("agent job is nil")
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return errors.New("agent job lease requires owner")
	}
	leaseToken = strings.TrimSpace(leaseToken)
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !j.CanLease(now) {
		return errors.New("agent job cannot be leased in current state")
	}
	if j.Attempts >= j.MaxAttempts {
		j.Status = AgentJobDeadLettered
		j.UpdatedAt = now
		return j.Validate()
	}
	j.Attempts++
	j.Status = AgentJobLeased
	j.LeaseOwner = owner
	j.LeaseToken = leaseToken
	j.LeaseExpiresAt = now.Add(ttl)
	j.ErrorMessage = ""
	j.UpdatedAt = now
	return j.Validate()
}

func (j AgentJob) ValidateLeaseToken(leaseToken string) error {
	leaseToken = strings.TrimSpace(leaseToken)
	if leaseToken == "" {
		return nil
	}
	if strings.TrimSpace(j.LeaseToken) == "" {
		return errors.New("agent job has no active lease token")
	}
	if leaseToken != j.LeaseToken {
		return errors.New("agent job lease token mismatch")
	}
	return nil
}

func (j *AgentJob) RenewLease(leaseToken string, ttl time.Duration, now time.Time) error {
	if j == nil {
		return errors.New("agent job is nil")
	}
	leaseToken = strings.TrimSpace(leaseToken)
	if leaseToken == "" {
		return errors.New("agent job lease renew requires lease token")
	}
	if err := j.ValidateLeaseToken(leaseToken); err != nil {
		return err
	}
	if j.Status != AgentJobLeased && j.Status != AgentJobRunning {
		return errors.New("only leased or running agent job can renew lease")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !j.LeaseExpiresAt.IsZero() && now.After(j.LeaseExpiresAt) {
		return errors.New("expired agent job lease cannot be renewed")
	}
	j.LeaseExpiresAt = now.Add(ttl)
	j.UpdatedAt = now
	return j.Validate()
}

func (j *AgentJob) MarkRunning(now time.Time) error {
	if j == nil {
		return errors.New("agent job is nil")
	}
	if j.Status != AgentJobLeased {
		return errors.New("only leased agent job can run")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	j.Status = AgentJobRunning
	j.UpdatedAt = now
	return j.Validate()
}

func (j *AgentJob) MarkSucceeded(result map[string]string, now time.Time) error {
	if j == nil {
		return errors.New("agent job is nil")
	}
	if j.Status == AgentJobCancelled || j.Status == AgentJobDeadLettered {
		return errors.New("terminal agent job cannot succeed")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	j.Status = AgentJobSucceeded
	j.Result = result
	j.ErrorMessage = ""
	j.LeaseOwner = ""
	j.LeaseToken = ""
	j.LeaseExpiresAt = time.Time{}
	j.UpdatedAt = now
	return j.Validate()
}

func (j *AgentJob) MarkFailed(message string, now time.Time) error {
	if j == nil {
		return errors.New("agent job is nil")
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("agent job failure requires error message")
	}
	if j.Status == AgentJobSucceeded || j.Status == AgentJobCancelled || j.Status == AgentJobDeadLettered {
		return errors.New("terminal agent job cannot fail")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	j.ErrorMessage = message
	j.LeaseOwner = ""
	j.LeaseToken = ""
	j.LeaseExpiresAt = time.Time{}
	j.UpdatedAt = now
	if j.Attempts >= j.MaxAttempts {
		j.Status = AgentJobDeadLettered
		return j.Validate()
	}
	j.Status = AgentJobFailed
	return j.Validate()
}

func (j *AgentJob) Retry(now time.Time) error {
	if j == nil {
		return errors.New("agent job is nil")
	}
	if j.Status != AgentJobFailed {
		return errors.New("only failed agent job can be retried")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	j.Status = AgentJobPending
	j.LeaseOwner = ""
	j.LeaseToken = ""
	j.LeaseExpiresAt = time.Time{}
	j.ErrorMessage = ""
	j.UpdatedAt = now
	return j.Validate()
}

func (j *AgentJob) Cancel(now time.Time) error {
	if j == nil {
		return errors.New("agent job is nil")
	}
	if j.Status == AgentJobSucceeded || j.Status == AgentJobDeadLettered {
		return errors.New("completed agent job cannot be cancelled")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	j.Status = AgentJobCancelled
	j.LeaseOwner = ""
	j.LeaseToken = ""
	j.LeaseExpiresAt = time.Time{}
	j.UpdatedAt = now
	return j.Validate()
}

func cleanList(items []string) []string {
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}
