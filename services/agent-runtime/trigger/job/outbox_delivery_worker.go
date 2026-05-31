package jobtrigger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboxDeliveryManager interface {
	LeaseNext(ctx context.Context, cmd command.LeaseNextOutboxCommand) (query.OutboxDeliveryView, error)
	MarkDispatching(ctx context.Context, cmd command.MarkOutboxDispatchingCommand) (query.OutboxDeliveryView, error)
	MarkSucceeded(ctx context.Context, cmd command.MarkOutboxSucceededCommand) (query.OutboxDeliveryView, error)
	MarkFailed(ctx context.Context, cmd command.MarkOutboxFailedCommand) (query.OutboxDeliveryView, error)
}

type OutboxDeliveryDispatcher interface {
	Dispatch(ctx context.Context, cmd command.DispatchDeliveryCommand) (query.DeliveryDispatchResultView, error)
}

type OutboxAccountLimiter interface {
	BlockedAccountKeys(now time.Time) []string
	Record(accountKey string, now time.Time)
}

type OutboxDeliveryWorkerConfig struct {
	Interval                     time.Duration
	BatchSize                    int
	WorkerID                     string
	LeaseTTLSeconds              int
	RunOnStart                   bool
	ChannelByAccount             map[string]string
	AccountMinInterval           time.Duration
	AccountWindow                time.Duration
	AccountMaxDispatchesInWindow int
	AccountLimiter               OutboxAccountLimiter
	Now                          func() time.Time
	Logf                         func(format string, args ...any)
}

type OutboxDeliveryWorkerResult struct {
	Processed          bool
	EventID            string
	DispatchCount      int
	Failed             bool
	ErrorKind          string
	ErrorMessage       string
	Reason             string
	BlockedAccountKeys []string
}

type OutboxDeliveryWorker struct {
	outbox           OutboxDeliveryManager
	dispatcher       OutboxDeliveryDispatcher
	interval         time.Duration
	batchSize        int
	workerID         string
	leaseTTLSeconds  int
	runOnStart       bool
	channelByAccount map[string]string
	accountLimiter   OutboxAccountLimiter
	now              func() time.Time
	logf             func(format string, args ...any)
}

func NewOutboxDeliveryWorker(
	outbox OutboxDeliveryManager,
	dispatcher OutboxDeliveryDispatcher,
	config OutboxDeliveryWorkerConfig,
) (*OutboxDeliveryWorker, error) {
	if outbox == nil {
		return nil, errors.New("outbox delivery worker requires outbox manager")
	}
	if dispatcher == nil {
		return nil, errors.New("outbox delivery worker requires dispatcher")
	}
	if config.Interval <= 0 {
		config.Interval = 2 * time.Second
	}
	if config.BatchSize <= 0 || config.BatchSize > 50 {
		config.BatchSize = 1
	}
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = "agent-runtime-outbox-worker"
	}
	if config.LeaseTTLSeconds <= 0 || config.LeaseTTLSeconds > 86400 {
		config.LeaseTTLSeconds = 300
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &OutboxDeliveryWorker{
		outbox:           outbox,
		dispatcher:       dispatcher,
		interval:         config.Interval,
		batchSize:        config.BatchSize,
		workerID:         strings.TrimSpace(config.WorkerID),
		leaseTTLSeconds:  config.LeaseTTLSeconds,
		runOnStart:       config.RunOnStart,
		channelByAccount: cloneStringMap(config.ChannelByAccount),
		accountLimiter:   config.AccountLimiter,
		now:              config.Now,
		logf:             config.Logf,
	}, nil
}

func (w *OutboxDeliveryWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("outbox delivery worker is nil")
	}
	if w.runOnStart {
		if err := w.ProcessBatch(ctx); err != nil {
			if ctx.Err() != nil {
				return err
			}
			w.log("outbox delivery worker batch failed: %v", err)
		}
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.ProcessBatch(ctx); err != nil {
				w.log("outbox delivery worker batch failed: %v", err)
			}
		}
	}
}

