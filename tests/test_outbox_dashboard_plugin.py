from __future__ import annotations

import json
from typing import Any
from urllib.parse import parse_qs, unquote, urlparse

from fastapi.testclient import TestClient

from bootstrap.dashboard_api import create_dashboard_app


class _MemoryAdmin:
    def describe(self):
        class _Desc:
            name = "default"

        return _Desc()

    def close(self) -> None:
        return None


def test_outbox_dashboard_plugin_reads_filters_and_actions(monkeypatch, tmp_path) -> None:
    deliveries = [
        _delivery("outbox:1", "failed", "1049511700", "2365524513", "platform timeout"),
        _delivery("outbox:2", "queued", "2365524513", "1049511700", ""),
    ]
    calls: list[tuple[str, str, dict[str, Any]]] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        path = unquote(parsed.path)
        method = getattr(request, "get_method", lambda: "GET")()
        query = parse_qs(parsed.query)

        if method == "GET" and path == "/v1/outbox":
            assert int(query.get("limit", ["0"])[0]) >= 1
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": deliveries}))
        if method == "GET" and path.startswith("/v1/outbox/"):
            event_id = path.removeprefix("/v1/outbox/")
            item = next((delivery for delivery in deliveries if delivery["event_id"] == event_id), None)
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": item or {}}))
        if method == "POST" and path.startswith("/v1/outbox/"):
            event_id, action = path.removeprefix("/v1/outbox/").rsplit("/", 1)
            raw = request.data.decode("utf-8") if getattr(request, "data", None) else "{}"
            body = json.loads(raw)
            calls.append((event_id, action, body))
            item = _delivery(event_id, "queued" if action == "retry" else action, "1049511700", "2365524513", body.get("error_message", ""))
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": item}))
        raise AssertionError(f"unhandled runtime call: {method} {path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        list_response = client.get(
            "/api/dashboard/outbox",
            params={
                "status": "failed",
                "account_id": "1049511700",
                "q": "timeout",
                "page": 1,
                "page_size": 10,
            },
        )
        assert list_response.status_code == 200
        payload = list_response.json()
        assert payload["total"] == 1
        item = payload["items"][0]
        assert item["event_id"] == "outbox:1"
        assert item["status"] == "failed"
        assert item["account_id"] == "1049511700"
        assert payload["status"]["runtime_available"] is True

        detail_response = client.get("/api/dashboard/outbox/outbox%3A1")
        assert detail_response.status_code == 200
        assert detail_response.json()["error_message"] == "platform timeout"

        retry_response = client.post("/api/dashboard/outbox/outbox%3A1/retry", json={})
        fail_response = client.post(
            "/api/dashboard/outbox/outbox%3A1/failed",
            json={"error_message": "manual test failure"},
        )
        assert retry_response.status_code == 200
        assert retry_response.json()["status"] == "queued"
        assert fail_response.status_code == 200
        assert fail_response.json()["status"] == "failed"

    assert ("outbox:1", "retry", {}) in calls
    assert ("outbox:1", "failed", {"error_message": "manual test failure"}) in calls


def test_outbox_panel_assets_are_exposed(monkeypatch, tmp_path) -> None:
    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        if parsed.path.startswith("/v1/outbox"):
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": []}))
        if parsed.path.startswith("/v1/"):
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": []}))
        raise AssertionError("unhandled request")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        plugins = client.get("/api/dashboard/plugins").json()
        plugin_panels = {
            item["id"]: item["panels"]
            for item in plugins
            if item["id"] == "outbox"
        }
        js_response = client.get("/plugins/outbox/dashboard_panel.js")
        css_response = client.get("/plugins/outbox/dashboard_panel.css")

    assert plugin_panels["outbox"][0]["name"] == "dashboard_panel"
    assert plugin_panels["outbox"][0]["has_css"] is True
    assert js_response.status_code == 200
    assert css_response.status_code == 200


def _delivery(
    event_id: str,
    status: str,
    account_id: str,
    conversation_id: str,
    error_message: str,
) -> dict[str, Any]:
    return {
        "event_id": event_id,
        "channel": {
            "kind": "qq",
            "account_id": account_id,
            "conversation_id": conversation_id,
            "conversation_type": "private",
        },
        "content": "generated image is ready",
        "attachments": [{"kind": "image", "name": "result.png"}],
        "status": status,
        "attempts": 1,
        "max_attempts": 2,
        "error_message": error_message,
        "metadata": {"source": "test"},
        "created_at": "2026-05-30T10:00:00+08:00",
        "updated_at": "2026-05-30T10:01:00+08:00",
    }


def _fake_urlopen_response(payload: str):
    class _Resp:
        def __enter__(self):
            return self

        def __exit__(self, *_args: Any):
            return None

        def read(self):
            return payload.encode("utf-8")

    return _Resp()
