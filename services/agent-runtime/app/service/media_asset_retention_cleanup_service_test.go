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

func TestMediaAssetRetentionCleanupDryRunDoesNotDeleteOrAudit(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetService(store)
	approvals := appservice.NewOperatorApprovalService()
	audits := appservice.NewControlMutationAuditService()
	preflight := appservice.NewMediaAssetRetentionCleanupPreflightService(mediaAssets, appservice.NewControlMutationPreflightService(approvals))
	cleanup := appservice.NewMediaAssetRetentionCleanupService(store, preflight, audits)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, mediaAssets, "asset:old", "old.txt", "ephemeral", now.Add(-2*time.Hour))

	view, err := cleanup.CleanupMediaAssetRetention(ctx, command.CleanupMediaAssetRetentionCommand{
		EphemeralTTLHours: 1,
		Timestamp:         now.Format(time.RFC3339Nano),
		TargetID:          "default-observed-group",
		OperatorID:        "qsyy",
		ApprovalID:        "missing",
		DryRun:            true,
	})
	if err != nil {
		t.Fatalf("dry run cleanup: %v", err)
	}
	if !view.DryRun || view.Applied || view.DeletedCount != 0 || view.SideEffect != "none" {
		t.Fatalf("unexpected dry run view: %+v", view)
	}
	if len(store.MediaAssets()) != 1 {
		t.Fatalf("dry run should not delete media metadata")
	}
	auditView, err := audits.ListControlMutationAudits(ctx, query.ControlMutationAuditFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list audits: %v", err)
	}
	if len(auditView.Mutations) != 0 {
		t.Fatalf("dry run should not record audit: %+v", auditView)
	}
}

func TestMediaAssetRetentionCleanupBlocksMissingApprovalWithoutDeleting(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetService(store)
	approvals := appservice.NewOperatorApprovalService()
	audits := appservice.NewControlMutationAuditService()
	preflight := appservice.NewMediaAssetRetentionCleanupPreflightService(mediaAssets, appservice.NewControlMutationPreflightService(approvals))
	cleanup := appservice.NewMediaAssetRetentionCleanupService(store, preflight, audits)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, mediaAssets, "asset:old", "old.txt", "ephemeral", now.Add(-2*time.Hour))

	view, err := cleanup.CleanupMediaAssetRetention(ctx, command.CleanupMediaAssetRetentionCommand{
		EphemeralTTLHours: 1,
		Timestamp:         now.Format(time.RFC3339Nano),
		TargetID:          "default-observed-group",
		OperatorID:        "qsyy",
		ApprovalID:        "missing",
	})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if view.Applied || view.Ready || view.DeletedCount != 0 || view.Reason != "approval_not_ready" {
		t.Fatalf("unexpected missing approval cleanup: %+v", view)
	}
	if len(store.MediaAssets()) != 1 {
		t.Fatalf("missing approval should not delete media metadata")
	}
}

func TestMediaAssetRetentionCleanupDeletesMetadataAndRecordsAppliedAudit(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetService(store)
	approvals := appservice.NewOperatorApprovalService()
	audits := appservice.NewControlMutationAuditService()
	preflight := appservice.NewMediaAssetRetentionCleanupPreflightService(mediaAssets, appservice.NewControlMutationPreflightService(approvals))
	cleanup := appservice.NewMediaAssetRetentionCleanupService(store, preflight, audits)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, mediaAssets, "asset:old", "old.txt", "ephemeral", now.Add(-2*time.Hour))
	registerTestMediaAssetWithRetention(t, mediaAssets, "asset:keep", "keep.txt", "permanent", now.Add(-90*24*time.Hour))
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

	view, err := cleanup.CleanupMediaAssetRetention(ctx, command.CleanupMediaAssetRetentionCommand{
		EphemeralTTLHours: 1,
		Timestamp:         now.Format(time.RFC3339Nano),
		TargetID:          "default-observed-group",
		OperatorID:        "qsyy",
		ApprovalID:        approval.ApprovalID,
		MutationID:        "mutation-media-cleanup-1",
	})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if !view.Applied || view.DeletedCount != 1 || view.DeletedAssetIDs[0] != "asset:old" ||
		view.AppliedAudit == nil || view.AppliedAudit.Status != "applied" ||
		view.SideEffect != "runtime_state_cleanup" {
		t.Fatalf("unexpected applied cleanup: %+v", view)
	}
	if _, ok, err := store.FindMediaAsset(ctx, "asset:old"); err != nil || ok {
		t.Fatalf("old media metadata should be deleted, ok=%t err=%v", ok, err)
	}
	if _, ok, err := store.FindMediaAsset(ctx, "asset:keep"); err != nil || !ok {
		t.Fatalf("permanent media metadata should remain, ok=%t err=%v", ok, err)
	}
}
