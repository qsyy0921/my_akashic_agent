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

func queryFilter(targetKind string, decision string) query.OperatorApprovalFilter {
	return query.OperatorApprovalFilter{TargetKind: targetKind, Decision: decision, Limit: 10}
}
