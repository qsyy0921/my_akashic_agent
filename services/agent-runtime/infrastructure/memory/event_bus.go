package memory

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type Store struct {
	mu            sync.Mutex
	observed      []ObservedEvent
	agentInbound  []model.MessageEnvelope
	outbound      []model.OutboundMessage
	sendRecords   []model.SendRecord
	nonces        map[string]time.Time
	audits        []AuditEvent
	imageJobs     map[string]model.ImageJob
	imageQueue    []string
	outbox        map[string]model.OutboxDelivery
	outboxOrder   []string
	outboxQueue   []string
	mediaAssets   map[string]model.MediaAsset
	mediaOrder    []string
	inboxEvents   map[string]model.InboxEvent
	inboxOrder    []string
	agentJobs     map[string]model.AgentJob
	agentJobOrder []string
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
		nonces:      make(map[string]time.Time),
		imageJobs:   make(map[string]model.ImageJob),
		outbox:      make(map[string]model.OutboxDelivery),
		mediaAssets: make(map[string]model.MediaAsset),
		inboxEvents: make(map[string]model.InboxEvent),
		agentJobs:   make(map[string]model.AgentJob),
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

func (s *Store) SaveOutboxDelivery(_ context.Context, delivery model.OutboxDelivery) error {
	if err := delivery.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.outbox[delivery.Message.EventID]; !exists {
		s.outboxOrder = append(s.outboxOrder, delivery.Message.EventID)
	}
	s.outbox[delivery.Message.EventID] = delivery
	return nil
}

func (s *Store) FindOutboxDelivery(_ context.Context, eventID string) (model.OutboxDelivery, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delivery, ok := s.outbox[eventID]
	return delivery, ok, nil
}

func (s *Store) ListOutboxDeliveries(_ context.Context, limit int) ([]model.OutboxDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	start := len(s.outboxOrder) - limit
	if start < 0 {
		start = 0
	}
	items := make([]model.OutboxDelivery, 0, len(s.outboxOrder)-start)
	for i := len(s.outboxOrder) - 1; i >= start; i-- {
		eventID := s.outboxOrder[i]
		if delivery, ok := s.outbox[eventID]; ok {
			items = append(items, delivery)
		}
	}
	return items, nil
}

func (s *Store) FindLeaseableOutboxDelivery(_ context.Context, now time.Time) (model.OutboxDelivery, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, eventID := range s.outboxQueue {
		if delivery, ok := s.outbox[eventID]; ok && delivery.CanLease(now) {
			return delivery, true, nil
		}
	}
	for _, eventID := range s.outboxOrder {
		if delivery, ok := s.outbox[eventID]; ok && delivery.CanLease(now) {
			return delivery, true, nil
		}
	}
	return model.OutboxDelivery{}, false, nil
}

func (s *Store) EnqueueOutboxDelivery(_ context.Context, delivery model.OutboxDelivery) error {
	if err := delivery.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, queued := range s.outboxQueue {
		if queued == delivery.Message.EventID {
			return nil
		}
	}
	s.outboxQueue = append(s.outboxQueue, delivery.Message.EventID)
	return nil
}

func (s *Store) SaveMediaAsset(_ context.Context, asset model.MediaAsset) error {
	if err := asset.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.mediaAssets[asset.AssetID]; !exists {
		s.mediaOrder = append(s.mediaOrder, asset.AssetID)
	}
	s.mediaAssets[asset.AssetID] = asset
	return nil
}

func (s *Store) FindMediaAsset(_ context.Context, assetID string) (model.MediaAsset, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.mediaAssets[assetID]
	return asset, ok, nil
}

func (s *Store) ListMediaAssets(_ context.Context, filter query.MediaAssetFilter) ([]model.MediaAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.MediaAsset, 0, limit)
	for i := len(s.mediaOrder) - 1; i >= 0 && len(items) < limit; i-- {
		assetID := s.mediaOrder[i]
		if asset, ok := s.mediaAssets[assetID]; ok && matchesMediaAssetFilter(asset, filter) {
			items = append(items, asset)
		}
	}
	return items, nil
}

