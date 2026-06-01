package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ControlMutationAuditService struct {
	mu         sync.RWMutex
	audits     map[string]model.ControlMutationAudit
	repository outport.ControlMutationAuditRepository
}

func NewControlMutationAuditService() *ControlMutationAuditService {
	return &ControlMutationAuditService{audits: make(map[string]model.ControlMutationAudit)}
}

func NewControlMutationAuditServiceWithRepository(
	ctx context.Context,
	repository outport.ControlMutationAuditRepository,
) (*ControlMutationAuditService, error) {
	service := NewControlMutationAuditService()
	service.repository = repository
	if repository != nil {
		items, err := repository.ListControlMutationAudits(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				continue
			}
			service.audits[item.MutationID] = item
		}
	}
	return service, nil
}

func (s *ControlMutationAuditService) RecordControlMutationAudit(ctx context.Context, cmd command.RecordControlMutationAuditCommand) (query.ControlMutationAuditView, error) {
	if err := ctx.Err(); err != nil {
		return query.ControlMutationAuditView{}, err
	}
	if s == nil {
		return query.ControlMutationAuditView{}, errors.New("control mutation audit service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	audit, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		MutationID:  cmd.MutationID,
		TargetKind:  cmd.TargetKind,
		TargetID:    cmd.TargetID,
		Action:      cmd.Action,
		Status:      cmd.Status,
		OperatorID:  cmd.OperatorID,
		ApprovalID:  cmd.ApprovalID,
		Reason:      cmd.Reason,
		RollbackOf:  cmd.RollbackOf,
		RollbackRef: cmd.RollbackRef,
		Metadata:    cmd.Metadata,
	}, now)
	if err != nil {
		return query.ControlMutationAuditView{}, err
	}

	s.mu.Lock()
	if s.audits == nil {
		s.audits = make(map[string]model.ControlMutationAudit)
	}
	if s.repository != nil {
		if err := s.repository.SaveControlMutationAudit(ctx, audit); err != nil {
			s.mu.Unlock()
			return query.ControlMutationAuditView{}, err
		}
	}
	s.audits[audit.MutationID] = audit
	s.mu.Unlock()

	return assembler.ToControlMutationAuditView(audit), nil
}

func (s *ControlMutationAuditService) ListControlMutationAudits(ctx context.Context, filter query.ControlMutationAuditFilter) (query.ControlMutationAuditsView, error) {
	if err := ctx.Err(); err != nil {
		return query.ControlMutationAuditsView{}, err
	}
	if s == nil {
		return query.ControlMutationAuditsView{}, errors.New("control mutation audit service is nil")
	}
	limit := boundedControlMutationAuditLimit(filter.Limit)
	targetKind := strings.TrimSpace(filter.TargetKind)
	targetID := strings.TrimSpace(filter.TargetID)
	status := strings.TrimSpace(filter.Status)
	approvalID := strings.TrimSpace(filter.ApprovalID)

	s.mu.RLock()
	items := make([]model.ControlMutationAudit, 0, len(s.audits))
	for _, item := range s.audits {
		if targetKind != "" && item.TargetKind != targetKind {
			continue
		}
		if targetID != "" && item.TargetID != targetID {
			continue
		}
		if status != "" && string(item.Status) != status {
			continue
		}
		if approvalID != "" && item.ApprovalID != approvalID {
			continue
		}
		items = append(items, item)
	}
	s.mu.RUnlock()

	items = model.SortedControlMutationAudits(items)
	if len(items) > limit {
		items = items[:limit]
	}
	totals := map[string]int{
		"mutations":   len(items),
		"planned":     0,
		"applied":     0,
		"failed":      0,
		"rolled_back": 0,
	}
	for _, item := range items {
		totals[string(item.Status)]++
	}
	return query.ControlMutationAuditsView{
		Mutations:  assembler.ToControlMutationAuditViews(items),
		Totals:     totals,
		Notes:      []string{"side_effect=runtime_state_only", "control mutation audit only; no runtime configuration is changed"},
		SideEffect: "runtime_state_only",
	}, nil
}

func boundedControlMutationAuditLimit(value int) int {
	if value <= 0 {
		return 100
	}
	if value > 500 {
		return 500
	}
	return value
}
