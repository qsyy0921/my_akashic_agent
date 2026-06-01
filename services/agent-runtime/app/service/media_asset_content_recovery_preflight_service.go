package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const (
	mediaAssetContentRecoveryTargetKind = "media_asset_content"
	mediaAssetContentRecoveryAction     = "recover_content"
)

type mediaAssetContentRecoveryPlanner interface {
	ContentRecoveryPlan(ctx context.Context, assetID string) (query.MediaAssetContentRecoveryPlanView, error)
}

type mediaAssetContentControlPreflightChecker interface {
	CheckControlMutationPreflight(ctx context.Context, preflight query.ControlMutationPreflight) (query.ControlMutationPreflightView, error)
}

type MediaAssetContentRecoveryPreflightService struct {
	planner   mediaAssetContentRecoveryPlanner
	preflight mediaAssetContentControlPreflightChecker
}

func NewMediaAssetContentRecoveryPreflightService(
	planner mediaAssetContentRecoveryPlanner,
	preflight mediaAssetContentControlPreflightChecker,
) *MediaAssetContentRecoveryPreflightService {
	return &MediaAssetContentRecoveryPreflightService{
		planner:   planner,
		preflight: preflight,
	}
}

func (s *MediaAssetContentRecoveryPreflightService) CheckMediaAssetContentRecoveryPreflight(
	ctx context.Context,
	filter query.MediaAssetContentRecoveryPreflightFilter,
) (query.MediaAssetContentRecoveryPreflightView, error) {
	if err := ctx.Err(); err != nil {
		return query.MediaAssetContentRecoveryPreflightView{}, err
	}
	if s == nil || s.planner == nil {
		return query.MediaAssetContentRecoveryPreflightView{}, errors.New("media asset content recovery preflight requires planner")
	}

	assetID := strings.TrimSpace(filter.AssetID)
	targetID := strings.TrimSpace(filter.TargetID)
	if targetID == "" {
		targetID = assetID
	}
	operatorID := strings.TrimSpace(filter.OperatorID)
	approvalID := strings.TrimSpace(filter.ApprovalID)
	plan, err := s.planner.ContentRecoveryPlan(ctx, assetID)
	if err != nil {
		return query.MediaAssetContentRecoveryPreflightView{}, err
	}

	base := query.MediaAssetContentRecoveryPreflightView{
		TargetKind:     mediaAssetContentRecoveryTargetKind,
		TargetID:       targetID,
		Action:         mediaAssetContentRecoveryAction,
		OperatorID:     operatorID,
		ApprovalID:     approvalID,
		AssetID:        assetID,
		RecoveryNeeded: !plan.Ready,
		ExecutorScope:  plan.FutureExecutorScope,
		Plan:           plan,
		SideEffect:     "none",
		Notes: []string{
			"read-only media content recovery preflight; no content is downloaded, restored, streamed or parsed",
			"approval and mutation audit records must be created explicitly through their own endpoints",
		},
	}
	if plan.Ready {
		base.Reason = "media_asset_content_recovery_not_required"
		base.Blockers = []string{"media_asset_content_recovery_not_required"}
		return base, nil
	}
	if strings.TrimSpace(plan.FutureExecutorScope) == "" {
		base.Reason = plan.Reason
		base.Blockers = append([]string(nil), plan.Blockers...)
		if len(base.Blockers) == 0 && plan.Reason != "" {
			base.Blockers = []string{plan.Reason}
		}
		return base, nil
	}
	if s.preflight == nil {
		base.Reason = "control_mutation_preflight_unavailable"
		base.Blockers = []string{"control_mutation_preflight_unavailable"}
		return base, nil
	}

	control, err := s.preflight.CheckControlMutationPreflight(ctx, query.ControlMutationPreflight{
		TargetKind: mediaAssetContentRecoveryTargetKind,
		TargetID:   targetID,
		Action:     mediaAssetContentRecoveryAction,
		OperatorID: operatorID,
		ApprovalID: approvalID,
	})
	if err != nil {
		return query.MediaAssetContentRecoveryPreflightView{}, err
	}
	base.ControlPreflight = control
	base.SuggestedAudit = control.SuggestedAudit
	if !control.Ready {
		base.Reason = control.Reason
		base.Blockers = append([]string(nil), control.Blockers...)
		if len(base.Blockers) == 0 && control.Reason != "" {
			base.Blockers = []string{control.Reason}
		}
		return base, nil
	}
	base.Ready = true
	base.Reason = "media_asset_content_recovery_preflight_ready"
	return base, nil
}