func matchesMediaAssetFilter(asset model.MediaAsset, filter query.MediaAssetFilter) bool {
	if filter.ChannelKind != "" && string(asset.Channel.Kind) != filter.ChannelKind {
		return false
	}
	if filter.AccountID != "" && asset.Channel.AccountID != filter.AccountID {
		return false
	}
	if filter.ConversationID != "" && asset.Channel.ConversationID != filter.ConversationID {
		return false
	}
	if filter.ConversationType != "" && string(asset.Channel.ConversationType) != filter.ConversationType {
		return false
	}
	if filter.SourceMessageID != "" && asset.SourceMessageID != filter.SourceMessageID {
		return false
	}
	if filter.SourceMessageIDSuffix != "" && !strings.HasSuffix(asset.SourceMessageID, ":"+filter.SourceMessageIDSuffix) && asset.SourceMessageID != filter.SourceMessageIDSuffix {
		return false
	}
	if filter.Kind != "" && string(asset.Kind) != filter.Kind {
		return false
	}
	return true
}

func (s *Store) SaveInboxEvent(_ context.Context, event model.InboxEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	eventID := event.EventID()
	if _, exists := s.inboxEvents[eventID]; exists {
		return nil
	}
	s.inboxOrder = append(s.inboxOrder, eventID)
	s.inboxEvents[eventID] = event
	return nil
}

func (s *Store) FindInboxEvent(_ context.Context, eventID string) (model.InboxEvent, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.inboxEvents[eventID]
	return event, ok, nil
}

func (s *Store) ListInboxEvents(_ context.Context, filter query.InboxEventFilter) ([]model.InboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.InboxEvent, 0, limit)
	if inboxOrderAscending(filter) {
		for i := 0; i < len(s.inboxOrder) && len(items) < limit; i++ {
			eventID := s.inboxOrder[i]
			if event, ok := s.inboxEvents[eventID]; ok && matchesInboxEventFilter(event, filter) {
				items = append(items, event)
			}
		}
	} else {
		for i := len(s.inboxOrder) - 1; i >= 0 && len(items) < limit; i-- {
			eventID := s.inboxOrder[i]
			if event, ok := s.inboxEvents[eventID]; ok && matchesInboxEventFilter(event, filter) {
				items = append(items, event)
			}
		}
	}
	return items, nil
}

func matchesInboxEventFilter(event model.InboxEvent, filter query.InboxEventFilter) bool {
	envelope := event.Envelope
	if filter.AfterSeqSet {
		seq, ok := inboxEventSeq(event)
		if !ok || seq <= filter.AfterSeq {
			return false
		}
	}
	if filter.ChannelKind != "" && string(envelope.Channel.Kind) != filter.ChannelKind {
		return false
	}
	if filter.AccountID != "" && envelope.Channel.AccountID != filter.AccountID {
		return false
	}
	if filter.ConversationID != "" && envelope.Channel.ConversationID != filter.ConversationID {
		return false
	}
	if filter.ConversationType != "" && string(envelope.Channel.ConversationType) != filter.ConversationType {
		return false
	}
	if filter.SenderID != "" && envelope.Sender.ID != filter.SenderID {
		return false
	}
	if filter.DecisionAction != "" && string(event.Decision.Action) != filter.DecisionAction {
		return false
	}
	if filter.ObserveOnly != "" && parseBoolFilter(filter.ObserveOnly) != event.ObserveOnly() {
		return false
	}
	return true
}

func parseBoolFilter(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes"
}

func inboxOrderAscending(filter query.InboxEventFilter) bool {
	order := strings.ToLower(strings.TrimSpace(filter.Order))
	return order == "asc" || order == "oldest"
}

func inboxEventSeq(event model.InboxEvent) (int, bool) {
	raw := strings.TrimSpace(event.Envelope.Metadata["seq"])
	if raw == "" {
		return 0, false
	}
	seq, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return seq, true
}

