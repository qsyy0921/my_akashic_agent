package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ProactiveStateService struct {
	repository outport.ProactiveStateRepository
}

func NewProactiveStateService(repository outport.ProactiveStateRepository) *ProactiveStateService {
	return &ProactiveStateService{repository: repository}
}

func (s *ProactiveStateService) RecordDelivery(ctx context.Context, cmd command.RecordProactiveDeliveryCommand) (query.ProactiveDeliveryView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveDeliveryView{}, errors.New("proactive state service requires repository")
	}
	record, err := model.NewProactiveDeliveryRecord(cmd.SessionKey, cmd.DeliveryKey, timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveDeliveryView{}, err
	}
	if err := s.repository.SaveProactiveDelivery(ctx, record); err != nil {
		return query.ProactiveDeliveryView{}, err
	}
	return assembler.ToProactiveDeliveryView(record), nil
}

func (s *ProactiveStateService) IsDeliveryDuplicate(ctx context.Context, cmd command.CheckProactiveDeliveryDuplicateCommand) (query.ProactiveDuplicateView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveDuplicateView{}, errors.New("proactive state service requires repository")
	}
	sessionKey := strings.TrimSpace(cmd.SessionKey)
	deliveryKey := strings.TrimSpace(cmd.DeliveryKey)
	if sessionKey == "" {
		return query.ProactiveDuplicateView{}, errors.New("session key required")
	}
	if deliveryKey == "" {
		return query.ProactiveDuplicateView{}, errors.New("delivery key required")
	}
	windowHours := positiveHoursOrDefault(cmd.WindowHours, 24)
	record, ok, err := s.repository.FindProactiveDelivery(ctx, sessionKey, deliveryKey)
	if err != nil {
		return query.ProactiveDuplicateView{}, err
	}
	duplicate := ok && record.WithinWindow(timestampOrNow(cmd.Timestamp), time.Duration(windowHours)*time.Hour)
	return query.ProactiveDuplicateView{
		Duplicate:   duplicate,
		SessionKey:  sessionKey,
		DeliveryKey: deliveryKey,
		WindowHours: windowHours,
	}, nil
}

func (s *ProactiveStateService) CountDeliveries(ctx context.Context, cmd command.CountProactiveDeliveriesCommand) (query.ProactiveCountView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveCountView{}, errors.New("proactive state service requires repository")
	}
	sessionKey := strings.TrimSpace(cmd.SessionKey)
	if sessionKey == "" {
		return query.ProactiveCountView{}, errors.New("session key required")
	}
	windowHours := positiveHoursOrDefault(cmd.WindowHours, 24)
	since := timestampOrNow(cmd.Timestamp).Add(-time.Duration(windowHours) * time.Hour)
	count, err := s.repository.CountProactiveDeliveriesSince(ctx, sessionKey, since)
	if err != nil {
		return query.ProactiveCountView{}, err
	}
	return query.ProactiveCountView{Count: count, SessionKey: sessionKey, WindowHours: windowHours}, nil
}

func (s *ProactiveStateService) ListDeliveries(ctx context.Context, filter query.ProactiveDeliveryFilter) ([]query.ProactiveDeliveryView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("proactive state service requires repository")
	}
	items, err := s.repository.ListProactiveDeliveries(ctx, filter)
	if err != nil {
		return nil, err
	}
	return assembler.ToProactiveDeliveryViews(items), nil
}

func (s *ProactiveStateService) IsItemSeen(ctx context.Context, cmd command.CheckProactiveItemSeenCommand) (query.ProactiveSeenView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveSeenView{}, errors.New("proactive state service requires repository")
	}
	sourceKey := model.NormalizeProactiveSourceKey(cmd.SourceKey)
	itemID := strings.TrimSpace(cmd.ItemID)
	if sourceKey == "" {
		return query.ProactiveSeenView{}, errors.New("source key required")
	}
	if itemID == "" {
		return query.ProactiveSeenView{}, errors.New("item id required")
	}
	ttlHours := positiveHoursOrDefault(cmd.TTLHours, 24)
	record, ok, err := s.repository.FindProactiveSeenItem(ctx, sourceKey, itemID)
	if err != nil {
		return query.ProactiveSeenView{}, err
	}
	seen := ok && record.WithinWindow(timestampOrNow(cmd.Timestamp), time.Duration(ttlHours)*time.Hour)
	if !ok {
		record = model.ProactiveSeenItemRecord{SourceKey: sourceKey, ItemID: itemID}
	}
	return assembler.ToProactiveSeenView(record, seen, ttlHours, "none"), nil
}

