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

func ToProactiveCleanupView(result model.ProactiveStateCleanupResult, timestamp time.Time, sideEffect string) query.ProactiveCleanupView {
	return query.ProactiveCleanupView{
		RemovedDeliveries:         result.RemovedDeliveries,
		RemovedSeenItems:          result.RemovedSeenItems,
		RemovedContextOnly:        result.RemovedContextOnly,
		RemovedRejectionCooldowns: result.RemovedRejectionCooldowns,
		Timestamp:                 formatProactiveTime(timestamp),
		SideEffect:                sideEffect,
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

func ToProactiveGlobalTimestampView(key string, timestamp time.Time, found bool) query.ProactiveTimestampView {
	return query.ProactiveTimestampView{
		Key:       key,
		Timestamp: formatProactiveTime(timestamp),
		Found:     found,
	}
}

func ToProactiveDriftSkillStateView(state model.ProactiveDriftSkillState, found bool, sideEffect string) query.ProactiveDriftSkillStateView {
	return query.ProactiveDriftSkillStateView{
		SkillName:  state.SkillName,
		LastRunAt:  formatProactiveTime(state.LastRunAt),
		RunCount:   state.RunCount,
		Status:     state.Status,
		Next:       state.Next,
		Found:      found,
		SideEffect: sideEffect,
	}
}

func ToProactiveDriftRecentRunView(run model.ProactiveDriftRecentRun) query.ProactiveDriftRecentRunView {
	return query.ProactiveDriftRecentRunView{
		SkillName:     run.SkillName,
		RunAt:         formatProactiveTime(run.RunAt),
		OneLine:       run.OneLine,
		MessageResult: run.MessageResult,
	}
}

func ToProactiveDriftRecentRunViews(items []model.ProactiveDriftRecentRun) []query.ProactiveDriftRecentRunView {
	views := make([]query.ProactiveDriftRecentRunView, 0, len(items))
	for _, item := range items {
		views = append(views, ToProactiveDriftRecentRunView(item))
	}
	return views
}

func ToProactiveDriftSummaryView(items []model.ProactiveDriftRecentRun, note string, sideEffect string) query.ProactiveDriftSummaryView {
	return query.ProactiveDriftSummaryView{
		Version:    1,
		RecentRuns: ToProactiveDriftRecentRunViews(items),
		Note:       note,
		SideEffect: sideEffect,
	}
}

func ToProactiveDriftFinishView(state model.ProactiveDriftSkillState, run model.ProactiveDriftRecentRun, note string, sideEffect string) query.ProactiveDriftFinishView {
	return query.ProactiveDriftFinishView{
		SkillState: ToProactiveDriftSkillStateView(state, true, sideEffect),
		RecentRun:  ToProactiveDriftRecentRunView(run),
		Note:       note,
		SideEffect: sideEffect,
	}
}

func ToProactiveTickLogView(log model.ProactiveTickLog, found bool, sideEffect string) query.ProactiveTickLogView {
	return query.ProactiveTickLogView{
		TickID:         log.TickID,
		SessionKey:     log.SessionKey,
		StartedAt:      formatProactiveTime(log.StartedAt),
		FinishedAt:     formatProactiveTime(log.FinishedAt),
		GateExit:       log.GateExit,
		TerminalAction: log.TerminalAction,
		SkipReason:     log.SkipReason,
		StepsTaken:     log.StepsTaken,
		AlertCount:     log.AlertCount,
		ContentCount:   log.ContentCount,
		ContextCount:   log.ContextCount,
		InterestingIDs: append([]string(nil), log.InterestingIDs...),
		DiscardedIDs:   append([]string(nil), log.DiscardedIDs...),
		CitedIDs:       append([]string(nil), log.CitedIDs...),
		DriftEntered:   log.DriftEntered,
		FinalMessage:   log.FinalMessage,
		Found:          found,
		SideEffect:     sideEffect,
	}
}

func ToProactiveTickLogViews(items []model.ProactiveTickLog) []query.ProactiveTickLogView {
	views := make([]query.ProactiveTickLogView, 0, len(items))
	for _, item := range items {
		views = append(views, ToProactiveTickLogView(item, true, "none"))
	}
	return views
}

func ToProactiveTickStepLogView(step model.ProactiveTickStepLog, sideEffect string) query.ProactiveTickStepLogView {
	toolArgs := map[string]any{}
	for key, value := range step.ToolArgs {
		toolArgs[key] = value
	}
	return query.ProactiveTickStepLogView{
		TickID:              step.TickID,
		StepIndex:           step.StepIndex,
		Phase:               step.Phase,
		ToolName:            step.ToolName,
		ToolCallID:          step.ToolCallID,
		ToolArgs:            toolArgs,
		ToolResultText:      step.ToolResultText,
		TerminalActionAfter: step.TerminalActionAfter,
		SkipReasonAfter:     step.SkipReasonAfter,
		InterestingIDsAfter: append([]string(nil), step.InterestingIDsAfter...),
		DiscardedIDsAfter:   append([]string(nil), step.DiscardedIDsAfter...),
		CitedIDsAfter:       append([]string(nil), step.CitedIDsAfter...),
		FinalMessageAfter:   step.FinalMessageAfter,
		SideEffect:          sideEffect,
	}
}

func ToProactiveTickStepLogViews(items []model.ProactiveTickStepLog) []query.ProactiveTickStepLogView {
	views := make([]query.ProactiveTickStepLogView, 0, len(items))
	for _, item := range items {
		views = append(views, ToProactiveTickStepLogView(item, "none"))
	}
	return views
}

func ToProactiveTickLogListView(items []model.ProactiveTickLog, total int, sideEffect string) query.ProactiveTickLogListView {
	return query.ProactiveTickLogListView{
		Items:      ToProactiveTickLogViews(items),
		Total:      total,
		SideEffect: sideEffect,
	}
}

func ToProactiveTickStepLogListView(items []model.ProactiveTickStepLog, sideEffect string) query.ProactiveTickStepLogListView {
	return query.ProactiveTickStepLogListView{
		Items:      ToProactiveTickStepLogViews(items),
		Total:      len(items),
		SideEffect: sideEffect,
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
