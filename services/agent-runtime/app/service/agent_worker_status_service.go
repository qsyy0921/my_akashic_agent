package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const defaultAgentWorkerStatusStaleSeconds = 180
const defaultAgentWorkerStatusLeaseTTLSeconds = 120

var ErrAgentWorkerLeaseConflict = errors.New("agent worker status lease conflict")

type AgentWorkerLeaseConflictError struct {
	WorkerID           string
	ExistingInstanceID string
	LeaseUntil         time.Time
}

func (e AgentWorkerLeaseConflictError) Error() string {
	return fmt.Sprintf(
		"%s: worker_id=%s existing_instance_id=%s lease_until=%s",
		ErrAgentWorkerLeaseConflict,
		e.WorkerID,
		e.ExistingInstanceID,
		e.LeaseUntil.UTC().Format(time.RFC3339Nano),
	)
}

func (e AgentWorkerLeaseConflictError) Unwrap() error {
	return ErrAgentWorkerLeaseConflict
}

type AgentWorkerStatusService struct {
	mu         sync.RWMutex
	workers    map[string]model.AgentWorkerStatus
	repository outport.AgentWorkerStatusRepository
	staleAfter time.Duration
	clock      func() time.Time
}

func NewAgentWorkerStatusService() *AgentWorkerStatusService {
	return &AgentWorkerStatusService{
		workers:    make(map[string]model.AgentWorkerStatus),
		staleAfter: time.Duration(defaultAgentWorkerStatusStaleSeconds) * time.Second,
		clock:      time.Now,
	}
}

func NewAgentWorkerStatusServiceWithRepository(
	ctx context.Context,
	repository outport.AgentWorkerStatusRepository,
	staleAfter time.Duration,
) (*AgentWorkerStatusService, error) {
	service := NewAgentWorkerStatusService()
	service.repository = repository
	if staleAfter > 0 {
		service.staleAfter = staleAfter
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if repository != nil {
		now := time.Now().UTC()
		if service.clock != nil {
			now = service.clock().UTC()
		}
		items, err := repository.ListAgentWorkerStatuses(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				continue
			}
			projected, stale := staleAgentWorkerStatus(item, now, service.staleAfter)
			if stale {
				if err := repository.DeleteAgentWorkerStatus(ctx, item.WorkerID); err != nil {
					return nil, err
				}
				continue
			}
			service.workers[item.WorkerID] = projected
		}
	}
	return service, nil
}

func (s *AgentWorkerStatusService) ReportAgentWorkerStatus(
	ctx context.Context,
	cmd command.ReportAgentWorkerStatusCommand,
) (query.AgentWorkerStatusView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentWorkerStatusView{}, err
	}
	if s == nil {
		return query.AgentWorkerStatusView{}, errors.New("agent worker status service is nil")
	}
	timestamp := cmd.Timestamp
	if timestamp.IsZero() {
		if s.clock != nil {
			timestamp = s.clock().UTC()
		} else {
			timestamp = time.Now().UTC()
		}
	}
	status, err := model.NewAgentWorkerStatus(model.AgentWorkerStatusSpec{
		WorkerID:       cmd.WorkerID,
		InstanceID:     cmd.InstanceID,
		WorkerType:     cmd.WorkerType,
		Status:         cmd.Status,
		CurrentJobID:   cmd.CurrentJobID,
		LastJobID:      cmd.LastJobID,
		LastError:      cmd.LastError,
		ProcessedTotal: cmd.ProcessedTotal,
		FailedTotal:    cmd.FailedTotal,
		Source:         cmd.Source,
		Metadata:       cmd.Metadata,
	}, timestamp)
	if err != nil {
		return query.AgentWorkerStatusView{}, err
	}
	if status.InstanceID != "" && status.HeartbeatActive() {
		status.LeaseUntil = timestamp.Add(agentWorkerStatusLeaseTTL(cmd.LeaseTTLSeconds))
	}

	s.mu.Lock()
	if s.workers == nil {
		s.workers = make(map[string]model.AgentWorkerStatus)
	}
	if existing, ok := s.workers[status.WorkerID]; ok {
		if err := rejectAgentWorkerStatusLeaseConflict(existing, status, timestamp); err != nil {
			if canReplaceAgentWorkerLeaseConflict(existing, status, cmd.ReplaceExistingInstanceID) {
				// allow controlled takeover after the caller proved the existing instance is gone
			} else {
				s.mu.Unlock()
				return query.AgentWorkerStatusView{}, err
			}
		}
	}
	if s.repository != nil {
		if err := s.repository.SaveAgentWorkerStatus(ctx, status); err != nil {
			s.mu.Unlock()
			return query.AgentWorkerStatusView{}, err
		}
	}
	s.workers[status.WorkerID] = status
	s.mu.Unlock()
	return assembler.ToAgentWorkerStatusView(status, false), nil
}