func (s *ProactiveStateService) MarkItemsSeen(ctx context.Context, cmd command.MarkProactiveItemsSeenCommand) (query.ProactiveMarkItemsView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveMarkItemsView{}, errors.New("proactive state service requires repository")
	}
	timestamp := timestampOrNow(cmd.Timestamp)
	records := make([]model.ProactiveSeenItemRecord, 0, len(cmd.Entries))
	for _, entry := range cmd.Entries {
		record, err := model.NewProactiveSeenItemRecord(entry.SourceKey, entry.ItemID, timestamp)
		if err != nil {
			return query.ProactiveMarkItemsView{}, err
		}
		records = append(records, record)
	}
	if len(records) == 0 {
		return assembler.ToProactiveMarkItemsView(0, timestamp, "runtime_state_write"), nil
	}
	if err := s.repository.SaveProactiveSeenItems(ctx, records); err != nil {
		return query.ProactiveMarkItemsView{}, err
	}
	return assembler.ToProactiveMarkItemsView(len(records), timestamp, "runtime_state_write"), nil
}

func (s *ProactiveStateService) IsRejectionCooled(ctx context.Context, cmd command.CheckProactiveRejectionCooldownCommand) (query.ProactiveRejectionCooldownView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveRejectionCooldownView{}, errors.New("proactive state service requires repository")
	}
	sourceKey := model.NormalizeProactiveSourceKey(cmd.SourceKey)
	itemID := strings.TrimSpace(cmd.ItemID)
	if sourceKey == "" {
		return query.ProactiveRejectionCooldownView{}, errors.New("source key required")
	}
	if itemID == "" {
		return query.ProactiveRejectionCooldownView{}, errors.New("item id required")
	}
	ttlHours := cmd.TTLHours
	if ttlHours <= 0 {
		record := model.ProactiveRejectionCooldownRecord{SourceKey: sourceKey, ItemID: itemID}
		return assembler.ToProactiveRejectionCooldownView(record, false, ttlHours, "none"), nil
	}
	record, ok, err := s.repository.FindProactiveRejectionCooldown(ctx, sourceKey, itemID)
	if err != nil {
		return query.ProactiveRejectionCooldownView{}, err
	}
	cooled := ok && record.WithinWindow(timestampOrNow(cmd.Timestamp), time.Duration(ttlHours)*time.Hour)
	if !ok {
		record = model.ProactiveRejectionCooldownRecord{SourceKey: sourceKey, ItemID: itemID}
	}
	return assembler.ToProactiveRejectionCooldownView(record, cooled, ttlHours, "none"), nil
}

func (s *ProactiveStateService) MarkRejectionCooldown(ctx context.Context, cmd command.MarkProactiveRejectionCooldownCommand) (query.ProactiveMarkItemsView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveMarkItemsView{}, errors.New("proactive state service requires repository")
	}
	timestamp := timestampOrNow(cmd.Timestamp)
	if cmd.Hours <= 0 || len(cmd.Entries) == 0 {
		return assembler.ToProactiveMarkItemsView(0, timestamp, "none"), nil
	}
	records := make([]model.ProactiveRejectionCooldownRecord, 0, len(cmd.Entries))
	for _, entry := range cmd.Entries {
		record, err := model.NewProactiveRejectionCooldownRecord(entry.SourceKey, entry.ItemID, timestamp)
		if err != nil {
			return query.ProactiveMarkItemsView{}, err
		}
		records = append(records, record)
	}
	if err := s.repository.SaveProactiveRejectionCooldowns(ctx, records); err != nil {
		return query.ProactiveMarkItemsView{}, err
	}
	return assembler.ToProactiveMarkItemsView(len(records), timestamp, "runtime_state_write"), nil
}

