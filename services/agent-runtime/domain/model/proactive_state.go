package model

import (
	"errors"
	"strings"
	"time"
)

const (
	ProactiveSessionMarkContextOnlyLastAt = "context_only_last_at"
	ProactiveSessionMarkDriftLastAt       = "drift_last_at"
	ProactiveGlobalMarkBGContextMainAt    = "bg_context_last_main_at"

	ProactiveDriftSkillStatusIdle       = "idle"
	ProactiveDriftSkillStatusInProgress = "in_progress"
	ProactiveDriftMessageResultSent     = "sent"
	ProactiveDriftMessageResultSilent   = "silent"
)

type ProactiveDeliveryRecord struct {
	SessionKey  string
	DeliveryKey string
	SentAt      time.Time
}

func NewProactiveDeliveryRecord(sessionKey string, deliveryKey string, sentAt time.Time) (ProactiveDeliveryRecord, error) {
	if sentAt.IsZero() {
		sentAt = time.Now().UTC()
	}
	record := ProactiveDeliveryRecord{
		SessionKey:  strings.TrimSpace(sessionKey),
		DeliveryKey: strings.TrimSpace(deliveryKey),
		SentAt:      sentAt,
	}
	if err := record.Validate(); err != nil {
		return ProactiveDeliveryRecord{}, err
	}
	return record, nil
}

func (r ProactiveDeliveryRecord) Validate() error {
	if strings.TrimSpace(r.SessionKey) == "" {
		return errors.New("proactive delivery requires session key")
	}
	if strings.TrimSpace(r.DeliveryKey) == "" {
		return errors.New("proactive delivery requires delivery key")
	}
	if r.SentAt.IsZero() {
		return errors.New("proactive delivery requires sent_at")
	}
	return nil
}

func (r ProactiveDeliveryRecord) WithinWindow(now time.Time, window time.Duration) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if window <= 0 {
		return false
	}
	return !r.SentAt.Before(now.Add(-window))
}

type ProactiveSeenItemRecord struct {
	SourceKey string
	ItemID    string
	SeenAt    time.Time
}

func NewProactiveSeenItemRecord(sourceKey string, itemID string, seenAt time.Time) (ProactiveSeenItemRecord, error) {
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}
	record := ProactiveSeenItemRecord{
		SourceKey: NormalizeProactiveSourceKey(sourceKey),
		ItemID:    strings.TrimSpace(itemID),
		SeenAt:    seenAt.UTC(),
	}
	if err := record.Validate(); err != nil {
		return ProactiveSeenItemRecord{}, err
	}
	return record, nil
}

func (r ProactiveSeenItemRecord) Validate() error {
	if strings.TrimSpace(r.SourceKey) == "" {
		return errors.New("proactive seen item requires source key")
	}
	if strings.TrimSpace(r.ItemID) == "" {
		return errors.New("proactive seen item requires item id")
	}
	if r.SeenAt.IsZero() {
		return errors.New("proactive seen item requires seen_at")
	}
	return nil
}

func (r ProactiveSeenItemRecord) WithinWindow(now time.Time, window time.Duration) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if window <= 0 {
		return false
	}
	return !r.SeenAt.Before(now.Add(-window))
}

type ProactiveRejectionCooldownRecord struct {
	SourceKey  string
	ItemID     string
	RejectedAt time.Time
}

func NewProactiveRejectionCooldownRecord(sourceKey string, itemID string, rejectedAt time.Time) (ProactiveRejectionCooldownRecord, error) {
	if rejectedAt.IsZero() {
		rejectedAt = time.Now().UTC()
	}
	record := ProactiveRejectionCooldownRecord{
		SourceKey:  NormalizeProactiveSourceKey(sourceKey),
		ItemID:     strings.TrimSpace(itemID),
		RejectedAt: rejectedAt.UTC(),
	}
	if err := record.Validate(); err != nil {
		return ProactiveRejectionCooldownRecord{}, err
	}
	return record, nil
}

func (r ProactiveRejectionCooldownRecord) Validate() error {
	if strings.TrimSpace(r.SourceKey) == "" {
		return errors.New("proactive rejection cooldown requires source key")
	}
	if strings.TrimSpace(r.ItemID) == "" {
		return errors.New("proactive rejection cooldown requires item id")
	}
	if r.RejectedAt.IsZero() {
		return errors.New("proactive rejection cooldown requires rejected_at")
	}
	return nil
}

func (r ProactiveRejectionCooldownRecord) WithinWindow(now time.Time, window time.Duration) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if window <= 0 {
		return false
	}
	return !r.RejectedAt.Before(now.Add(-window))
}

