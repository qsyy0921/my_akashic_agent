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
            "queue_backend_provider": "nats_jetstream",
            "queue_backend_mode": "external_lease",
            "queue_consumer_concurrency": 8,
            "queue_max_in_flight": 64,
            "queue_external_lease_ready": False,
            "runtime_workers": 3,
            "runtime_workers_enabled": 2,
            "runtime_workers_running": 1,
            "send_ledger_records": 4,
            "send_ledger_repeated_hashes": 1,
            "inbox_metric_events": 9,
            "inbox_metric_observe_only": 7,
            "inbox_metric_with_attachments": 3,
            "agent_job_metric_events": 12,
            "agent_job_metric_dead_letters": 1,
            "outbox_metric_events": 7,
            "outbox_metric_dead_letters": 1,
        },
        "cards": [
            {"id": "delivery_adapters", "label": "Delivery Adapters", "value": 1, "status": "warn"},
            {
                "id": "queue_backend",
                "label": "Queue Backend",
                "value": "nats_jetstream/external_lease",
                "status": "warn",
            },
            {"id": "runtime_workers", "label": "Runtime Workers", "value": 1, "status": "warn"},
            {"id": "send_ledger_metrics", "label": "Send Ledger Metrics", "value": 4, "status": "warn"},
            {"id": "inbox_metrics", "label": "Inbox Metrics", "value": 9, "status": "ok"},
            {"id": "agent_job_metrics", "label": "Agent Job Metrics", "value": 12, "status": "danger"},
            {"id": "outbox_metrics", "label": "Outbox Metrics", "value": 7, "status": "danger"},
        ],
        "delivery_adapters": delivery_adapters,
        "queue_backend": queue_backend,
        "runtime_workers": runtime_workers,
        "send_ledger_metrics": send_ledger_metrics,
        "inbox_metrics": inbox_metrics,
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
    assert payload["summary"]["queue_backend_provider"] == "nats_jetstream"
    assert payload["summary"]["queue_backend_mode"] == "external_lease"
    assert payload["summary"]["queue_consumer_concurrency"] == 8
    assert payload["summary"]["queue_max_in_flight"] == 64
    assert payload["summary"]["queue_external_lease_ready"] is False
    assert payload["summary"]["runtime_workers"] == 3
    assert payload["summary"]["runtime_workers_enabled"] == 2
    assert payload["summary"]["runtime_workers_running"] == 1
    assert payload["summary"]["send_ledger_records"] == 4
    assert payload["summary"]["send_ledger_repeated_hashes"] == 1
    assert payload["summary"]["inbox_metric_events"] == 9
    assert payload["summary"]["inbox_metric_observe_only"] == 7
    assert payload["summary"]["inbox_metric_with_attachments"] == 3
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
    queue_card = next(item for item in payload["cards"] if item["id"] == "queue_backend")
    assert queue_card["value"] == "nats_jetstream/external_lease"
    assert queue_card["status"] == "warn"
    runtime_worker_card = next(
        item for item in payload["cards"] if item["id"] == "runtime_workers"
    )
    assert runtime_worker_card["status"] == "warn"
    assert payload["runtime_workers"]["totals"]["running"] == 1
    assert payload["runtime_workers"]["workers"][1]["worker_id"] == "runtime-outbox-a"
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
