package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToSendRecordView(record model.SendRecord) query.SendRecordView {
	return query.SendRecordView{
		FromBotID:      record.FromBotID,
		ConversationID: record.ConversationID,
		ContentHash:    record.ContentHash,
		Timestamp:      formatSendLedgerTime(record.Timestamp),
	}
}

func ToSendRecordViews(items []model.SendRecord) []query.SendRecordView {
	views := make([]query.SendRecordView, 0, len(items))
	for _, item := range items {
		views = append(views, ToSendRecordView(item))
	}
	return views
}

func formatSendLedgerTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
