package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type controlMutationApprovalChecker interface {
	CheckOperatorApproval(ctx context.Context, check query.OperatorApprovalCheck) (query.OperatorApprovalCheckView, error)
}

type ControlMutationPreflightService struct {
	approvals controlMutationApprovalChecker
	policy    domainservice.ControlMutationPolicy
}

func NewControlMutationPreflightService(approvals controlMutationApprovalChecker) *ControlMutationPreflightService {
	return &ControlMutationPreflightService{
		approvals: approvals,
		policy:    domainservice.NewControlMutationPolicy(),
	}
}

func (s *ControlMutationPreflightService) CheckControlMutationPreflight(ctx context.Context, preflight query.ControlMutationPreflight) (query.ControlMutationPreflightView, error) {
	if err := ctx.Err(); err != nil {
		return query.ControlMutationPreflightView{}, err
	}
	if s == nil {
		return query.ControlMutationPreflightView{}, errors.New("control mutation preflight service is nil")
	}

	targetKind := strings.TrimSpace(preflight.TargetKind)
	targetID := strings.TrimSpace(preflight.TargetID)
	action := strings.TrimSpace(preflight.Action)
	operatorID := strings.TrimSpace(preflight.OperatorID)
	approvalID := strings.TrimSpace(preflight.ApprovalID)

	base := query.ControlMutationPreflightView{
		TargetKind: targetKind,
		TargetID:   targetID,
		Action:     action,
		OperatorID: operatorID,
		ApprovalID: approvalID,
		SideEffect: "none",
		Notes:      []string{"preflight only; no runtime configuration is changed"},
	}

	blockers := make([]string, 0, 5)
	if targetKind == "" {
		blockers = append(blockers, "missing_target_kind")
	}
	if targetID == "" {
		blockers = append(blockers, "missing_target_id")
	}
	if action == "" {
		blockers = append(blockers, "missing_action")
	}
	if operatorID == "" {
		blockers = append(blockers, "missing_operator_id")
	}
	if approvalID == "" {
		blockers = append(blockers, "missing_approval_id")
	}
	if len(blockers) > 0 {
		base.Reason = blockers[0]
		base.Blockers = blockers
		return base, nil
	}
	policyResult := s.policy.Check(targetKind, action)
	base.SupportedActions = policyResult.SupportedActions
	if !policyResult.Allowed {
		base.Reason = policyResult.Reason
		base.Blockers = []string{policyResult.Reason}
		return base, nil
	}
	if s.approvals == nil {
		base.Reason = "operator_approval_checker_unavailable"
		base.Blockers = []string{"operator_approval_checker_unavailable"}
		return base, nil
	}

	approvalCheck, err := s.approvals.CheckOperatorApproval(ctx, query.OperatorApprovalCheck{
		ApprovalID: approvalID,
		TargetKind: targetKind,
		TargetID:   targetID,
	})
	if err != nil {
		return query.ControlMutationPreflightView{}, err
	}
	base.ApprovalCheck = approvalCheck
	if !approvalCheck.Approved {
		base.Reason = "approval_not_ready"
		base.Blockers = append([]string(nil), approvalCheck.Blockers...)
		if len(base.Blockers) == 0 && approvalCheck.Reason != "" {
			base.Blockers = []string{approvalCheck.Reason}
		}
		return base, nil
	}

	base.Ready = true
	base.Reason = approvalCheck.Reason
	base.SuggestedAudit = &query.ControlMutationSuggestedAuditView{
		TargetKind: targetKind,
		TargetID:   targetID,
		Action:     action,
		Status:     "planned",
		OperatorID: operatorID,
		ApprovalID: approvalID,
		Metadata: map[string]string{
			"preflight": "control_mutation_preflight",
		},
	}
	return base, nil
}
