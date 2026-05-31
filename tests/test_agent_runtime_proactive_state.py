from __future__ import annotations

import json
from datetime import datetime, timezone
from types import SimpleNamespace
from typing import Any, cast

import httpx

from agent.config_models import AgentRuntimeIntegrationConfig
from bootstrap.proactive import _build_proactive_state_store
from integrations.agent_runtime_proactive_state import AgentRuntimeProactiveStateStore
from proactive_v2.anyaction import QuotaStore
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
        if request.url.path == "/v1/proactive/tick-logs/start":
            assert body["tick_id"] == "tick-1"
            assert body["session_key"] == "telegram:1"
            return httpx.Response(202, json={"code": "OK", "data": body})
        if request.url.path == "/v1/proactive/tick-logs/finish":
            assert body["tick_id"] == "tick-1"
            assert body["terminal_action"] == "reply"
            assert body["interesting_ids"] == ["feed:1"]
            return httpx.Response(202, json={"code": "OK", "data": body})
        if request.url.path == "/v1/proactive/tick-steps":
            assert body["tick_id"] == "tick-1"
            assert body["tool_name"] == "message_push"
            assert body["tool_args"] == {"message": "hello"}
            return httpx.Response(202, json={"code": "OK", "data": body})
        if request.url.path == "/v1/proactive/deliveries/duplicate":
            assert params["window_hours"] == "24"
            return _ok({"duplicate": True})
        if request.url.path == "/v1/proactive/deliveries/count":
            return _ok({"count": 2})
        if request.url.path == "/v1/proactive/seen-items":
            assert body["entries"] == [
                {"source_key": "mcp:news:feed-a", "item_id": "item-a"}
            ]
            return httpx.Response(202, json={"code": "OK", "data": {"count": 1}})
        if request.url.path == "/v1/proactive/seen-items/seen":
            assert params["source_key"] == "mcp:news:feed-b"
            assert params["item_id"] == "item-a"
            return _ok({"seen": True, "source_key": "mcp:news", "item_id": "item-a"})
        if request.url.path == "/v1/proactive/rejection-cooldowns":
            assert body["hours"] == 2
            return httpx.Response(202, json={"code": "OK", "data": {"count": 1}})
        if request.url.path == "/v1/proactive/rejection-cooldowns/cooled":
            assert params["ttl_hours"] == "2"
            return _ok(
                {"cooled": True, "source_key": "qq:group:1", "item_id": "item-b"}
            )
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
        if request.url.path == "/v1/proactive/drift/finish":
            assert body["skill_used"] == "explore-curiosity"
            assert body["one_line"] == "整理攻略"
            assert body["next"] == "继续核验"
            return httpx.Response(202, json={"code": "OK", "data": {"side_effect": "runtime_state_write"}})
        if request.url.path == "/v1/proactive/drift/summary":
            assert params["limit"] == "10"
            return _ok(
                {
                    "version": 1,
                    "recent_runs": [
                        {
                            "skill": "explore-curiosity",
                            "run_at": "2026-05-30T10:10:00Z",
                            "one_line": "整理攻略",
                            "message_result": "silent",
                        }
                    ],
                    "note": "runtime note",
                    "side_effect": "none",
                }
            )
        if request.url.path == "/v1/proactive/drift/skills/explore-curiosity":
            return _ok(
                {
                    "skill_name": "explore-curiosity",
                    "last_run_at": "2026-05-30T10:10:00Z",
                    "run_count": 3,
                    "status": "in_progress",
                    "next": "继续核验",
                    "found": True,
                    "side_effect": "none",
                }
            )
        if request.url.path == "/v1/proactive/bg-context/main":
            assert body["timestamp"] == now.isoformat()
            return httpx.Response(202, json={"code": "OK", "data": body})
        if request.url.path == "/v1/proactive/bg-context/main/last":
            return _ok({"found": True, "timestamp": "2026-05-30T10:30:00Z"})
        if request.url.path == "/v1/proactive/cleanup":
            assert body["seen_ttl_hours"] == 24
            assert body["delivery_ttl_hours"] == 48
            assert body["context_only_ttl_hours"] == 24
            assert body["rejection_cooldown_ttl_hours"] == 2
            return httpx.Response(202, json={"code": "OK", "data": {"removed_seen_items": 0}})
        return httpx.Response(404, text="not found")

    fallback = ProactiveStateStore(tmp_path / "proactive.db")
    store = AgentRuntimeProactiveStateStore(
        _config(),
        fallback,
        transport=httpx.MockTransport(handler),
    )
    now = datetime(2026, 5, 30, 8, 30, tzinfo=timezone.utc)

    store.mark_delivery("telegram:1", "delivery-a", now)
    store.record_tick_log_start(
        tick_id="tick-1",
        session_key="telegram:1",
        started_at=now.isoformat(),
        gate_exit=None,
    )
    store.record_tick_step_log(
        tick_id="tick-1",
        step_index=1,
        phase="loop",
        tool_name="message_push",
        tool_call_id="call-1",
        tool_args={"message": "hello"},
        tool_result_text='{"ok":true}',
        terminal_action_after=None,
        skip_reason_after="",
        interesting_ids_after=["feed:1"],
        discarded_ids_after=[],
        cited_ids_after=[],
        final_message_after="hello",
    )
    store.record_tick_log_finish(
        tick_id="tick-1",
        session_key="telegram:1",
        started_at=now.isoformat(),
        finished_at=now.isoformat(),
        gate_exit=None,
        terminal_action="reply",
        skip_reason="",
        steps_taken=1,
        alert_count=0,
        content_count=1,
        context_count=0,
        interesting_ids=["feed:1"],
        discarded_ids=[],
        cited_ids=[],
        drift_entered=False,
        final_message="hello",
    )

    assert store.is_delivery_duplicate("telegram:1", "delivery-a", 24, now) is True
    assert store.count_deliveries_in_window("telegram:1", 24, now) == 2

    store.mark_items_seen([("mcp:news:feed-a", "item-a")], now)
    assert store.is_item_seen("mcp:news:feed-b", "item-a", 24, now) is True

    store.mark_rejection_cooldown([("qq:group:1", "item-b")], 2, now)
    assert store.is_rejection_cooled("qq:group:1", "item-b", 2, now) is True

    store.mark_context_only_send("telegram:1", now)

    assert store.count_context_only_in_window("telegram:1", 24, now) == 1
    assert store.get_last_context_only_at("telegram:1") == datetime(
        2026, 5, 30, 9, 0, tzinfo=timezone.utc
    )

    store.mark_drift_run("telegram:1", now)

    assert store.get_last_drift_at("telegram:1") == datetime(
        2026, 5, 30, 10, 0, tzinfo=timezone.utc
    )

    finish = store.record_drift_finish(
        skill_used="explore-curiosity",
        one_line="整理攻略",
        next_action="继续核验",
        message_result="silent",
        note=None,
        now_utc=now,
    )
    assert finish["side_effect"] == "runtime_state_write"
    assert store.get_drift_summary()["note"] == "runtime note"
    assert store.get_drift_skill_state("explore-curiosity")["run_count"] == 3

    store.mark_bg_context_main_send(now)

    assert store.get_bg_context_last_main_at() == datetime(
        2026, 5, 30, 10, 30, tzinfo=timezone.utc
    )

    assert fallback.count_deliveries_in_window("telegram:1", 24, now) == 1
    assert fallback.is_item_seen("mcp:news:feed-b", "item-a", 24, now) is True
    assert fallback.is_rejection_cooled("qq:group:1", "item-b", 2, now) is True
    assert fallback._count_rows("tick_log") == 1
    assert fallback._count_rows("tick_step_log") == 1

    store.cleanup(24, 48, 72, 2)
    assert [call[1] for call in calls] == [
        "/v1/proactive/deliveries",
        "/v1/proactive/tick-logs/start",
        "/v1/proactive/tick-steps",
        "/v1/proactive/tick-logs/finish",
        "/v1/proactive/deliveries/duplicate",
        "/v1/proactive/deliveries/count",
        "/v1/proactive/seen-items",
        "/v1/proactive/seen-items/seen",
        "/v1/proactive/rejection-cooldowns",
        "/v1/proactive/rejection-cooldowns/cooled",
        "/v1/proactive/context-only",
        "/v1/proactive/context-only/count",
        "/v1/proactive/context-only/last",
        "/v1/proactive/drift-runs",
        "/v1/proactive/drift-runs/last",
        "/v1/proactive/drift/finish",
        "/v1/proactive/drift/summary",
        "/v1/proactive/drift/skills/explore-curiosity",
        "/v1/proactive/bg-context/main",
        "/v1/proactive/bg-context/main/last",
        "/v1/proactive/cleanup",
    ]

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

    store.mark_items_seen([("mcp:news:feed-a", "item-a")], now)
    assert store.is_item_seen("mcp:news:feed-b", "item-a", 24, now) is True

    store.mark_rejection_cooldown([("qq:group:1", "item-b")], 2, now)
    assert store.is_rejection_cooled("qq:group:1", "item-b", 2, now) is True

    store.cleanup(24, 48, 72, 2)

    store.mark_context_only_send("telegram:1", now)
    assert store.get_last_context_only_at("telegram:1") == now

    store.mark_drift_run("telegram:1", now)
    assert store.get_last_drift_at("telegram:1") == now

    store.mark_bg_context_main_send(now)
    assert store.get_bg_context_last_main_at() == now

    store.close()


