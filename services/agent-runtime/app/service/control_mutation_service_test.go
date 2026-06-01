package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
)

func TestControlMutationAuditServiceRecordsAndFiltersLedger(t *testing.T) {
	ctx := context.Background()
	service := appservice.NewControlMutationAuditService()
	now := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)

	item, err := service.RecordControlMutationAudit(ctx, command.RecordControlMutationAuditCommand{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Action:     "enable",
		Status:     "planned",
		OperatorID: "qsyy",
		ApprovalID: "approval-a",
		Timestamp:  now,
		Metadata:   map[string]string{"source": "test"},
	})
	if err != nil {
		t.Fatalf("record mutation: %v", err)
	}
	if item.Status != "planned" || item.MutationID == "" || item.Metadata["source"] != "test" {
		t.Fatalf("unexpected mutation audit: %+v", item)
	}
	if _, err := service.RecordControlMutationAudit(ctx, command.RecordControlMutationAuditCommand{
		TargetKind: "outbound_cutover",
		TargetID:   "cutover-a",
		Action:     "rollback",
		Status:     "rolled_back",
		OperatorID: "qsyy",
		ApprovalID: "approval-b",
		Reason:     "restore state-store",
		RollbackOf: item.MutationID,
		Timestamp:  now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("record rollback: %v", err)
	}

	view, err := service.ListControlMutationAudits(ctx, query.ControlMutationAuditFilter{
		TargetKind: "outbound_cutover",
		ApprovalID: "approval-a",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("list mutations: %v", err)
	}
	if view.Totals["mutations"] != 1 || view.Totals["planned"] != 1 || view.SideEffect != "runtime_state_only" {
		t.Fatalf("unexpected list view: %+v", view)
	}
	if view.Mutations[0].ApprovalID != "approval-a" {
		t.Fatalf("unexpected filtered mutation: %+v", view.Mutations)
	}
}

func TestControlMutationAuditServiceRejectsInvalidAudit(t *testing.T) {
	service := appservice.NewControlMutationAuditService()
	_, err := service.RecordControlMutationAudit(context.Background(), command.RecordControlMutationAuditCommand{
		TargetKind: "agent_job_priority",
		TargetID:   "priority-a",
		Action:     "update_limit",
		Status:     "failed",
		OperatorID: "qsyy",
		ApprovalID: "approval-a",
	})
	if err == nil {
		t.Fatal("expected failed audit without reason to fail")
	}
}
