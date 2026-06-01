package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestOperatorApprovalValidatesDecisionReasonAndExpiry(t *testing.T) {
	now := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	approval, err := model.NewOperatorApproval(model.OperatorApprovalSpec{
		TargetKind: "agent_job_priority_plan",
		TargetID:   "plan-a",
		Decision:   "ack",
		OperatorID: "qsyy",
		ExpiresAt:  now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("new approval: %v", err)
	}
	if approval.Decision != model.OperatorApprovalApproved || !approval.ActiveAt(now.Add(time.Minute)) {
		t.Fatalf("unexpected approval: %+v", approval)
	}
	if approval.ActiveAt(now.Add(2 * time.Hour)) {
		t.Fatalf("expected expired approval inactive: %+v", approval)
	}

	if _, err := model.NewOperatorApproval(model.OperatorApprovalSpec{
		TargetKind: "outbound_cutover_plan",
		TargetID:   "plan-b",
		Decision:   "rejected",
		OperatorID: "qsyy",
	}, now); err == nil {
		t.Fatal("expected rejected approval without reason to fail")
	}
}

func TestSortedOperatorApprovalsNewestFirst(t *testing.T) {
	oldItem, err := model.NewOperatorApproval(model.OperatorApprovalSpec{
		TargetKind: "a",
		TargetID:   "1",
		Decision:   "approved",
		OperatorID: "op",
	}, time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	newItem, err := model.NewOperatorApproval(model.OperatorApprovalSpec{
		TargetKind: "a",
		TargetID:   "2",
		Decision:   "approved",
		OperatorID: "op",
	}, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	sorted := model.SortedOperatorApprovals([]model.OperatorApproval{oldItem, newItem})
	if sorted[0].TargetID != "2" || sorted[1].TargetID != "1" {
		t.Fatalf("unexpected sort order: %+v", sorted)
	}
}
