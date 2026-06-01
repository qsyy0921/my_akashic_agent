package operatorapprovalstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/operatorapprovalstore"
)

func TestStorePersistsOperatorApprovals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator-approvals.json")
	store, err := operatorapprovalstore.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	approval, err := model.NewOperatorApproval(model.OperatorApprovalSpec{
		TargetKind: "outbound_cutover_plan",
		TargetID:   "plan-a",
		Decision:   "approved",
		OperatorID: "qsyy",
	}, time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveOperatorApproval(context.Background(), approval); err != nil {
		t.Fatalf("save approval: %v", err)
	}
	reopened, err := operatorapprovalstore.NewStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	items, err := reopened.ListOperatorApprovals(context.Background())
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(items) != 1 || items[0].ApprovalID != approval.ApprovalID {
		t.Fatalf("unexpected approvals: %+v", items)
	}
}
