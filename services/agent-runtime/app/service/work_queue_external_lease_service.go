package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

const (
	QueueLeaseDispositionAck  = "ack"
	QueueLeaseDispositionNack = "nack"
	QueueLeaseDispositionTerm = "term"

	externalLeaseDiagnosticsSampleLimit = 100
)

type WorkQueueExternalLeaseService struct {
	outbox           *OutboxService
	dispatch         *DeliveryDispatchService
	agentJobs        *AgentJobService
	channelByAccount map[string]string
	workerID         string
	leaseTTLSeconds  int
	accountLimiter   *domainservice.OutboxAccountRateLimiter
	diagnosticsMu    sync.Mutex
	diagnostics      queueExternalLeaseDiagnostics
}

type queueExternalLeaseDiagnostics struct {
	executedTotal int
	errorTotal    int
	dispositions  map[string]int
	reasons       map[string]int
	workKinds     map[string]int
	recent        []query.QueueExternalLeaseExecutionView
}

type WorkQueueExternalLeaseOption func(*WorkQueueExternalLeaseService)

func NewWorkQueueExternalLeaseService(
	outbox *OutboxService,
	dispatch *DeliveryDispatchService,
	options ...WorkQueueExternalLeaseOption,
) *WorkQueueExternalLeaseService {
	service := &WorkQueueExternalLeaseService{
		outbox:          outbox,
		dispatch:        dispatch,
		workerID:        "agent-runtime-external-lease",
		leaseTTLSeconds: 300,
		diagnostics: queueExternalLeaseDiagnostics{
			dispositions: make(map[string]int),
			reasons:      make(map[string]int),
			workKinds:    make(map[string]int),
			recent:       make([]query.QueueExternalLeaseExecutionView, 0, externalLeaseDiagnosticsSampleLimit),
		},
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func WithExternalLeaseChannelByAccount(channelByAccount map[string]string) WorkQueueExternalLeaseOption {
	return func(service *WorkQueueExternalLeaseService) {
		service.channelByAccount = cloneStringMap(channelByAccount)
	}
}

func WithExternalLeaseWorker(workerID string, ttlSeconds int) WorkQueueExternalLeaseOption {
	return func(service *WorkQueueExternalLeaseService) {
		if workerID = strings.TrimSpace(workerID); workerID != "" {
			service.workerID = workerID
		}
		if ttlSeconds > 0 {
			service.leaseTTLSeconds = ttlSeconds
		}
	}
}

func WithExternalLeaseAgentJobs(agentJobs *AgentJobService) WorkQueueExternalLeaseOption {
	return func(service *WorkQueueExternalLeaseService) {
		service.agentJobs = agentJobs
	}
}

func WithExternalLeaseAccountRateLimit(config domainservice.OutboxAccountRateLimitConfig) WorkQueueExternalLeaseOption {
	return func(service *WorkQueueExternalLeaseService) {
		service.accountLimiter = domainservice.NewOutboxAccountRateLimiter(config)
	}
}

func (s *WorkQueueExternalLeaseService) ExecuteWorkQueueLease(
	ctx context.Context,
	cmd command.ExecuteWorkQueueLeaseCommand,
) (query.QueueExternalLeaseExecutionView, error) {
	result, err := s.executeWorkQueueLease(ctx, cmd)
	if s != nil {
		s.recordExternalLeaseExecution(cmd, result, err)
	}
	return result, err
}

func (s *WorkQueueExternalLeaseService) executeWorkQueueLease(
	ctx context.Context,
	cmd command.ExecuteWorkQueueLeaseCommand,
) (query.QueueExternalLeaseExecutionView, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueExternalLeaseExecutionView{}, err
	}
	if s == nil {
		return query.QueueExternalLeaseExecutionView{}, errors.New("work queue external lease service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	workKind := normalizeWorkKind(cmd.WorkKind)
	workID := firstNonBlank(cmd.WorkID, cmd.AggregateID)
	view := query.QueueExternalLeaseExecutionView{
		WorkKind:    workKind,
		WorkID:      workID,
		AggregateID: strings.TrimSpace(cmd.AggregateID),
		Subject:     strings.TrimSpace(cmd.Subject),
		ExecutedAt:  formatQueueCompareTime(now),
	}
	switch workKind {
	case "outbox_delivery":
		return s.executeOutboxWork(ctx, cmd, view, workID, now)
	case "agent_job":
		return s.executeAgentJobWork(ctx, view, workID, now)
	default:
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "unsupported_work_kind"
		return view, nil
	}
}

func (s *WorkQueueExternalLeaseService) SnapshotExternalLeaseDiagnostics(
	ctx context.Context,
) (query.QueueExternalLeaseDiagnostics, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueExternalLeaseDiagnostics{}, err
	}
	if s == nil {
		return query.QueueExternalLeaseDiagnostics{}, errors.New("work queue external lease service is nil")
	}
	s.diagnosticsMu.Lock()
	defer s.diagnosticsMu.Unlock()
	return query.QueueExternalLeaseDiagnostics{
		Enabled:          true,
		SampleLimit:      externalLeaseDiagnosticsSampleLimit,
		ExecutedTotal:    s.diagnostics.executedTotal,
		ErrorTotal:       s.diagnostics.errorTotal,
		Dispositions:     sortedExternalLeaseCounters(s.diagnostics.dispositions),
		Reasons:          sortedExternalLeaseCounters(s.diagnostics.reasons),
		WorkKinds:        sortedExternalLeaseCounters(s.diagnostics.workKinds),
		RecentExecutions: cloneExternalLeaseExecutions(s.diagnostics.recent),
		Notes: []string{
			"external_lease diagnostics are in-memory runtime observations",
			"state store and lifecycle event streams remain authoritative",
		},
	}, nil
}

func (s *WorkQueueExternalLeaseService) recordExternalLeaseExecution(
	cmd command.ExecuteWorkQueueLeaseCommand,
	view query.QueueExternalLeaseExecutionView,
	err error,
) {
	if s == nil {
		return
	}
	if view.WorkKind == "" {
		view.WorkKind = normalizeWorkKind(cmd.WorkKind)
	}
	if view.WorkID == "" {
		view.WorkID = firstNonBlank(cmd.WorkID, cmd.AggregateID)
	}
	if view.AggregateID == "" {
		view.AggregateID = strings.TrimSpace(cmd.AggregateID)
	}
	if view.Subject == "" {
		view.Subject = strings.TrimSpace(cmd.Subject)
	}
	if view.ExecutedAt == "" {
		timestamp := cmd.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now().UTC()
		}
		view.ExecutedAt = formatQueueCompareTime(timestamp)
	}
	if err != nil {
		view.Disposition = QueueLeaseDispositionNack
		view.Reason = "executor_error"
	}
	if view.Disposition == "" {
		view.Disposition = QueueLeaseDispositionTerm
	}
	if view.Reason == "" {
		view.Reason = "unspecified"
	}

	s.diagnosticsMu.Lock()
	defer s.diagnosticsMu.Unlock()
	if s.diagnostics.dispositions == nil {
		s.diagnostics.dispositions = make(map[string]int)
	}
	if s.diagnostics.reasons == nil {
		s.diagnostics.reasons = make(map[string]int)
	}
	if s.diagnostics.workKinds == nil {
		s.diagnostics.workKinds = make(map[string]int)
	}
	s.diagnostics.executedTotal++
	if err != nil {
		s.diagnostics.errorTotal++
	}
	s.diagnostics.dispositions[view.Disposition]++
	s.diagnostics.reasons[view.Reason]++
	s.diagnostics.workKinds[view.WorkKind]++
	s.diagnostics.recent = append([]query.QueueExternalLeaseExecutionView{view}, s.diagnostics.recent...)
	if len(s.diagnostics.recent) > externalLeaseDiagnosticsSampleLimit {
		s.diagnostics.recent = s.diagnostics.recent[:externalLeaseDiagnosticsSampleLimit]
	}
}

func (s *WorkQueueExternalLeaseService) executeOutboxWork(
	ctx context.Context,
	cmd command.ExecuteWorkQueueLeaseCommand,
	view query.QueueExternalLeaseExecutionView,
	workID string,
	now time.Time,
) (query.QueueExternalLeaseExecutionView, error) {
	if s.outbox == nil || s.dispatch == nil {
		return query.QueueExternalLeaseExecutionView{}, errors.New("work queue external lease service requires outbox and dispatch services")
	}
	if workID == "" {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "missing_work_id"
		return view, nil
	}
	current, err := s.outbox.Get(ctx, workID)
	if err != nil {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "missing_state"
		return view, nil
	}
	if current.Status == string(model.DeliverySucceeded) || current.Status == string(model.DeliveryDeadLettered) {
		return s.leaseRejectedView(ctx, view, workID)
	}
	if s.outboxAccountBlocked(current, now) {
		view.Disposition = QueueLeaseDispositionNack
		view.Reason = "delivery_rate_limited"
		view.StateStatus = current.Status
		view.Attempts = current.Attempts
		return view, nil
	}

	leased, err := s.outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    workID,
		WorkerID:   firstNonBlank(cmd.WorkerID, s.workerID),
		TTLSeconds: firstPositive(cmd.LeaseTTLSeconds, s.leaseTTLSeconds),
		Timestamp:  now,
	})
	if err != nil {
		return s.leaseRejectedView(ctx, view, workID)
	}
	view.StateStatus = leased.Status
	view.Attempts = leased.Attempts

	_, dispatchErr := s.dispatch.Dispatch(ctx, command.DispatchDeliveryCommand{
		EventID:          workID,
		ChannelByAccount: mergeStringMaps(s.channelByAccount, cmd.ChannelByAccount),
	})
	s.recordOutboxDispatchAttempt(leased, now)
	if dispatchErr == nil {
		succeeded, err := s.outbox.MarkSucceeded(ctx, command.MarkOutboxSucceededCommand{
			EventID:   workID,
			Timestamp: now,
		})
		if err != nil {
			return query.QueueExternalLeaseExecutionView{}, err
		}
		view.Disposition = QueueLeaseDispositionAck
		view.Reason = "delivery_succeeded"
		view.StateStatus = succeeded.Status
		view.Attempts = succeeded.Attempts
		return view, nil
	}

	failureKind, failureMessage := deliveryFailure(dispatchErr)
	failed, err := s.outbox.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      workID,
		ErrorKind:    string(failureKind),
		ErrorMessage: failureMessage,
		Timestamp:    now,
	})
	if err != nil {
		return query.QueueExternalLeaseExecutionView{}, err
	}
	view.StateStatus = failed.Status
	view.Attempts = failed.Attempts
	if failed.Status == string(model.DeliveryFailed) {
		retried, err := s.outbox.Retry(ctx, command.RetryOutboxCommand{
			EventID:   workID,
			Timestamp: now,
		})
		if err != nil {
			return query.QueueExternalLeaseExecutionView{}, err
		}
		view.Disposition = QueueLeaseDispositionNack
		view.Reason = "delivery_retry_scheduled"
		view.StateStatus = retried.Status
		view.Attempts = retried.Attempts
		return view, nil
	}
	view.Disposition = QueueLeaseDispositionAck
	view.Reason = "delivery_terminal_failure"
	return view, nil
}