func (s *AgentWorkerStatusService) ListAgentWorkerStatuses(
	ctx context.Context,
	filter query.AgentWorkerStatusFilter,
) (query.AgentWorkerStatusesView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentWorkerStatusesView{}, err
	}
	if s == nil {
		return query.AgentWorkerStatusesView{}, errors.New("agent worker status service is nil")
	}
	staleAfter := s.staleAfter
	if filter.StaleAfterSeconds > 0 {
		staleAfter = time.Duration(filter.StaleAfterSeconds) * time.Second
	}
	now := time.Now().UTC()
	if s.clock != nil {
		now = s.clock().UTC()
	}

	s.mu.RLock()
	items := s.snapshotLocked()
	s.mu.RUnlock()

	staleByWorkerID := make(map[string]bool)
	for index, item := range items {
		next := item.WithStaleHeartbeat(now, staleAfter)
		if next.Status != item.Status {
			staleByWorkerID[item.WorkerID] = true
			items[index] = next
		}
	}
	return agentWorkerStatusesView(items, staleByWorkerID), nil
}

func (s *AgentWorkerStatusService) CleanupStaleAgentWorkerStatuses(
	ctx context.Context,
	cmd command.CleanupStaleAgentWorkerStatusesCommand,
) (query.AgentWorkerStatusCleanupView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentWorkerStatusCleanupView{}, err
	}
	if s == nil {
		return query.AgentWorkerStatusCleanupView{}, errors.New("agent worker status service is nil")
	}

	now := cmd.Timestamp
	if now.IsZero() {
		if s.clock != nil {
			now = s.clock().UTC()
		} else {
			now = time.Now().UTC()
		}
	}
	staleAfter := s.staleAfter
	if cmd.StaleAfterSeconds > 0 {
		staleAfter = agentWorkerStatusStaleAfter(cmd.StaleAfterSeconds)
	}
	workerIDFilter := strings.TrimSpace(cmd.WorkerID)
	instanceIDFilter := strings.TrimSpace(cmd.InstanceID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.workers == nil {
		s.workers = make(map[string]model.AgentWorkerStatus)
	}

	deletedProjected := make([]model.AgentWorkerStatus, 0)
	for _, item := range model.SortedAgentWorkerStatuses(s.snapshotLocked()) {
		if workerIDFilter != "" && item.WorkerID != workerIDFilter {
			continue
		}
		if instanceIDFilter != "" && item.InstanceID != instanceIDFilter {
			continue
		}
		projected, stale := staleAgentWorkerStatus(item, now, staleAfter)
		if !stale {
			continue
		}
		if s.repository != nil {
			if err := s.repository.DeleteAgentWorkerStatus(ctx, item.WorkerID); err != nil {
				return query.AgentWorkerStatusCleanupView{}, err
			}
		}
		delete(s.workers, item.WorkerID)
		deletedProjected = append(deletedProjected, projected)
	}

	remainingOriginal := s.snapshotLocked()
	remainingProjected := make([]model.AgentWorkerStatus, len(remainingOriginal))
	staleByWorkerID := make(map[string]bool)
	for index, item := range remainingOriginal {
		projected, stale := staleAgentWorkerStatus(item, now, staleAfter)
		remainingProjected[index] = projected
		if stale {
			staleByWorkerID[item.WorkerID] = true
		}
	}
	return agentWorkerStatusCleanupView(deletedProjected, remainingProjected, staleByWorkerID), nil
}

func (s *AgentWorkerStatusService) snapshotLocked() []model.AgentWorkerStatus {
	items := make([]model.AgentWorkerStatus, 0, len(s.workers))
	for _, item := range s.workers {
		items = append(items, item)
	}
	return model.SortedAgentWorkerStatuses(items)
}

func staleAgentWorkerStatus(item model.AgentWorkerStatus, now time.Time, staleAfter time.Duration) (model.AgentWorkerStatus, bool) {
	projected := item.WithStaleHeartbeat(now, staleAfter)
	return projected, projected.Status != item.Status
}