type ProactiveStateRetentionCutoffs struct {
	DeliveriesBefore         time.Time
	SeenItemsBefore          time.Time
	ContextOnlyBefore        time.Time
	RejectionCooldownsBefore time.Time
}

type ProactiveStateCleanupResult struct {
	RemovedDeliveries         int
	RemovedSeenItems          int
	RemovedContextOnly        int
	RemovedRejectionCooldowns int
}

type ProactiveContextOnlyRecord struct {
	SessionKey string
	SentAt     time.Time
}

func NewProactiveContextOnlyRecord(sessionKey string, sentAt time.Time) (ProactiveContextOnlyRecord, error) {
	if sentAt.IsZero() {
		sentAt = time.Now().UTC()
	}
	record := ProactiveContextOnlyRecord{
		SessionKey: strings.TrimSpace(sessionKey),
		SentAt:     sentAt,
	}
	if err := record.Validate(); err != nil {
		return ProactiveContextOnlyRecord{}, err
	}
	return record, nil
}

func (r ProactiveContextOnlyRecord) Validate() error {
	if strings.TrimSpace(r.SessionKey) == "" {
		return errors.New("proactive context-only record requires session key")
	}
	if r.SentAt.IsZero() {
		return errors.New("proactive context-only record requires sent_at")
	}
	return nil
}

type ProactiveSessionMark struct {
	SessionKey string
	Key        string
	MarkedAt   time.Time
}

func NewProactiveSessionMark(sessionKey string, key string, markedAt time.Time) (ProactiveSessionMark, error) {
	if markedAt.IsZero() {
		markedAt = time.Now().UTC()
	}
	mark := ProactiveSessionMark{
		SessionKey: strings.TrimSpace(sessionKey),
		Key:        strings.TrimSpace(key),
		MarkedAt:   markedAt,
	}
	if err := mark.Validate(); err != nil {
		return ProactiveSessionMark{}, err
	}
	return mark, nil
}

func (m ProactiveSessionMark) Validate() error {
	if strings.TrimSpace(m.SessionKey) == "" {
		return errors.New("proactive session mark requires session key")
	}
	switch strings.TrimSpace(m.Key) {
	case ProactiveSessionMarkContextOnlyLastAt, ProactiveSessionMarkDriftLastAt:
	default:
		return errors.New("proactive session mark has invalid key")
	}
	if m.MarkedAt.IsZero() {
		return errors.New("proactive session mark requires marked_at")
	}
	return nil
}

type ProactiveGlobalMark struct {
	Key      string
	MarkedAt time.Time
}

func NewProactiveGlobalMark(key string, markedAt time.Time) (ProactiveGlobalMark, error) {
	if markedAt.IsZero() {
		markedAt = time.Now().UTC()
	}
	mark := ProactiveGlobalMark{
		Key:      strings.TrimSpace(key),
		MarkedAt: markedAt.UTC(),
	}
	if err := mark.Validate(); err != nil {
		return ProactiveGlobalMark{}, err
	}
	return mark, nil
}

func (m ProactiveGlobalMark) Validate() error {
	switch strings.TrimSpace(m.Key) {
	case ProactiveGlobalMarkBGContextMainAt:
	default:
		return errors.New("proactive global mark has invalid key")
	}
	if m.MarkedAt.IsZero() {
		return errors.New("proactive global mark requires marked_at")
	}
	return nil
}

type ProactiveAnyActionQuota struct {
	QuotaKey     string
	WindowKey    string
	NextResetAt  time.Time
	Used         int
	LastActionAt time.Time
}

func NewProactiveAnyActionQuota(quotaKey string, windowKey string, nextResetAt time.Time, used int, lastActionAt time.Time) (ProactiveAnyActionQuota, error) {
	quota := ProactiveAnyActionQuota{
		QuotaKey:     proactiveQuotaKeyOrDefault(quotaKey),
		WindowKey:    strings.TrimSpace(windowKey),
		NextResetAt:  nextResetAt.UTC(),
		Used:         used,
		LastActionAt: lastActionAt.UTC(),
	}
	if err := quota.Validate(); err != nil {
		return ProactiveAnyActionQuota{}, err
	}
	return quota, nil
}