func (w *OutboxDeliveryWorker) ProcessBatch(ctx context.Context) error {
	if w == nil {
		return errors.New("outbox delivery worker is nil")
	}
	for i := 0; i < w.batchSize; i++ {
		result, err := w.ProcessOnce(ctx)
		if err != nil {
			return err
		}
		if !result.Processed {
			return nil
		}
		if result.Failed {
			w.log(
				"outbox delivery worker failed event_id=%s error_kind=%s error=%s",
				result.EventID,
				result.ErrorKind,
				result.ErrorMessage,
			)
		}
	}
	return nil
}

func (w *OutboxDeliveryWorker) ProcessOnce(ctx context.Context) (OutboxDeliveryWorkerResult, error) {
	if w == nil {
		return OutboxDeliveryWorkerResult{}, errors.New("outbox delivery worker is nil")
	}
	if err := ctx.Err(); err != nil {
		return OutboxDeliveryWorkerResult{}, err
	}
	now := w.now()
	blockedAccountKeys := w.blockedAccountKeys(now)
	delivery, err := w.outbox.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:           w.workerID,
		TTLSeconds:         w.leaseTTLSeconds,
		Timestamp:          now,
		BlockedAccountKeys: blockedAccountKeys,
	})
	if err != nil {
		if isNoLeaseableOutboxDelivery(err) {
			reason := "no_delivery"
			if len(blockedAccountKeys) > 0 {
				reason = "no_delivery_or_rate_limited"
			}
			return OutboxDeliveryWorkerResult{
				Processed:          false,
				Reason:             reason,
				BlockedAccountKeys: blockedAccountKeys,
			}, nil
		}
		return OutboxDeliveryWorkerResult{}, err
	}
	result := OutboxDeliveryWorkerResult{
		Processed: true,
		EventID:   delivery.EventID,
	}
	if _, err := w.outbox.MarkDispatching(ctx, command.MarkOutboxDispatchingCommand{
		EventID:   delivery.EventID,
		Timestamp: w.now(),
	}); err != nil {
		return result, err
	}
	dispatch, err := w.dispatcher.Dispatch(ctx, command.DispatchDeliveryCommand{
		EventID:          delivery.EventID,
		ChannelByAccount: cloneStringMap(w.channelByAccount),
	})
	w.recordDispatchAttempt(delivery, w.now())
	if err != nil {
		kind := deliveryErrorKind(err)
		message := err.Error()
		if _, markErr := w.outbox.MarkFailed(ctx, command.MarkOutboxFailedCommand{
			EventID:      delivery.EventID,
			ErrorKind:    kind,
			ErrorMessage: message,
			Timestamp:    w.now(),
		}); markErr != nil {
			return result, fmt.Errorf("dispatch failed with %s: %s; mark failed: %w", kind, message, markErr)
		}
		result.Failed = true
		result.ErrorKind = kind
		result.ErrorMessage = message
		return result, nil
	}
	if _, err := w.outbox.MarkSucceeded(ctx, command.MarkOutboxSucceededCommand{
		EventID:   delivery.EventID,
		Timestamp: w.now(),
	}); err != nil {
		return result, err
	}
	result.DispatchCount = dispatch.StepCount
	return result, nil
}

func (w *OutboxDeliveryWorker) blockedAccountKeys(now time.Time) []string {
	if w == nil || w.accountLimiter == nil {
		return nil
	}
	return w.accountLimiter.BlockedAccountKeys(now)
}

func (w *OutboxDeliveryWorker) recordDispatchAttempt(delivery query.OutboxDeliveryView, now time.Time) {
	if w == nil || w.accountLimiter == nil {
		return
	}
	w.accountLimiter.Record(outboxDeliveryViewAccountKey(delivery), now)
}

func (w *OutboxDeliveryWorker) log(format string, args ...any) {
	if w != nil && w.logf != nil {
		w.logf(format, args...)
	}
}

func deliveryErrorKind(err error) string {
	if err == nil {
		return "unknown"
	}
	var typed interface{ DeliveryErrorKind() string }
	if errors.As(err, &typed) {
		kind := strings.TrimSpace(typed.DeliveryErrorKind())
		if kind != "" {
			return kind
		}
	}
	return "unknown"
}

func isNoLeaseableOutboxDelivery(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no leaseable outbox delivery")
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

func outboxDeliveryViewAccountKey(delivery query.OutboxDeliveryView) string {
	return strings.TrimSpace(delivery.Channel.Kind) + ":" + strings.TrimSpace(delivery.Channel.AccountID)
}