func (s *WorkQueueExternalLeaseService) outboxAccountBlocked(delivery query.OutboxDeliveryView, now time.Time) bool {
	if s == nil || s.accountLimiter == nil {
		return false
	}
	return s.accountLimiter.AccountBlocked(outboxDeliveryViewAccountKey(delivery), now)
}

func (s *WorkQueueExternalLeaseService) recordOutboxDispatchAttempt(delivery query.OutboxDeliveryView, now time.Time) {
	if s == nil || s.accountLimiter == nil {
		return
	}
	s.accountLimiter.Record(outboxDeliveryViewAccountKey(delivery), now)
}

func (s *WorkQueueExternalLeaseService) executeAgentJobWork(
	ctx context.Context,
	view query.QueueExternalLeaseExecutionView,
	workID string,
	now time.Time,
) (query.QueueExternalLeaseExecutionView, error) {
	if workID == "" {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "missing_work_id"
		return view, nil
	}
	if s.agentJobs == nil {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "agent_job_executor_unavailable"
		return view, nil
	}
	job, err := s.agentJobs.getModel(ctx, workID)
	if err != nil {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "missing_state"
		return view, nil
	}
	return s.agentJobResultAckView(ctx, view, job, now)
}

func (s *WorkQueueExternalLeaseService) agentJobResultAckView(
	ctx context.Context,
	view query.QueueExternalLeaseExecutionView,
	job model.AgentJob,
	now time.Time,
) (query.QueueExternalLeaseExecutionView, error) {
	view.StateStatus = string(job.Status)
	view.Attempts = job.Attempts
	switch job.Status {
	case model.AgentJobSucceeded, model.AgentJobDeadLettered, model.AgentJobCancelled:
		view.Disposition = QueueLeaseDispositionAck
		view.Reason = "agent_job_terminal"
		return view, nil
	case model.AgentJobFailed:
		retried, err := s.agentJobs.Retry(ctx, command.RetryAgentJobCommand{
			JobID:     job.JobID,
			Timestamp: now,
		})
		if err != nil {
			return query.QueueExternalLeaseExecutionView{}, err
		}
		view.Disposition = QueueLeaseDispositionNack
		view.Reason = "agent_job_retry_scheduled"
		view.StateStatus = retried.Status
		view.Attempts = retried.Attempts
		return view, nil
	case model.AgentJobLeased, model.AgentJobRunning:
		if job.LeaseExpired(now) {
			recovered, err := s.agentJobs.recoverExpiredJob(ctx, job, now)
			if err != nil {
				return query.QueueExternalLeaseExecutionView{}, err
			}
			view.StateStatus = recovered.Job.Status
			view.Attempts = recovered.Job.Attempts
			if recovered.Job.Status == string(model.AgentJobDeadLettered) {
				view.Disposition = QueueLeaseDispositionAck
				view.Reason = "agent_job_terminal_after_recovery"
				return view, nil
			}
			view.Disposition = QueueLeaseDispositionNack
			view.Reason = "agent_job_expired_lease_recovered"
			return view, nil
		}
		view.Disposition = QueueLeaseDispositionNack
		view.Reason = "agent_job_waiting_for_result"
		return view, nil
	case model.AgentJobPending:
		view.Disposition = QueueLeaseDispositionNack
		view.Reason = "agent_job_pending_for_python_worker"
		return view, nil
	default:
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "agent_job_invalid_state"
		return view, nil
	}
}

