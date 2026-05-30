from __future__ import annotations

import json
from typing import Any
from urllib.parse import parse_qs, urlparse

from fastapi.testclient import TestClient

from bootstrap.dashboard_api import create_dashboard_app


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
    diagnostics = {
        "generated_at": "2026-05-30T08:50:00Z",
        "stale_after_seconds": 60,
        "totals": {
            "jobs": 3,
            "checkpoints": 1,
            "stale_leases": 1,
            "leaseable_jobs": 0,
        },
        "workers": [],
    }
    seen_paths: list[str] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        seen_paths.append(parsed.path)
        query = parse_qs(parsed.query)
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
    assert payload["summary"]["job_events"] == 2
    assert payload["summary"]["rag_eval_failures"] == 1
    assert payload["jobs_by_status"]["dead_lettered"] == 1
    assert payload["outbox_by_status"]["dead_lettered"] == 1
    assert payload["checkpoint_lag"][0]["checkpoint_lag_messages"] == 17
    assert {"/healthz", "/v1/jobs", "/v1/outbox", "/v1/job-events"}.issubset(set(seen_paths))


def test_runtime_overview_panel_assets_are_exposed(monkeypatch, tmp_path) -> None:
    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        if parsed.path == "/healthz":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": {"status": "ok"}}))
        if parsed.path.startswith("/v1/"):
            payload: dict[str, Any] | list[Any]
            payload = {} if parsed.path == "/v1/knowledge-worker-diagnostics" else []
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


def _fake_urlopen_response(payload: str):
    class _Resp:
        def __enter__(self):
            return self

        def __exit__(self, *_args: Any):
            return None

        def read(self):
            return payload.encode("utf-8")

    return _Resp()
