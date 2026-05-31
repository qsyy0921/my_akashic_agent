package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultOutboxMetricLimit   = 200
	maxOutboxMetricLimit       = 200
	maxOutboxDeadLetterSamples = 10
	outboxPressureQueuedWarn   = 10
	outboxPressureDispatchWarn = 5
	outboxPressureActiveWarn   = 10
)

type OutboxMetricsService struct {
	deliveries outport.OutboxRepository
	events     outport.OutboxDeliveryEventStore
}

func NewOutboxMetricsService(
	deliveries outport.OutboxRepository,
	events outport.OutboxDeliveryEventStore,
) *OutboxMetricsService {
	return &OutboxMetricsService{deliveries: deliveries, events: events}
}

func (s *OutboxMetricsService) Get(ctx context.Context, filter query.OutboxMetricsFilter) (query.OutboxMetricsView, error) {
	if s == nil || s.deliveries == nil {
		return query.OutboxMetricsView{}, errors.New("outbox metrics service requires delivery repository")
	}
	deliveryLimit := boundedOutboxMetricLimit(filter.DeliveryLimit)
	eventLimit := boundedOutboxMetricLimit(filter.EventLimit)

	deliveries, err := s.deliveries.ListOutboxDeliveries(ctx, deliveryLimit)
	if err != nil {
		return query.OutboxMetricsView{}, err
	}
	events := make([]model.OutboxDeliveryEvent, 0)
	notes := make([]string, 0)
	if s.events == nil {
		notes = append(notes, "outbox delivery event stream unavailable")
	} else {
		events, err = s.events.ListOutboxDeliveryEvents(ctx, query.OutboxDeliveryEventFilter{Limit: eventLimit})
		if err != nil {
			return query.OutboxMetricsView{}, err
		}
	}

	return summarizeOutboxMetrics(deliveries, events, notes), nil
}

func summarizeOutboxMetrics(
	deliveries []model.OutboxDelivery,
	events []model.OutboxDeliveryEvent,
	notes []string,
) query.OutboxMetricsView {
	deliveriesByStatus := make(map[string]int)
	deliveriesByChannelKind := make(map[string]query.OutboxChannelKindMetricsView)
	deadByChannelKind := make(map[string]int)
	pressureByAccount := make(map[string]*query.OutboxAccountPressureView)
	throughput := query.OutboxThroughputMetricsView{
		EventsByType: make(map[string]int),
	}
	recentDeadLetters := make([]query.OutboxDeadLetterSampleView, 0, maxOutboxDeadLetterSamples)

	for _, delivery := range deliveries {
		status := string(delivery.Status)
		channelKind := string(delivery.Message.Channel.Kind)
		deliveriesByStatus[status]++
		channelMetrics := deliveriesByChannelKind[channelKind]
		if channelMetrics.ByStatus == nil {
			channelMetrics.ByStatus = make(map[string]int)
		}
		channelMetrics.Total++
		channelMetrics.ByStatus[status]++
		deliveriesByChannelKind[channelKind] = channelMetrics
		pressure := outboxPressureForDelivery(pressureByAccount, delivery)
		switch delivery.Status {
		case model.DeliveryQueued:
			pressure.Queued++
		case model.DeliveryDispatching:
			pressure.Dispatching++
		case model.DeliveryDeadLettered:
			pressure.DeadLettered++
		}
		if delivery.Status == model.DeliveryDeadLettered {
			deadByChannelKind[channelKind]++
		}
	}

	for _, event := range events {
		eventType := string(event.EventType)
		throughput.EventsByType[eventType]++
		switch event.EventType {
		case model.OutboxDeliveryEventQueued:
			throughput.Queued++
		case model.OutboxDeliveryEventLeased:
			throughput.Leased++
		case model.OutboxDeliveryEventDispatching:
			throughput.Dispatching++
		case model.OutboxDeliveryEventSucceeded:
			throughput.Succeeded++
		case model.OutboxDeliveryEventFailed:
			throughput.Failed++
		case model.OutboxDeliveryEventRetry:
			throughput.Retry++
		}
		if event.Status == model.DeliveryDeadLettered {
			throughput.DeadLettered++
		}
		if isTerminalOutboxStatus(event.Status) {
			throughput.TerminalEvents++
		}
		if event.Status == model.DeliveryDeadLettered && len(recentDeadLetters) < maxOutboxDeadLetterSamples {
			recentDeadLetters = append(recentDeadLetters, query.OutboxDeadLetterSampleView{
				DeliveryID:   event.DeliveryID,
				ChannelKind:  string(event.Channel.Kind),
				EventType:    eventType,
				Status:       string(event.Status),
				ErrorKind:    string(event.ErrorKind),
				ErrorMessage: event.ErrorMessage,
				Attempt:      event.Attempt,
				MaxAttempts:  event.MaxAttempts,
				OccurredAt:   formatOutboxMetricTime(event.OccurredAt),
			})
		}
	}

	return query.OutboxMetricsView{
		SampledDeliveries:       len(deliveries),
		SampledEvents:           len(events),
		DeliveriesByStatus:      deliveriesByStatus,
		DeliveriesByChannelKind: deliveriesByChannelKind,
		Throughput:              throughput,
		DeadLetters: query.OutboxDeadLetterMetricsView{
			CurrentTotal:  deliveriesByStatus[string(model.DeliveryDeadLettered)],
			ByChannelKind: deadByChannelKind,
			Recent:        recentDeadLetters,
		},
		Pressure: summarizeOutboxPressure(pressureByAccount),
		Notes:    notes,
	}
}