func (s *ProactiveStateService) RecordContextOnly(ctx context.Context, cmd command.RecordProactiveContextOnlyCommand) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	record, err := model.NewProactiveContextOnlyRecord(cmd.SessionKey, timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if err := s.repository.SaveProactiveContextOnly(ctx, record); err != nil {
		return query.ProactiveTimestampView{}, err
	}
	mark, err := model.NewProactiveSessionMark(record.SessionKey, model.ProactiveSessionMarkContextOnlyLastAt, record.SentAt)
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if err := s.repository.SaveProactiveSessionMark(ctx, mark); err != nil {
		return query.ProactiveTimestampView{}, err
	}
	return assembler.ToProactiveTimestampView(record.SessionKey, model.ProactiveSessionMarkContextOnlyLastAt, record.SentAt, true), nil
}

func (s *ProactiveStateService) LastContextOnly(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error) {
	return s.lastSessionMark(ctx, sessionKey, model.ProactiveSessionMarkContextOnlyLastAt)
}

func (s *ProactiveStateService) CountContextOnly(ctx context.Context, cmd command.CountProactiveContextOnlyCommand) (query.ProactiveCountView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveCountView{}, errors.New("proactive state service requires repository")
	}
	sessionKey := strings.TrimSpace(cmd.SessionKey)
	if sessionKey == "" {
		return query.ProactiveCountView{}, errors.New("session key required")
	}
	windowHours := positiveHoursOrDefault(cmd.WindowHours, 24)
	since := timestampOrNow(cmd.Timestamp).Add(-time.Duration(windowHours) * time.Hour)
	count, err := s.repository.CountProactiveContextOnlySince(ctx, sessionKey, since)
	if err != nil {
		return query.ProactiveCountView{}, err
	}
	return query.ProactiveCountView{Count: count, SessionKey: sessionKey, WindowHours: windowHours}, nil
}

func (s *ProactiveStateService) RecordDriftRun(ctx context.Context, cmd command.RecordProactiveDriftRunCommand) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	mark, err := model.NewProactiveSessionMark(cmd.SessionKey, model.ProactiveSessionMarkDriftLastAt, timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if err := s.repository.SaveProactiveSessionMark(ctx, mark); err != nil {
		return query.ProactiveTimestampView{}, err
	}
	return assembler.ToProactiveTimestampView(mark.SessionKey, mark.Key, mark.MarkedAt, true), nil
}

func (s *ProactiveStateService) LastDriftRun(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error) {
	return s.lastSessionMark(ctx, sessionKey, model.ProactiveSessionMarkDriftLastAt)
}

func (s *ProactiveStateService) RecordDriftFinish(ctx context.Context, cmd command.RecordProactiveDriftFinishCommand) (query.ProactiveDriftFinishView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveDriftFinishView{}, errors.New("proactive state service requires repository")
	}
	skillName := clipProactiveText(cmd.SkillUsed, 80)
	if skillName == "" {
		return query.ProactiveDriftFinishView{}, errors.New("skill_used required")
	}
	timestamp := timestampOrNow(cmd.Timestamp)
	existing, found, err := s.repository.FindProactiveDriftSkillState(ctx, skillName)
	if err != nil {
		return query.ProactiveDriftFinishView{}, err
	}
	if !found {
		existing, err = model.NewProactiveDriftSkillState(skillName, time.Time{}, 0, model.ProactiveDriftSkillStatusIdle, "")
		if err != nil {
			return query.ProactiveDriftFinishView{}, err
		}
	}
	state, err := existing.WithFinish(timestamp, clipProactiveText(cmd.Next, 100))
	if err != nil {
		return query.ProactiveDriftFinishView{}, err
	}
	run, err := model.NewProactiveDriftRecentRun(
		skillName,
		timestamp,
		clipProactiveText(cmd.OneLine, 150),
		cmd.MessageResult,
	)
	if err != nil {
		return query.ProactiveDriftFinishView{}, err
	}
	note := clipProactiveText(cmd.Note, 150)
	if note == "" {
		_, existingNote, err := s.repository.ListProactiveDriftRecentRuns(ctx, 1)
		if err != nil {
			return query.ProactiveDriftFinishView{}, err
		}
		note = existingNote
	}
	if err := s.repository.SaveProactiveDriftFinish(ctx, state, run, note, 10); err != nil {
		return query.ProactiveDriftFinishView{}, err
	}
	return assembler.ToProactiveDriftFinishView(state, run, note, "runtime_state_write"), nil
}

