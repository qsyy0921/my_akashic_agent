package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultInboxMetricLimit   = 200
	maxInboxMetricLimit       = 200
	maxInboxRecentSampleCount = 10
)

type InboxMetricsService struct {
	repository outport.InboxEventRepository
}

func NewInboxMetricsService(repository outport.InboxEventRepository) *InboxMetricsService {
	return &InboxMetricsService{repository: repository}
}

func (s *InboxMetricsService) Get(ctx context.Context, filter query.InboxMetricsFilter) (query.InboxMetricsView, error) {
	if s == nil || s.repository == nil {
		return query.InboxMetricsView{}, errors.New("inbox metrics service requires repository")
	}
	events, err := s.repository.ListInboxEvents(ctx, query.InboxEventFilter{
		Limit:            boundedInboxMetricLimit(filter.Limit),
		ChannelKind:      strings.TrimSpace(filter.ChannelKind),
		AccountID:        strings.TrimSpace(filter.AccountID),
		ConversationID:   strings.TrimSpace(filter.ConversationID),
		ConversationType: strings.TrimSpace(filter.ConversationType),
		DecisionAction:   strings.TrimSpace(filter.DecisionAction),
		ObserveOnly:      strings.TrimSpace(filter.ObserveOnly),
	})
	if err != nil {
		return query.InboxMetricsView{}, err
	}
	return summarizeInboxMetrics(events), nil
}

func summarizeInboxMetrics(events []model.InboxEvent) query.InboxMetricsView {
	view := query.InboxMetricsView{
		EventsByChannelKind:    make(map[string]query.InboxChannelKindMetricsView),
		EventsByConversation:   make(map[string]query.InboxConversationMetricsView),
		EventsByDecisionAction: make(map[string]int),
		EventsBySenderKind:     make(map[string]int),
		Recent:                 make([]query.InboxRecentSampleView, 0, maxInboxRecentSampleCount),
	}
	senders := make(map[string]struct{})
	conversationSenders := make(map[string]map[string]struct{})

	for _, event := range events {
		envelope := event.Envelope
		channelKind := string(envelope.Channel.Kind)
		conversationType := string(envelope.Channel.ConversationType)
		decisionAction := string(event.Decision.Action)
		senderKind := string(envelope.Sender.Kind)
		conversationKey := inboxConversationKey(envelope.Channel)
		attachmentCount := len(envelope.Attachments)
		observeOnly := event.ObserveOnly()
		replyEligible := decisionAction == string(model.LoopActionAllow) && !observeOnly

		view.SampledEvents++
		if observeOnly {
			view.ObserveOnlyTotal++
		}
		if replyEligible {
			view.ReplyEligibleTotal++
		}
		if attachmentCount > 0 {
			view.WithAttachments++
		}
		view.AttachmentCount += attachmentCount
		if envelope.Sender.ID != "" {
			senders[envelope.Sender.ID] = struct{}{}
		}
		view.EventsByDecisionAction[decisionAction]++
		view.EventsBySenderKind[senderKind]++

		channelMetrics := view.EventsByChannelKind[channelKind]
		if channelMetrics.ByConversationType == nil {
			channelMetrics.ByConversationType = make(map[string]int)
		}
		channelMetrics.Total++
		channelMetrics.ByConversationType[conversationType]++
		if observeOnly {
			channelMetrics.ObserveOnly++
		}
		if replyEligible {
			channelMetrics.ReplyEligible++
		}
		if attachmentCount > 0 {
			channelMetrics.WithAttachments++
		}
		channelMetrics.AttachmentCount += attachmentCount
		view.EventsByChannelKind[channelKind] = channelMetrics

		conversationMetrics := view.EventsByConversation[conversationKey]
		conversationMetrics.Channel = toInboxMetricsChannelView(envelope.Channel)
		conversationMetrics.Total++
		if observeOnly {
			conversationMetrics.ObserveOnly++
		}
		if replyEligible {
			conversationMetrics.ReplyEligible++
		}
		if attachmentCount > 0 {
			conversationMetrics.WithAttachments++
		}
		conversationMetrics.AttachmentCount += attachmentCount
		if event.ReceivedAt.After(parseInboxMetricTime(conversationMetrics.LatestReceivedAt)) {
			conversationMetrics.LatestReceivedAt = formatInboxMetricTime(event.ReceivedAt)
		}
		if seq, ok := inboxMetricSeq(event); ok {
			conversationMetrics.SequencedEvents++
			if conversationMetrics.SequencedEvents == 1 || seq > conversationMetrics.LatestSeq {
				conversationMetrics.LatestSeq = seq
			}
		}
		if _, ok := conversationSenders[conversationKey]; !ok {
			conversationSenders[conversationKey] = make(map[string]struct{})
		}
		if envelope.Sender.ID != "" {
			conversationSenders[conversationKey][envelope.Sender.ID] = struct{}{}
		}
		conversationMetrics.UniqueSenders = len(conversationSenders[conversationKey])
		view.EventsByConversation[conversationKey] = conversationMetrics

		if len(view.Recent) < maxInboxRecentSampleCount {
			seq, _ := inboxMetricSeq(event)
			view.Recent = append(view.Recent, query.InboxRecentSampleView{
				EventID:         envelope.EventID,
				Channel:         toInboxMetricsChannelView(envelope.Channel),
				SenderID:        envelope.Sender.ID,
				SenderKind:      senderKind,
				DecisionAction:  decisionAction,
				ObserveOnly:     observeOnly,
				AttachmentCount: attachmentCount,
				Seq:             seq,
				ReceivedAt:      formatInboxMetricTime(event.ReceivedAt),
			})
		}
	}

	view.UniqueSenders = len(senders)
	return view
}

func inboxConversationKey(channel model.ChannelRef) string {
	return fmt.Sprintf(
		"%s/%s/%s/%s",
		channel.Kind,
		channel.AccountID,
		channel.ConversationType,
		channel.ConversationID,
	)
}

func toInboxMetricsChannelView(channel model.ChannelRef) query.InboxChannelView {
	return query.InboxChannelView{
		Kind:             string(channel.Kind),
		AccountID:        channel.AccountID,
		ConversationID:   channel.ConversationID,
		ConversationType: string(channel.ConversationType),
	}
}

func inboxMetricSeq(event model.InboxEvent) (int, bool) {
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

func boundedInboxMetricLimit(value int) int {
	if value <= 0 {
		return defaultInboxMetricLimit
	}
	if value > maxInboxMetricLimit {
		return maxInboxMetricLimit
	}
	return value
}

func formatInboxMetricTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func parseInboxMetricTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
