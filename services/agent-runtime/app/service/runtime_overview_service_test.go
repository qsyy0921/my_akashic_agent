package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics(t *testing.T) {
	service := NewRuntimeOverviewService(RuntimeOverviewDeps{
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:                "nats_jetstream",
			Mode:                    "external_lease",
			ExternalQueueConfigured: true,
			ConsumerConcurrency:     8,
			MaxInFlight:             64,
			ExternalLease:           &query.QueueExternalLeaseGate{AllowExecution: false},
		}},
		DeliveryAdapters: staticRuntimeDeliveryAdapters{items: []query.DeliveryAdapterDiagnosticsView{
			{Provider: "onebot", Channel: "qq_2365524513", Enabled: true},
			{Provider: "onebot", Channel: "qq_1049511700", Enabled: false},
		}},
		SendLedger: staticRuntimeSendLedgerMetrics{view: query.SendLedgerMetricsView{
			SampledRecords:        4,
			RepeatedContentHashes: 1,
		}},
		InboxMetrics: staticRuntimeInboxMetrics{view: query.InboxMetricsView{
			SampledEvents:    9,
			ObserveOnlyTotal: 7,
			WithAttachments:  3,
		}},
		AgentJobMetrics: staticRuntimeAgentJobMetrics{view: query.AgentJobMetricsView{
			SampledJobs:   3,
			SampledEvents: 12,
			JobsByStatus: map[string]int{
				"running":       1,
				"dead_lettered": 1,
			},
			DeadLetters: query.AgentJobDeadLetterMetricsView{
				CurrentTotal: 1,
				ByType:       map[string]int{"rag_eval": 1},
			},
		}},
		OutboxMetrics: staticRuntimeOutboxMetrics{view: query.OutboxMetricsView{
			SampledDeliveries: 2,
			SampledEvents:     7,
			DeliveriesByStatus: map[string]int{
				"dispatching":   1,
				"dead_lettered": 1,
			},
			DeadLetters: query.OutboxDeadLetterMetricsView{CurrentTotal: 1},
		}},
		KnowledgeDiagnostics: staticRuntimeKnowledgeDiagnostics{view: query.KnowledgeWorkerDiagnosticsView{
			Totals: map[string]int{"checkpoints": 1, "stale_leases": 1},
			Workers: []query.KnowledgeWorkerDiagnosticView{{
				Checkpoints: []query.KnowledgeCheckpointView{{
					CheckpointID: "ragflow:qq:27234224:ds-main",
					Cursor:       102,
					Metadata:     map[string]string{"latest_source_seq": "119"},
				}},
			}},
		}},
	})

	view, err := service.Get(context.Background(), query.RuntimeOverviewFilter{
		Limit:             50,
		EventLimit:        10,
		StaleAfterSeconds: 60,
	})
	if err != nil {
		t.Fatalf("runtime overview: %v", err)
	}
	if view.Summary["delivery_adapters_enabled"] != 1 {
		t.Fatalf("unexpected adapter summary: %#v", view.Summary)
	}
	if view.Summary["worker_leases"] != 2 {
		t.Fatalf("unexpected worker lease count: %#v", view.Summary)
	}
	if view.Summary["dead_letters"] != 2 {
		t.Fatalf("unexpected dead letter count: %#v", view.Summary)
	}
	if view.Summary["checkpoint_lag_max"] != 17 {
		t.Fatalf("unexpected checkpoint lag: %#v", view.Summary)
	}
	if view.Summary["queue_backend_provider"] != "nats_jetstream" {
		t.Fatalf("unexpected queue backend: %#v", view.Summary)
	}
	assertRuntimeOverviewCardStatus(t, view.Cards, "delivery_adapters", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "queue_backend", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "send_ledger_metrics", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "agent_job_metrics", "danger")
	assertRuntimeOverviewCardStatus(t, view.Cards, "outbox_metrics", "danger")
}

func assertRuntimeOverviewCardStatus(t *testing.T, cards []query.RuntimeOverviewCardView, id string, status string) {
	t.Helper()
	for _, card := range cards {
		if card.ID == id {
			if card.Status != status {
				t.Fatalf("card %s status = %s, want %s", id, card.Status, status)
			}
			return
		}
	}
	t.Fatalf("card %s not found in %#v", id, cards)
}

type staticRuntimeQueueBackend struct {
	view query.QueueBackendView
}

func (s staticRuntimeQueueBackend) Get(context.Context) (query.QueueBackendView, error) {
	return s.view, nil
}

type staticRuntimeDeliveryAdapters struct {
	items []query.DeliveryAdapterDiagnosticsView
}

func (s staticRuntimeDeliveryAdapters) ListDeliveryAdapters(context.Context) ([]query.DeliveryAdapterDiagnosticsView, error) {
	return s.items, nil
}

type staticRuntimeSendLedgerMetrics struct {
	view query.SendLedgerMetricsView
}

func (s staticRuntimeSendLedgerMetrics) Metrics(context.Context, query.SendLedgerMetricsFilter) (query.SendLedgerMetricsView, error) {
	return s.view, nil
}

type staticRuntimeInboxMetrics struct {
	view query.InboxMetricsView
}

func (s staticRuntimeInboxMetrics) Get(context.Context, query.InboxMetricsFilter) (query.InboxMetricsView, error) {
	return s.view, nil
}

type staticRuntimeAgentJobMetrics struct {
	view query.AgentJobMetricsView
}

func (s staticRuntimeAgentJobMetrics) Get(context.Context, query.AgentJobMetricsFilter) (query.AgentJobMetricsView, error) {
	return s.view, nil
}

type staticRuntimeOutboxMetrics struct {
	view query.OutboxMetricsView
}

func (s staticRuntimeOutboxMetrics) Get(context.Context, query.OutboxMetricsFilter) (query.OutboxMetricsView, error) {
	return s.view, nil
}

type staticRuntimeKnowledgeDiagnostics struct {
	view query.KnowledgeWorkerDiagnosticsView
}

func (s staticRuntimeKnowledgeDiagnostics) Get(context.Context, query.KnowledgeWorkerDiagnosticsFilter) (query.KnowledgeWorkerDiagnosticsView, error) {
	return s.view, nil
}