func (q ProactiveAnyActionQuota) Validate() error {
	if strings.TrimSpace(q.QuotaKey) == "" {
		return errors.New("proactive anyaction quota requires quota key")
	}
	if strings.TrimSpace(q.WindowKey) == "" {
		return errors.New("proactive anyaction quota requires window key")
	}
	if q.NextResetAt.IsZero() {
		return errors.New("proactive anyaction quota requires next_reset_at")
	}
	if q.Used < 0 {
		return errors.New("proactive anyaction quota requires non-negative used")
	}
	return nil
}

func (q ProactiveAnyActionQuota) WithAction(timestamp time.Time) (ProactiveAnyActionQuota, error) {
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	next := q
	next.Used++
	next.LastActionAt = timestamp.UTC()
	return next, next.Validate()
}

func NewProactiveAnyActionQuotaForWindow(quotaKey string, windowKey string, nextResetAt time.Time, lastActionAt time.Time) (ProactiveAnyActionQuota, error) {
	return NewProactiveAnyActionQuota(quotaKey, windowKey, nextResetAt, 0, lastActionAt)
}

func proactiveQuotaKeyOrDefault(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	return value
}

type ProactiveDriftSkillState struct {
	SkillName string
	LastRunAt time.Time
	RunCount  int
	Status    string
	Next      string
}

func NewProactiveDriftSkillState(skillName string, lastRunAt time.Time, runCount int, status string, next string) (ProactiveDriftSkillState, error) {
	state := ProactiveDriftSkillState{
		SkillName: strings.TrimSpace(skillName),
		LastRunAt: lastRunAt.UTC(),
		RunCount:  runCount,
		Status:    normalizeProactiveDriftSkillStatus(status),
		Next:      strings.TrimSpace(next),
	}
	if err := state.Validate(); err != nil {
		return ProactiveDriftSkillState{}, err
	}
	return state, nil
}

func (s ProactiveDriftSkillState) Validate() error {
	if strings.TrimSpace(s.SkillName) == "" {
		return errors.New("proactive drift skill state requires skill_name")
	}
	if s.RunCount < 0 {
		return errors.New("proactive drift skill state requires non-negative run_count")
	}
	switch strings.TrimSpace(s.Status) {
	case ProactiveDriftSkillStatusIdle, ProactiveDriftSkillStatusInProgress:
	default:
		return errors.New("proactive drift skill state has invalid status")
	}
	return nil
}

func (s ProactiveDriftSkillState) WithFinish(runAt time.Time, next string) (ProactiveDriftSkillState, error) {
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}
	return NewProactiveDriftSkillState(
		s.SkillName,
		runAt,
		max(0, s.RunCount)+1,
		ProactiveDriftSkillStatusInProgress,
		next,
	)
}

type ProactiveDriftRecentRun struct {
	SkillName     string
	RunAt         time.Time
	OneLine       string
	MessageResult string
}

func NewProactiveDriftRecentRun(skillName string, runAt time.Time, oneLine string, messageResult string) (ProactiveDriftRecentRun, error) {
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}
	run := ProactiveDriftRecentRun{
		SkillName:     strings.TrimSpace(skillName),
		RunAt:         runAt.UTC(),
		OneLine:       strings.TrimSpace(oneLine),
		MessageResult: normalizeProactiveDriftMessageResult(messageResult),
	}
	if err := run.Validate(); err != nil {
		return ProactiveDriftRecentRun{}, err
	}
	return run, nil
}

func (r ProactiveDriftRecentRun) Validate() error {
	if strings.TrimSpace(r.SkillName) == "" {
		return errors.New("proactive drift recent run requires skill_name")
	}
	if r.RunAt.IsZero() {
		return errors.New("proactive drift recent run requires run_at")
	}
	if strings.TrimSpace(r.OneLine) == "" {
		return errors.New("proactive drift recent run requires one_line")
	}
	switch strings.TrimSpace(r.MessageResult) {
	case ProactiveDriftMessageResultSent, ProactiveDriftMessageResultSilent:
	default:
		return errors.New("proactive drift recent run has invalid message_result")
	}
	return nil
}

func normalizeProactiveDriftSkillStatus(value string) string {
	value = strings.TrimSpace(value)
	if value == ProactiveDriftSkillStatusInProgress {
		return value
	}
	return ProactiveDriftSkillStatusIdle
}

func normalizeProactiveDriftMessageResult(value string) string {
	value = strings.TrimSpace(value)
	if value == ProactiveDriftMessageResultSent {
		return value
	}
	return ProactiveDriftMessageResultSilent
}

func NormalizeProactiveSourceKey(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "mcp:") {
		return value
	}
	parts := strings.SplitN(value, ":", 3)
	if len(parts) < 2 {
		return value
	}
	return strings.Join(parts[:2], ":")
}
