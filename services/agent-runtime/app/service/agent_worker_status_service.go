package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const defaultAgentWorkerStatusStaleSeconds = 180

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
		items, err := repository.ListAgentWorkerStatuses(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				continue
			}
			service.workers[item.WorkerID] = item
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
		timestamp = time.Now().UTC()
	}
	status, err := model.NewAgentWorkerStatus(model.AgentWorkerStatusSpec{
		WorkerID:       cmd.WorkerID,
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

	s.mu.Lock()
	if s.workers == nil {
		s.workers = make(map[string]model.AgentWorkerStatus)
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

func (s *AgentWorkerStatusService) snapshotLocked() []model.AgentWorkerStatus {
	items := make([]model.AgentWorkerStatus, 0, len(s.workers))
	for _, item := range s.workers {
		items = append(items, item)
	}
	return model.SortedAgentWorkerStatuses(items)
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

func agentWorkerStatusMetadataWithStale(items map[string]string, staleAfter time.Duration) map[string]string {
	cloned := make(map[string]string, len(items)+1)
	for key, value := range items {
		cloned[key] = value
	}
	cloned["stale_after_seconds"] = strconv.Itoa(int(staleAfter.Seconds()))
	return cloned
}
