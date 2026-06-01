package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestMediaAssetRetentionCleanupPreflightAllowsApprovedCleanup(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetService(store)
	approvals := appservice.NewOperatorApprovalService()
	controlPreflight := appservice.NewControlMutationPreflightService(approvals)
	service := appservice.NewMediaAssetRetentionCleanupPreflightService(mediaAssets, controlPreflight)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, mediaAssets, "asset:old", "old.txt", "ephemeral", now.Add(-2*time.Hour))
	approval, err := approvals.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "media_asset_retention",
		TargetID:   "default-observed-group",
		Decision:   "approved",
		OperatorID: "qsyy",
		Timestamp:  now,
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}

	view, err := service.CheckMediaAssetRetentionCleanupPreflight(ctx, query.MediaAssetRetentionCleanupPreflightFilter{
		RetentionFilter: query.MediaAssetRetentionDiagnosticsFilter{
			Limit:             10,
			Timestamp:         now.Format(time.RFC3339Nano),
			EphemeralTTLHours: 1,
		},
		TargetID:   "default-observed-group",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
	})
	if err != nil {
		t.Fatalf("cleanup preflight: %v", err)
	}
	if !view.Ready || view.Reason != "media_asset_retention_cleanup_preflight_ready" ||
		view.CandidateCount != 1 || view.ControlPreflight.SuggestedAudit == nil ||
		view.SuggestedAudit == nil || view.SideEffect != "none" {
		t.Fatalf("unexpected ready cleanup preflight: %+v", view)
	}
	if view.SuggestedAudit.TargetKind != "media_asset_retention" ||
		view.SuggestedAudit.Action != "cleanup_expired" ||
		view.SuggestedAudit.Status != "planned" {
		t.Fatalf("unexpected suggested audit: %+v", view.SuggestedAudit)
	}
}

func TestMediaAssetRetentionCleanupPreflightBlocksNoCandidates(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetService(store)
	service := appservice.NewMediaAssetRetentionCleanupPreflightService(mediaAssets, nil)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, mediaAssets, "asset:keep", "keep.txt", "permanent", now.Add(-90*24*time.Hour))

	view, err := service.CheckMediaAssetRetentionCleanupPreflight(ctx, query.MediaAssetRetentionCleanupPreflightFilter{
		RetentionFilter: query.MediaAssetRetentionDiagnosticsFilter{
			Limit:     10,
			Timestamp: now.Format(time.RFC3339Nano),
		},
		TargetID:   "default-observed-group",
		OperatorID: "qsyy",
		ApprovalID: "approval-a",
	})
	if err != nil {
		t.Fatalf("cleanup preflight: %v", err)
	}
	if view.Ready || view.Reason != "media_asset_retention_no_cleanup_candidates" ||
		len(view.Blockers) == 0 || view.CandidateCount != 0 || view.ControlPreflight.SideEffect != "" {
		t.Fatalf("unexpected no-candidate cleanup preflight: %+v", view)
	}
}

func TestMediaAssetRetentionCleanupPreflightBlocksMissingApproval(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetService(store)
	approvals := appservice.NewOperatorApprovalService()
	controlPreflight := appservice.NewControlMutationPreflightService(approvals)
	service := appservice.NewMediaAssetRetentionCleanupPreflightService(mediaAssets, controlPreflight)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, mediaAssets, "asset:old", "old.txt", "ephemeral", now.Add(-2*time.Hour))

	view, err := service.CheckMediaAssetRetentionCleanupPreflight(ctx, query.MediaAssetRetentionCleanupPreflightFilter{
		RetentionFilter: query.MediaAssetRetentionDiagnosticsFilter{
			Limit:             10,
			Timestamp:         now.Format(time.RFC3339Nano),
			EphemeralTTLHours: 1,
		},
		TargetID:   "default-observed-group",
		OperatorID: "qsyy",
		ApprovalID: "missing",
	})
	if err != nil {
		t.Fatalf("cleanup preflight: %v", err)
	}
	if view.Ready || view.Reason != "approval_not_ready" ||
		len(view.Blockers) != 1 || view.Blockers[0] != "approval_not_found" ||
		view.SuggestedAudit != nil {
		t.Fatalf("unexpected missing approval cleanup preflight: %+v", view)
	}
}
