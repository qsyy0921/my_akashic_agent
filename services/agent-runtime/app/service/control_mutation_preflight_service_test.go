package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
)

func TestControlMutationPreflightServiceAllowsActiveApproval(t *testing.T) {
	ctx := context.Background()
	approvals := appservice.NewOperatorApprovalService()
	approval, err := approvals.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Decision:   "approved",
		OperatorID: "qsyy",
		Timestamp:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	preflight := appservice.NewControlMutationPreflightService(approvals)

	view, err := preflight.CheckControlMutationPreflight(ctx, query.ControlMutationPreflight{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Action:     "enable",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
	})
	if err != nil {
		t.Fatalf("check preflight: %v", err)
	}
	if !view.Ready || view.Reason != "approval_active" || view.SideEffect != "none" {
		t.Fatalf("unexpected preflight view: %+v", view)
	}
	if view.SuggestedAudit == nil || view.SuggestedAudit.Status != "planned" || view.SuggestedAudit.ApprovalID != approval.ApprovalID {
		t.Fatalf("unexpected suggested audit: %+v", view.SuggestedAudit)
	}
	if !view.ApprovalCheck.Approved {
		t.Fatalf("expected approved check: %+v", view.ApprovalCheck)
	}
}

func TestControlMutationPreflightServiceAllowsMediaRetentionCleanupApproval(t *testing.T) {
	ctx := context.Background()
	approvals := appservice.NewOperatorApprovalService()
	approval, err := approvals.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "media_asset_retention",
		TargetID:   "default-observed-group",
		Decision:   "approved",
		OperatorID: "qsyy",
		Timestamp:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	preflight := appservice.NewControlMutationPreflightService(approvals)

	view, err := preflight.CheckControlMutationPreflight(ctx, query.ControlMutationPreflight{
		TargetKind: "media_asset_retention",
		TargetID:   "default-observed-group",
		Action:     "cleanup_expired",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
	})
	if err != nil {
		t.Fatalf("check preflight: %v", err)
	}
	if !view.Ready || view.Reason != "approval_active" || view.SideEffect != "none" {
		t.Fatalf("unexpected media retention preflight view: %+v", view)
	}
	if len(view.SupportedActions) != 1 || view.SupportedActions[0] != "cleanup_expired" {
		t.Fatalf("expected cleanup_expired supported action, got %+v", view.SupportedActions)
	}
	if view.SuggestedAudit == nil ||
		view.SuggestedAudit.TargetKind != "media_asset_retention" ||
		view.SuggestedAudit.Action != "cleanup_expired" ||
		view.SuggestedAudit.Status != "planned" {
		t.Fatalf("unexpected suggested audit: %+v", view.SuggestedAudit)
	}
}

func TestControlMutationPreflightServiceBlocksMissingApproval(t *testing.T) {
	ctx := context.Background()
	approvals := appservice.NewOperatorApprovalService()
	preflight := appservice.NewControlMutationPreflightService(approvals)

	view, err := preflight.CheckControlMutationPreflight(ctx, query.ControlMutationPreflight{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Action:     "enable",
		OperatorID: "qsyy",
		ApprovalID: "missing",
	})
	if err != nil {
		t.Fatalf("check preflight: %v", err)
	}
	if view.Ready || view.Reason != "approval_not_ready" || len(view.Blockers) != 1 || view.Blockers[0] != "approval_not_found" {
		t.Fatalf("unexpected blocked preflight view: %+v", view)
	}
	if view.SuggestedAudit != nil {
		t.Fatalf("blocked preflight must not suggest audit: %+v", view.SuggestedAudit)
	}
}

func TestControlMutationPreflightServiceBlocksUnsupportedAction(t *testing.T) {
	ctx := context.Background()
	approvals := appservice.NewOperatorApprovalService()
	approval, err := approvals.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Decision:   "approved",
		OperatorID: "qsyy",
		Timestamp:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	preflight := appservice.NewControlMutationPreflightService(approvals)

	view, err := preflight.CheckControlMutationPreflight(ctx, query.ControlMutationPreflight{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Action:     "apply",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
	})
	if err != nil {
		t.Fatalf("check preflight: %v", err)
	}
	if view.Ready || view.Reason != "unsupported_control_mutation_action" || view.Blockers[0] != "unsupported_control_mutation_action" {
		t.Fatalf("unexpected unsupported action view: %+v", view)
	}
	if len(view.SupportedActions) != 2 || view.SupportedActions[0] != "enable" || view.SupportedActions[1] != "rollback" {
		t.Fatalf("expected supported action hints, got %+v", view.SupportedActions)
	}
}

func TestControlMutationPreflightServiceBlocksUnsupportedMediaRetentionAction(t *testing.T) {
	ctx := context.Background()
	approvals := appservice.NewOperatorApprovalService()
	approval, err := approvals.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "media_asset_retention",
		TargetID:   "default-observed-group",
		Decision:   "approved",
		OperatorID: "qsyy",
		Timestamp:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	preflight := appservice.NewControlMutationPreflightService(approvals)

	view, err := preflight.CheckControlMutationPreflight(ctx, query.ControlMutationPreflight{
		TargetKind: "media_asset_retention",
		TargetID:   "default-observed-group",
		Action:     "delete_all",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
	})
	if err != nil {
		t.Fatalf("check preflight: %v", err)
	}
	if view.Ready || view.Reason != "unsupported_control_mutation_action" ||
		len(view.SupportedActions) != 1 || view.SupportedActions[0] != "cleanup_expired" {
		t.Fatalf("unexpected unsupported media retention action view: %+v", view)
	}
	if view.SuggestedAudit != nil {
		t.Fatalf("blocked media retention preflight must not suggest audit: %+v", view.SuggestedAudit)
	}
}
