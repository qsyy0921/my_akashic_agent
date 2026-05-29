package memory

import (
	"context"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
)

type Store struct {
	mu           sync.Mutex
	observed     []ObservedEvent
	agentInbound []model.MessageEnvelope
	outbound     []model.OutboundMessage
	sendRecords  []model.SendRecord
	nonces       map[string]time.Time
	audits       []AuditEvent
	imageJobs    map[string]model.ImageJob
	imageQueue   []string
}

type ObservedEvent struct {
	Envelope model.MessageEnvelope
	Decision model.LoopDecision
}

type AuditEvent struct {
	Envelope model.MessageEnvelope
	Decision model.LoopDecision
}

func NewStore() *Store {
	return &Store{
		nonces:    make(map[string]time.Time),
		imageJobs: make(map[string]model.ImageJob),
	}
}

func NewEventBus() *Store {
	return NewStore()
}

func (s *Store) PublishObserved(_ context.Context, envelope model.MessageEnvelope, decision model.LoopDecision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.observed = append(s.observed, ObservedEvent{Envelope: envelope, Decision: decision})
	return nil
}

func (s *Store) PublishAgentInbound(_ context.Context, envelope model.MessageEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agentInbound = append(s.agentInbound, envelope)
	return nil
}

func (s *Store) PublishOutbound(_ context.Context, message model.OutboundMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.outbound = append(s.outbound, message)
	return nil
}

func (s *Store) RecordMessageDecision(_ context.Context, envelope model.MessageEnvelope, decision model.LoopDecision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.audits = append(s.audits, AuditEvent{Envelope: envelope, Decision: decision})
	return nil
}

func (s *Store) RecordSent(_ context.Context, record model.SendRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sendRecords = append(s.sendRecords, record)
	return nil
}

func (s *Store) SaveImageJob(_ context.Context, job model.ImageJob) error {
	if err := job.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.imageJobs[job.JobID] = job
	return nil
}

func (s *Store) FindImageJob(_ context.Context, jobID string) (model.ImageJob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.imageJobs[jobID]
	return job, ok, nil
}

func (s *Store) EnqueueImageJob(_ context.Context, job model.ImageJob) error {
	if err := job.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, queued := range s.imageQueue {
		if queued == job.JobID {
			return nil
		}
	}
	s.imageQueue = append(s.imageQueue, job.JobID)
	return nil
}

func (s *Store) RecentlySent(botID string, conversationID string, contentHash string, window time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-window)
	for i := len(s.sendRecords) - 1; i >= 0; i-- {
		record := s.sendRecords[i]
		if record.Timestamp.Before(cutoff) {
			continue
		}
		if record.FromBotID == botID && record.ConversationID == conversationID && record.ContentHash == contentHash {
			return true
		}
	}
	return false
}

func (s *Store) RecordNonce(_ context.Context, nonce string, seenAt time.Time) error {
	if nonce == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nonces[nonce] = seenAt
	return nil
}

func (s *Store) SeenNonce(nonce string, window time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	seenAt, ok := s.nonces[nonce]
	if !ok {
		return false
	}
	return seenAt.After(time.Now().Add(-window))
}

func (s *Store) Observed() []ObservedEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make([]ObservedEvent, len(s.observed))
	copy(events, s.observed)
	return events
}

func (s *Store) AgentInbound() []model.MessageEnvelope {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make([]model.MessageEnvelope, len(s.agentInbound))
	copy(events, s.agentInbound)
	return events
}

func (s *Store) Outbound() []model.OutboundMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make([]model.OutboundMessage, len(s.outbound))
	copy(events, s.outbound)
	return events
}

func (s *Store) Audits() []AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make([]AuditEvent, len(s.audits))
	copy(events, s.audits)
	return events
}

func (s *Store) ImageJobs() []model.ImageJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := make([]model.ImageJob, 0, len(s.imageJobs))
	for _, job := range s.imageJobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Store) ImageQueue() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue := make([]string, len(s.imageQueue))
	copy(queue, s.imageQueue)
	return queue
}
