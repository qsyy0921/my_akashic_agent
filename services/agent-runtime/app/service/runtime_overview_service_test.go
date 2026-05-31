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
			OutboxExecutionOwner:    "nats_external_lease",
			AgentJobExecutionOwner:  "python_ai_worker_with_nats_result_ack",
			SelectedProviderCapability: &query.QueueProviderCapabilityView{
				Provider:                    "nats_jetstream",
				Status:                      "selected",
				Recommended:                 true,
				RecommendedPhase:            "first_external_mq",
				Implemented:                 true,
				SupportsExternalLease:       true,
				SupportsAgentJobResultAck:   true,
				SupportsConcurrentConsumers: true,
				SupportsDelayedNack:         true,
			},
			ExternalLease: &query.QueueExternalLeaseGate{
				AllowExecution: false,
				Diagnostics: &query.QueueExternalLeaseDiagnostics{
					Enabled:       true,
					ExecutedTotal: 5,
					ErrorTotal:    1,
					Dispositions: []query.QueueExternalLeaseCounter{
						{Name: "ack", Count: 2},
						{Name: "nack", Count: 2},
						{Name: "term", Count: 1},
					},
				},
			},
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
		InboundDedupe: staticRuntimeInboundDedupeMetrics{view: query.InboundDedupeMetricsView{
			SampledRecords:     2,
			ActiveRecords:      2,
			DuplicateRecords:   1,
			SeenTotal:          3,
			DuplicateSeenTotal: 1,
			Scopes: []query.InboundDedupeScopeMetricsView{{
				Scope:              "qq:qq_2365524513:2365524513",
				Records:            1,
				ActiveRecords:      1,
				DuplicateRecords:   1,
				SeenTotal:          2,
				DuplicateSeenTotal: 1,
			}},
			SideEffect: "none",
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
			Pressure: query.AgentJobPressureMetricsView{
				JobTypes:                4,
				HighPressureJobTypes:    1,
				MaxPending:              11,
				MaxActive:               2,
				OldestPendingAgeSeconds: 1800,
				ByType: []query.AgentJobTypePressureView{
					{
						JobType:                 "group_memory_extract",
						Pending:                 11,
						Active:                  0,
						OldestPendingAgeSeconds: 1800,
						HighPressure:            true,
						PressureReason:          "pending>=10",
					},
					{
						JobType:        "rag_ingest",
						Pending:        1,
						Active:         0,
						HighPressure:   false,
						PressureReason: "",
					},
					{
						JobType:        "image_generation",
						Pending:        0,
						Active:         1,
						HighPressure:   false,
						PressureReason: "",
					},
					{
						JobType:        "rag_eval",
						Pending:        1,
						Active:         0,
						HighPressure:   false,
						PressureReason: "",
					},
				},
			},
		}},
		AgentWorkers: staticRuntimeAgentWorkers{view: query.AgentWorkerStatusesView{
			Workers: []query.AgentWorkerStatusView{
				{
					WorkerID:   "knowledge-worker-1",
					WorkerType: "knowledge",
					Status:     "stopped",
					Stale:      true,
					UpdatedAt:  "2026-05-31T11:20:00Z",
				},
				{
					WorkerID:    "image-worker-1",
					WorkerType:  "image_generation",
					Status:      "running",
					LeaseActive: true,
					UpdatedAt:   "2026-05-31T11:59:00Z",
				},
				{
					WorkerID:   "rag-eval-worker-1",
					WorkerType: "rag_eval",
					Status:     "failed",
					UpdatedAt:  "2026-05-31T11:58:00Z",
				},
			},
			Totals: map[string]int{
				"workers":          3,
				"starting":         0,
				"idle":             0,
				"running":          1,
				"failed":           1,
				"stopped":          1,
				"stale":            1,
				"knowledge":        1,
				"image_generation": 1,
				"rag_eval":         1,
				"outbox_delivery":  0,
				"other_type":       0,
			},
			SideEffect: "none",
		}},
		OutboxMetrics: staticRuntimeOutboxMetrics{view: query.OutboxMetricsView{
			SampledDeliveries: 2,
			SampledEvents:     7,
			DeliveriesByStatus: map[string]int{
				"dispatching":   1,
				"dead_lettered": 1,
			},
			DeadLetters: query.OutboxDeadLetterMetricsView{CurrentTotal: 1},
			Pressure: query.OutboxPressureMetricsView{
				Accounts:             2,
				HighPressureAccounts: 1,
				MaxActive:            12,
				MaxQueued:            10,
				ByAccount: []query.OutboxAccountPressureView{{
					AccountKey:     "qq:1049511700",
					ChannelKind:    "qq",
					AccountID:      "1049511700",
					Queued:         10,
					Dispatching:    2,
					Active:         12,
					HighPressure:   true,
					PressureReason: "active>=10",
				}},
			},
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
		ObserveCapture: staticObserveCaptureDiagnostics{view: query.ObserveCaptureDiagnosticsView{
			Targets: []query.ObserveCaptureTargetDiagnosticsView{{
				TargetID:          "qq:1049511700:group:27234224",
				Enabled:           true,
				ObserveOnly:       true,
				ReceiverConnected: true,
				Status:            "warn",
				TextEvents:        1,
				ImageAssets:       1,
			}},
			Totals:     map[string]int{"targets": 1, "enabled": 1, "ready": 0, "warning": 1, "blocked": 0, "text_covered": 1, "image_covered": 1, "file_covered": 0},
			SideEffect: "none",
		}},
		KnowledgePipelines: staticKnowledgePipelineDiagnostics{view: query.KnowledgePipelineDiagnosticsView{
			Totals: map[string]int{
				"targets":              2,
				"enabled":              2,
				"ready":                1,
				"warning":              0,
				"blocked":              1,
				"receiver_connected":   2,
				"group_memory_pending": 1,
				"rag_ingest_pending":   1,
				"high_pressure":        1,
				"memory_checkpoints":   1,
				"rag_checkpoints":      2,
			},
			Pipelines: []query.KnowledgePipelineView{
				{
					TargetID: "qq:1049511700:group:27234224",
					Channel: query.ObserveTargetChannelView{
						Kind:             "qq",
						AccountID:        "1049511700",
						ConversationID:   "27234224",
						ConversationType: "group",
					},
					Enabled:           true,
					ObserveOnly:       true,
					CaptureStatus:     "ok",
					ReceiverConnected: true,
					Status:            "ok",
					Reasons:           []string{"pipeline_ready"},
				},
				{
					TargetID: "qq:1049511700:group:3219982",
					Channel: query.ObserveTargetChannelView{
						Kind:             "qq",
						AccountID:        "1049511700",
						ConversationID:   "3219982",
						ConversationType: "group",
					},
					Enabled:           true,
					ObserveOnly:       true,
					CaptureStatus:     "warn",
					ReceiverConnected: true,
					Status:            "blocked",
					Reasons:           []string{"rag_ingest_worker_blocked"},
				},
			},
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
		SchedulerJobs: staticSchedulerJobDiagnostics{view: query.SchedulerJobDiagnosticsView{
			SampledJobs:  2,
			EnabledJobs:  2,
			OverdueJobs:  1,
			DueSoonJobs:  1,
			InstantJobs:  1,
			SoftJobs:     1,
			JobsByStatus: map[string]int{"overdue": 1, "due_soon": 1},
			SideEffect:   "none",
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
	if view.Summary["queue_outbox_execution_owner"] != "nats_external_lease" ||
		view.Summary["queue_agent_job_execution_owner"] != "python_ai_worker_with_nats_result_ack" {
		t.Fatalf("unexpected queue execution owners: %#v", view.Summary)
	}
	if view.Summary["queue_provider_status"] != "selected" ||
		view.Summary["queue_provider_recommended"] != true ||
		view.Summary["queue_provider_recommended_phase"] != "first_external_mq" ||
		view.Summary["queue_provider_implemented"] != true ||
		view.Summary["queue_provider_supports_concurrent_consumers"] != true ||
		view.Summary["queue_provider_supports_delayed_nack"] != true ||
		view.Summary["queue_provider_supports_external_lease"] != true ||
		view.Summary["queue_provider_supports_agent_job_result_ack"] != true {
		t.Fatalf("unexpected queue provider capability summary: %#v", view.Summary)
	}
	if view.Summary["queue_external_lease_executed_total"] != 5 ||
		view.Summary["queue_external_lease_error_total"] != 1 ||
		view.Summary["queue_external_lease_ack"] != 2 ||
		view.Summary["queue_external_lease_nack"] != 2 ||
		view.Summary["queue_external_lease_term"] != 1 {
		t.Fatalf("unexpected external lease summary: %#v", view.Summary)
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
	if view.Summary["observe_capture_targets"] != 1 || view.Summary["observe_capture_warning"] != 1 || view.Summary["observe_capture_file"] != 0 {
		t.Fatalf("unexpected observe capture summary: %#v", view.Summary)
	}
	if view.Summary["knowledge_pipeline_targets"] != 2 ||
		view.Summary["knowledge_pipeline_ready"] != 1 ||
		view.Summary["knowledge_pipeline_warning"] != 0 ||
		view.Summary["knowledge_pipeline_blocked"] != 1 {
		t.Fatalf("unexpected knowledge pipeline summary: %#v", view.Summary)
	}
	if view.Summary["receiver_statuses"] != 2 || view.Summary["receiver_status_suspended"] != 1 {
		t.Fatalf("unexpected receiver status summary: %#v", view.Summary)
	}
	if view.Summary["receiver_leases"] != 1 || view.Summary["receiver_leases_active"] != 1 {
		t.Fatalf("unexpected receiver lease summary: %#v", view.Summary)
	}
	if view.Summary["scheduler_jobs"] != 2 || view.Summary["scheduler_jobs_overdue"] != 1 {
		t.Fatalf("unexpected scheduler summary: %#v", view.Summary)
	}
	if view.Summary["inbound_dedupe_records"] != 2 || view.Summary["inbound_dedupe_duplicate_seen_total"] != 1 {
		t.Fatalf("unexpected inbound dedupe summary: %#v", view.Summary)
	}
	if view.Summary["agent_job_pressure_job_types"] != 4 ||
		view.Summary["agent_job_pressure_high_job_types"] != 1 ||
		view.Summary["agent_job_pressure_max_pending"] != 11 ||
		view.Summary["agent_job_pressure_max_active"] != 2 ||
		view.Summary["agent_job_pressure_oldest_pending_age_seconds"] != 1800 {
		t.Fatalf("unexpected agent job pressure summary: %#v", view.Summary)
	}
	if view.Summary["agent_job_worker_coverage_job_types"] != 4 ||
		view.Summary["agent_job_worker_coverage_uncovered_job_types"] != 3 ||
		view.Summary["agent_job_worker_coverage_stale_job_types"] != 2 ||
		view.Summary["agent_job_worker_coverage_failed_job_types"] != 1 {
		t.Fatalf("unexpected agent job worker coverage summary: %#v", view.Summary)
	}
	if view.Summary["outbox_pressure_accounts"] != 2 ||
		view.Summary["outbox_pressure_high_accounts"] != 1 ||
		view.Summary["outbox_pressure_max_active"] != 12 ||
		view.Summary["outbox_pressure_max_queued"] != 10 {
		t.Fatalf("unexpected outbox pressure summary: %#v", view.Summary)
	}
	assertRuntimeOverviewCardStatus(t, view.Cards, "delivery_adapters", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "queue_backend", "warn")
	assertRuntimeOverviewCardValue(t, view.Cards, "queue_backend", "nats_jetstream/external_lease (first_external_mq)")
	assertRuntimeOverviewCardStatus(t, view.Cards, "external_lease_diagnostics", "danger")
	assertRuntimeOverviewCardStatus(t, view.Cards, "runtime_config", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "runtime_workers", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "observe_targets", "ok")
	assertRuntimeOverviewCardStatus(t, view.Cards, "observe_capture", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "knowledge_pipelines", "danger")
	assertRuntimeOverviewCardValue(t, view.Cards, "knowledge_pipelines", "1/2")
	assertRuntimeOverviewCardStatus(t, view.Cards, "receiver_statuses", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "receiver_leases", "ok")
	assertRuntimeOverviewCardStatus(t, view.Cards, "scheduler_jobs", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "send_ledger_metrics", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "inbound_dedupe_metrics", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "agent_job_metrics", "danger")
	assertRuntimeOverviewCardStatus(t, view.Cards, "agent_job_pressure", "warn")
	assertRuntimeOverviewCardValue(t, view.Cards, "agent_job_pressure", "1/4")
	assertRuntimeOverviewCardStatus(t, view.Cards, "agent_job_worker_coverage", "danger")
	assertRuntimeOverviewCardValue(t, view.Cards, "agent_job_worker_coverage", "3/4")
	assertRuntimeOverviewCardStatus(t, view.Cards, "outbox_metrics", "danger")
	assertRuntimeOverviewCardStatus(t, view.Cards, "outbox_pressure", "warn")
	if len(view.AgentJobWorkerCoverage) != 4 {
		t.Fatalf("unexpected agent job worker coverage items: %+v", view.AgentJobWorkerCoverage)
	}
	if view.AgentJobWorkerCoverage[0].JobType != "group_memory_extract" ||
		view.AgentJobWorkerCoverage[0].CoverageStatus != "danger" ||
		view.AgentJobWorkerCoverage[0].CoverageReason != "high_pressure_no_active_worker" {
		t.Fatalf("unexpected first agent job worker coverage item: %+v", view.AgentJobWorkerCoverage[0])
	}
	if view.AgentJobWorkerCoverage[2].JobType != "image_generation" ||
		view.AgentJobWorkerCoverage[2].CoverageStatus != "ok" ||
		view.AgentJobWorkerCoverage[2].ActiveWorkers != 1 ||
		view.AgentJobWorkerCoverage[2].RunningWorkers != 1 {
		t.Fatalf("unexpected image coverage item: %+v", view.AgentJobWorkerCoverage[2])
	}
	if view.AgentJobWorkerCoverage[3].JobType != "rag_eval" ||
		view.AgentJobWorkerCoverage[3].CoverageStatus != "warn" ||
		view.AgentJobWorkerCoverage[3].CoverageReason != "failed_worker_present" {
		t.Fatalf("unexpected rag_eval coverage item: %+v", view.AgentJobWorkerCoverage[3])
	}
}

func assertRuntimeOverviewCardValue(t *testing.T, cards []query.RuntimeOverviewCardView, id string, value string) {
	t.Helper()
	for _, card := range cards {
		if card.ID == id {
			if card.Value != value {
				t.Fatalf("card %s value = %#v, want %q", id, card.Value, value)
			}
			return
		}
	}
	t.Fatalf("card %s not found in %#v", id, cards)
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

type staticRuntimeInboundDedupeMetrics struct {
	view query.InboundDedupeMetricsView
}

func (s staticRuntimeInboundDedupeMetrics) Metrics(context.Context, query.InboundDedupeMetricsFilter) (query.InboundDedupeMetricsView, error) {
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

type staticRuntimeAgentWorkers struct {
	view query.AgentWorkerStatusesView
}

func (s staticRuntimeAgentWorkers) ListAgentWorkerStatuses(context.Context, query.AgentWorkerStatusFilter) (query.AgentWorkerStatusesView, error) {
	return s.view, nil
}

type staticObserveTargets struct {
	view query.ObserveTargetsView
}

func (s staticObserveTargets) ListObserveTargets(context.Context) (query.ObserveTargetsView, error) {
	return s.view, nil
}

type staticObserveCaptureDiagnostics struct {
	view query.ObserveCaptureDiagnosticsView
}

func (s staticObserveCaptureDiagnostics) GetObserveCaptureDiagnostics(context.Context, query.ObserveCaptureDiagnosticsFilter) (query.ObserveCaptureDiagnosticsView, error) {
	return s.view, nil
}

type staticKnowledgePipelineDiagnostics struct {
	view query.KnowledgePipelineDiagnosticsView
}

func (s staticKnowledgePipelineDiagnostics) GetKnowledgePipelineDiagnostics(context.Context, query.KnowledgePipelineDiagnosticsFilter) (query.KnowledgePipelineDiagnosticsView, error) {
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

type staticSchedulerJobDiagnostics struct {
	view query.SchedulerJobDiagnosticsView
}

func (s staticSchedulerJobDiagnostics) GetSchedulerJobDiagnostics(context.Context, query.SchedulerJobDiagnosticsFilter) (query.SchedulerJobDiagnosticsView, error) {
	return s.view, nil
}
