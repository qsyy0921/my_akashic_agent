package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const (
	mediaAssetRetentionCleanupTargetKind = "media_asset_retention"
	mediaAssetRetentionCleanupAction     = "cleanup_expired"
)

type mediaAssetRetentionPlanner interface {
	RetentionPlan(ctx context.Context, filter query.MediaAssetRetentionDiagnosticsFilter) (query.MediaAssetRetentionPlanView, error)
}

type mediaAssetRetentionControlPreflightChecker interface {
	CheckControlMutationPreflight(ctx context.Context, preflight query.ControlMutationPreflight) (query.ControlMutationPreflightView, error)
}

type MediaAssetRetentionCleanupPreflightService struct {
	planner   mediaAssetRetentionPlanner
	preflight mediaAssetRetentionControlPreflightChecker
}

func NewMediaAssetRetentionCleanupPreflightService(
	planner mediaAssetRetentionPlanner,
	preflight mediaAssetRetentionControlPreflightChecker,
) *MediaAssetRetentionCleanupPreflightService {
	return &MediaAssetRetentionCleanupPreflightService{
		planner:   planner,
		preflight: preflight,
	}
}

func (s *MediaAssetRetentionCleanupPreflightService) CheckMediaAssetRetentionCleanupPreflight(
	ctx context.Context,
	filter query.MediaAssetRetentionCleanupPreflightFilter,
) (query.MediaAssetRetentionCleanupPreflightView, error) {
	if err := ctx.Err(); err != nil {
		return query.MediaAssetRetentionCleanupPreflightView{}, err
	}
	if s == nil || s.planner == nil {
		return query.MediaAssetRetentionCleanupPreflightView{}, errors.New("media asset retention cleanup preflight requires planner")
	}

	targetID := strings.TrimSpace(filter.TargetID)
	operatorID := strings.TrimSpace(filter.OperatorID)
	approvalID := strings.TrimSpace(filter.ApprovalID)
	plan, err := s.planner.RetentionPlan(ctx, filter.RetentionFilter)
	if err != nil {
		return query.MediaAssetRetentionCleanupPreflightView{}, err
	}

	base := query.MediaAssetRetentionCleanupPreflightView{
		TargetKind:     mediaAssetRetentionCleanupTargetKind,
		TargetID:       targetID,
		Action:         mediaAssetRetentionCleanupAction,
		OperatorID:     operatorID,
		ApprovalID:     approvalID,
		CandidateCount: plan.CandidateCount,
		AssetCount:     plan.AssetCount,
		Plan:           plan,
		SideEffect:     "none",
		Notes: []string{
			"read-only media retention cleanup preflight; no media metadata or file content is deleted",
			"approval and mutation audit records must be created explicitly through their own endpoints",
		},
	}
	if !plan.Ready {
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
		TargetKind: mediaAssetRetentionCleanupTargetKind,
		TargetID:   targetID,
		Action:     mediaAssetRetentionCleanupAction,
		OperatorID: operatorID,
		ApprovalID: approvalID,
	})
	if err != nil {
		return query.MediaAssetRetentionCleanupPreflightView{}, err
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
	base.Reason = "media_asset_retention_cleanup_preflight_ready"
	return base, nil
}
