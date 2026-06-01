package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestControlMutationAuditValidatesStatusApprovalAndRollback(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	item, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Action:     "enable",
		Status:     "applied",
		OperatorID: "qsyy",
		ApprovalID: "approval-a",
		Metadata:   map[string]string{"source": "test"},
	}, now)
	if err != nil {
		t.Fatalf("new audit: %v", err)
	}
	if item.Status != model.ControlMutationApplied || item.MutationID == "" || item.Metadata["source"] != "test" {
		t.Fatalf("unexpected audit: %+v", item)
	}

	if _, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-b",
		Action:     "enable",
		Status:     "applied",
		OperatorID: "qsyy",
	}, now); err == nil {
		t.Fatal("expected missing approval_id to fail")
	}

	if _, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-c",
		Action:     "enable",
		Status:     "failed",
		OperatorID: "qsyy",
		ApprovalID: "approval-c",
	}, now); err == nil {
		t.Fatal("expected failed status without reason to fail")
	}

	if _, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-d",
		Action:     "rollback",
		Status:     "rolled_back",
		OperatorID: "qsyy",
		ApprovalID: "approval-d",
		Reason:     "restore state-store",
	}, now); err == nil {
		t.Fatal("expected rolled_back without rollback metadata to fail")
	}
}

func TestSortedControlMutationAuditsNewestFirst(t *testing.T) {
	oldItem, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		TargetKind: "agent_job_priority",
		TargetID:   "priority-a",
		Action:     "update_limit",
		Status:     "planned",
		OperatorID: "qsyy",
		ApprovalID: "approval-a",
	}, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	newItem, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		TargetKind: "agent_job_priority",
		TargetID:   "priority-b",
		Action:     "update_limit",
		Status:     "planned",
		OperatorID: "qsyy",
		ApprovalID: "approval-b",
	}, time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	sorted := model.SortedControlMutationAudits([]model.ControlMutationAudit{oldItem, newItem})
	if sorted[0].MutationID != newItem.MutationID {
		t.Fatalf("expected newest first: %+v", sorted)
	}
}
