package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const (
	defaultSendLedgerMetricLimit = 200
	maxSendLedgerMetricLimit     = 200
	maxSendLedgerRepeatedHashes  = 10
	maxSendLedgerRecentRecords   = 10
)

func (s *SendLedgerService) Metrics(ctx context.Context, filter query.SendLedgerMetricsFilter) (query.SendLedgerMetricsView, error) {
	if s == nil || s.ledger == nil {
		return query.SendLedgerMetricsView{}, errors.New("send ledger service requires ledger")
	}
	return summarizeSendLedgerMetrics(ctx, s.ledger, filter)
}

func summarizeSendLedgerMetrics(
	ctx context.Context,
	ledger outport.SendLedger,
	filter query.SendLedgerMetricsFilter,
) (query.SendLedgerMetricsView, error) {
	records, err := ledger.ListSentRecords(ctx, query.SendRecordFilter{
		Limit:          boundedSendLedgerMetricLimit(filter.Limit),
		FromBotID:      strings.TrimSpace(filter.FromBotID),
		ConversationID: strings.TrimSpace(filter.ConversationID),
	})
	if err != nil {
		return query.SendLedgerMetricsView{}, err
	}

	view := query.SendLedgerMetricsView{
		RecordsByBot:          make(map[string]query.SendLedgerBotMetricsView),
		RecordsByConversation: make(map[string]query.SendLedgerConversationMetricsView),
		RepeatedHashes:        make([]query.SendLedgerRepeatedHashView, 0, maxSendLedgerRepeatedHashes),
		Recent:                make([]query.SendRecordView, 0, maxSendLedgerRecentRecords),
	}
	bots := make(map[string]struct{})
	conversations := make(map[string]struct{})
	contentHashes := make(map[string]struct{})
	botConversations := make(map[string]map[string]struct{})
	botContentHashes := make(map[string]map[string]struct{})
	conversationContentHashes := make(map[string]map[string]int)
	conversationLatestByHash := make(map[string]map[string]time.Time)

	for _, record := range records {
		view.SampledRecords++
		bots[record.FromBotID] = struct{}{}
		conversationKey := sendLedgerConversationKey(record.FromBotID, record.ConversationID)
		conversations[conversationKey] = struct{}{}
		contentHashes[record.ContentHash] = struct{}{}

		if _, ok := botConversations[record.FromBotID]; !ok {
			botConversations[record.FromBotID] = make(map[string]struct{})
		}
		botConversations[record.FromBotID][conversationKey] = struct{}{}
		if _, ok := botContentHashes[record.FromBotID]; !ok {
			botContentHashes[record.FromBotID] = make(map[string]struct{})
		}
		botContentHashes[record.FromBotID][record.ContentHash] = struct{}{}

		botMetrics := view.RecordsByBot[record.FromBotID]
		botMetrics.Total++
		if record.Timestamp.After(parseSendLedgerMetricTime(botMetrics.LatestTimestamp)) {
			botMetrics.LatestTimestamp = formatSendLedgerMetricTime(record.Timestamp)
		}
		view.RecordsByBot[record.FromBotID] = botMetrics

		conversationMetrics := view.RecordsByConversation[conversationKey]
		conversationMetrics.FromBotID = record.FromBotID
		conversationMetrics.ConversationID = record.ConversationID
		conversationMetrics.Total++
		if record.Timestamp.After(parseSendLedgerMetricTime(conversationMetrics.LatestTimestamp)) {
			conversationMetrics.LatestTimestamp = formatSendLedgerMetricTime(record.Timestamp)
		}
		view.RecordsByConversation[conversationKey] = conversationMetrics

		if _, ok := conversationContentHashes[conversationKey]; !ok {
			conversationContentHashes[conversationKey] = make(map[string]int)
		}
		conversationContentHashes[conversationKey][record.ContentHash]++
		if _, ok := conversationLatestByHash[conversationKey]; !ok {
			conversationLatestByHash[conversationKey] = make(map[string]time.Time)
		}
		if record.Timestamp.After(conversationLatestByHash[conversationKey][record.ContentHash]) {
			conversationLatestByHash[conversationKey][record.ContentHash] = record.Timestamp
		}

		if len(view.Recent) < maxSendLedgerRecentRecords {
			view.Recent = append(view.Recent, assembler.ToSendRecordView(record))
		}
	}

	for botID, metrics := range view.RecordsByBot {
		metrics.UniqueConversations = len(botConversations[botID])
		metrics.UniqueContentHashes = len(botContentHashes[botID])
		view.RecordsByBot[botID] = metrics
	}

	for conversationKey, metrics := range view.RecordsByConversation {
		hashCounts := conversationContentHashes[conversationKey]
		metrics.UniqueContentHashes = len(hashCounts)
		for contentHash, count := range hashCounts {
			if count <= 1 {
				continue
			}
			metrics.RepeatedHashes++
			view.RepeatedContentHashes++
			if len(view.RepeatedHashes) < maxSendLedgerRepeatedHashes {
				view.RepeatedHashes = append(view.RepeatedHashes, query.SendLedgerRepeatedHashView{
					FromBotID:       metrics.FromBotID,
					ConversationID:  metrics.ConversationID,
					ContentHash:     contentHash,
					Count:           count,
					LatestTimestamp: formatSendLedgerMetricTime(conversationLatestByHash[conversationKey][contentHash]),
				})
			}
		}
		view.RecordsByConversation[conversationKey] = metrics
	}

	view.UniqueBots = len(bots)
	view.UniqueConversations = len(conversations)
	view.UniqueContentHashes = len(contentHashes)
	return view, nil
}

func sendLedgerConversationKey(fromBotID string, conversationID string) string {
	return fmt.Sprintf("%s/%s", fromBotID, conversationID)
}

func boundedSendLedgerMetricLimit(value int) int {
	if value <= 0 {
		return defaultSendLedgerMetricLimit
	}
	if value > maxSendLedgerMetricLimit {
		return maxSendLedgerMetricLimit
	}
	return value
}

func formatSendLedgerMetricTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func parseSendLedgerMetricTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