func agentWorkerStatusesView(items []model.AgentWorkerStatus, staleByWorkerID map[string]bool) query.AgentWorkerStatusesView {
	totals := map[string]int{
		"workers":          len(items),
		"starting":         0,
		"idle":             0,
		"running":          0,
		"failed":           0,
		"stopped":          0,
		"stale":            0,
		"image_generation": 0,
		"knowledge":        0,
		"rag_eval":         0,
		"outbox_delivery":  0,
		"other_type":       0,
	}
	for _, item := range items {
		switch item.Status {
		case model.AgentWorkerStatusStarting:
			totals["starting"]++
		case model.AgentWorkerStatusIdle:
			totals["idle"]++
		case model.AgentWorkerStatusRunning:
			totals["running"]++
		case model.AgentWorkerStatusFailed:
			totals["failed"]++
		case model.AgentWorkerStatusStopped:
			totals["stopped"]++
		}
		switch item.WorkerType {
		case "image_generation":
			totals["image_generation"]++
		case "knowledge":
			totals["knowledge"]++
		case "rag_eval":
			totals["rag_eval"]++
		case "outbox_delivery":
			totals["outbox_delivery"]++
		default:
			totals["other_type"]++
		}
		if staleByWorkerID[item.WorkerID] {
			totals["stale"]++
		}
	}
	return query.AgentWorkerStatusesView{
		Workers:    assembler.ToAgentWorkerStatusViews(items, staleByWorkerID),
		Totals:     totals,
		Notes:      []string{"side_effect=none", "python_ai_worker_liveness"},
		SideEffect: "none",
	}
}

func agentWorkerStatusCleanupView(
	deleted []model.AgentWorkerStatus,
	remaining []model.AgentWorkerStatus,
	staleByWorkerID map[string]bool,
) query.AgentWorkerStatusCleanupView {
	deletedStaleByWorkerID := make(map[string]bool, len(deleted))
	totals := map[string]int{
		"deleted":         len(deleted),
		"remaining":       len(remaining),
		"remaining_stale": 0,
	}
	for _, item := range deleted {
		deletedStaleByWorkerID[item.WorkerID] = true
	}
	for _, item := range remaining {
		if staleByWorkerID[item.WorkerID] {
			totals["remaining_stale"]++
		}
	}
	return query.AgentWorkerStatusCleanupView{
		Deleted:    assembler.ToAgentWorkerStatusViews(deleted, deletedStaleByWorkerID),
		Remaining:  assembler.ToAgentWorkerStatusViews(remaining, staleByWorkerID),
		Totals:     totals,
		Notes:      []string{"side_effect=runtime_state_only", "stale_agent_worker_statuses_removed"},
		SideEffect: "runtime_state_only",
	}
}

func agentWorkerStatusStaleAfter(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = defaultAgentWorkerStatusStaleSeconds
	}
	if seconds < 30 {
		seconds = 30
	}
	if seconds > 24*60*60 {
		seconds = 24 * 60 * 60
	}
	return time.Duration(seconds) * time.Second
}

func agentWorkerStatusLeaseTTL(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = defaultAgentWorkerStatusLeaseTTLSeconds
	}
	if seconds < 30 {
		seconds = 30
	}
	if seconds > 60*60 {
		seconds = 60 * 60
	}
	return time.Duration(seconds) * time.Second
}

func rejectAgentWorkerStatusLeaseConflict(existing model.AgentWorkerStatus, incoming model.AgentWorkerStatus, now time.Time) error {
	if !existing.LeaseActive(now) {
		return nil
	}
	if existing.InstanceID == "" || incoming.InstanceID == existing.InstanceID {
		return nil
	}
	return AgentWorkerLeaseConflictError{
		WorkerID:           existing.WorkerID,
		ExistingInstanceID: existing.InstanceID,
		LeaseUntil:         existing.LeaseUntil,
	}
}

func canReplaceAgentWorkerLeaseConflict(
	existing model.AgentWorkerStatus,
	incoming model.AgentWorkerStatus,
	replaceExistingInstanceID string,
) bool {
	if replaceExistingInstanceID == "" {
		return false
	}
	if existing.InstanceID == "" || incoming.InstanceID == "" {
		return false
	}
	if existing.InstanceID == incoming.InstanceID {
		return false
	}
	return existing.InstanceID == replaceExistingInstanceID
}

func agentWorkerStatusMetadataWithStale(items map[string]string, staleAfter time.Duration) map[string]string {
	cloned := make(map[string]string, len(items)+1)
	for key, value := range items {
		cloned[key] = value
	}
	cloned["stale_after_seconds"] = strconv.Itoa(int(staleAfter.Seconds()))
	return cloned
}
