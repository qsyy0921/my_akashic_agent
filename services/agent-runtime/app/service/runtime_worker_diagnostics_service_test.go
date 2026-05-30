package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestRuntimeWorkerDiagnosticsServiceComputesTotals(t *testing.T) {
	service := NewRuntimeWorkerDiagnosticsService(query.RuntimeWorkerDiagnosticsView{
		Workers: []query.RuntimeWorkerView{
			{Name: "agent_job_recovery", Enabled: false, Running: false},
			{Name: "outbox_delivery_worker", Enabled: true, Running: true},
			{Name: "nats_dual_read_compare", Enabled: true, Running: false},
		},
	})

	view, err := service.GetRuntimeWorkers(context.Background())
	if err != nil {
		t.Fatalf("get runtime workers: %v", err)
	}

	if view.Totals["workers"] != 3 ||
		view.Totals["enabled"] != 2 ||
		view.Totals["running"] != 1 ||
		view.Totals["disabled"] != 1 {
		t.Fatalf("unexpected totals: %#v", view.Totals)
	}
}
