from __future__ import annotations

import json
from typing import Any
from urllib.parse import parse_qs, urlparse

from fastapi.testclient import TestClient

from bootstrap.dashboard_api import create_dashboard_app
from plugins.runtime_overview.dashboard import RuntimeOverviewDashboardReader


class _MemoryAdmin:
    def describe(self):
        class _Desc:
            name = "default"

        return _Desc()

    def close(self) -> None:
        return None


def test_runtime_overview_dashboard_plugin_aggregates_runtime_state(
    monkeypatch,
    tmp_path,
) -> None:
    jobs = [
        {
            "job_id": "group_memory_extract:qq:27234224:1",
            "job_type": "group_memory_extract",
            "agent_id": "worker-1",
            "route": {
                "kind": "qq",
                "account_id": "2365524513",
                "conversation_id": "27234224",
                "conversation_type": "group",
            },
            "status": "running",
            "attempts": 1,
            "max_attempts": 3,
            "lease_owner": "knowledge-worker",
            "lease_expires_at": "2020-01-01T00:00:00Z",
            "result": {},
            "metadata": {},
            "created_at": "2026-05-30T08:00:00Z",
            "updated_at": "2026-05-30T08:01:00Z",
        },
        {
            "job_id": "rag_eval:fixture:failed",
            "job_type": "rag_eval",
            "agent_id": "eval-worker",
            "status": "succeeded",
            "attempts": 1,
            "max_attempts": 1,
            "lease_owner": "",
            "lease_expires_at": "",
            "result": {"passed": False, "faithfulness": 0.42},
            "metadata": {},
            "created_at": "2026-05-30T08:10:00Z",
            "updated_at": "2026-05-30T08:12:00Z",
        },
        {
            "job_id": "rag_ingest:qq:3219982:dead",
            "job_type": "rag_ingest",
            "agent_id": "worker-2",
            "status": "dead_lettered",
            "attempts": 3,
            "max_attempts": 3,
            "lease_owner": "",
            "lease_expires_at": "",
            "error_message": "ragflow timeout",
            "metadata": {},
            "created_at": "2026-05-30T08:20:00Z",
            "updated_at": "2026-05-30T08:22:00Z",
        },
    ]
    outbox = [
        {
            "event_id": "outbox:dispatching",
            "channel": {
                "kind": "telegram",
                "account_id": "bot",
                "conversation_id": "123",
                "conversation_type": "private",
            },
            "status": "dispatching",
            "attempts": 1,
            "max_attempts": 3,
            "lease_owner": "outbox-worker",
            "lease_expires_at": "2030-01-01T00:00:00Z",
            "created_at": "2030-01-01T00:00:00Z",
            "updated_at": "2030-01-01T00:00:30Z",
        },
        {
            "event_id": "outbox:dead",
            "channel": {
                "kind": "qq",
                "account_id": "2365524513",
                "conversation_id": "1049511700",
                "conversation_type": "private",
            },
            "status": "dead_lettered",
            "attempts": 3,
            "max_attempts": 3,
            "error_kind": "route_error",
            "error_message": "missing adapter",
            "created_at": "2026-05-30T08:40:00Z",
            "updated_at": "2026-05-30T08:41:00Z",
        },
    ]
    checkpoints = [
        {
            "checkpoint_id": "ragflow:qq:27234224:ds-main",
            "cursor": 102,
            "updated_at": "2026-05-30T08:45:00Z",
            "metadata": {
                "source": "qq",
                "group_id": "27234224",
                "dataset_id": "ds-main",
                "latest_source_seq": 119,
            },
        }
    ]
    events = [
        {
            "event_id": "evt-1",
            "job_id": "group_memory_extract:qq:27234224:1",
            "job_type": "group_memory_extract",
            "event_type": "leased",
            "status": "running",
            "attempt": 1,
            "max_attempts": 3,
            "lease_owner": "knowledge-worker",
            "occurred_at": "2026-05-30T08:01:00Z",
        },
        {
            "event_id": "evt-2",
            "job_id": "rag_eval:fixture:failed",
            "job_type": "rag_eval",
            "event_type": "succeeded",
            "status": "succeeded",
            "attempt": 1,
            "max_attempts": 1,
            "occurred_at": "2026-05-30T08:12:00Z",
        },
    ]
    outbox_events = [
        {
            "event_id": "outbox-event:outbox:dispatching:leased:1",
            "delivery_id": "outbox:dispatching",
            "channel": {
                "kind": "telegram",
                "account_id": "bot",
                "conversation_id": "123",
                "conversation_type": "private",
            },
            "event_type": "leased",
            "status": "dispatching",
            "attempt": 1,
            "max_attempts": 3,
            "lease_owner": "outbox-worker",
            "occurred_at": "2026-05-30T08:31:00Z",
        }
    ]
    diagnostics = {
        "generated_at": "2026-05-30T08:50:00Z",
        "stale_after_seconds": 60,
        "totals": {
            "jobs": 3,
            "checkpoints": 1,
            "stale_leases": 1,
            "leaseable_jobs": 0,
        },
        "workers": [
            {
                "job_type": "rag_ingest",
                "checkpoints": checkpoints,
            }
        ],
    }
    delivery_adapters = [
        {
            "provider": "onebot",
            "channel": "qq_2365524513",
            "transport": "websocket",
            "enabled": True,
            "endpoint_configured": True,
            "access_token_configured": True,
            "endpoint": "ws://127.0.0.1:3002",
        },
        {
            "provider": "onebot",
            "channel": "qq_1049511700",
            "transport": "websocket",
            "enabled": False,
            "endpoint_configured": False,
            "access_token_configured": False,
        },
    ]
    queue_backend = {
        "provider": "nats_jetstream",
        "mode": "external_lease",
        "migration_phase": "external_lease_gate",
        "stream": "AKASHIC_WORK",
        "subject_prefix": "akashic.work",
        "external_queue_configured": True,
        "external_queue_active": False,
        "state_store_authoritative": True,
        "lease_owner": "go_state_store",
        "consumer_model": "goroutine_worker_pool",
        "consumer_concurrency": 8,
        "max_in_flight": 64,
        "outbox_queue_source": "outbox_state_store",
        "agent_job_queue_source": "agent_job_state_store",
        "dsn_configured": True,
        "recommended_first_backend": "nats_jetstream",
        "external_lease": {
            "enabled": True,
            "allow_execution": False,
            "gate_state": "blocked",
            "execution_scope": "none",
            "blockers": ["explicit_cutover", "state_lease_workers_disabled"],
        },
    }
    send_ledger_metrics = {
        "sampled_records": 4,
        "unique_bots": 2,
        "unique_conversations": 2,
        "unique_content_hashes": 3,
        "repeated_content_hashes": 1,
        "records_by_bot": {
            "1049511700": {
                "total": 3,
                "unique_conversations": 1,
                "unique_content_hashes": 2,
                "latest_timestamp": "2026-05-30T08:49:00Z",
            }
        },
        "records_by_conversation": {
            "1049511700/2365524513": {
                "from_bot_id": "1049511700",
                "conversation_id": "2365524513",
                "total": 3,
                "unique_content_hashes": 2,
                "repeated_hashes": 1,
                "latest_timestamp": "2026-05-30T08:49:00Z",
            }
        },
        "repeated_hashes": [
            {
                "from_bot_id": "1049511700",
                "conversation_id": "2365524513",
                "content_hash": "hash-image",
                "count": 2,
                "latest_timestamp": "2026-05-30T08:49:00Z",
            }
        ],
        "recent": [
            {
                "from_bot_id": "1049511700",
                "conversation_id": "2365524513",
                "content_hash": "hash-image",
                "timestamp": "2026-05-30T08:49:00Z",
            }
        ],
    }
    inbox_metrics = {
        "sampled_events": 9,
        "observe_only_total": 7,
        "reply_eligible_total": 2,
        "with_attachments": 3,
        "attachment_count": 4,
        "unique_senders": 5,
        "events_by_channel_kind": {
            "qq": {
                "total": 7,
                "observe_only": 7,
                "reply_eligible": 0,
                "with_attachments": 3,
                "attachment_count": 4,
                "by_conversation_type": {"group": 7},
            }
        },
        "events_by_conversation": {
            "qq/1049511700/group/27234224": {
                "channel": {
                    "kind": "qq",
                    "account_id": "1049511700",
                    "conversation_id": "27234224",
                    "conversation_type": "group",
                },
                "total": 7,
                "observe_only": 7,
                "reply_eligible": 0,
                "with_attachments": 3,
                "attachment_count": 4,
                "unique_senders": 5,
                "sequenced_events": 7,
                "latest_seq": 119,
                "latest_received_at": "2026-05-30T08:49:00Z",
            }
        },
        "events_by_decision_action": {"allow": 9},
        "events_by_sender_kind": {"human": 9},
        "recent": [
            {
                "event_id": "qq:1049511700:group:27234224:119",
                "channel": {
                    "kind": "qq",
                    "account_id": "1049511700",
                    "conversation_id": "27234224",
                    "conversation_type": "group",
                },
                "sender_id": "2948770636",
                "sender_kind": "human",
                "decision_action": "allow",
                "observe_only": True,
                "attachment_count": 1,
                "seq": 119,
                "received_at": "2026-05-30T08:49:00Z",
            }
        ],
    }
    inbound_dedupe_metrics = {
        "sampled_records": 2,
        "active_records": 2,
        "expired_records": 0,
        "duplicate_records": 1,
        "seen_total": 3,
        "duplicate_seen_total": 1,
        "scopes": [
            {
                "scope": "qq:qq_2365524513:2365524513",
                "records": 1,
                "active_records": 1,
                "expired_records": 0,
                "duplicate_records": 1,
                "seen_total": 2,
                "duplicate_seen_total": 1,
                "latest_seen_at": "2026-05-31T11:00:01Z",
            }
        ],
        "totals": {
            "records": 2,
            "active_records": 2,
            "expired_records": 0,
            "duplicate_records": 1,
            "seen_total": 3,
            "duplicate_seen_total": 1,
            "scopes": 2,
        },
        "notes": ["side_effect=none"],
        "side_effect": "none",
    }
    agent_job_metrics = {
        "sampled_jobs": 3,
        "sampled_events": 12,
        "jobs_by_status": {"running": 1, "succeeded": 1, "dead_lettered": 1},
        "jobs_by_type": {
            "rag_ingest": {"total": 1, "by_status": {"dead_lettered": 1}},
        },
        "throughput": {
            "events_by_type": {"created": 3, "succeeded": 1, "failed": 1},
            "created": 3,
            "leased": 2,
            "renewed": 0,
            "running": 1,
            "succeeded": 1,
            "failed": 1,
            "retry": 0,
            "lease_expired": 0,
            "cancelled": 0,
            "terminal_events": 2,
        },
        "dead_letters": {
            "current_total": 1,
            "by_type": {"rag_ingest": 1},
            "recent": [
                {
                    "job_id": "rag_ingest:qq:3219982:dead",
                    "job_type": "rag_ingest",
                    "event_type": "failed",
                    "status": "dead_lettered",
                    "attempt": 3,
                    "max_attempts": 3,
                    "occurred_at": "2026-05-30T08:22:00Z",
                }
            ],
        },
    }
    outbox_metrics = {
        "sampled_deliveries": 2,
        "sampled_events": 7,
        "deliveries_by_status": {"dispatching": 1, "dead_lettered": 1},
        "deliveries_by_channel_kind": {
            "qq": {"total": 1, "by_status": {"dead_lettered": 1}},
            "telegram": {"total": 1, "by_status": {"dispatching": 1}},
        },
        "throughput": {
            "events_by_type": {"leased": 2, "succeeded": 1, "failed": 1},
            "queued": 2,
            "leased": 2,
            "dispatching": 1,
            "succeeded": 1,
            "failed": 1,
            "retry": 0,
            "dead_lettered": 1,
            "terminal_events": 2,
        },
        "dead_letters": {
            "current_total": 1,
            "by_channel_kind": {"qq": 1},
            "recent": [
                {
                    "delivery_id": "outbox:dead",
                    "channel_kind": "qq",
                    "event_type": "failed",
                    "status": "dead_lettered",
                    "error_kind": "route_error",
                    "error_message": "missing adapter",
                    "attempt": 3,
                    "max_attempts": 3,
                    "occurred_at": "2026-05-30T08:41:00Z",
                }
            ],
        },
    }
    runtime_workers = {
        "workers": [
            {
                "name": "agent_job_recovery",
                "kind": "agent_job_recovery",
                "enabled": False,
                "running": False,
            },
            {
                "name": "outbox_delivery_worker",
                "kind": "outbox_delivery",
                "enabled": True,
                "running": True,
                "worker_id": "runtime-outbox-a",
            },
            {
                "name": "nats_dual_read_compare",
                "kind": "work_queue_compare",
                "enabled": True,
                "running": False,
                "consumer_concurrency": 8,
                "max_in_flight": 64,
            },
        ],
        "totals": {"workers": 3, "enabled": 2, "running": 1, "disabled": 1},
        "notes": ["read-only runtime diagnostics"],
    }
    observe_targets = {
        "targets": [
            {
                "target_id": "qq:1049511700:group:27234224",
                "channel": {
                    "kind": "qq",
                    "account_id": "1049511700",
                    "conversation_id": "27234224",
                    "conversation_type": "group",
                },
                "observe_only": True,
                "reply_allowed": False,
                "require_at": False,
                "enabled": True,
                "source": "python_config",
                "metadata": {"channel_name": "qq"},
            }
        ],
        "totals": {
            "targets": 1,
            "enabled": 1,
            "disabled": 0,
            "observe_only": 1,
            "reply_allowed": 0,
            "groups": 1,
        },
        "side_effect": "none",
    }
    observe_capture = {
        "targets": [
            {
                "target_id": "qq:1049511700:group:27234224",
                "channel": {
                    "kind": "qq",
                    "account_id": "1049511700",
                    "conversation_id": "27234224",
                    "conversation_type": "group",
                },
                "enabled": True,
                "observe_only": True,
                "receiver_connected": True,
                "receiver_id": "qq:1049511700:qq",
                "receiver_status": "connected",
                "status": "warn",
                "blockers": ["file_not_seen"],
                "inbox_events": 3,
                "text_events": 3,
                "attachment_events": 1,
                "attachment_count": 1,
                "media_assets": 1,
                "image_assets": 1,
                "file_assets": 0,
                "content_ready_assets": 1,
                "coverage": {
                    "text_seen": True,
                    "attachment_seen": True,
                    "image_seen": True,
                    "file_seen": False,
                    "media_content_ready": True,
                },
            }
        ],
        "totals": {
            "targets": 1,
            "enabled": 1,
            "ready": 0,
            "warning": 1,
            "blocked": 0,
            "receiver_connected": 1,
            "text_covered": 1,
            "attachment_covered": 1,
            "image_covered": 1,
            "file_covered": 0,
            "media_assets": 1,
            "image_assets": 1,
            "file_assets": 0,
            "content_ready_assets": 1,
        },
        "side_effect": "none",
    }
    receiver_statuses = {
        "receivers": [
            {
                "receiver_id": "qq:1049511700:qq",
                "kind": "qq",
                "channel_name": "qq",
                "account_id": "1049511700",
                "endpoint": "ws://127.0.0.1:3001",
                "status": "connected",
                "reason": "ncatbot_started",
                "source": "python_channel",
            },
            {
                "receiver_id": "telegram:7689386159:telegram",
                "kind": "telegram",
                "channel_name": "telegram",
                "account_id": "7689386159",
                "status": "suspended",
                "reason": "getupdates_conflict",
                "source": "python_channel",
                "metadata": {"polling": "stopped"},
            },
        ],
        "totals": {
            "receivers": 2,
            "starting": 0,
            "connected": 1,
            "suspended": 1,
            "failed": 0,
            "stopped": 0,
            "qq": 1,
            "telegram": 1,
        },
        "side_effect": "none",
    }
    receiver_leases = {
        "leases": [
            {
                "receiver_id": "telegram:7689386159:telegram",
                "kind": "telegram",
                "channel_name": "telegram",
                "account_id": "7689386159",
                "holder_id": "telegram:telegram:host:123",
                "lease_token_present": True,
                "active": True,
                "expires_at": "2026-05-31T00:10:00Z",
            }
        ],
        "totals": {
            "leases": 1,
            "active": 1,
            "expired": 0,
            "telegram": 1,
        },
        "side_effect": "runtime_state_only",
    }
    scheduler_jobs = {
        "sampled_jobs": 2,
        "enabled_jobs": 1,
        "disabled_jobs": 1,
        "overdue_jobs": 1,
        "due_soon_jobs": 0,
        "instant_jobs": 1,
        "soft_jobs": 1,
        "next_fire_at": "2026-06-01T08:58:00Z",
        "jobs_by_trigger": {"instant": 1, "soft": 1},
        "jobs_by_tier": {"instant": 1, "soft": 1},
        "jobs_by_channel": {"telegram": 1, "qq": 1},
        "jobs_by_status": {"overdue": 1, "disabled": 1},
        "recent": [
            {
                "id": "schedule:overdue",
                "trigger": "instant",
                "tier": "instant",
                "channel": "telegram",
                "chat_id": "123",
                "fire_at": "2026-06-01T08:58:00Z",
                "status": "overdue",
                "run_count": 0,
                "enabled": True,
                "overdue_by_seconds": 60,
            }
        ],
        "due_soon_seconds": 300,
        "side_effect": "none",
    }
    go_overview = {
        "summary": {
            "jobs_total": 3,
            "outbox_total": 2,
            "checkpoints_total": 1,
            "worker_leases": 2,
            "agent_job_leases": 1,
            "outbox_leases": 1,
            "stale_jobs": 1,
            "dead_letters": 2,
            "checkpoint_lag_max": 17,
            "job_events": 12,
            "outbox_events": 7,
            "rag_eval_failures": 1,
            "delivery_adapters": 2,
            "delivery_adapters_enabled": 1,
            "delivery_adapters_disabled": 1,
            "delivery_smoke_ready": False,
            "delivery_smoke_reason": "delivery_smoke_not_ready",
            "delivery_smoke_cases": 2,
            "delivery_smoke_ready_cases": 1,
            "delivery_smoke_not_ready_cases": 1,
            "delivery_smoke_blockers": 1,
            "queue_backend_provider": "nats_jetstream",
            "queue_backend_mode": "external_lease",
            "queue_consumer_concurrency": 8,
            "queue_max_in_flight": 64,
            "queue_external_lease_ready": False,
            "queue_topology_nodes": 3,
            "queue_topology_edges": 2,
            "queue_topology_work_kinds": 2,
            "queue_topology_blockers": 1,
            "queue_topology_external_lease_ready": False,
            "queue_topology_outbox_execution_owner": "nats_external_lease",
            "queue_topology_agent_job_execution_owner": (
                "python_ai_worker_with_nats_result_ack"
            ),
            "queue_topology_agent_job_ack_owner": "nats_external_lease_result_ack",
            "runtime_workers": 3,
            "runtime_workers_enabled": 2,
            "runtime_workers_running": 1,
            "observe_targets": 1,
            "observe_targets_enabled": 1,
            "observe_targets_observe_only": 1,
            "observe_targets_reply_allowed": 0,
            "observe_target_groups": 1,
            "observe_capture_targets": 1,
            "observe_capture_ready": 0,
            "observe_capture_warning": 1,
            "observe_capture_blocked": 0,
            "observe_capture_text": 1,
            "observe_capture_image": 1,
            "observe_capture_file": 0,
            "observe_capture_content_ready": 1,
            "media_asset_content_assets": 3,
            "media_asset_content_ready": 1,
            "media_asset_content_forbidden": 1,
            "media_asset_content_unavailable": 1,
            "media_asset_content_disabled": 0,
            "media_asset_content_error": 0,
            "media_asset_retention_assets": 3,
            "media_asset_retention_cleanup_due": 1,
            "media_asset_retention_permanent": 1,
            "media_asset_retention_default": 1,
            "media_asset_retention_ephemeral": 1,
            "media_asset_retention_unknown": 0,
            "agent_job_capacity_ready": False,
            "agent_job_capacity_reason": "agent_job_capacity_attention_required",
            "agent_job_capacity_blockers": 3,
            "agent_job_capacity_job_types": 4,
            "agent_job_capacity_mapped_job_types": 4,
            "agent_job_capacity_unmapped_job_types": 0,
            "agent_job_capacity_high_pressure_job_types": 1,
            "agent_job_capacity_blocked_job_types": 1,
            "agent_job_capacity_worker_warning_job_types": 2,
            "agent_job_capacity_active_worker_job_types": 1,
            "agent_job_capacity_stale_worker_job_types": 2,
            "agent_job_capacity_failed_worker_job_types": 1,
            "agent_job_capacity_max_pending": 11,
            "agent_job_capacity_max_active": 2,
            "agent_job_capacity_oldest_pending_age_seconds": 1800,
            "agent_job_priority_ready": False,
            "agent_job_priority_reason": "agent_job_priority_attention_required",
            "agent_job_priority_blockers": 2,
            "agent_job_priority_job_types": 4,
            "agent_job_priority_high_priority_job_types": 2,
            "agent_job_priority_blocked_job_types": 1,
            "agent_job_priority_warning_job_types": 1,
            "agent_job_priority_max_priority_score": 100,
            "agent_job_external_lease_ready": False,
            "agent_job_external_lease_reason": "agent_job_external_lease_not_ready",
            "agent_job_external_lease_blockers": 3,
            "agent_job_external_lease_result_ack_ready": False,
            "agent_job_external_lease_worker_ready": False,
            "agent_job_external_lease_strict_token": True,
            "agent_job_external_lease_execution_owner": (
                "python_ai_worker_with_nats_result_ack"
            ),
            "agent_job_external_lease_execution_scope": "agent_job_result_ack_only",
            "agent_job_external_lease_plan_ready": False,
            "agent_job_external_lease_plan_decision": "blocked",
            "agent_job_external_lease_plan_blockers": 2,
            "agent_job_external_lease_plan_current_owner": (
                "python_ai_worker_state_store_lease"
            ),
            "agent_job_external_lease_plan_desired_owner": (
                "python_ai_worker_with_nats_result_ack"
            ),
            "agent_job_external_lease_plan_recommended_owner": (
                "python_ai_worker_with_nats_result_ack"
            ),
            "outbound_cutover_plan_ready": False,
            "outbound_cutover_plan_decision": "blocked",
            "outbound_cutover_plan_blockers": 2,
            "outbound_cutover_plan_current_owner": "go_state_store_api",
            "outbound_cutover_plan_desired_owner": "nats_external_lease",
            "outbound_cutover_plan_recommended_owner": "nats_external_lease",
            "control_mutation_policy_allowed": True,
            "control_mutation_policy_reason": "control_mutation_policy_listed",
            "control_mutation_policy_targets": 2,
            "control_mutation_policy_actions": 4,
            "operator_approvals_total": 2,
            "operator_approvals_active": 1,
            "operator_approvals_approved": 1,
            "operator_approvals_rejected": 1,
            "operator_approvals_revoked": 0,
            "control_mutations_total": 2,
            "control_mutations_planned": 1,
            "control_mutations_applied": 0,
            "control_mutations_failed": 1,
            "control_mutations_rolled_back": 0,
            "knowledge_job_planner_cutover_plan_ready": False,
            "knowledge_job_planner_cutover_plan_decision": "blocked",
            "knowledge_job_planner_cutover_plan_blockers": 2,
            "knowledge_job_planner_cutover_plan_current_owner": (
                "python_legacy_knowledge_enqueue"
            ),
            "knowledge_job_planner_cutover_plan_desired_owner": (
                "go_runtime_knowledge_job_planner"
            ),
            "knowledge_job_planner_cutover_plan_recommended_owner": (
                "go_runtime_knowledge_job_planner"
            ),
            "receiver_statuses": 2,
            "receiver_status_connected": 1,
            "receiver_status_suspended": 1,
            "receiver_status_failed": 0,
            "receiver_status_qq": 1,
            "receiver_status_telegram": 1,
            "receiver_leases": 1,
            "receiver_leases_active": 1,
            "receiver_leases_expired": 0,
            "receiver_lease_cleanup_required": False,
            "receiver_lease_cleanup_endpoint": "/v1/receiver-leases/cleanup-expired",
            "scheduler_jobs": 2,
            "scheduler_jobs_enabled": 1,
            "scheduler_jobs_disabled": 1,
            "scheduler_jobs_overdue": 1,
            "scheduler_jobs_due_soon": 0,
            "scheduler_jobs_soft": 1,
            "scheduler_jobs_instant": 1,
            "send_ledger_records": 4,
            "send_ledger_repeated_hashes": 1,
            "inbox_metric_events": 9,
            "inbox_metric_observe_only": 7,
            "inbox_metric_with_attachments": 3,
            "inbound_dedupe_records": 2,
            "inbound_dedupe_active_records": 2,
            "inbound_dedupe_duplicate_records": 1,
            "inbound_dedupe_seen_total": 3,
            "inbound_dedupe_duplicate_seen_total": 1,
            "inbound_dedupe_scopes": 2,
            "agent_job_metric_events": 12,
            "agent_job_metric_dead_letters": 1,
            "outbox_metric_events": 7,
            "outbox_metric_dead_letters": 1,
        },
        "cards": [
            {"id": "delivery_adapters", "label": "Delivery Adapters", "value": 1, "status": "warn"},
            {"id": "delivery_smoke", "label": "Delivery Smoke", "value": "1/2", "status": "danger"},
            {
                "id": "queue_backend",
                "label": "Queue Backend",
                "value": "nats_jetstream/external_lease",
                "status": "warn",
            },
            {
                "id": "queue_topology",
                "label": "Queue Topology",
                "value": "1/2",
                "status": "warn",
            },
            {"id": "runtime_workers", "label": "Runtime Workers", "value": 1, "status": "warn"},
            {"id": "observe_targets", "label": "Observe Targets", "value": 1, "status": "ok"},
            {"id": "observe_capture", "label": "Observe Capture", "value": "0/1", "status": "warn"},
            {"id": "media_asset_content", "label": "Media Asset Content", "value": "1/3", "status": "danger"},
            {"id": "media_asset_retention", "label": "Media Asset Retention", "value": "1/3", "status": "warn"},
            {
                "id": "agent_job_capacity_plan",
                "label": "Agent Job Capacity",
                "value": "attention:3",
                "status": "danger",
            },
            {
                "id": "agent_job_priority_plan",
                "label": "Agent Job Priority",
                "value": "attention:2",
                "status": "danger",
            },
            {
                "id": "agent_job_external_lease_readiness",
                "label": "Agent Job External Lease",
                "value": "agent_job_external_lease_not_ready:3",
                "status": "danger",
            },
            {
                "id": "agent_job_external_lease_plan",
                "label": "Agent Job External Lease Plan",
                "value": "blocked:2",
                "status": "warn",
            },
            {
                "id": "outbound_cutover_plan",
                "label": "Outbound Cutover",
                "value": "blocked:2",
                "status": "warn",
            },
            {
                "id": "control_audit",
                "label": "Control Audit",
                "value": "1/2",
                "status": "warn",
                "detail": {
                    "operator_approvals": {
                        "approvals": [
                            {
                                "approval_id": "approval-a",
                                "target_kind": "outbound_cutover_plan",
                                "target_id": "cutover-a",
                                "decision": "approved",
                                "operator_id": "qsyy",
                                "active": True,
                                "created_at": "2026-06-01T12:00:00Z",
                                "metadata": {"source": "test"},
                            },
                            {
                                "approval_id": "approval-b",
                                "target_kind": "agent_job_priority_plan",
                                "target_id": "priority-a",
                                "decision": "rejected",
                                "operator_id": "qsyy",
                                "reason": "blocked",
                                "active": False,
                                "created_at": "2026-06-01T12:01:00Z",
                            },
                        ],
                        "totals": {
                            "approvals": 2,
                            "active": 1,
                            "approved": 1,
                            "rejected": 1,
                            "revoked": 0,
                        },
                        "side_effect": "runtime_state_only",
                    },
                    "control_mutations": {
                        "mutations": [
                            {
                                "mutation_id": "mutation-a",
                                "target_kind": "outbound_cutover",
                                "target_id": "cutover-a",
                                "action": "enable",
                                "status": "planned",
                                "operator_id": "qsyy",
                                "approval_id": "approval-a",
                                "created_at": "2026-06-01T12:02:00Z",
                            },
                            {
                                "mutation_id": "mutation-b",
                                "target_kind": "outbound_cutover",
                                "target_id": "cutover-a",
                                "action": "enable",
                                "status": "failed",
                                "operator_id": "qsyy",
                                "approval_id": "approval-a",
                                "reason": "smoke failed",
                                "rollback_ref": "manual",
                                "created_at": "2026-06-01T12:03:00Z",
                            },
                        ],
                        "totals": {
                            "mutations": 2,
                            "planned": 1,
                            "applied": 0,
                            "failed": 1,
                            "rolled_back": 0,
                        },
                        "side_effect": "runtime_state_only",
                    },
                },
            },
            {
                "id": "control_mutation_policy",
                "label": "Control Mutation Policy",
                "value": "2/4",
                "status": "ok",
                "detail": {
                    "control_mutation_policy": {
                        "allowed": True,
                        "reason": "control_mutation_policy_listed",
                        "intents": [
                            {
                                "target_kind": "agent_job_capacity",
                                "actions": ["apply", "rollback"],
                            },
                            {
                                "target_kind": "outbound_cutover",
                                "actions": ["enable", "rollback"],
                            },
                        ],
                        "side_effect": "none",
                    }
                },
            },
            {
                "id": "knowledge_job_planner_cutover_plan",
                "label": "Knowledge Planner Cutover",
                "value": "blocked:2",
                "status": "warn",
            },
            {"id": "receiver_statuses", "label": "Receiver Statuses", "value": 1, "status": "warn"},
            {"id": "receiver_leases", "label": "Receiver Leases", "value": "1/0", "status": "ok"},
            {
                "id": "scheduler_jobs",
                "label": "Scheduler Jobs",
                "value": "1/2",
                "status": "warn",
            },
            {"id": "send_ledger_metrics", "label": "Send Ledger Metrics", "value": 4, "status": "warn"},
            {"id": "inbox_metrics", "label": "Inbox Metrics", "value": 9, "status": "ok"},
            {"id": "inbound_dedupe_metrics", "label": "Inbound Dedupe", "value": 1, "status": "warn"},
            {"id": "agent_job_metrics", "label": "Agent Job Metrics", "value": 12, "status": "danger"},
            {"id": "outbox_metrics", "label": "Outbox Metrics", "value": 7, "status": "danger"},
        ],
        "delivery_adapters": delivery_adapters,
        "delivery_smoke_readiness": {
            "ready": False,
            "reason": "delivery_smoke_not_ready",
            "cases": [
                {
                    "name": "qq_private_text_2365524513_to_1049511700",
                    "ready": True,
                    "reason": "delivery_adapter_ready",
                },
                {
                    "name": "qq_group_text_1049511700_to_27234224",
                    "ready": False,
                    "reason": "delivery_adapter_unavailable",
                    "missing_channels": ["qq_1049511700"],
                },
            ],
            "totals": {"cases": 2, "ready": 1, "not_ready": 1},
            "blockers": [
                "qq_group_text_1049511700_to_27234224:missing_channel:qq_1049511700"
            ],
            "side_effect": "none",
        },
        "queue_backend": queue_backend,
        "queue_topology": {
            "provider": "nats_jetstream",
            "mode": "external_lease",
            "migration_phase": "first_external_mq",
            "selected_provider": "nats_jetstream",
            "recommended_provider": "nats_jetstream",
            "state_store_authoritative": True,
            "external_queue_active": True,
            "external_lease_ready": False,
            "execution_scope": "outbox_delivery_and_agent_job_result_ack",
            "nodes": [
                {"id": "state_store", "kind": "state_store", "status": "ok"},
                {"id": "external_queue", "kind": "external_queue", "status": "ok"},
                {"id": "python_ai_worker", "kind": "python_worker", "status": "ok"},
            ],
            "edges": [
                {
                    "from": "state_store",
                    "to": "external_lease_executor",
                    "work_kind": "outbox_delivery",
                    "queue_source": "outbox_state_store",
                    "execution_owner": "nats_external_lease",
                    "ack_owner": "nats_external_lease",
                    "status": "ok",
                },
                {
                    "from": "state_store",
                    "to": "python_ai_worker",
                    "work_kind": "agent_job",
                    "queue_source": "agent_job_state_store_with_nats_result_ack",
                    "execution_owner": "python_ai_worker_with_nats_result_ack",
                    "ack_owner": "nats_external_lease_result_ack",
                    "status": "blocked",
                },
            ],
            "work_kinds": [
                {
                    "work_kind": "outbox_delivery",
                    "queue_source": "outbox_state_store",
                    "execution_owner": "nats_external_lease",
                    "ack_owner": "nats_external_lease",
                    "allowed": True,
                },
                {
                    "work_kind": "agent_job",
                    "queue_source": "agent_job_state_store_with_nats_result_ack",
                    "execution_owner": "python_ai_worker_with_nats_result_ack",
                    "ack_owner": "nats_external_lease_result_ack",
                    "allowed": False,
                    "blockers": ["agent_job_result_ack_disabled"],
                },
            ],
            "blockers": ["agent_job:agent_job_result_ack_disabled"],
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "runtime_workers": runtime_workers,
        "observe_targets": observe_targets,
        "observe_capture": observe_capture,
        "media_asset_content_diagnostics": {
            "items": [
                {
                    "asset_id": "asset:qq:1049511700:group:27234224:1",
                    "channel": {
                        "kind": "qq",
                        "account_id": "1049511700",
                        "conversation_id": "27234224",
                        "conversation_type": "group",
                    },
                    "source_message_id": "qq:gqq:27234224:119",
                    "sender_id": "2952887906",
                    "kind": "image",
                    "mime_type": "image/jpeg",
                    "name": "akashic_qq_image.jpg",
                    "size_bytes": 120,
                    "content_status": "ready",
                    "content_reason": "media_asset_content_ready",
                    "content_endpoint": "/v1/media-assets/asset%3Aqq%3A1049511700%3Agroup%3A27234224%3A1/content",
                    "content_mime_type": "image/jpeg",
                    "content_size_bytes": 120,
                    "updated_at": "2026-05-31T12:00:00Z",
                },
                {
                    "asset_id": "asset:forbidden",
                    "channel": {"kind": "qq"},
                    "content_status": "forbidden",
                    "content_reason": "media_asset_content_forbidden",
                },
                {
                    "asset_id": "asset:missing",
                    "channel": {"kind": "qq"},
                    "content_status": "unavailable",
                    "content_reason": "media_asset_content_unavailable",
                },
            ],
            "totals": {
                "assets": 3,
                "ready": 1,
                "forbidden": 1,
                "unavailable": 1,
                "disabled": 0,
                "error": 0,
            },
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "media_asset_retention_diagnostics": {
            "items": [
                {
                    "asset_id": "asset:qq:1049511700:group:27234224:1",
                    "channel": {
                        "kind": "qq",
                        "account_id": "1049511700",
                        "conversation_id": "27234224",
                        "conversation_type": "group",
                    },
                    "source_message_id": "qq:gqq:27234224:119",
                    "sender_id": "2952887906",
                    "kind": "image",
                    "retention": "default-observed-group",
                    "retention_class": "default",
                    "cleanup_due": False,
                    "age_seconds": 3600,
                    "ttl_seconds": 2592000,
                    "cleanup_reason": "media_asset_retention_not_due",
                    "created_at": "2026-05-31T12:00:00Z",
                    "updated_at": "2026-05-31T12:00:00Z",
                },
                {
                    "asset_id": "asset:old",
                    "channel": {"kind": "qq"},
                    "retention": "ephemeral",
                    "retention_class": "ephemeral",
                    "cleanup_due": True,
                    "cleanup_reason": "media_asset_retention_due",
                },
                {
                    "asset_id": "asset:keep",
                    "channel": {"kind": "qq"},
                    "retention": "permanent",
                    "retention_class": "permanent",
                    "cleanup_due": False,
                    "cleanup_reason": "media_asset_retention_permanent",
                },
            ],
            "totals": {
                "assets": 3,
                "cleanup_due": 1,
                "permanent": 1,
                "default": 1,
                "ephemeral": 1,
                "unknown": 0,
            },
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "agent_job_capacity_plan": {
            "ready": False,
            "reason": "agent_job_capacity_attention_required",
            "summary": {
                "job_types": 4,
                "mapped_job_types": 4,
                "unmapped_job_types": 0,
                "high_pressure_job_types": 1,
                "capacity_blocked_job_types": 1,
                "worker_warning_job_types": 2,
                "active_worker_job_types": 1,
                "stale_worker_job_types": 2,
                "failed_worker_job_types": 1,
                "max_pending": 11,
                "max_active": 2,
                "oldest_pending_age_seconds": 1800,
            },
            "items": [
                {
                    "job_type": "group_memory_extract",
                    "severity": "blocked",
                    "action": "start_or_recover_python_worker",
                    "recommendation": "restart knowledge worker",
                    "pending": 11,
                    "leased": 1,
                    "running": 1,
                    "active": 2,
                    "oldest_pending_age_seconds": 1800,
                    "high_pressure": True,
                    "pressure_reason": "pending_backlog_high",
                    "coverage": {"job_type": "group_memory_extract", "status": "danger"},
                }
            ],
            "verification_steps": [
                {
                    "step_index": 1,
                    "phase": "verify",
                    "action": "read_agent_job_metrics",
                    "method": "GET",
                    "endpoint": "/v1/job-metrics",
                }
            ],
            "blockers": [
                "agent_job_capacity_blocked",
                "worker_coverage_danger",
                "pending_backlog_high",
            ],
            "attributes": {"planned_by": "agent_runtime_agent_job_capacity_plan"},
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "agent_job_priority_plan": {
            "ready": False,
            "reason": "agent_job_priority_attention_required",
            "summary": {
                "job_types": 4,
                "high_priority_job_types": 2,
                "blocked_job_types": 1,
                "warning_job_types": 1,
                "max_priority_score": 100,
            },
            "items": [
                {
                    "rank": 1,
                    "job_type": "group_memory_extract",
                    "priority_class": "critical",
                    "priority_score": 100,
                    "action": "recover_worker_before_priority_tuning",
                    "recommendation": "recover knowledge worker",
                    "pending": 11,
                    "leased": 1,
                    "running": 1,
                    "active": 2,
                    "oldest_pending_age_seconds": 1800,
                    "high_pressure": True,
                    "pressure_reason": "pending_backlog_high",
                    "coverage": {"job_type": "group_memory_extract", "status": "danger"},
                }
            ],
            "verification_steps": [
                {
                    "step_index": 1,
                    "phase": "verify",
                    "action": "compare_capacity_plan",
                    "method": "GET",
                    "endpoint": "/v1/agent-job-capacity/plan",
                }
            ],
            "blockers": [
                "agent_job_priority_attention_required",
                "agent_job_priority_blocked",
            ],
            "attributes": {"planned_by": "agent_runtime_agent_job_priority_plan"},
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "agent_job_external_lease_readiness": {
            "ready": False,
            "reason": "agent_job_external_lease_not_ready",
            "external_lease_ready": False,
            "agent_job_result_ack_ready": False,
            "strict_lease_token_enabled": True,
            "agent_job_worker_ready": False,
            "execution_owner": "python_ai_worker_with_nats_result_ack",
            "queue_provider": "nats_jetstream",
            "queue_mode": "external_lease",
            "execution_scope": "agent_job_result_ack_only",
            "allowed_work_kinds": ["outbox"],
            "blocked_work_kinds": [{"kind": "agent_job", "reason": "not_enabled"}],
            "required_checks": [{"name": "strict_token", "ready": True}],
            "worker_coverage": [{"job_type": "group_memory_extract", "status": "danger"}],
            "blockers": [
                "agent_job_external_lease_smoke_missing",
                "agent_job_worker_coverage_blocked",
                "agent_job_result_ack_disabled",
            ],
            "attributes": {
                "planned_by": "agent_runtime_agent_job_external_lease_readiness"
            },
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "agent_job_external_lease_plan": {
            "ready": False,
            "decision": "blocked",
            "desired_execution_owner": "python_ai_worker_with_nats_result_ack",
            "recommended_execution_owner": "python_ai_worker_with_nats_result_ack",
            "current_execution_owner": "python_ai_worker_state_store_lease",
            "readiness": {
                "ready": False,
                "reason": "agent_job_external_lease_not_ready",
                "agent_job_result_ack_ready": False,
                "strict_lease_token_enabled": True,
                "agent_job_worker_ready": False,
                "execution_owner": "python_ai_worker_with_nats_result_ack",
                "execution_scope": "agent_job_result_ack_only",
                "blockers": ["agent_job_result_ack_disabled"],
                "side_effect": "none",
            },
            "required_checks": [
                {
                    "step_index": 1,
                    "phase": "precheck",
                    "action": "check_agent_job_external_lease_readiness",
                    "method": "GET",
                    "endpoint": "/v1/agent-job-external-lease/readiness",
                }
            ],
            "enable_steps": [
                {
                    "step_index": 1,
                    "phase": "enable",
                    "action": "enable_agent_job_result_ack",
                    "env": {"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "true"},
                }
            ],
            "verification_steps": [
                {
                    "step_index": 1,
                    "phase": "verify",
                    "action": "read_queue_backend",
                    "method": "GET",
                    "endpoint": "/v1/queue-backend",
                }
            ],
            "rollback_steps": [
                {
                    "step_index": 1,
                    "phase": "rollback",
                    "action": "disable_agent_job_result_ack",
                    "env": {"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "false"},
                }
            ],
            "blockers": [
                "agent_job_external_lease_readiness_not_ready",
                "agent_job_worker_coverage_blocked",
            ],
            "attributes": {"planned_by": "agent_runtime_agent_job_external_lease_plan"},
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "outbound_cutover_plan": {
            "ready": False,
            "decision": "blocked",
            "desired_execution_owner": "nats_external_lease",
            "recommended_execution_owner": "nats_external_lease",
            "current_execution_owner": "go_state_store_api",
            "readiness": {
                "ready": False,
                "reason": "outbound_cutover_not_ready",
                "onebot_ready": True,
                "smoke_ready": False,
                "execution_ready": False,
                "execution_owner": "go_state_store_api",
                "local_outbox_worker_ready": False,
                "external_lease_outbox_ready": False,
                "queue_provider": "nats_jetstream",
                "queue_mode": "external_lease",
                "external_lease_scope": "outbox_only",
                "expected_onebot_channels": ["qq_2365524513"],
                "missing_onebot_channels": [],
                "delivery_smoke_readiness": {
                    "ready": False,
                    "reason": "delivery_smoke_not_ready",
                    "totals": {"cases": 2, "ready": 1, "not_ready": 1},
                    "side_effect": "none",
                },
                "blockers": ["delivery_smoke_not_ready", "outbox_execution_path_not_ready"],
                "side_effect": "none",
            },
            "required_checks": [
                {
                    "step_index": 1,
                    "phase": "precheck",
                    "action": "check_outbound_cutover_readiness",
                    "method": "GET",
                    "endpoint": "/v1/outbound-cutover/readiness",
                }
            ],
            "enable_steps": [
                {
                    "step_index": 1,
                    "phase": "enable",
                    "action": "enable_nats_outbox_external_lease",
                    "env": {"AKASHIC_QUEUE_EXTERNAL_LEASE_OUTBOX_ENABLED": "true"},
                }
            ],
            "verification_steps": [
                {
                    "step_index": 1,
                    "phase": "verify",
                    "action": "read_outbox_metrics",
                    "method": "GET",
                    "endpoint": "/v1/outbox-metrics",
                }
            ],
            "rollback_steps": [
                {
                    "step_index": 1,
                    "phase": "rollback",
                    "action": "disable_nats_outbox_external_lease",
                    "env": {"AKASHIC_QUEUE_EXTERNAL_LEASE_OUTBOX_ENABLED": "false"},
                }
            ],
            "blockers": ["outbound_cutover_readiness_not_ready", "delivery_smoke_not_ready"],
            "attributes": {"planned_by": "agent_runtime_outbound_cutover_plan"},
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "control_mutation_policy": {
            "allowed": True,
            "reason": "control_mutation_policy_listed",
            "intents": [
                {
                    "target_kind": "agent_job_capacity",
                    "actions": ["apply", "rollback"],
                },
                {
                    "target_kind": "outbound_cutover",
                    "actions": ["enable", "rollback"],
                },
            ],
            "notes": ["policy query only"],
            "side_effect": "none",
        },
        "operator_approvals": {
            "approvals": [
                {
                    "approval_id": "approval-a",
                    "target_kind": "outbound_cutover_plan",
                    "target_id": "cutover-a",
                    "decision": "approved",
                    "operator_id": "qsyy",
                    "active": True,
                    "created_at": "2026-06-01T12:00:00Z",
                    "metadata": {"source": "test"},
                },
                {
                    "approval_id": "approval-b",
                    "target_kind": "agent_job_priority_plan",
                    "target_id": "priority-a",
                    "decision": "rejected",
                    "operator_id": "qsyy",
                    "reason": "blocked",
                    "active": False,
                    "created_at": "2026-06-01T12:01:00Z",
                },
            ],
            "totals": {
                "approvals": 2,
                "active": 1,
                "approved": 1,
                "rejected": 1,
                "revoked": 0,
            },
            "notes": ["approval ledger only"],
            "side_effect": "runtime_state_only",
        },
        "control_mutations": {
            "mutations": [
                {
                    "mutation_id": "mutation-a",
                    "target_kind": "outbound_cutover",
                    "target_id": "cutover-a",
                    "action": "enable",
                    "status": "planned",
                    "operator_id": "qsyy",
                    "approval_id": "approval-a",
                    "created_at": "2026-06-01T12:02:00Z",
                },
                {
                    "mutation_id": "mutation-b",
                    "target_kind": "outbound_cutover",
                    "target_id": "cutover-a",
                    "action": "enable",
                    "status": "failed",
                    "operator_id": "qsyy",
                    "approval_id": "approval-a",
                    "reason": "smoke failed",
                    "rollback_ref": "manual",
                    "created_at": "2026-06-01T12:03:00Z",
                },
            ],
            "totals": {
                "mutations": 2,
                "planned": 1,
                "applied": 0,
                "failed": 1,
                "rolled_back": 0,
            },
            "notes": ["control mutation audit only"],
            "side_effect": "runtime_state_only",
        },
        "knowledge_job_planner_cutover_plan": {
            "ready": False,
            "decision": "blocked",
            "desired_admission_owner": "go_runtime_knowledge_job_planner",
            "recommended_admission_owner": "go_runtime_knowledge_job_planner",
            "current_admission_owner": "python_legacy_knowledge_enqueue",
            "readiness": {
                "ready": False,
                "reason": "knowledge_job_planner_not_ready",
                "planner_enabled": True,
                "planner_running": False,
                "knowledge_worker_ready": False,
                "knowledge_worker_active": 0,
                "knowledge_worker_stale": 1,
                "knowledge_worker_failed": 0,
                "knowledge_worker_stopped": 1,
                "blockers": [
                    "knowledge_job_planner_worker_not_running",
                    "knowledge_worker_unavailable",
                ],
                "side_effect": "none",
            },
            "required_checks": [
                {
                    "step_index": 1,
                    "phase": "precheck",
                    "action": "check_knowledge_job_planner_readiness",
                    "method": "GET",
                    "endpoint": "/v1/knowledge-job-planner/readiness",
                }
            ],
            "enable_steps": [
                {
                    "step_index": 1,
                    "phase": "enable",
                    "action": "enable_go_knowledge_job_planner",
                    "env": {"AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED": "true"},
                }
            ],
            "verification_steps": [
                {
                    "step_index": 1,
                    "phase": "verify",
                    "action": "read_runtime_workers",
                    "method": "GET",
                    "endpoint": "/v1/runtime-workers",
                }
            ],
            "rollback_steps": [
                {
                    "step_index": 1,
                    "phase": "rollback",
                    "action": "disable_go_knowledge_job_planner",
                    "env": {"AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED": "false"},
                }
            ],
            "blockers": [
                "knowledge_job_planner_worker_not_running",
                "knowledge_worker_unavailable",
            ],
            "attributes": {
                "planned_by": "agent_runtime_knowledge_job_planner_cutover_plan"
            },
            "notes": ["read-only"],
            "side_effect": "none",
        },
        "receiver_statuses": receiver_statuses,
        "receiver_leases": receiver_leases,
        "scheduler_jobs": scheduler_jobs,
        "send_ledger_metrics": send_ledger_metrics,
        "inbox_metrics": inbox_metrics,
        "inbound_dedupe_metrics": inbound_dedupe_metrics,
        "agent_job_metrics": agent_job_metrics,
        "outbox_metrics": outbox_metrics,
        "diagnostics": diagnostics,
        "status": {
            "runtime_available": True,
            "health_available": True,
            "partial": False,
            "errors": [],
        },
    }
    seen_paths: list[str] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        seen_paths.append(parsed.path)
        query = parse_qs(parsed.query)
        if parsed.path == "/v1/runtime-overview":
            assert query["limit"] == ["50"]
            assert query["event_limit"] == ["10"]
            assert query["stale_after_seconds"] == ["60"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": go_overview}))
        if parsed.path == "/healthz":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": {"status": "ok"}}))
        if parsed.path == "/v1/jobs":
            assert query["limit"] == ["50"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": jobs}))
        if parsed.path == "/v1/outbox":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": outbox}))
        if parsed.path == "/v1/knowledge-checkpoints":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": checkpoints}))
        if parsed.path == "/v1/knowledge-worker-diagnostics":
            assert query["stale_after_seconds"] == ["60"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": diagnostics}))
        if parsed.path == "/v1/job-events":
            assert query["limit"] == ["10"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": events}))
        if parsed.path == "/v1/outbox-events":
            assert query["limit"] == ["10"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": outbox_events}))
        if parsed.path == "/v1/delivery-adapters":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": delivery_adapters}))
        if parsed.path == "/v1/queue-backend":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": queue_backend}))
        if parsed.path == "/v1/send-ledger/metrics":
            assert query["limit"] == ["50"]
            return _fake_urlopen_response(
                json.dumps({"code": "OK", "data": send_ledger_metrics})
            )
        if parsed.path == "/v1/inbox-metrics":
            assert query["limit"] == ["50"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": inbox_metrics}))
        if parsed.path == "/v1/job-metrics":
            assert query["job_limit"] == ["50"]
            assert query["event_limit"] == ["10"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": agent_job_metrics}))
        if parsed.path == "/v1/outbox-metrics":
            assert query["delivery_limit"] == ["50"]
            assert query["event_limit"] == ["10"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": outbox_metrics}))
        raise AssertionError(f"unhandled runtime call: {parsed.path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        response = client.get(
            "/api/dashboard/runtime-overview",
            params={
                "limit": 50,
                "event_limit": 10,
                "stale_after_seconds": 60,
            },
        )

    assert response.status_code == 200
    payload = response.json()
    assert payload["status"]["runtime_available"] is True
    assert payload["status"]["errors"] == []
    assert payload["summary"]["jobs_total"] == 3
    assert payload["summary"]["worker_leases"] == 2
    assert payload["summary"]["stale_jobs"] == 1
    assert payload["summary"]["dead_letters"] == 2
    assert payload["summary"]["checkpoint_lag_max"] == 17
    assert payload["summary"]["job_events"] == 12
    assert payload["summary"]["outbox_events"] == 7
    assert payload["summary"]["rag_eval_failures"] == 1
    assert payload["summary"]["delivery_adapters"] == 2
    assert payload["summary"]["delivery_adapters_enabled"] == 1
    assert payload["summary"]["delivery_adapters_disabled"] == 1
    assert payload["summary"]["delivery_smoke_ready"] is False
    assert payload["summary"]["delivery_smoke_reason"] == "delivery_smoke_not_ready"
    assert payload["summary"]["delivery_smoke_cases"] == 2
    assert payload["summary"]["delivery_smoke_ready_cases"] == 1
    assert payload["summary"]["delivery_smoke_not_ready_cases"] == 1
    assert payload["summary"]["delivery_smoke_blockers"] == 1
    assert payload["summary"]["queue_backend_provider"] == "nats_jetstream"
    assert payload["summary"]["queue_backend_mode"] == "external_lease"
    assert payload["summary"]["queue_consumer_concurrency"] == 8
    assert payload["summary"]["queue_max_in_flight"] == 64
    assert payload["summary"]["queue_external_lease_ready"] is False
    assert payload["summary"]["queue_topology_nodes"] == 3
    assert payload["summary"]["queue_topology_edges"] == 2
    assert payload["summary"]["queue_topology_work_kinds"] == 2
    assert payload["summary"]["queue_topology_blockers"] == 1
    assert payload["summary"]["queue_topology_external_lease_ready"] is False
    assert payload["summary"]["queue_topology_outbox_execution_owner"] == (
        "nats_external_lease"
    )
    assert payload["summary"]["queue_topology_agent_job_execution_owner"] == (
        "python_ai_worker_with_nats_result_ack"
    )
    assert payload["summary"]["queue_topology_agent_job_ack_owner"] == (
        "nats_external_lease_result_ack"
    )
    assert payload["summary"]["runtime_workers"] == 3
    assert payload["summary"]["runtime_workers_enabled"] == 2
    assert payload["summary"]["runtime_workers_running"] == 1
    assert payload["summary"]["observe_targets"] == 1
    assert payload["summary"]["observe_targets_observe_only"] == 1
    assert payload["summary"]["observe_target_groups"] == 1
    assert payload["summary"]["observe_capture_targets"] == 1
    assert payload["summary"]["observe_capture_warning"] == 1
    assert payload["summary"]["observe_capture_file"] == 0
    assert payload["summary"]["media_asset_content_assets"] == 3
    assert payload["summary"]["media_asset_content_ready"] == 1
    assert payload["summary"]["media_asset_content_forbidden"] == 1
    assert payload["summary"]["media_asset_content_unavailable"] == 1
    assert payload["summary"]["media_asset_retention_assets"] == 3
    assert payload["summary"]["media_asset_retention_cleanup_due"] == 1
    assert payload["summary"]["media_asset_retention_permanent"] == 1
    assert payload["summary"]["media_asset_retention_default"] == 1
    assert payload["summary"]["media_asset_retention_ephemeral"] == 1
    assert payload["summary"]["agent_job_capacity_ready"] is False
    assert payload["summary"]["agent_job_capacity_reason"] == (
        "agent_job_capacity_attention_required"
    )
    assert payload["summary"]["agent_job_capacity_blockers"] == 3
    assert payload["summary"]["agent_job_capacity_job_types"] == 4
    assert payload["summary"]["agent_job_capacity_max_pending"] == 11
    assert payload["summary"]["agent_job_capacity_oldest_pending_age_seconds"] == 1800
    assert payload["summary"]["agent_job_priority_ready"] is False
    assert payload["summary"]["agent_job_priority_reason"] == (
        "agent_job_priority_attention_required"
    )
    assert payload["summary"]["agent_job_priority_blockers"] == 2
    assert payload["summary"]["agent_job_priority_job_types"] == 4
    assert payload["summary"]["agent_job_priority_high_priority_job_types"] == 2
    assert payload["summary"]["agent_job_priority_blocked_job_types"] == 1
    assert payload["summary"]["agent_job_priority_warning_job_types"] == 1
    assert payload["summary"]["agent_job_priority_max_priority_score"] == 100
    assert payload["summary"]["agent_job_external_lease_ready"] is False
    assert payload["summary"]["agent_job_external_lease_reason"] == (
        "agent_job_external_lease_not_ready"
    )
    assert payload["summary"]["agent_job_external_lease_blockers"] == 3
    assert payload["summary"]["agent_job_external_lease_result_ack_ready"] is False
    assert payload["summary"]["agent_job_external_lease_worker_ready"] is False
    assert payload["summary"]["agent_job_external_lease_strict_token"] is True
    assert payload["summary"]["agent_job_external_lease_execution_owner"] == (
        "python_ai_worker_with_nats_result_ack"
    )
    assert payload["summary"]["agent_job_external_lease_plan_ready"] is False
    assert payload["summary"]["agent_job_external_lease_plan_decision"] == "blocked"
    assert payload["summary"]["agent_job_external_lease_plan_blockers"] == 2
    assert payload["summary"]["agent_job_external_lease_plan_current_owner"] == (
        "python_ai_worker_state_store_lease"
    )
    assert payload["summary"]["outbound_cutover_plan_ready"] is False
    assert payload["summary"]["outbound_cutover_plan_decision"] == "blocked"
    assert payload["summary"]["outbound_cutover_plan_blockers"] == 2
    assert payload["summary"]["outbound_cutover_plan_current_owner"] == (
        "go_state_store_api"
    )
    assert payload["summary"]["control_mutation_policy_allowed"] is True
    assert (
        payload["summary"]["control_mutation_policy_reason"]
        == "control_mutation_policy_listed"
    )
    assert payload["summary"]["control_mutation_policy_targets"] == 2
    assert payload["summary"]["control_mutation_policy_actions"] == 4
    assert payload["summary"]["operator_approvals_total"] == 2
    assert payload["summary"]["operator_approvals_active"] == 1
    assert payload["summary"]["operator_approvals_approved"] == 1
    assert payload["summary"]["operator_approvals_rejected"] == 1
    assert payload["summary"]["control_mutations_total"] == 2
    assert payload["summary"]["control_mutations_planned"] == 1
    assert payload["summary"]["control_mutations_failed"] == 1
    assert payload["summary"]["knowledge_job_planner_cutover_plan_ready"] is False
    assert payload["summary"]["knowledge_job_planner_cutover_plan_decision"] == "blocked"
    assert payload["summary"]["knowledge_job_planner_cutover_plan_blockers"] == 2
    assert payload["summary"]["knowledge_job_planner_cutover_plan_current_owner"] == (
        "python_legacy_knowledge_enqueue"
    )
    assert payload["summary"]["knowledge_job_planner_cutover_plan_desired_owner"] == (
        "go_runtime_knowledge_job_planner"
    )
    assert payload["summary"]["knowledge_job_planner_cutover_plan_recommended_owner"] == (
        "go_runtime_knowledge_job_planner"
    )
    assert payload["summary"]["receiver_statuses"] == 2
    assert payload["summary"]["receiver_status_suspended"] == 1
    assert payload["summary"]["receiver_leases"] == 1
    assert payload["summary"]["receiver_leases_active"] == 1
    assert payload["summary"]["receiver_lease_cleanup_required"] is False
    assert (
        payload["summary"]["receiver_lease_cleanup_endpoint"]
        == "/v1/receiver-leases/cleanup-expired"
    )
    assert payload["summary"]["scheduler_jobs"] == 2
    assert payload["summary"]["scheduler_jobs_enabled"] == 1
    assert payload["summary"]["scheduler_jobs_overdue"] == 1
    assert payload["summary"]["send_ledger_records"] == 4
    assert payload["summary"]["send_ledger_repeated_hashes"] == 1
    assert payload["summary"]["inbox_metric_events"] == 9
    assert payload["summary"]["inbox_metric_observe_only"] == 7
    assert payload["summary"]["inbox_metric_with_attachments"] == 3
    assert payload["summary"]["inbound_dedupe_records"] == 2
    assert payload["summary"]["inbound_dedupe_duplicate_seen_total"] == 1
    assert payload["summary"]["agent_job_metric_events"] == 12
    assert payload["summary"]["agent_job_metric_dead_letters"] == 1
    assert payload["summary"]["outbox_metric_events"] == 7
    assert payload["summary"]["outbox_metric_dead_letters"] == 1
    assert payload["jobs_by_status"]["dead_lettered"] == 1
    assert payload["outbox_by_status"]["dead_lettered"] == 1
    assert payload["checkpoint_lag"][0]["checkpoint_lag_messages"] == 17
    assert payload["delivery_adapters"][0]["channel"] == "qq_2365524513"
    adapter_card = next(item for item in payload["cards"] if item["id"] == "delivery_adapters")
    assert adapter_card["status"] == "warn"
    smoke_card = next(item for item in payload["cards"] if item["id"] == "delivery_smoke")
    assert smoke_card["value"] == "1/2"
    assert smoke_card["status"] == "danger"
    assert payload["delivery_smoke_readiness"]["totals"]["not_ready"] == 1
    assert payload["delivery_smoke_readiness"]["cases"][1]["missing_channels"] == [
        "qq_1049511700"
    ]
    queue_card = next(item for item in payload["cards"] if item["id"] == "queue_backend")
    assert queue_card["value"] == "nats_jetstream/external_lease"
    assert queue_card["status"] == "warn"
    queue_topology_card = next(
        item for item in payload["cards"] if item["id"] == "queue_topology"
    )
    assert queue_topology_card["value"] == "1/2"
    assert queue_topology_card["status"] == "warn"
    queue_topology = payload["queue_topology"]
    assert queue_topology["provider"] == "nats_jetstream"
    assert queue_topology["work_kinds"][1]["work_kind"] == "agent_job"
    assert queue_topology["work_kinds"][1]["ack_owner"] == (
        "nats_external_lease_result_ack"
    )
    assert queue_topology["work_kinds"][1]["allowed"] is False
    assert queue_topology["side_effect"] == "none"
    runtime_worker_card = next(
        item for item in payload["cards"] if item["id"] == "runtime_workers"
    )
    assert runtime_worker_card["status"] == "warn"
    assert payload["runtime_workers"]["totals"]["running"] == 1
    assert payload["runtime_workers"]["workers"][1]["worker_id"] == "runtime-outbox-a"
    observe_card = next(item for item in payload["cards"] if item["id"] == "observe_targets")
    assert observe_card["status"] == "ok"
    assert payload["observe_targets"]["targets"][0]["channel"]["conversation_id"] == "27234224"
    assert payload["observe_targets"]["side_effect"] == "none"
    observe_capture_card = next(item for item in payload["cards"] if item["id"] == "observe_capture")
    assert observe_capture_card["status"] == "warn"
    assert payload["observe_capture"]["targets"][0]["blockers"] == ["file_not_seen"]
    assert payload["observe_capture"]["totals"]["image_covered"] == 1
    media_card = next(item for item in payload["cards"] if item["id"] == "media_asset_content")
    assert media_card["value"] == "1/3"
    assert media_card["status"] == "danger"
    media_retention_card = next(
        item for item in payload["cards"] if item["id"] == "media_asset_retention"
    )
    assert media_retention_card["value"] == "1/3"
    assert media_retention_card["status"] == "warn"
    assert payload["media_asset_content_diagnostics"]["totals"]["forbidden"] == 1
    assert payload["media_asset_retention_diagnostics"]["totals"]["cleanup_due"] == 1
    assert payload["media_asset_retention_diagnostics"]["items"][1]["asset_id"] == "asset:old"
    assert (
        payload["media_asset_content_diagnostics"]["items"][0]["content_endpoint"]
        == "/v1/media-assets/asset%3Aqq%3A1049511700%3Agroup%3A27234224%3A1/content"
    )
    capacity_card = next(
        item for item in payload["cards"] if item["id"] == "agent_job_capacity_plan"
    )
    assert capacity_card["value"] == "attention:3"
    assert capacity_card["status"] == "danger"
    capacity_plan = payload["agent_job_capacity_plan"]
    assert capacity_plan["summary"]["max_pending"] == 11
    assert capacity_plan["items"][0]["job_type"] == "group_memory_extract"
    assert capacity_plan["items"][0]["coverage"]["status"] == "danger"
    assert capacity_plan["verification_steps"][0]["endpoint"] == "/v1/job-metrics"
    assert capacity_plan["side_effect"] == "none"
    priority_card = next(
        item for item in payload["cards"] if item["id"] == "agent_job_priority_plan"
    )
    assert priority_card["value"] == "attention:2"
    assert priority_card["status"] == "danger"
    priority_plan = payload["agent_job_priority_plan"]
    assert priority_plan["summary"]["max_priority_score"] == 100
    assert priority_plan["items"][0]["job_type"] == "group_memory_extract"
    assert priority_plan["items"][0]["priority_class"] == "critical"
    assert priority_plan["items"][0]["coverage"]["status"] == "danger"
    assert priority_plan["verification_steps"][0]["endpoint"] == (
        "/v1/agent-job-capacity/plan"
    )
    assert priority_plan["side_effect"] == "none"
    external_readiness_card = next(
        item
        for item in payload["cards"]
        if item["id"] == "agent_job_external_lease_readiness"
    )
    assert external_readiness_card["status"] == "danger"
    external_readiness = payload["agent_job_external_lease_readiness"]
    assert external_readiness["execution_scope"] == "agent_job_result_ack_only"
    assert external_readiness["blocked_work_kinds"][0]["kind"] == "agent_job"
    assert external_readiness["worker_coverage"][0]["status"] == "danger"
    external_plan_card = next(
        item for item in payload["cards"] if item["id"] == "agent_job_external_lease_plan"
    )
    assert external_plan_card["value"] == "blocked:2"
    external_plan = payload["agent_job_external_lease_plan"]
    assert external_plan["current_execution_owner"] == "python_ai_worker_state_store_lease"
    assert external_plan["readiness"]["reason"] == "agent_job_external_lease_not_ready"
    assert external_plan["enable_steps"][0]["env"] == {
        "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "true"
    }
    assert external_plan["rollback_steps"][0]["action"] == "disable_agent_job_result_ack"
    outbound_card = next(
        item for item in payload["cards"] if item["id"] == "outbound_cutover_plan"
    )
    assert outbound_card["status"] == "warn"
    outbound_plan = payload["outbound_cutover_plan"]
    assert outbound_plan["desired_execution_owner"] == "nats_external_lease"
    assert outbound_plan["readiness"]["expected_onebot_channels"] == ["qq_2365524513"]
    assert outbound_plan["readiness"]["delivery_smoke_readiness"]["totals"]["not_ready"] == 1
    assert outbound_plan["enable_steps"][0]["env"] == {
        "AKASHIC_QUEUE_EXTERNAL_LEASE_OUTBOX_ENABLED": "true"
    }
    assert outbound_plan["side_effect"] == "none"
    policy_card = next(
        item for item in payload["cards"] if item["id"] == "control_mutation_policy"
    )
    assert policy_card["value"] == "2/4"
    assert policy_card["status"] == "ok"
    control_mutation_policy = payload["control_mutation_policy"]
    assert control_mutation_policy["allowed"] is True
    assert control_mutation_policy["reason"] == "control_mutation_policy_listed"
    assert control_mutation_policy["intents"][0]["target_kind"] == "agent_job_capacity"
    assert control_mutation_policy["intents"][1]["actions"] == ["enable", "rollback"]
    assert control_mutation_policy["side_effect"] == "none"
    control_audit_card = next(
        item for item in payload["cards"] if item["id"] == "control_audit"
    )
    assert control_audit_card["value"] == "1/2"
    assert control_audit_card["status"] == "warn"
    operator_approvals = payload["operator_approvals"]
    assert operator_approvals["totals"]["active"] == 1
    assert operator_approvals["approvals"][0]["approval_id"] == "approval-a"
    assert operator_approvals["approvals"][0]["metadata"]["source"] == "test"
    assert operator_approvals["side_effect"] == "runtime_state_only"
    control_mutations = payload["control_mutations"]
    assert control_mutations["totals"]["failed"] == 1
    assert control_mutations["mutations"][1]["status"] == "failed"
    assert control_mutations["mutations"][1]["rollback_ref"] == "manual"
    assert control_mutations["side_effect"] == "runtime_state_only"
    cutover_card = next(
        item for item in payload["cards"] if item["id"] == "knowledge_job_planner_cutover_plan"
    )
    assert cutover_card["value"] == "blocked:2"
    assert cutover_card["status"] == "warn"
    cutover_plan = payload["knowledge_job_planner_cutover_plan"]
    assert cutover_plan["decision"] == "blocked"
    assert cutover_plan["current_admission_owner"] == "python_legacy_knowledge_enqueue"
    assert cutover_plan["desired_admission_owner"] == "go_runtime_knowledge_job_planner"
    assert cutover_plan["readiness"]["reason"] == "knowledge_job_planner_not_ready"
    assert cutover_plan["readiness"]["knowledge_worker_stale"] == 1
    assert cutover_plan["required_checks"][0]["endpoint"] == (
        "/v1/knowledge-job-planner/readiness"
    )
    assert cutover_plan["enable_steps"][0]["env"] == {
        "AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED": "true"
    }
    assert cutover_plan["rollback_steps"][0]["action"] == (
        "disable_go_knowledge_job_planner"
    )
    assert cutover_plan["side_effect"] == "none"
    receiver_card = next(item for item in payload["cards"] if item["id"] == "receiver_statuses")
    assert receiver_card["status"] == "warn"
    assert payload["receiver_statuses"]["receivers"][1]["reason"] == "getupdates_conflict"
    assert payload["receiver_statuses"]["totals"]["telegram"] == 1
    lease_card = next(item for item in payload["cards"] if item["id"] == "receiver_leases")
    assert lease_card["status"] == "ok"
    assert lease_card["value"] == "1/0"
    assert payload["receiver_leases"]["leases"][0]["lease_token_present"] is True
    assert payload["receiver_leases"]["side_effect"] == "runtime_state_only"
    scheduler_card = next(item for item in payload["cards"] if item["id"] == "scheduler_jobs")
    assert scheduler_card["value"] == "1/2"
    assert scheduler_card["status"] == "warn"
    assert payload["scheduler_jobs"]["jobs_by_status"]["overdue"] == 1
    assert payload["scheduler_jobs"]["recent"][0]["id"] == "schedule:overdue"
    assert payload["scheduler_jobs"]["side_effect"] == "none"
    send_ledger_card = next(
        item for item in payload["cards"] if item["id"] == "send_ledger_metrics"
    )
    assert send_ledger_card["status"] == "warn"
    assert payload["send_ledger_metrics"]["repeated_hashes"][0]["content_hash"] == (
        "hash-image"
    )
    inbox_metrics_card = next(item for item in payload["cards"] if item["id"] == "inbox_metrics")
    assert inbox_metrics_card["status"] == "ok"
    assert payload["inbox_metrics"]["events_by_conversation"][
        "qq/1049511700/group/27234224"
    ]["latest_seq"] == 119
    dedupe_card = next(
        item for item in payload["cards"] if item["id"] == "inbound_dedupe_metrics"
    )
    assert dedupe_card["status"] == "warn"
    assert payload["inbound_dedupe_metrics"]["duplicate_seen_total"] == 1
    assert payload["inbound_dedupe_metrics"]["scopes"][0]["scope"] == (
        "qq:qq_2365524513:2365524513"
    )
    metrics_card = next(item for item in payload["cards"] if item["id"] == "agent_job_metrics")
    assert metrics_card["status"] == "danger"
    assert payload["agent_job_metrics"]["throughput"]["succeeded"] == 1
    assert payload["agent_job_metrics"]["dead_letters"]["recent"][0]["job_id"] == (
        "rag_ingest:qq:3219982:dead"
    )
    outbox_metrics_card = next(item for item in payload["cards"] if item["id"] == "outbox_metrics")
    assert outbox_metrics_card["status"] == "danger"
    assert payload["outbox_metrics"]["throughput"]["failed"] == 1
    assert payload["outbox_metrics"]["dead_letters"]["recent"][0]["delivery_id"] == "outbox:dead"
    assert payload["queue_backend"]["external_lease_blockers"] == [
        "explicit_cutover",
        "state_lease_workers_disabled",
    ]
    assert seen_paths == ["/v1/runtime-overview"]


def test_runtime_overview_dashboard_exposes_manual_adapter_health_probe(
    monkeypatch,
    tmp_path,
) -> None:
    seen_paths: list[str] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        seen_paths.append(parsed.path)
        query = parse_qs(parsed.query)
        if parsed.path == "/v1/delivery-adapters/health":
            assert query["timeout_seconds"] == ["2"]
            assert timeout >= 2
            return _fake_urlopen_response(
                json.dumps(
                    {
                        "code": "OK",
                        "data": {
                            "items": [
                                {
                                    "provider": "onebot",
                                    "channel": "qq_2365524513",
                                    "transport": "websocket",
                                    "healthy": True,
                                    "reachable": True,
                                    "authenticated": True,
                                    "account_id": "2365524513",
                                    "account_name": "bot-236",
                                    "checked_at": "2026-05-31T12:00:00Z",
                                    "latency_ms": 15,
                                    "side_effect": "none",
                                    "access_token_present": True,
                                }
                            ],
                            "totals": {
                                "adapters": 1,
                                "healthy": 1,
                                "unhealthy": 0,
                                "authenticated": 1,
                            },
                        },
                    }
                )
            )
        raise AssertionError(f"unhandled runtime call: {parsed.path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        response = client.get(
            "/api/dashboard/runtime-overview/delivery-adapter-health",
            params={"timeout_seconds": 2},
        )

    assert response.status_code == 200
    payload = response.json()
    assert payload["status"]["available"] is True
    assert payload["status"]["side_effect"] == "none"
    assert payload["totals"]["healthy"] == 1
    assert payload["items"][0]["channel"] == "qq_2365524513"
    assert payload["items"][0]["account_id"] == "2365524513"
    assert payload["items"][0]["side_effect"] == "none"
    assert seen_paths == ["/v1/delivery-adapters/health"]


def test_runtime_overview_dashboard_exposes_manual_delivery_smoke_readiness(
    monkeypatch,
    tmp_path,
) -> None:
    seen: dict[str, Any] = {}

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        seen["path"] = parsed.path
        seen["method"] = request.get_method()
        seen["timeout"] = timeout
        seen["body"] = json.loads(request.data.decode("utf-8"))
        if parsed.path == "/v1/delivery-smoke/readiness":
            return _fake_urlopen_response(
                json.dumps(
                    {
                        "code": "OK",
                        "data": {
                            "ready": True,
                            "reason": "delivery_smoke_ready",
                            "cases": [
                                {
                                    "name": "qq_group_text_2365524513_to_27234224",
                                    "ready": True,
                                    "reason": "delivery_adapter_ready",
                                    "plan": {
                                        "event_id": "smoke-qq-group",
                                        "channel": "qq_2365524513",
                                        "chat_id": "27234224",
                                        "step_count": 1,
                                        "steps": [
                                            {
                                                "step_index": 1,
                                                "kind": "text",
                                                "channel": "qq_2365524513",
                                                "chat_id": "27234224",
                                                "conversation_type": "group",
                                            }
                                        ],
                                    },
                                    "attributes": {"side_effect": "none"},
                                }
                            ],
                            "totals": {"cases": 1, "ready": 1, "not_ready": 0},
                            "side_effect": "none",
                        },
                    }
                )
            )
        raise AssertionError(f"unhandled runtime call: {parsed.path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        response = client.get(
            "/api/dashboard/runtime-overview/delivery-smoke-readiness",
            params={"group_ids": "27234224,3219982", "include_synthetic_media": "true"},
        )

    assert response.status_code == 200
    payload = response.json()
    assert payload["ready"] is True
    assert payload["status"]["available"] is True
    assert payload["status"]["side_effect"] == "none"
    assert payload["totals"]["ready"] == 1
    assert payload["cases"][0]["plan"]["channel"] == "qq_2365524513"
    assert seen["path"] == "/v1/delivery-smoke/readiness"
    assert seen["method"] == "POST"
    assert seen["body"]["group_ids"] == ["27234224", "3219982"]
    assert seen["body"]["include_synthetic_media"] is True
    assert seen["timeout"] >= 5


def test_runtime_overview_panel_assets_are_exposed(monkeypatch, tmp_path) -> None:
    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        if parsed.path == "/healthz":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": {"status": "ok"}}))
        if parsed.path.startswith("/v1/"):
            payload: dict[str, Any] | list[Any]
            payload = (
                {}
                if parsed.path
                in {
                    "/v1/knowledge-worker-diagnostics",
                    "/v1/queue-backend",
                    "/v1/send-ledger/metrics",
                    "/v1/inbox-metrics",
                    "/v1/job-metrics",
                    "/v1/outbox-metrics",
                    "/v1/observe-targets",
                    "/v1/observe-capture-diagnostics",
                    "/v1/receiver-statuses",
                    "/v1/receiver-leases",
                    "/v1/scheduler/diagnostics",
                }
                else []
            )
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": payload}))
        raise AssertionError(f"unhandled runtime call: {parsed.path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        plugins = client.get("/api/dashboard/plugins").json()
        plugin_panels = {
            item["id"]: item["panels"]
            for item in plugins
            if item["id"] == "runtime_overview"
        }
        js_response = client.get("/plugins/runtime_overview/dashboard_panel.js")
        css_response = client.get("/plugins/runtime_overview/dashboard_panel.css")

    assert plugin_panels["runtime_overview"][0]["name"] == "dashboard_panel"
    assert plugin_panels["runtime_overview"][0]["has_css"] is True
    assert js_response.status_code == 200
    assert css_response.status_code == 200
    assert "/api/dashboard/runtime-overview/delivery-adapter-health" in js_response.text
    assert "/api/dashboard/runtime-overview/delivery-smoke-readiness" in js_response.text
    assert "Probe Health" in js_response.text
    assert "Smoke Readiness" in js_response.text


def test_runtime_overview_reader_falls_back_when_go_aggregate_is_unavailable(
    monkeypatch,
    tmp_path,
) -> None:
    seen_paths: list[str] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        seen_paths.append(parsed.path)
        if parsed.path == "/v1/runtime-overview":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": []}))
        if parsed.path == "/healthz":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": {"status": "ok"}}))
        if parsed.path.startswith("/v1/"):
            payload: dict[str, Any] | list[Any]
            payload = (
                {}
                if parsed.path
                in {
                    "/v1/knowledge-worker-diagnostics",
                    "/v1/queue-backend",
                    "/v1/send-ledger/metrics",
                    "/v1/inbox-metrics",
                    "/v1/job-metrics",
                    "/v1/outbox-metrics",
                    "/v1/observe-targets",
                    "/v1/observe-capture-diagnostics",
                    "/v1/receiver-statuses",
                    "/v1/receiver-leases",
                    "/v1/scheduler/diagnostics",
                }
                else []
            )
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": payload}))
        raise AssertionError(f"unhandled runtime call: {parsed.path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    payload = RuntimeOverviewDashboardReader(tmp_path).get_overview(
        limit=10,
        event_limit=5,
        stale_after_seconds=30,
    )

    assert payload["status"]["runtime_available"] is True
    assert "/v1/runtime-overview" in seen_paths
    assert "/v1/jobs" in seen_paths
    assert "/v1/outbox-metrics" in seen_paths


def _fake_urlopen_response(payload: str):
    class _Resp:
        def __enter__(self):
            return self

        def __exit__(self, *_args: Any):
            return None

        def read(self):
            return payload.encode("utf-8")

    return _Resp()
