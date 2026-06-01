package controlmutationstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/controlmutationstore"
)

func TestStorePersistsControlMutationAudits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control-mutations.json")
	store, err := controlmutationstore.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	audit, err := model.NewControlMutationAudit(model.ControlMutationAuditSpec{
		TargetKind: "queue_execution_owner",
		TargetID:   "outbox",
		Action:     "switch_owner",
		Status:     "planned",
		OperatorID: "qsyy",
		ApprovalID: "approval-a",
	}, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveControlMutationAudit(context.Background(), audit); err != nil {
		t.Fatalf("save audit: %v", err)
	}
	reopened, err := controlmutationstore.NewStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	items, err := reopened.ListControlMutationAudits(context.Background())
	if err != nil {
		t.Fatalf("list audits: %v", err)
	}
	if len(items) != 1 || items[0].MutationID != audit.MutationID {
		t.Fatalf("unexpected audits: %+v", items)
	}
}
