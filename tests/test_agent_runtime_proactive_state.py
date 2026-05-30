from __future__ import annotations

import json
from datetime import datetime, timezone
from types import SimpleNamespace
from typing import Any, cast

import httpx

from agent.config_models import AgentRuntimeIntegrationConfig
from bootstrap.proactive import _build_proactive_state_store
from integrations.agent_runtime_proactive_state import AgentRuntimeProactiveStateStore
from proactive_v2.state import ProactiveStateStore


def _config() -> AgentRuntimeIntegrationConfig:
    return AgentRuntimeIntegrationConfig(
        enabled=True,
        base_url="http://agent-runtime.local",
        request_timeout_seconds=1,
    )


def _ok(data: Any) -> httpx.Response:
    return httpx.Response(200, json={"code": "OK", "data": data})


def test_agent_runtime_proactive_state_uses_go_for_scheduling_calls(tmp_path):
    calls: list[tuple[str, str, dict[str, str], dict[str, Any]]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        params = dict(request.url.params)
        calls.append((request.method, request.url.path, params, body))
        if request.url.path == "/v1/proactive/deliveries":
            assert body["session_key"] == "telegram:1"
            assert body["delivery_key"] == "delivery-a"
            return httpx.Response(202, json={"code": "OK", "data": body})
        if request.url.path == "/v1/proactive/deliveries/duplicate":
            assert params["window_hours"] == "24"
            return _ok({"duplicate": True})
        if request.url.path == "/v1/proactive/deliveries/count":
            return _ok({"count": 2})
        if request.url.path == "/v1/proactive/context-only":
            return httpx.Response(202, json={"code": "OK", "data": body})
        if request.url.path == "/v1/proactive/context-only/count":
            return _ok({"count": 1})
        if request.url.path == "/v1/proactive/context-only/last":
            return _ok({"found": True, "timestamp": "2026-05-30T09:00:00Z"})
        if request.url.path == "/v1/proactive/drift-runs":
            return httpx.Response(202, json={"code": "OK", "data": body})
        if request.url.path == "/v1/proactive/drift-runs/last":
            return _ok({"found": True, "timestamp": "2026-05-30T10:00:00Z"})
        return httpx.Response(404, text="not found")

    fallback = ProactiveStateStore(tmp_path / "proactive.db")
    store = AgentRuntimeProactiveStateStore(
        _config(),
        fallback,
        transport=httpx.MockTransport(handler),
    )
    now = datetime(2026, 5, 30, 8, 30, tzinfo=timezone.utc)

    store.mark_delivery("telegram:1", "delivery-a", now)

    assert store.is_delivery_duplicate("telegram:1", "delivery-a", 24, now) is True
    assert store.count_deliveries_in_window("telegram:1", 24, now) == 2

    store.mark_context_only_send("telegram:1", now)

    assert store.count_context_only_in_window("telegram:1", 24, now) == 1
    assert store.get_last_context_only_at("telegram:1") == datetime(
        2026, 5, 30, 9, 0, tzinfo=timezone.utc
    )

    store.mark_drift_run("telegram:1", now)

    assert store.get_last_drift_at("telegram:1") == datetime(
        2026, 5, 30, 10, 0, tzinfo=timezone.utc
    )
    assert [call[1] for call in calls] == [
        "/v1/proactive/deliveries",
        "/v1/proactive/deliveries/duplicate",
        "/v1/proactive/deliveries/count",
        "/v1/proactive/context-only",
        "/v1/proactive/context-only/count",
        "/v1/proactive/context-only/last",
        "/v1/proactive/drift-runs",
        "/v1/proactive/drift-runs/last",
    ]
    assert fallback.count_deliveries_in_window("telegram:1", 24, now) == 1

    store.close()


def test_agent_runtime_proactive_state_falls_back_to_sqlite_on_runtime_error(tmp_path):
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(500, text="runtime down")

    fallback = ProactiveStateStore(tmp_path / "proactive.db")
    store = AgentRuntimeProactiveStateStore(
        _config(),
        fallback,
        transport=httpx.MockTransport(handler),
    )
    now = datetime(2026, 5, 30, 8, 30, tzinfo=timezone.utc)

    store.mark_delivery("telegram:1", "delivery-a", now)

    assert store.is_delivery_duplicate("telegram:1", "delivery-a", 24, now) is True
    assert store.count_deliveries_in_window("telegram:1", 24, now) == 1

    store.mark_context_only_send("telegram:1", now)
    assert store.get_last_context_only_at("telegram:1") == now

    store.mark_drift_run("telegram:1", now)
    assert store.get_last_drift_at("telegram:1") == now

    store.close()


def test_build_proactive_state_store_wraps_sqlite_when_agent_runtime_enabled(tmp_path):
    cfg = SimpleNamespace(agent_runtime=_config())

    store = _build_proactive_state_store(cast(Any, cfg), tmp_path)

    assert isinstance(store, AgentRuntimeProactiveStateStore)
    store.close()
