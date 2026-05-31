package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
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
		DeliverySmoke: staticRuntimeDeliverySmoke{view: query.DeliverySmokeReadinessView{
			Ready:  false,
			Reason: "delivery_smoke_not_ready",
			Cases: []query.DeliverySmokeCaseReadinessView{
				{Name: "qq_private_text_2365524513_to_1049511700", Ready: true, Reason: "delivery_adapter_ready"},
				{
					Name:            "qq_group_text_1049511700_to_27234224",
					Ready:           false,
					Reason:          "delivery_adapter_unavailable",
					MissingChannels: []string{"qq_1049511700"},
				},
			},
			Totals:     map[string]int{"cases": 2, "ready": 1, "not_ready": 1},
			Blockers:   []string{"qq_group_text_1049511700_to_27234224:missing_channel:qq_1049511700"},
			SideEffect: "none",
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
		MediaAssetContent: staticRuntimeMediaAssetContentDiagnostics{view: query.MediaAssetContentDiagnosticsView{
			Items: []query.MediaAssetContentDiagnosticItemView{
				{AssetID: "asset:ready", Kind: "image", ContentStatus: "ready", ContentReason: "media_asset_content_ready"},
				{AssetID: "asset:forbidden", Kind: "image", ContentStatus: "forbidden", ContentReason: "media_asset_content_forbidden"},
				{AssetID: "asset:missing", Kind: "file", ContentStatus: "unavailable", ContentReason: "media_asset_content_unavailable"},
				{AssetID: "asset:disabled", Kind: "file", ContentStatus: "disabled", ContentReason: "media_asset_content_disabled"},
			},
			Totals: map[string]int{
				"assets":      4,
				"ready":       1,
				"forbidden":   1,
				"unavailable": 1,
				"disabled":    1,
				"error":       0,
			},
			SideEffect: "none",
		}},
		KnowledgePipelines: staticKnowledgePipelineDiagnostics{view: query.KnowledgePipelineDiagnosticsView{
			Totals: map[string]int{
				"targets":                            2,
				"enabled":                            2,
				"ready":                              1,
				"warning":                            0,
				"blocked":                            1,
				"receiver_connected":                 2,
				"group_memory_pending":               1,
				"rag_ingest_pending":                 1,
				"high_pressure":                      1,
				"memory_checkpoints":                 1,
				"rag_checkpoints":                    2,
				"rag_datasets":                       2,
				"rag_dataset_ingest_snapshots":       1,
				"rag_dataset_index_ready":            1,
				"rag_dataset_index_missing_snapshot": 1,
				"rag_dataset_index_empty":            1,
				"rag_dataset_index_lagging":          1,
				"rag_dataset_warning":                1,
				"rag_dataset_blocked":                1,
				"lagging":                            1,
				"stale_checkpoints":                  1,
				"stagnant":                           1,
				"expired_active_leases":              1,
				"stale_active_leases":                1,
				"stalled":                            1,
				"configured_rag_datasets":            2,
				"configured_rag_dataset_not_started": 1,
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
		KnowledgeJobPlanner: staticKnowledgeJobPlannerPreview{view: query.KnowledgeJobPlannerPreviewView{
			Timestamp:       "2026-05-31T08:02:00Z",
			Bucket:          29670242,
			IntervalSeconds: 60,
			Targets:         1,
			SkippedTargets:  2,
			TotalJobs:       3,
			GroupMemoryJobs: 1,
			RagIngestJobs:   2,
			Groups:          []string{"27234224"},
			Plans: []query.KnowledgeJobPlannerTargetPlanView{{
				TargetID: "qq:1049511700:group:27234224",
				Channel: query.ObserveTargetChannelView{
					Kind:             "qq",
					AccountID:        "1049511700",
					ConversationID:   "27234224",
					ConversationType: "group",
				},
				Datasets: []string{"ds-main", "ds-guides"},
				Jobs: []query.KnowledgeJobPlannerJobPlanView{{
					JobID:     "group_memory_extract:qq:27234224:29670242",
					JobType:   "group_memory_extract",
					AgentID:   "akashic-python-worker",
					DedupeKey: "knowledge:group_memory_extract:qq:1049511700:27234224",
				}},
			}},
			SideEffect: "none",
		}},
		KnowledgePlannerReady: staticKnowledgeJobPlannerReadiness{view: query.KnowledgeJobPlannerReadinessView{
			Ready:                  false,
			Reason:                 "knowledge_job_planner_not_ready",
			PlannerEnabled:         true,
			PlannerRunning:         false,
			KnowledgeWorkerReady:   false,
			KnowledgeWorkerActive:  0,
			KnowledgeWorkerStale:   1,
			KnowledgeWorkerFailed:  0,
			KnowledgeWorkerStopped: 1,
			Blockers:               []string{"knowledge_job_planner_worker_not_running", "knowledge_worker_unavailable"},
			Preview:                query.KnowledgeJobPlannerPreviewView{TotalJobs: 3, Targets: 1, SideEffect: "none"},
			SideEffect:             "none",
		}},
		KnowledgePlannerCutover: staticRuntimeKnowledgeJobPlannerCutover{view: query.KnowledgeJobPlannerCutoverPlanView{
			Ready:                     false,
			Decision:                  "blocked",
			DesiredAdmissionOwner:     "go_runtime_knowledge_job_planner",
			RecommendedAdmissionOwner: "go_runtime_knowledge_job_planner",
			CurrentAdmissionOwner:     "python_legacy_knowledge_enqueue",
			Readiness: query.KnowledgeJobPlannerReadinessView{
				Ready:      false,
				Reason:     "knowledge_job_planner_not_ready",
				SideEffect: "none",
			},
			RequiredChecks: []query.KnowledgeJobPlannerCutoverStep{{
				StepIndex: 1,
				Phase:     "precheck",
				Action:    "check_knowledge_job_planner_readiness",
				Method:    "GET",
				Endpoint:  "/v1/knowledge-job-planner/readiness",
			}},
			EnableSteps: []query.KnowledgeJobPlannerCutoverStep{{
				StepIndex: 1,
				Phase:     "enable",
				Action:    "enable_go_knowledge_job_planner",
				Env:       map[string]string{"AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED": "true"},
			}},
			VerificationSteps: []query.KnowledgeJobPlannerCutoverStep{{
				StepIndex: 1,
				Phase:     "verify",
				Action:    "read_runtime_workers",
				Method:    "GET",
				Endpoint:  "/v1/runtime-workers",
			}},
			RollbackSteps: []query.KnowledgeJobPlannerCutoverStep{{
				StepIndex: 1,
				Phase:     "rollback",
				Action:    "disable_go_knowledge_job_planner",
				Env:       map[string]string{"AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED": "false"},
			}},
			Blockers:   []string{"knowledge_job_planner_readiness_not_ready", "knowledge_worker_unavailable"},
			SideEffect: "none",
		}},
		AgentJobCapacityPlan: staticRuntimeAgentJobCapacityPlan{view: query.AgentJobCapacityPlanView{
			Ready:  false,
			Reason: "agent_job_capacity_attention_required",
			Summary: query.AgentJobCapacitySummaryView{
				JobTypes:                4,
				MappedJobTypes:          4,
				HighPressureJobTypes:    1,
				CapacityBlockedJobTypes: 1,
				WorkerWarningJobTypes:   2,
				ActiveWorkerJobTypes:    1,
				StaleWorkerJobTypes:     2,
				FailedWorkerJobTypes:    1,
				MaxPending:              11,
				MaxActive:               2,
				OldestPendingAgeSeconds: 1800,
			},
			Items: []query.AgentJobCapacityPlanItemView{
				{
					JobType:                 "group_memory_extract",
					Severity:                "danger",
					Action:                  "start_or_recover_python_worker",
					Recommendation:          "Start or recover a Python knowledge worker.",
					Pending:                 11,
					OldestPendingAgeSeconds: 1800,
					HighPressure:            true,
					PressureReason:          "pending>=10",
				},
				{
					JobType:        "image_generation",
					Severity:       "ok",
					Action:         "monitor",
					Recommendation: "Capacity coverage is available.",
					Active:         1,
				},
			},
			Blockers: []string{
				"agent_job_capacity_blocked",
				"agent_job_high_pressure",
				"agent_job_worker_warning",
			},
			SideEffect: "none",
		}},
		AgentJobExternalLeaseReady: staticRuntimeAgentJobExternalLeaseReadiness{view: query.AgentJobExternalLeaseReadinessView{
			Ready:                   false,
			Reason:                  "agent_job_external_lease_not_ready",
			ExternalLeaseReady:      true,
			AgentJobResultAckReady:  false,
			StrictLeaseTokenEnabled: true,
			AgentJobWorkerReady:     false,
			ExecutionOwner:          "python_ai_worker_with_nats_result_ack",
			QueueProvider:           "nats_jetstream",
			QueueMode:               "external_lease",
			ExecutionScope:          "agent_job_result_ack_only",
			AllowedWorkKinds:        []string{"agent_job"},
			Blockers: []string{
				"agent_job_result_ack_gate_missing",
				"agent_job_worker_unavailable",
				"agent_job_external_lease_smoke_missing",
			},
			SideEffect: "none",
		}},
		AgentJobExternalLeasePlan: staticRuntimeAgentJobExternalLeasePlan{view: query.AgentJobExternalLeasePlanView{
			Ready:                     false,
			Decision:                  "blocked",
			DesiredExecutionOwner:     "python_ai_worker_with_nats_result_ack",
			RecommendedExecutionOwner: "python_ai_worker_with_nats_result_ack",
			CurrentExecutionOwner:     "python_ai_worker_state_store_lease",
			Readiness: query.AgentJobExternalLeaseReadinessView{
				Ready:          false,
				Reason:         "agent_job_external_lease_not_ready",
				ExecutionOwner: "python_ai_worker_state_store_lease",
				SideEffect:     "none",
			},
			RequiredChecks: []query.AgentJobExternalLeasePlanStep{{
				StepIndex: 1,
				Phase:     "precheck",
				Action:    "check_agent_job_external_lease_readiness",
				Method:    "GET",
				Endpoint:  "/v1/agent-job-external-lease/readiness",
			}},
			EnableSteps: []query.AgentJobExternalLeasePlanStep{{
				StepIndex: 1,
				Phase:     "enable",
				Action:    "configure_agent_job_nats_result_ack",
				Env:       map[string]string{"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "true"},
			}},
			VerificationSteps: []query.AgentJobExternalLeasePlanStep{{
				StepIndex: 1,
				Phase:     "verify",
				Action:    "read_queue_backend",
				Method:    "GET",
				Endpoint:  "/v1/queue-backend",
			}},
			RollbackSteps: []query.AgentJobExternalLeasePlanStep{{
				StepIndex: 1,
				Phase:     "rollback",
				Action:    "disable_agent_job_result_ack_scope",
				Env:       map[string]string{"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "false"},
			}},
			Blockers:   []string{"agent_job_external_lease_readiness_not_ready", "agent_job_worker_coverage_blocked"},
			SideEffect: "none",
		}},
		OutboundCutoverPlan: staticRuntimeOutboundCutoverPlan{view: query.OutboundCutoverPlanView{
			Ready:                     false,
			Decision:                  "blocked",
			DesiredExecutionOwner:     "nats_external_lease",
			RecommendedExecutionOwner: "nats_external_lease",
			CurrentExecutionOwner:     "go_state_store_api",
			Readiness: query.OutboundCutoverReadinessView{
				Ready:          false,
				Reason:         "outbound_cutover_not_ready",
				ExecutionOwner: "go_state_store_api",
				SideEffect:     "none",
			},
			RequiredChecks: []query.OutboundCutoverPlanStep{{
				StepIndex: 1,
				Phase:     "precheck",
				Action:    "check_outbound_cutover_readiness",
				Method:    "POST",
				Endpoint:  "/v1/outbound-cutover/readiness",
			}},
			EnableSteps: []query.OutboundCutoverPlanStep{{
				StepIndex: 1,
				Phase:     "enable",
				Action:    "configure_nats_external_lease",
				Env:       map[string]string{"AKASHIC_QUEUE_MODE": "external_lease"},
			}},
			VerificationSteps: []query.OutboundCutoverPlanStep{{
				StepIndex: 1,
				Phase:     "verify",
				Action:    "read_queue_backend",
				Method:    "GET",
				Endpoint:  "/v1/queue-backend",
			}},
			RollbackSteps: []query.OutboundCutoverPlanStep{{
				StepIndex: 1,
				Phase:     "rollback",
				Action:    "disable_external_lease_cutover",
				Env:       map[string]string{"AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER": "false"},
			}},
			Blockers:   []string{"external_lease_outbox_not_ready", "dual_read_smoke_passed"},
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
	if view.Summary["delivery_smoke_ready"] != false ||
		view.Summary["delivery_smoke_reason"] != "delivery_smoke_not_ready" ||
		view.Summary["delivery_smoke_cases"] != 2 ||
		view.Summary["delivery_smoke_ready_cases"] != 1 ||
		view.Summary["delivery_smoke_not_ready_cases"] != 1 ||
		view.Summary["delivery_smoke_blockers"] != 1 {
		t.Fatalf("unexpected delivery smoke summary: %#v", view.Summary)
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
	if view.Summary["media_asset_content_assets"] != 4 ||
		view.Summary["media_asset_content_ready"] != 1 ||
		view.Summary["media_asset_content_forbidden"] != 1 ||
		view.Summary["media_asset_content_unavailable"] != 1 ||
		view.Summary["media_asset_content_disabled"] != 1 ||
		view.Summary["media_asset_content_error"] != 0 {
		t.Fatalf("unexpected media asset content summary: %#v", view.Summary)
	}
	if view.Summary["knowledge_pipeline_targets"] != 2 ||
		view.Summary["knowledge_pipeline_ready"] != 1 ||
		view.Summary["knowledge_pipeline_warning"] != 0 ||
		view.Summary["knowledge_pipeline_blocked"] != 1 ||
		view.Summary["knowledge_pipeline_lagging_targets"] != 1 ||
		view.Summary["knowledge_pipeline_stale_checkpoint_targets"] != 1 ||
		view.Summary["knowledge_pipeline_stagnant_targets"] != 1 ||
		view.Summary["knowledge_pipeline_expired_active_lease_targets"] != 1 ||
		view.Summary["knowledge_pipeline_stale_active_lease_targets"] != 1 ||
		view.Summary["knowledge_pipeline_rag_datasets"] != 2 ||
		view.Summary["knowledge_pipeline_rag_dataset_ingest_snapshots"] != 1 ||
		view.Summary["knowledge_pipeline_rag_dataset_index_ready"] != 1 ||
		view.Summary["knowledge_pipeline_rag_dataset_index_missing_snapshot"] != 1 ||
		view.Summary["knowledge_pipeline_rag_dataset_index_empty"] != 1 ||
		view.Summary["knowledge_pipeline_rag_dataset_index_lagging"] != 1 ||
		view.Summary["knowledge_pipeline_configured_rag_datasets"] != 2 ||
		view.Summary["knowledge_pipeline_configured_rag_dataset_not_started"] != 1 ||
		view.Summary["knowledge_pipeline_rag_dataset_warning"] != 1 ||
		view.Summary["knowledge_pipeline_rag_dataset_blocked"] != 1 ||
		view.Summary["knowledge_pipeline_stalled_targets"] != 1 {
		t.Fatalf("unexpected knowledge pipeline summary: %#v", view.Summary)
	}
	if view.Summary["knowledge_job_planner_preview_targets"] != 1 ||
		view.Summary["knowledge_job_planner_preview_skipped_targets"] != 2 ||
		view.Summary["knowledge_job_planner_preview_groups"] != 1 ||
		view.Summary["knowledge_job_planner_preview_total_jobs"] != 3 ||
		view.Summary["knowledge_job_planner_preview_group_memory_jobs"] != 1 ||
		view.Summary["knowledge_job_planner_preview_rag_ingest_jobs"] != 2 {
		t.Fatalf("unexpected knowledge planner preview summary: %#v", view.Summary)
	}
	if view.Summary["knowledge_job_planner_readiness_ready"] != false ||
		view.Summary["knowledge_job_planner_readiness_blockers"] != 2 ||
		view.Summary["knowledge_job_planner_readiness_planner_enabled"] != true ||
		view.Summary["knowledge_job_planner_readiness_planner_running"] != false ||
		view.Summary["knowledge_job_planner_readiness_worker_ready"] != false ||
		view.Summary["knowledge_job_planner_readiness_worker_active"] != 0 ||
		view.Summary["knowledge_job_planner_readiness_worker_stale"] != 1 ||
		view.Summary["knowledge_job_planner_readiness_worker_failed"] != 0 ||
		view.Summary["knowledge_job_planner_readiness_worker_stopped"] != 1 {
		t.Fatalf("unexpected knowledge planner readiness summary: %#v", view.Summary)
	}
	if view.Summary["knowledge_job_planner_cutover_plan_ready"] != false ||
		view.Summary["knowledge_job_planner_cutover_plan_decision"] != "blocked" ||
		view.Summary["knowledge_job_planner_cutover_plan_blockers"] != 2 ||
		view.Summary["knowledge_job_planner_cutover_plan_current_owner"] != "python_legacy_knowledge_enqueue" ||
		view.Summary["knowledge_job_planner_cutover_plan_desired_owner"] != "go_runtime_knowledge_job_planner" ||
		view.Summary["knowledge_job_planner_cutover_plan_recommended_owner"] != "go_runtime_knowledge_job_planner" {
		t.Fatalf("unexpected knowledge planner cutover summary: %#v", view.Summary)
	}
	if view.Summary["agent_job_capacity_ready"] != false ||
		view.Summary["agent_job_capacity_reason"] != "agent_job_capacity_attention_required" ||
		view.Summary["agent_job_capacity_blockers"] != 3 ||
		view.Summary["agent_job_capacity_job_types"] != 4 ||
		view.Summary["agent_job_capacity_mapped_job_types"] != 4 ||
		view.Summary["agent_job_capacity_high_pressure_job_types"] != 1 ||
		view.Summary["agent_job_capacity_blocked_job_types"] != 1 ||
		view.Summary["agent_job_capacity_worker_warning_job_types"] != 2 ||
		view.Summary["agent_job_capacity_active_worker_job_types"] != 1 ||
		view.Summary["agent_job_capacity_stale_worker_job_types"] != 2 ||
		view.Summary["agent_job_capacity_failed_worker_job_types"] != 1 ||
		view.Summary["agent_job_capacity_max_pending"] != 11 ||
		view.Summary["agent_job_capacity_oldest_pending_age_seconds"] != 1800 {
		t.Fatalf("unexpected agent job capacity summary: %#v", view.Summary)
	}
	if view.Summary["agent_job_external_lease_ready"] != false ||
		view.Summary["agent_job_external_lease_reason"] != "agent_job_external_lease_not_ready" ||
		view.Summary["agent_job_external_lease_blockers"] != 3 ||
		view.Summary["agent_job_external_lease_result_ack_ready"] != false ||
		view.Summary["agent_job_external_lease_worker_ready"] != false ||
		view.Summary["agent_job_external_lease_strict_token"] != true ||
		view.Summary["agent_job_external_lease_execution_owner"] != "python_ai_worker_with_nats_result_ack" ||
		view.Summary["agent_job_external_lease_execution_scope"] != "agent_job_result_ack_only" {
		t.Fatalf("unexpected agent job external lease readiness summary: %#v", view.Summary)
	}
	if view.Summary["agent_job_external_lease_plan_ready"] != false ||
		view.Summary["agent_job_external_lease_plan_decision"] != "blocked" ||
		view.Summary["agent_job_external_lease_plan_blockers"] != 2 ||
		view.Summary["agent_job_external_lease_plan_current_owner"] != "python_ai_worker_state_store_lease" ||
		view.Summary["agent_job_external_lease_plan_desired_owner"] != "python_ai_worker_with_nats_result_ack" ||
		view.Summary["agent_job_external_lease_plan_recommended_owner"] != "python_ai_worker_with_nats_result_ack" {
		t.Fatalf("unexpected agent job external lease plan summary: %#v", view.Summary)
	}
	if view.Summary["outbound_cutover_plan_ready"] != false ||
		view.Summary["outbound_cutover_plan_decision"] != "blocked" ||
		view.Summary["outbound_cutover_plan_blockers"] != 2 ||
		view.Summary["outbound_cutover_plan_current_owner"] != "go_state_store_api" ||
		view.Summary["outbound_cutover_plan_desired_owner"] != "nats_external_lease" ||
		view.Summary["outbound_cutover_plan_recommended_owner"] != "nats_external_lease" {
		t.Fatalf("unexpected outbound cutover plan summary: %#v", view.Summary)
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
	assertRuntimeOverviewCardStatus(t, view.Cards, "delivery_smoke", "danger")
	assertRuntimeOverviewCardValue(t, view.Cards, "delivery_smoke", "1/2")
	assertRuntimeOverviewCardStatus(t, view.Cards, "queue_backend", "warn")
	assertRuntimeOverviewCardValue(t, view.Cards, "queue_backend", "nats_jetstream/external_lease (first_external_mq)")
	assertRuntimeOverviewCardStatus(t, view.Cards, "external_lease_diagnostics", "danger")
	assertRuntimeOverviewCardStatus(t, view.Cards, "runtime_config", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "runtime_workers", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "observe_targets", "ok")
	assertRuntimeOverviewCardStatus(t, view.Cards, "observe_capture", "warn")
	assertRuntimeOverviewCardStatus(t, view.Cards, "media_asset_content", "danger")
	assertRuntimeOverviewCardValue(t, view.Cards, "media_asset_content", "1/4")
	assertRuntimeOverviewCardStatus(t, view.Cards, "knowledge_pipelines", "danger")
	assertRuntimeOverviewCardValue(t, view.Cards, "knowledge_pipelines", "1/2")
	assertRuntimeOverviewCardStatus(t, view.Cards, "knowledge_job_planner_preview", "ok")
	assertRuntimeOverviewCardValue(t, view.Cards, "knowledge_job_planner_preview", "3/1")
	assertRuntimeOverviewCardStatus(t, view.Cards, "knowledge_job_planner_readiness", "warn")
	assertRuntimeOverviewCardValue(t, view.Cards, "knowledge_job_planner_readiness", "blocked:2")
	assertRuntimeOverviewCardStatus(t, view.Cards, "knowledge_job_planner_cutover_plan", "warn")
	assertRuntimeOverviewCardValue(t, view.Cards, "knowledge_job_planner_cutover_plan", "blocked:2")
	assertRuntimeOverviewCardStatus(t, view.Cards, "agent_job_external_lease_readiness", "danger")
	assertRuntimeOverviewCardValue(t, view.Cards, "agent_job_external_lease_readiness", "agent_job_external_lease_not_ready:3")
	assertRuntimeOverviewCardStatus(t, view.Cards, "agent_job_external_lease_plan", "warn")
	assertRuntimeOverviewCardValue(t, view.Cards, "agent_job_external_lease_plan", "blocked:2")
	assertRuntimeOverviewCardStatus(t, view.Cards, "outbound_cutover_plan", "warn")
	assertRuntimeOverviewCardValue(t, view.Cards, "outbound_cutover_plan", "blocked:2")
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
	assertRuntimeOverviewCardStatus(t, view.Cards, "agent_job_capacity_plan", "danger")
	assertRuntimeOverviewCardValue(t, view.Cards, "agent_job_capacity_plan", "attention:3")
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
	if view.DeliverySmokeReadiness.Ready || len(view.DeliverySmokeReadiness.Blockers) != 1 {
		t.Fatalf("unexpected delivery smoke detail: %+v", view.DeliverySmokeReadiness)
	}
	if intFromMap(view.MediaAssetContent.Totals, "assets") != 4 ||
		intFromMap(view.MediaAssetContent.Totals, "forbidden") != 1 {
		t.Fatalf("unexpected media asset content detail: %+v", view.MediaAssetContent)
	}
	if view.KnowledgeJobPlanner.TotalJobs != 3 || len(view.KnowledgeJobPlanner.Plans) != 1 {
		t.Fatalf("unexpected knowledge planner preview detail: %+v", view.KnowledgeJobPlanner)
	}
	if view.KnowledgePlannerReady.Ready || len(view.KnowledgePlannerReady.Blockers) != 2 {
		t.Fatalf("unexpected knowledge planner readiness detail: %+v", view.KnowledgePlannerReady)
	}
	if view.KnowledgePlannerCutover.Decision != "blocked" ||
		view.KnowledgePlannerCutover.DesiredAdmissionOwner != "go_runtime_knowledge_job_planner" ||
		len(view.KnowledgePlannerCutover.RollbackSteps) == 0 {
		t.Fatalf("unexpected knowledge planner cutover detail: %+v", view.KnowledgePlannerCutover)
	}
	if view.AgentJobCapacityPlan.Ready ||
		view.AgentJobCapacityPlan.Summary.CapacityBlockedJobTypes != 1 ||
		len(view.AgentJobCapacityPlan.Blockers) != 3 {
		t.Fatalf("unexpected agent job capacity plan detail: %+v", view.AgentJobCapacityPlan)
	}
	if view.AgentJobExternalLease.Ready ||
		view.AgentJobExternalLease.ExecutionScope != "agent_job_result_ack_only" ||
		len(view.AgentJobExternalLease.Blockers) != 3 {
		t.Fatalf("unexpected agent job external lease readiness detail: %+v", view.AgentJobExternalLease)
	}
	if view.AgentJobExternalPlan.Decision != "blocked" ||
		view.AgentJobExternalPlan.DesiredExecutionOwner != "python_ai_worker_with_nats_result_ack" ||
		len(view.AgentJobExternalPlan.RollbackSteps) == 0 {
		t.Fatalf("unexpected agent job external lease plan detail: %+v", view.AgentJobExternalPlan)
	}
	if view.OutboundCutoverPlan.Decision != "blocked" || len(view.OutboundCutoverPlan.RollbackSteps) == 0 {
		t.Fatalf("unexpected outbound cutover plan detail: %+v", view.OutboundCutoverPlan)
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

type staticRuntimeDeliverySmoke struct {
	view query.DeliverySmokeReadinessView
}

func (s staticRuntimeDeliverySmoke) CheckDeliverySmokeReadiness(context.Context, command.CheckDeliverySmokeReadinessCommand) (query.DeliverySmokeReadinessView, error) {
	return s.view, nil
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

type staticRuntimeMediaAssetContentDiagnostics struct {
	view query.MediaAssetContentDiagnosticsView
}

func (s staticRuntimeMediaAssetContentDiagnostics) ContentDiagnostics(context.Context, query.MediaAssetContentDiagnosticsFilter) (query.MediaAssetContentDiagnosticsView, error) {
	return s.view, nil
}

type staticKnowledgePipelineDiagnostics struct {
	view query.KnowledgePipelineDiagnosticsView
}

func (s staticKnowledgePipelineDiagnostics) GetKnowledgePipelineDiagnostics(context.Context, query.KnowledgePipelineDiagnosticsFilter) (query.KnowledgePipelineDiagnosticsView, error) {
	return s.view, nil
}

type staticKnowledgeJobPlannerPreview struct {
	view query.KnowledgeJobPlannerPreviewView
}

func (s staticKnowledgeJobPlannerPreview) PreviewKnowledgeJobs(context.Context, command.PlanKnowledgeJobsCommand) (query.KnowledgeJobPlannerPreviewView, error) {
	return s.view, nil
}

type staticKnowledgeJobPlannerReadiness struct {
	view query.KnowledgeJobPlannerReadinessView
}

func (s staticKnowledgeJobPlannerReadiness) CheckKnowledgeJobPlannerReadiness(context.Context, command.CheckKnowledgeJobPlannerReadinessCommand) (query.KnowledgeJobPlannerReadinessView, error) {
	return s.view, nil
}

type staticRuntimeKnowledgeJobPlannerCutover struct {
	view query.KnowledgeJobPlannerCutoverPlanView
}

func (s staticRuntimeKnowledgeJobPlannerCutover) PlanKnowledgeJobPlannerCutover(context.Context, command.PlanKnowledgeJobPlannerCutoverCommand) (query.KnowledgeJobPlannerCutoverPlanView, error) {
	return s.view, nil
}

type staticRuntimeAgentJobExternalLeaseReadiness struct {
	view query.AgentJobExternalLeaseReadinessView
}

type staticRuntimeAgentJobCapacityPlan struct {
	view query.AgentJobCapacityPlanView
}

func (s staticRuntimeAgentJobCapacityPlan) PlanAgentJobCapacity(context.Context, command.PlanAgentJobCapacityCommand) (query.AgentJobCapacityPlanView, error) {
	return s.view, nil
}

func (s staticRuntimeAgentJobExternalLeaseReadiness) CheckAgentJobExternalLeaseReadiness(context.Context, command.CheckAgentJobExternalLeaseReadinessCommand) (query.AgentJobExternalLeaseReadinessView, error) {
	return s.view, nil
}

type staticRuntimeAgentJobExternalLeasePlan struct {
	view query.AgentJobExternalLeasePlanView
}

func (s staticRuntimeAgentJobExternalLeasePlan) PlanAgentJobExternalLease(context.Context, command.PlanAgentJobExternalLeaseCommand) (query.AgentJobExternalLeasePlanView, error) {
	return s.view, nil
}

type staticRuntimeOutboundCutoverPlan struct {
	view query.OutboundCutoverPlanView
}

func (s staticRuntimeOutboundCutoverPlan) PlanOutboundCutover(context.Context, command.PlanOutboundCutoverCommand) (query.OutboundCutoverPlanView, error) {
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
