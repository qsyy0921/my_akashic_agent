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
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Runtime: query.RuntimeProcessConfigView{
				Address:       ":8780",
				AddressSource: "AKASHIC_RUNTIME_ADDR",
				BotIDs:        []string{"1049511700", "2365524513"},
			},
			Delivery: query.RuntimeDeliveryConfigView{
				OneBotMissingChannels: []string{"qq_2365524513"},
			},
			Readiness:  query.RuntimeConfigReadinessView{Blockers: []string{"onebot_expected_channels_missing"}},
			SideEffect: "none",
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
		RuntimeWorkers: staticRuntimeWorkerDiagnostics{view: query.RuntimeWorkerDiagnosticsView{
			Totals: map[string]int{"workers": 3, "enabled": 2, "running": 1},
			Workers: []query.RuntimeWorkerView{
				{Name: "agent_job_recovery", Enabled: false, Running: false},
				{Name: "outbox_delivery_worker", Enabled: true, Running: true},
				{Name: "nats_dual_read_compare", Enabled: true, Running: false},
			},
		}},
		ObserveTargets: staticObserveTargets{view: query.ObserveTargetsView{
			Targets: []query.ObserveTargetView{{
				TargetID: "qq:1049511700:group:27234224",
				Channel: query.ObserveTargetChannelView{
					Kind:             "qq",
					AccountID:        "1049511700",
					ConversationID:   "27234224",
					ConversationType: "group",
				},
				ObserveOnly: true,
				Enabled:     true,
				Source:      "python_config",
			}},
			Totals:     map[string]int{"targets": 1, "enabled": 1, "observe_only": 1, "reply_allowed": 0, "groups": 1},
			SideEffect: "none",
		}},
		ReceiverStatuses: staticReceiverStatuses{view: query.ReceiverStatusesView{
			Receivers: []query.ReceiverStatusView{
				{ReceiverID: "qq:1049511700:qq", Kind: "qq", ChannelName: "qq", AccountID: "1049511700", Status: "connected"},
				{ReceiverID: "telegram:7689386159:telegram", Kind: "telegram", ChannelName: "telegram", AccountID: "7689386159", Status: "suspended"},
			},
			Totals:     map[string]int{"receivers": 2, "connected": 1, "suspended": 1, "failed": 0, "qq": 1, "telegram": 1},
			SideEffect: "none",
		}},
		ReceiverLeases: staticReceiverLeases{view: query.ReceiverLeasesView{
			Leases: []query.ReceiverLeaseView{{
				ReceiverID:        "telegram:7689386159:telegram",
				Kind:              "telegram",
				ChannelName:       "telegram",
				AccountID:         "7689386159",
				HolderID:          "python:1",
				LeaseTokenPresent: true,
				Active:            true,
			}},
			Totals:     map[string]int{"leases": 1, "active": 1, "expired": 0, "telegram": 1},
			SideEffect: "runtime_state_only",
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
	if view.Summary["runtime_workers_running"] != 1 {
		t.Fatalf("unexpected runtime worker summary: %#v", view.Summary)
	}
	if view.Summary["runtime_config_blockers"] != 1 || view.Summary["runtime_config_onebot_missing"] != 1 {
		t.Fatalf("unexpected runtime config summary: %#v", view.Summary)
	}
	if view.Summary["observe_targets"] != 1 || view.Summary["observe_target_groups"] != 1 {
		t.Fatalf("unexpected observe target summary: %#v", view.Summary)
	}
	if view.Summary["receiver_statuses"] != 2 || view.Summary["receiver_status_suspended"] != 1 {
		t.Fatalf("unexpected receiver status summary: %#v", view.Summary)
	}
	if view.Summary["receiver_leases"] != 1 || view.Summary["receiver_leases_active"] != 1 {
		t.Fatalf("unexpected receiver lease summary: %#v", view.Summary)
	}
	assertRuntimeOverviewCardStatus(t, view.Cards, "delivery_adapters", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "queue_backend", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "runtime_config", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "runtime_workers", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "observe_targets", "ok")
	assertRuntimeOverviewCardStatus(t, view.Cards, "receiver_statuses", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "receiver_leases", "ok")
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

type staticRuntimeConfig struct {
	view query.RuntimeConfigView
}

func (s staticRuntimeConfig) GetRuntimeConfig(context.Context) (query.RuntimeConfigView, error) {
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

type staticRuntimeWorkerDiagnostics struct {
	view query.RuntimeWorkerDiagnosticsView
}

func (s staticRuntimeWorkerDiagnostics) GetRuntimeWorkers(context.Context) (query.RuntimeWorkerDiagnosticsView, error) {
	return s.view, nil
}

type staticObserveTargets struct {
	view query.ObserveTargetsView
}

func (s staticObserveTargets) ListObserveTargets(context.Context) (query.ObserveTargetsView, error) {
	return s.view, nil
}

type staticReceiverStatuses struct {
	view query.ReceiverStatusesView
}

func (s staticReceiverStatuses) ListReceiverStatuses(context.Context) (query.ReceiverStatusesView, error) {
	return s.view, nil
}

type staticReceiverLeases struct {
	view query.ReceiverLeasesView
}

func (s staticReceiverLeases) ListReceiverLeases(context.Context) (query.ReceiverLeasesView, error) {
	return s.view, nil
}