func (s *WorkQueueExternalLeaseService) leaseRejectedView(
	ctx context.Context,
	view query.QueueExternalLeaseExecutionView,
	workID string,
) (query.QueueExternalLeaseExecutionView, error) {
	current, err := s.outbox.Get(ctx, workID)
	if err != nil {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "missing_state"
		return view, nil
	}
	view.StateStatus = current.Status
	view.Attempts = current.Attempts
	switch current.Status {
	case string(model.DeliverySucceeded), string(model.DeliveryDeadLettered):
		view.Disposition = QueueLeaseDispositionAck
		view.Reason = "already_terminal"
	default:
		view.Disposition = QueueLeaseDispositionNack
		view.Reason = "go_lease_rejected"
	}
	return view, nil
}

func deliveryFailure(err error) (model.DeliveryErrorKind, string) {
	if err == nil {
		return model.DeliveryErrorUnknown, "delivery dispatch failed"
	}
	kind := model.DeliveryErrorUnknown
	if kinded, ok := err.(interface{ DeliveryErrorKind() string }); ok {
		kind = model.NormalizeDeliveryErrorKind(kinded.DeliveryErrorKind())
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "delivery dispatch failed"
	}
	return kind, message
}

func outboxDeliveryViewAccountKey(delivery query.OutboxDeliveryView) string {
	return strings.TrimSpace(delivery.Channel.Kind) + ":" + strings.TrimSpace(delivery.Channel.AccountID)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func cloneStringMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}

func mergeStringMaps(left map[string]string, right map[string]string) map[string]string {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	merged := cloneStringMap(left)
	if merged == nil {
		merged = make(map[string]string)
	}
	for key, value := range right {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		merged[key] = value
	}
	return merged
}

func sortedExternalLeaseCounters(items map[string]int) []query.QueueExternalLeaseCounter {
	if len(items) == 0 {
		return nil
	}
	result := make([]query.QueueExternalLeaseCounter, 0, len(items))
	for name, count := range items {
		result = append(result, query.QueueExternalLeaseCounter{Name: name, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Name < result[j].Name
		}
		return result[i].Count > result[j].Count
	})
	return result
}

func cloneExternalLeaseExecutions(items []query.QueueExternalLeaseExecutionView) []query.QueueExternalLeaseExecutionView {
	if len(items) == 0 {
		return nil
	}
	result := make([]query.QueueExternalLeaseExecutionView, len(items))
	copy(result, items)
	return result
}