func (s *ProactiveStateService) DriftSummary(ctx context.Context, limit int) (query.ProactiveDriftSummaryView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveDriftSummaryView{}, errors.New("proactive state service requires repository")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	items, note, err := s.repository.ListProactiveDriftRecentRuns(ctx, limit)
	if err != nil {
		return query.ProactiveDriftSummaryView{}, err
	}
	return assembler.ToProactiveDriftSummaryView(items, note, "none"), nil
}

func (s *ProactiveStateService) DriftSkillState(ctx context.Context, skillName string) (query.ProactiveDriftSkillStateView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveDriftSkillStateView{}, errors.New("proactive state service requires repository")
	}
	skillName = clipProactiveText(skillName, 80)
	if skillName == "" {
		return query.ProactiveDriftSkillStateView{}, errors.New("skill_name required")
	}
	state, found, err := s.repository.FindProactiveDriftSkillState(ctx, skillName)
	if err != nil {
		return query.ProactiveDriftSkillStateView{}, err
	}
	if !found {
		state, err = model.NewProactiveDriftSkillState(skillName, time.Time{}, 0, model.ProactiveDriftSkillStatusIdle, "")
		if err != nil {
			return query.ProactiveDriftSkillStateView{}, err
		}
	}
	return assembler.ToProactiveDriftSkillStateView(state, found, "none"), nil
}

func (s *ProactiveStateService) RecordBGContextMain(ctx context.Context, cmd command.RecordProactiveBGContextMainCommand) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	mark, err := model.NewProactiveGlobalMark(model.ProactiveGlobalMarkBGContextMainAt, timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if err := s.repository.SaveProactiveGlobalMark(ctx, mark); err != nil {
		return query.ProactiveTimestampView{}, err
	}
	return assembler.ToProactiveGlobalTimestampView(mark.Key, mark.MarkedAt, true), nil
}

func (s *ProactiveStateService) LastBGContextMain(ctx context.Context) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	mark, ok, err := s.repository.FindProactiveGlobalMark(ctx, model.ProactiveGlobalMarkBGContextMainAt)
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if !ok {
		return assembler.ToProactiveGlobalTimestampView(model.ProactiveGlobalMarkBGContextMainAt, time.Time{}, false), nil
	}
	return assembler.ToProactiveGlobalTimestampView(mark.Key, mark.MarkedAt, true), nil
}

func (s *ProactiveStateService) SnapshotAnyActionQuota(ctx context.Context, cmd command.SnapshotProactiveAnyActionQuotaCommand) (query.ProactiveAnyActionQuotaView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveAnyActionQuotaView{}, errors.New("proactive state service requires repository")
	}
	quotaKey := proactiveQuotaKey(cmd.QuotaKey)
	now := timestampOrNow(cmd.Timestamp)
	windowKey, nextResetAt, err := proactiveAnyActionWindow(now, cmd.ResetHour, cmd.Timezone)
	if err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	existing, found, err := s.repository.FindProactiveAnyActionQuota(ctx, quotaKey)
	if err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	if found && existing.WindowKey == windowKey {
		return assembler.ToProactiveAnyActionQuotaView(existing, true, "runtime_state_read_rollover_if_needed"), nil
	}
	lastActionAt := time.Time{}
	if found {
		lastActionAt = existing.LastActionAt
	}
	quota, err := model.NewProactiveAnyActionQuotaForWindow(quotaKey, windowKey, nextResetAt, lastActionAt)
	if err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	if err := s.repository.SaveProactiveAnyActionQuota(ctx, quota); err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	return assembler.ToProactiveAnyActionQuotaView(quota, found, "runtime_state_rollover"), nil
}