func outboxPressureForDelivery(
	items map[string]*query.OutboxAccountPressureView,
	delivery model.OutboxDelivery,
) *query.OutboxAccountPressureView {
	channelKind := string(delivery.Message.Channel.Kind)
	accountID := string(delivery.Message.Channel.AccountID)
	key := channelKind + ":" + accountID
	item := items[key]
	if item == nil {
		item = &query.OutboxAccountPressureView{
			AccountKey:  key,
			ChannelKind: channelKind,
			AccountID:   accountID,
		}
		items[key] = item
	}
	return item
}

func summarizeOutboxPressure(items map[string]*query.OutboxAccountPressureView) query.OutboxPressureMetricsView {
	if len(items) == 0 {
		return query.OutboxPressureMetricsView{}
	}
	accounts := make([]query.OutboxAccountPressureView, 0, len(items))
	view := query.OutboxPressureMetricsView{Accounts: len(items)}
	for _, item := range items {
		current := *item
		current.Active = current.Queued + current.Dispatching
		current.HighPressure, current.PressureReason = outboxPressureStatus(current)
		if current.HighPressure {
			view.HighPressureAccounts++
		}
		if current.Active > view.MaxActive {
			view.MaxActive = current.Active
		}
		if current.Queued > view.MaxQueued {
			view.MaxQueued = current.Queued
		}
		accounts = append(accounts, current)
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Active == accounts[j].Active {
			return accounts[i].AccountKey < accounts[j].AccountKey
		}
		return accounts[i].Active > accounts[j].Active
	})
	view.ByAccount = accounts
	return view
}

func outboxPressureStatus(item query.OutboxAccountPressureView) (bool, string) {
	if item.Active >= outboxPressureActiveWarn {
		return true, fmt.Sprintf("active>=%d", outboxPressureActiveWarn)
	}
	if item.Queued >= outboxPressureQueuedWarn {
		return true, fmt.Sprintf("queued>=%d", outboxPressureQueuedWarn)
	}
	if item.Dispatching >= outboxPressureDispatchWarn {
		return true, fmt.Sprintf("dispatching>=%d", outboxPressureDispatchWarn)
	}
	return false, ""
}

func isTerminalOutboxStatus(status model.DeliveryStatus) bool {
	switch status {
	case model.DeliverySucceeded, model.DeliveryDeadLettered:
		return true
	default:
		return false
	}
}

func boundedOutboxMetricLimit(value int) int {
	if value <= 0 {
		return defaultOutboxMetricLimit
	}
	if value > maxOutboxMetricLimit {
		return maxOutboxMetricLimit
	}
	return value
}

func formatOutboxMetricTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
