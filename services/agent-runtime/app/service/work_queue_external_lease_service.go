package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	QueueLeaseDispositionAck  = "ack"
	QueueLeaseDispositionNack = "nack"
	QueueLeaseDispositionTerm = "term"
)

type WorkQueueExternalLeaseService struct {
	outbox           *OutboxService
	dispatch         *DeliveryDispatchService
	channelByAccount map[string]string
	workerID         string
	leaseTTLSeconds  int
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

func (s *WorkQueueExternalLeaseService) ExecuteWorkQueueLease(
	ctx context.Context,
	cmd command.ExecuteWorkQueueLeaseCommand,
) (query.QueueExternalLeaseExecutionView, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueExternalLeaseExecutionView{}, err
	}
	if s == nil || s.outbox == nil || s.dispatch == nil {
		return query.QueueExternalLeaseExecutionView{}, errors.New("work queue external lease service requires outbox and dispatch services")
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
	if workKind != "outbox_delivery" {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "unsupported_work_kind"
		return view, nil
	}
	if workID == "" {
		view.Disposition = QueueLeaseDispositionTerm
		view.Reason = "missing_work_id"
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
