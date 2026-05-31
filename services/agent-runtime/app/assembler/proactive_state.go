package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToProactiveDeliveryView(record model.ProactiveDeliveryRecord) query.ProactiveDeliveryView {
	return query.ProactiveDeliveryView{
		SessionKey:  record.SessionKey,
		DeliveryKey: record.DeliveryKey,
		SentAt:      formatProactiveTime(record.SentAt),
	}
}

func ToProactiveDeliveryViews(items []model.ProactiveDeliveryRecord) []query.ProactiveDeliveryView {
	views := make([]query.ProactiveDeliveryView, 0, len(items))
	for _, item := range items {
		views = append(views, ToProactiveDeliveryView(item))
	}
	return views
}

func ToProactiveTimestampView(sessionKey string, key string, timestamp time.Time, found bool) query.ProactiveTimestampView {
	return query.ProactiveTimestampView{
		SessionKey: sessionKey,
		Key:        key,
		Timestamp:  formatProactiveTime(timestamp),
		Found:      found,
	}
}

func ToProactiveAnyActionQuotaView(record model.ProactiveAnyActionQuota, found bool, sideEffect string) query.ProactiveAnyActionQuotaView {
	return query.ProactiveAnyActionQuotaView{
		QuotaKey:     record.QuotaKey,
		WindowKey:    record.WindowKey,
		NextResetAt:  formatProactiveTime(record.NextResetAt),
		Used:         record.Used,
		LastActionAt: formatProactiveTime(record.LastActionAt),
		Found:        found,
		SideEffect:   sideEffect,
	}
}

func formatProactiveTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