func (s *Store) SaveAgentJob(_ context.Context, job model.AgentJob) error {
	if err := job.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agentJobs[job.JobID]; !exists {
		s.agentJobOrder = append(s.agentJobOrder, job.JobID)
	}
	s.agentJobs[job.JobID] = job
	return nil
}

func (s *Store) FindAgentJob(_ context.Context, jobID string) (model.AgentJob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.agentJobs[jobID]
	return job, ok, nil
}

func (s *Store) ListAgentJobs(_ context.Context, filter query.AgentJobFilter) ([]model.AgentJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.AgentJob, 0, limit)
	for i := len(s.agentJobOrder) - 1; i >= 0 && len(items) < limit; i-- {
		jobID := s.agentJobOrder[i]
		job, ok := s.agentJobs[jobID]
		if !ok {
			continue
		}
		if filter.JobType != "" && string(job.JobType) != filter.JobType {
			continue
		}
		if filter.Status != "" && string(job.Status) != filter.Status {
			continue
		}
		items = append(items, job)
	}
	return items, nil
}

func (s *Store) FindLeaseableAgentJob(_ context.Context, jobType string, now time.Time) (model.AgentJob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, jobID := range s.agentJobOrder {
		job, ok := s.agentJobs[jobID]
		if !ok {
			continue
		}
		if jobType != "" && string(job.JobType) != jobType {
			continue
		}
		if job.CanLease(now) {
			return job, true, nil
		}
	}
	return model.AgentJob{}, false, nil
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

func (s *Store) ListSentRecords(_ context.Context, filter query.SendRecordFilter) ([]model.SendRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.SendRecord, 0, limit)
	for i := len(s.sendRecords) - 1; i >= 0 && len(items) < limit; i-- {
		record := s.sendRecords[i]
		if filter.FromBotID != "" && record.FromBotID != filter.FromBotID {
			continue
		}
		if filter.ConversationID != "" && record.ConversationID != filter.ConversationID {
			continue
		}
		if filter.ContentHash != "" && record.ContentHash != filter.ContentHash {
			continue
		}
		items = append(items, record)
	}
	return items, nil
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

func (s *Store) ListShadowObserved(_ context.Context, limit int) ([]outport.ShadowObservedRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	start := len(s.observed) - limit
	if start < 0 {
		start = 0
	}
	records := make([]outport.ShadowObservedRecord, 0, len(s.observed)-start)
	for i := len(s.observed) - 1; i >= start; i-- {
		item := s.observed[i]
		records = append(records, outport.ShadowObservedRecord{
			Envelope: item.Envelope,
			Decision: item.Decision,
		})
	}
	return records, nil
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

func (s *Store) OutboxDeliveries() []model.OutboxDelivery {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]model.OutboxDelivery, 0, len(s.outboxOrder))
	for _, eventID := range s.outboxOrder {
		if delivery, ok := s.outbox[eventID]; ok {
			items = append(items, delivery)
		}
	}
	return items
}

func (s *Store) OutboxQueue() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue := make([]string, len(s.outboxQueue))
	copy(queue, s.outboxQueue)
	return queue
}

func (s *Store) MediaAssets() []model.MediaAsset {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]model.MediaAsset, 0, len(s.mediaOrder))
	for _, assetID := range s.mediaOrder {
		if asset, ok := s.mediaAssets[assetID]; ok {
			items = append(items, asset)
		}
	}
	return items
}

func (s *Store) InboxEvents() []model.InboxEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]model.InboxEvent, 0, len(s.inboxOrder))
	for _, eventID := range s.inboxOrder {
		if event, ok := s.inboxEvents[eventID]; ok {
			items = append(items, event)
		}
	}
	return items
}

func (s *Store) AgentJobs() []model.AgentJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]model.AgentJob, 0, len(s.agentJobOrder))
	for _, jobID := range s.agentJobOrder {
		if job, ok := s.agentJobs[jobID]; ok {
			items = append(items, job)
		}
	}
	return items
}
