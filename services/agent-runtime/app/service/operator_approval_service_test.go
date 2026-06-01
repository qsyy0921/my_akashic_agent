package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
)

func TestOperatorApprovalServiceRecordsAndFiltersLedger(t *testing.T) {
	ctx := context.Background()
	service := appservice.NewOperatorApprovalService()
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	approval, err := service.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "agent_job_priority_plan",
		TargetID:   "priority-plan-a",
		Decision:   "approved",
		OperatorID: "qsyy",
		ExpiresAt:  now.Add(time.Hour),
		Timestamp:  now,
		Metadata:   map[string]string{"source": "test"},
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	if !approval.Active || approval.Decision != "approved" || approval.Metadata["source"] != "test" {
		t.Fatalf("unexpected approval: %+v", approval)
	}
	if _, err := service.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "agent_job_priority_plan",
		TargetID:   "priority-plan-b",
		Decision:   "rejected",
		OperatorID: "qsyy",
		Reason:     "workers unhealthy",
		Timestamp:  now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("record rejection: %v", err)
	}

	view, err := service.ListOperatorApprovals(ctx, queryFilter("agent_job_priority_plan", "approved"))
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if view.Totals["approvals"] != 1 || view.Totals["approved"] != 1 || view.SideEffect != "runtime_state_only" {
		t.Fatalf("unexpected list view: %+v", view)
	}
	if view.Approvals[0].TargetID != "priority-plan-a" {
		t.Fatalf("unexpected filtered approval: %+v", view.Approvals)
	}
}

func TestOperatorApprovalServiceChecksApprovalReadiness(t *testing.T) {
	ctx := context.Background()
	service := appservice.NewOperatorApprovalService()
	now := time.Now().UTC()

	active, err := service.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "outbound_cutover_plan",
		TargetID:   "plan-active",
		Decision:   "approved",
		OperatorID: "qsyy",
		ExpiresAt:  now.Add(time.Hour),
		Timestamp:  now,
	})
	if err != nil {
		t.Fatalf("record active approval: %v", err)
	}
	check, err := service.CheckOperatorApproval(ctx, query.OperatorApprovalCheck{
		ApprovalID: active.ApprovalID,
		TargetKind: "outbound_cutover_plan",
		TargetID:   "plan-active",
	})
	if err != nil {
		t.Fatalf("check active approval: %v", err)
	}
	if !check.Approved || check.Reason != "approval_active" || check.SideEffect != "none" || check.Approval == nil {
		t.Fatalf("unexpected active check: %+v", check)
	}

	missing, err := service.CheckOperatorApproval(ctx, query.OperatorApprovalCheck{TargetKind: "outbound_cutover_plan", TargetID: "missing"})
	if err != nil {
		t.Fatalf("check missing approval: %v", err)
	}
	if missing.Approved || missing.Reason != "approval_not_found" || missing.Blockers[0] != "approval_not_found" {
		t.Fatalf("unexpected missing check: %+v", missing)
	}

	if _, err := service.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "outbound_cutover_plan",
		TargetID:   "plan-rejected",
		Decision:   "rejected",
		OperatorID: "qsyy",
		Reason:     "smoke failed",
		Timestamp:  now,
	}); err != nil {
		t.Fatalf("record rejection: %v", err)
	}
	rejected, err := service.CheckOperatorApproval(ctx, query.OperatorApprovalCheck{TargetKind: "outbound_cutover_plan", TargetID: "plan-rejected"})
	if err != nil {
		t.Fatalf("check rejected approval: %v", err)
	}
	if rejected.Approved || rejected.Reason != "approval_not_approved" || rejected.Approval == nil {
		t.Fatalf("unexpected rejected check: %+v", rejected)
	}

	if _, err := service.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "outbound_cutover_plan",
		TargetID:   "plan-expired",
		Decision:   "approved",
		OperatorID: "qsyy",
		ExpiresAt:  now.Add(-time.Hour),
		Timestamp:  now.Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("record expired approval: %v", err)
	}
	expired, err := service.CheckOperatorApproval(ctx, query.OperatorApprovalCheck{TargetKind: "outbound_cutover_plan", TargetID: "plan-expired"})
	if err != nil {
		t.Fatalf("check expired approval: %v", err)
	}
	if expired.Approved || expired.Reason != "approval_expired" || expired.Blockers[0] != "approval_expired" {
		t.Fatalf("unexpected expired check: %+v", expired)
	}
}

func queryFilter(targetKind string, decision string) query.OperatorApprovalFilter {
	return query.OperatorApprovalFilter{TargetKind: targetKind, Decision: decision, Limit: 10}
}
