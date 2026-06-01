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

type OperatorApprovalService struct {
	mu         sync.RWMutex
	approvals  map[string]model.OperatorApproval
	repository outport.OperatorApprovalRepository
}

func NewOperatorApprovalService() *OperatorApprovalService {
	return &OperatorApprovalService{approvals: make(map[string]model.OperatorApproval)}
}

func NewOperatorApprovalServiceWithRepository(
	ctx context.Context,
	repository outport.OperatorApprovalRepository,
) (*OperatorApprovalService, error) {
	service := NewOperatorApprovalService()
	service.repository = repository
	if repository != nil {
		items, err := repository.ListOperatorApprovals(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				continue
			}
			service.approvals[item.ApprovalID] = item
		}
	}
	return service, nil
}

func (s *OperatorApprovalService) RecordOperatorApproval(ctx context.Context, cmd command.RecordOperatorApprovalCommand) (query.OperatorApprovalView, error) {
	if err := ctx.Err(); err != nil {
		return query.OperatorApprovalView{}, err
	}
	if s == nil {
		return query.OperatorApprovalView{}, errors.New("operator approval service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	approval, err := model.NewOperatorApproval(model.OperatorApprovalSpec{
		ApprovalID: cmd.ApprovalID,
		TargetKind: cmd.TargetKind,
		TargetID:   cmd.TargetID,
		Decision:   cmd.Decision,
		OperatorID: cmd.OperatorID,
		Reason:     cmd.Reason,
		ExpiresAt:  cmd.ExpiresAt,
		Metadata:   cmd.Metadata,
	}, now)
	if err != nil {
		return query.OperatorApprovalView{}, err
	}

	s.mu.Lock()
	if s.approvals == nil {
		s.approvals = make(map[string]model.OperatorApproval)
	}
	if s.repository != nil {
		if err := s.repository.SaveOperatorApproval(ctx, approval); err != nil {
			s.mu.Unlock()
			return query.OperatorApprovalView{}, err
		}
	}
	s.approvals[approval.ApprovalID] = approval
	s.mu.Unlock()

	return assembler.ToOperatorApprovalView(approval, now), nil
}

func (s *OperatorApprovalService) ListOperatorApprovals(ctx context.Context, filter query.OperatorApprovalFilter) (query.OperatorApprovalsView, error) {
	if err := ctx.Err(); err != nil {
		return query.OperatorApprovalsView{}, err
	}
	if s == nil {
		return query.OperatorApprovalsView{}, errors.New("operator approval service is nil")
	}
	now := time.Now().UTC()
	limit := boundedOperatorApprovalLimit(filter.Limit)
	targetKind := strings.TrimSpace(filter.TargetKind)
	targetID := strings.TrimSpace(filter.TargetID)
	decision := strings.TrimSpace(filter.Decision)

	s.mu.RLock()
	items := make([]model.OperatorApproval, 0, len(s.approvals))
	for _, item := range s.approvals {
		if targetKind != "" && item.TargetKind != targetKind {
			continue
		}
		if targetID != "" && item.TargetID != targetID {
			continue
		}
		if decision != "" && string(item.Decision) != decision {
			continue
		}
		items = append(items, item)
	}
	s.mu.RUnlock()
	items = model.SortedOperatorApprovals(items)
	if len(items) > limit {
		items = items[:limit]
	}

	totals := map[string]int{
		"approvals": len(items),
		"active":    0,
		"approved":  0,
		"rejected":  0,
		"revoked":   0,
	}
	for _, item := range items {
		if item.ActiveAt(now) {
			totals["active"]++
		}
		totals[string(item.Decision)]++
	}
	return query.OperatorApprovalsView{
		Approvals:  assembler.ToOperatorApprovalViews(items, now),
		Totals:     totals,
		Notes:      []string{"side_effect=runtime_state_only", "approval ledger only; no runtime configuration is changed"},
		SideEffect: "runtime_state_only",
	}, nil
}

func boundedOperatorApprovalLimit(value int) int {
	if value <= 0 {
		return 100
	}
	if value > 500 {
		return 500
	}
	return value
}
