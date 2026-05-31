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

func ToProactiveSeenView(record model.ProactiveSeenItemRecord, found bool, ttlHours int, sideEffect string) query.ProactiveSeenView {
	return query.ProactiveSeenView{
		Seen:       found,
		SourceKey:  record.SourceKey,
		ItemID:     record.ItemID,
		TTLHours:   ttlHours,
		SeenAt:     formatProactiveTime(record.SeenAt),
		SideEffect: sideEffect,
	}
}

func ToProactiveRejectionCooldownView(record model.ProactiveRejectionCooldownRecord, found bool, ttlHours int, sideEffect string) query.ProactiveRejectionCooldownView {
	return query.ProactiveRejectionCooldownView{
		Cooled:     found,
		SourceKey:  record.SourceKey,
		ItemID:     record.ItemID,
		TTLHours:   ttlHours,
		RejectedAt: formatProactiveTime(record.RejectedAt),
		SideEffect: sideEffect,
	}
}

func ToProactiveMarkItemsView(count int, timestamp time.Time, sideEffect string) query.ProactiveMarkItemsView {
	return query.ProactiveMarkItemsView{
		Count:      count,
		Timestamp:  formatProactiveTime(timestamp),
		SideEffect: sideEffect,
	}
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
