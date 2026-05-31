package proactivestate

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const storeVersion = "2026-05-30.proactivestate.v1"

// Store persists deterministic proactive scheduling state for local runtime use.
type Store struct {
	mu            sync.Mutex
	path          string
	deliveries    map[string]model.ProactiveDeliveryRecord
	deliveryOrder []string
	seenItems     map[string]model.ProactiveSeenItemRecord
	rejections    map[string]model.ProactiveRejectionCooldownRecord
	contextOnly   []model.ProactiveContextOnlyRecord
	sessionMarks  map[string]model.ProactiveSessionMark
	globalMarks   map[string]model.ProactiveGlobalMark
	anyAction     map[string]model.ProactiveAnyActionQuota
	driftSkills   map[string]model.ProactiveDriftSkillState
	driftRuns     []model.ProactiveDriftRecentRun
	driftNote     string
}

type persistedState struct {
	Version      string                                   `json:"version"`
	Deliveries   []model.ProactiveDeliveryRecord          `json:"deliveries"`
	SeenItems    []model.ProactiveSeenItemRecord          `json:"seen_items,omitempty"`
	Rejections   []model.ProactiveRejectionCooldownRecord `json:"rejection_cooldowns,omitempty"`
	ContextOnly  []model.ProactiveContextOnlyRecord       `json:"context_only"`
	SessionMarks []model.ProactiveSessionMark             `json:"session_marks"`
	GlobalMarks  []model.ProactiveGlobalMark              `json:"global_marks,omitempty"`
	AnyAction    []model.ProactiveAnyActionQuota          `json:"anyaction_quotas,omitempty"`
	DriftSkills  []model.ProactiveDriftSkillState         `json:"drift_skills,omitempty"`
	DriftRuns    []model.ProactiveDriftRecentRun          `json:"drift_recent_runs,omitempty"`
	DriftNote    string                                   `json:"drift_note,omitempty"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("proactive state store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:         cleanPath,
		deliveries:   make(map[string]model.ProactiveDeliveryRecord),
		seenItems:    make(map[string]model.ProactiveSeenItemRecord),
		rejections:   make(map[string]model.ProactiveRejectionCooldownRecord),
		contextOnly:  make([]model.ProactiveContextOnlyRecord, 0),
		sessionMarks: make(map[string]model.ProactiveSessionMark),
		globalMarks:  make(map[string]model.ProactiveGlobalMark),
		anyAction:    make(map[string]model.ProactiveAnyActionQuota),
		driftSkills:  make(map[string]model.ProactiveDriftSkillState),
		driftRuns:    make([]model.ProactiveDriftRecentRun, 0),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveProactiveDelivery(_ context.Context, record model.ProactiveDeliveryRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := deliveryKey(record.SessionKey, record.DeliveryKey)
	if _, exists := s.deliveries[key]; !exists {
		s.deliveryOrder = append(s.deliveryOrder, key)
	}
	s.deliveries[key] = record
	return s.flush()
}

func (s *Store) FindProactiveDelivery(_ context.Context, sessionKey string, deliveryKeyValue string) (model.ProactiveDeliveryRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.deliveries[deliveryKey(sessionKey, deliveryKeyValue)]
	return record, ok, nil
}

func (s *Store) CountProactiveDeliveriesSince(_ context.Context, sessionKey string, since time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, record := range s.deliveries {
		if record.SessionKey == sessionKey && !record.SentAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (s *Store) ListProactiveDeliveries(_ context.Context, filter query.ProactiveDeliveryFilter) ([]model.ProactiveDeliveryRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.ProactiveDeliveryRecord, 0, limit)
	for i := len(s.deliveryOrder) - 1; i >= 0 && len(items) < limit; i-- {
		key := s.deliveryOrder[i]
		record, ok := s.deliveries[key]
		if !ok {
			continue
		}
		if filter.SessionKey != "" && record.SessionKey != filter.SessionKey {
			continue
		}
		if filter.DeliveryKey != "" && record.DeliveryKey != filter.DeliveryKey {
			continue
		}
		items = append(items, record)
	}
	return items, nil
}

func (s *Store) SaveProactiveSeenItems(_ context.Context, records []model.ProactiveSeenItemRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range records {
		if err := record.Validate(); err != nil {
			return err
		}
		s.seenItems[sourceItemKey(record.SourceKey, record.ItemID)] = record
	}
	return s.flush()
}

func (s *Store) FindProactiveSeenItem(_ context.Context, sourceKey string, itemID string) (model.ProactiveSeenItemRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.seenItems[sourceItemKey(sourceKey, itemID)]
	return record, ok, nil
}

func (s *Store) SaveProactiveRejectionCooldowns(_ context.Context, records []model.ProactiveRejectionCooldownRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range records {
		if err := record.Validate(); err != nil {
			return err
		}
		s.rejections[sourceItemKey(record.SourceKey, record.ItemID)] = record
	}
	return s.flush()
}

func (s *Store) FindProactiveRejectionCooldown(_ context.Context, sourceKey string, itemID string) (model.ProactiveRejectionCooldownRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.rejections[sourceItemKey(sourceKey, itemID)]
	return record, ok, nil
}

func (s *Store) SaveProactiveContextOnly(_ context.Context, record model.ProactiveContextOnlyRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.contextOnly = append(s.contextOnly, record)
	return s.flush()
}

func (s *Store) CountProactiveContextOnlySince(_ context.Context, sessionKey string, since time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, record := range s.contextOnly {
		if record.SessionKey == sessionKey && !record.SentAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (s *Store) SaveProactiveSessionMark(_ context.Context, mark model.ProactiveSessionMark) error {
	if err := mark.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionMarks[sessionMarkKey(mark.SessionKey, mark.Key)] = mark
	return s.flush()
}

func (s *Store) FindProactiveSessionMark(_ context.Context, sessionKey string, key string) (model.ProactiveSessionMark, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mark, ok := s.sessionMarks[sessionMarkKey(sessionKey, key)]
	return mark, ok, nil
}

func (s *Store) SaveProactiveGlobalMark(_ context.Context, mark model.ProactiveGlobalMark) error {
	if err := mark.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.globalMarks[strings.TrimSpace(mark.Key)] = mark
	return s.flush()
}

func (s *Store) FindProactiveGlobalMark(_ context.Context, key string) (model.ProactiveGlobalMark, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mark, ok := s.globalMarks[strings.TrimSpace(key)]
	return mark, ok, nil
}

func (s *Store) SaveProactiveAnyActionQuota(_ context.Context, quota model.ProactiveAnyActionQuota) error {
	if err := quota.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anyAction[quota.QuotaKey] = quota
	return s.flush()
}

func (s *Store) FindProactiveAnyActionQuota(_ context.Context, quotaKey string) (model.ProactiveAnyActionQuota, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quota, ok := s.anyAction[strings.TrimSpace(quotaKey)]
	return quota, ok, nil
}

func (s *Store) SaveProactiveDriftFinish(_ context.Context, state model.ProactiveDriftSkillState, run model.ProactiveDriftRecentRun, note string, recentLimit int) error {
	if err := state.Validate(); err != nil {
		return err
	}
	if err := run.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.driftSkills[strings.TrimSpace(state.SkillName)] = state
	s.driftRuns = append(s.driftRuns, run)
	if recentLimit <= 0 || recentLimit > 50 {
		recentLimit = 10
	}
	if len(s.driftRuns) > recentLimit {
		s.driftRuns = append([]model.ProactiveDriftRecentRun(nil), s.driftRuns[len(s.driftRuns)-recentLimit:]...)
	}
	if strings.TrimSpace(note) != "" {
		s.driftNote = strings.TrimSpace(note)
	}
	return s.flush()
}

func (s *Store) FindProactiveDriftSkillState(_ context.Context, skillName string) (model.ProactiveDriftSkillState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.driftSkills[strings.TrimSpace(skillName)]
	return state, ok, nil
}

func (s *Store) ListProactiveDriftRecentRuns(_ context.Context, limit int) ([]model.ProactiveDriftRecentRun, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	start := len(s.driftRuns) - limit
	if start < 0 {
		start = 0
	}
	items := append([]model.ProactiveDriftRecentRun(nil), s.driftRuns[start:]...)
	return items, s.driftNote, nil
}

func (s *Store) CleanupProactiveState(_ context.Context, cutoffs model.ProactiveStateRetentionCutoffs) (model.ProactiveStateCleanupResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := model.ProactiveStateCleanupResult{}
	if !cutoffs.DeliveriesBefore.IsZero() {
		nextOrder := make([]string, 0, len(s.deliveryOrder))
		for _, key := range s.deliveryOrder {
			record, ok := s.deliveries[key]
			if !ok {
				continue
			}
			if record.SentAt.Before(cutoffs.DeliveriesBefore) {
				delete(s.deliveries, key)
				result.RemovedDeliveries++
				continue
			}
			nextOrder = append(nextOrder, key)
		}
		s.deliveryOrder = nextOrder
	}
	if !cutoffs.SeenItemsBefore.IsZero() {
		for key, record := range s.seenItems {
			if record.SeenAt.Before(cutoffs.SeenItemsBefore) {
				delete(s.seenItems, key)
				result.RemovedSeenItems++
			}
		}
	}
	if !cutoffs.ContextOnlyBefore.IsZero() {
		next := make([]model.ProactiveContextOnlyRecord, 0, len(s.contextOnly))
		for _, record := range s.contextOnly {
			if record.SentAt.Before(cutoffs.ContextOnlyBefore) {
				result.RemovedContextOnly++
				continue
			}
			next = append(next, record)
		}
		s.contextOnly = next
	}
	if !cutoffs.RejectionCooldownsBefore.IsZero() {
		for key, record := range s.rejections {
			if record.RejectedAt.Before(cutoffs.RejectionCooldownsBefore) {
				delete(s.rejections, key)
				result.RemovedRejectionCooldowns++
			}
		}
	}
	if result.RemovedDeliveries == 0 &&
		result.RemovedSeenItems == 0 &&
		result.RemovedContextOnly == 0 &&
		result.RemovedRejectionCooldowns == 0 {
		return result, nil
	}
	return result, s.flush()
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}

	var state persistedState
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	for _, record := range state.Deliveries {
		if err := record.Validate(); err != nil {
			continue
		}
		key := deliveryKey(record.SessionKey, record.DeliveryKey)
		if _, exists := s.deliveries[key]; !exists {
			s.deliveryOrder = append(s.deliveryOrder, key)
		}
		s.deliveries[key] = record
	}
	for _, record := range state.SeenItems {
		if err := record.Validate(); err != nil {
			continue
		}
		s.seenItems[sourceItemKey(record.SourceKey, record.ItemID)] = record
	}
	for _, record := range state.Rejections {
		if err := record.Validate(); err != nil {
			continue
		}
		s.rejections[sourceItemKey(record.SourceKey, record.ItemID)] = record
	}
	for _, record := range state.ContextOnly {
		if err := record.Validate(); err != nil {
			continue
		}
		s.contextOnly = append(s.contextOnly, record)
	}
	for _, mark := range state.SessionMarks {
		if err := mark.Validate(); err != nil {
			continue
		}
		s.sessionMarks[sessionMarkKey(mark.SessionKey, mark.Key)] = mark
	}
	for _, mark := range state.GlobalMarks {
		if err := mark.Validate(); err != nil {
			continue
		}
		s.globalMarks[strings.TrimSpace(mark.Key)] = mark
	}
	for _, quota := range state.AnyAction {
		if err := quota.Validate(); err != nil {
			continue
		}
		s.anyAction[quota.QuotaKey] = quota
	}
	for _, skill := range state.DriftSkills {
		if err := skill.Validate(); err != nil {
			continue
		}
		s.driftSkills[strings.TrimSpace(skill.SkillName)] = skill
	}
	for _, run := range state.DriftRuns {
		if err := run.Validate(); err != nil {
			continue
		}
		s.driftRuns = append(s.driftRuns, run)
	}
	s.driftNote = strings.TrimSpace(state.DriftNote)
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version:      storeVersion,
		Deliveries:   make([]model.ProactiveDeliveryRecord, 0, len(s.deliveryOrder)),
		SeenItems:    make([]model.ProactiveSeenItemRecord, 0, len(s.seenItems)),
		Rejections:   make([]model.ProactiveRejectionCooldownRecord, 0, len(s.rejections)),
		ContextOnly:  append(make([]model.ProactiveContextOnlyRecord, 0, len(s.contextOnly)), s.contextOnly...),
		SessionMarks: make([]model.ProactiveSessionMark, 0, len(s.sessionMarks)),
		GlobalMarks:  make([]model.ProactiveGlobalMark, 0, len(s.globalMarks)),
		AnyAction:    make([]model.ProactiveAnyActionQuota, 0, len(s.anyAction)),
		DriftSkills:  make([]model.ProactiveDriftSkillState, 0, len(s.driftSkills)),
		DriftRuns:    append([]model.ProactiveDriftRecentRun(nil), s.driftRuns...),
		DriftNote:    s.driftNote,
	}
	for _, key := range s.deliveryOrder {
		if record, ok := s.deliveries[key]; ok {
			state.Deliveries = append(state.Deliveries, record)
		}
	}
	for _, record := range s.seenItems {
		state.SeenItems = append(state.SeenItems, record)
	}
	for _, record := range s.rejections {
		state.Rejections = append(state.Rejections, record)
	}
	for _, mark := range s.sessionMarks {
		state.SessionMarks = append(state.SessionMarks, mark)
	}
	for _, mark := range s.globalMarks {
		state.GlobalMarks = append(state.GlobalMarks, mark)
	}
	for _, quota := range s.anyAction {
		state.AnyAction = append(state.AnyAction, quota)
	}
	for _, skill := range s.driftSkills {
		state.DriftSkills = append(state.DriftSkills, skill)
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return err
	}
	if _, err := os.Stat(s.path); err == nil {
		if err := os.Remove(s.path); err != nil {
			return err
		}
	}
	return os.Rename(tmpPath, s.path)
}

func deliveryKey(sessionKey string, deliveryKeyValue string) string {
	return strings.TrimSpace(sessionKey) + "\x00" + strings.TrimSpace(deliveryKeyValue)
}

func sourceItemKey(sourceKey string, itemID string) string {
	return model.NormalizeProactiveSourceKey(sourceKey) + "\x00" + strings.TrimSpace(itemID)
}

func sessionMarkKey(sessionKey string, key string) string {
	return strings.TrimSpace(sessionKey) + "\x00" + strings.TrimSpace(key)
}