def test_agent_runtime_proactive_state_uses_go_for_anyaction_quota(tmp_path):
    calls: list[tuple[str, str, dict[str, str], dict[str, Any]]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        params = dict(request.url.params)
        calls.append((request.method, request.url.path, params, body))
        if request.url.path == "/v1/proactive/anyaction/quota":
            assert params["quota_key"] == "default"
            assert params["reset_hour"] == "12"
            assert params["timezone"] == "Asia/Shanghai"
            return _ok(
                {
                    "quota_key": "default",
                    "window_key": "2026-05-30@12@Asia/Shanghai",
                    "next_reset_at": "2026-05-31T04:00:00Z",
                    "used": 2,
                    "last_action_at": "2026-05-30T08:00:00Z",
                }
            )
        if request.url.path == "/v1/proactive/anyaction/actions":
            assert body["quota_key"] == "default"
            assert body["reset_hour"] == 12
            assert body["timezone"] == "Asia/Shanghai"
            return httpx.Response(202, json={"code": "OK", "data": body})
        return httpx.Response(404, text="not found")

    fallback = ProactiveStateStore(tmp_path / "proactive.db")
    store = AgentRuntimeProactiveStateStore(
        _config(),
        fallback,
        transport=httpx.MockTransport(handler),
    )
    quota = store.anyaction_quota_store(QuotaStore(tmp_path / "quota.json"))
    now = datetime(2026, 5, 30, 8, 30, tzinfo=timezone.utc)

    snapshot = quota.snapshot(
        now_utc=now,
        reset_hour=12,
        timezone_name="Asia/Shanghai",
    )
    quota.record_action(
        now_utc=now,
        reset_hour=12,
        timezone_name="Asia/Shanghai",
    )

    assert snapshot.used == 2
    assert snapshot.window_key == "2026-05-30@12@Asia/Shanghai"
    assert [call[1] for call in calls] == [
        "/v1/proactive/anyaction/quota",
        "/v1/proactive/anyaction/actions",
    ]
    assert QuotaStore(tmp_path / "quota.json").snapshot(
        now_utc=now,
        reset_hour=12,
        timezone_name="Asia/Shanghai",
    ).used == 1
    store.close()


def test_build_proactive_state_store_wraps_sqlite_when_agent_runtime_enabled(tmp_path):
    cfg = SimpleNamespace(agent_runtime=_config())

    store = _build_proactive_state_store(cast(Any, cfg), tmp_path)

    assert isinstance(store, AgentRuntimeProactiveStateStore)
    store.close()