func (s *ProactiveStateService) RecordAnyAction(ctx context.Context, cmd command.RecordProactiveAnyActionCommand) (query.ProactiveAnyActionQuotaView, error) {
	if _, err := s.SnapshotAnyActionQuota(ctx, command.SnapshotProactiveAnyActionQuotaCommand{
		QuotaKey:  cmd.QuotaKey,
		ResetHour: cmd.ResetHour,
		Timezone:  cmd.Timezone,
		Timestamp: cmd.Timestamp,
	}); err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	quota, found, err := s.repository.FindProactiveAnyActionQuota(ctx, proactiveQuotaKey(cmd.QuotaKey))
	if err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	if !found {
		return query.ProactiveAnyActionQuotaView{}, errors.New("proactive anyaction quota not found after snapshot")
	}
	updated, err := quota.WithAction(timestampOrNow(cmd.Timestamp))
	if err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	if err := s.repository.SaveProactiveAnyActionQuota(ctx, updated); err != nil {
		return query.ProactiveAnyActionQuotaView{}, err
	}
	return assembler.ToProactiveAnyActionQuotaView(updated, true, "runtime_state_write"), nil
}

func (s *ProactiveStateService) Cleanup(ctx context.Context, cmd command.CleanupProactiveStateCommand) (query.ProactiveCleanupView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveCleanupView{}, errors.New("proactive state service requires repository")
	}
	timestamp := timestampOrNow(cmd.Timestamp)
	cutoffs := model.ProactiveStateRetentionCutoffs{
		DeliveriesBefore:  timestamp.Add(-time.Duration(positiveHoursOrDefault(cmd.DeliveryTTLHours, 24)) * time.Hour),
		SeenItemsBefore:   timestamp.Add(-time.Duration(positiveHoursOrDefault(cmd.SeenTTLHours, 24)) * time.Hour),
		ContextOnlyBefore: timestamp.Add(-time.Duration(positiveHoursOrDefault(cmd.ContextOnlyTTLHours, 24)) * time.Hour),
	}
	if cmd.RejectionCooldownTTLHours > 0 {
		cutoffs.RejectionCooldownsBefore = timestamp.Add(-time.Duration(cmd.RejectionCooldownTTLHours) * time.Hour)
	}
	result, err := s.repository.CleanupProactiveState(ctx, cutoffs)
	if err != nil {
		return query.ProactiveCleanupView{}, err
	}
	return assembler.ToProactiveCleanupView(result, timestamp, "runtime_state_cleanup"), nil
}

func (s *ProactiveStateService) lastSessionMark(ctx context.Context, sessionKey string, key string) (query.ProactiveTimestampView, error) {
	if s == nil || s.repository == nil {
		return query.ProactiveTimestampView{}, errors.New("proactive state service requires repository")
	}
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return query.ProactiveTimestampView{}, errors.New("session key required")
	}
	mark, ok, err := s.repository.FindProactiveSessionMark(ctx, sessionKey, key)
	if err != nil {
		return query.ProactiveTimestampView{}, err
	}
	if !ok {
		return assembler.ToProactiveTimestampView(sessionKey, key, time.Time{}, false), nil
	}
	return assembler.ToProactiveTimestampView(mark.SessionKey, mark.Key, mark.MarkedAt, true), nil
}

func timestampOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

func positiveHoursOrDefault(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func proactiveQuotaKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	return value
}

func clipProactiveText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func proactiveAnyActionWindow(now time.Time, resetHour int, timezoneName string) (string, time.Time, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if resetHour < 0 || resetHour > 23 {
		return "", time.Time{}, errors.New("reset_hour must be between 0 and 23")
	}
	timezoneName = strings.TrimSpace(timezoneName)
	if timezoneName == "" {
		timezoneName = "Asia/Shanghai"
	}
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return "", time.Time{}, err
	}
	localNow := now.In(location)
	resetToday := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), resetHour, 0, 0, 0, location)
	var start time.Time
	var nextReset time.Time
	if !localNow.Before(resetToday) {
		start = resetToday
		nextReset = resetToday.AddDate(0, 0, 1)
	} else {
		start = resetToday.AddDate(0, 0, -1)
		nextReset = resetToday
	}
	windowKey := fmt.Sprintf("%s@%02d@%s", start.Format("2006-01-02"), resetHour, timezoneName)
	return windowKey, nextReset.UTC(), nil
}
