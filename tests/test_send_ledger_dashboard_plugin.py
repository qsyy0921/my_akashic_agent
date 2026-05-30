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


def test_send_ledger_dashboard_plugin_reads_records_and_recent(monkeypatch, tmp_path) -> None:
    records = [
        {
            "from_bot_id": "1049511700",
            "conversation_id": "2365524513",
            "content_hash": "sha256:reply-hash-a",
            "timestamp": "2026-05-30T00:40:00+08:00",
        },
        {
            "from_bot_id": "2365524513",
            "conversation_id": "1049511700",
            "content_hash": "sha256:reply-hash-b",
            "timestamp": "2026-05-30T00:41:00+08:00",
        },
    ]

    seen_queries: list[dict[str, list[str]]] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        path = parsed.path
        query = parse_qs(parsed.query)
        seen_queries.append(query)

        if path == "/v1/send-ledger/records":
            if query.get("from_bot_id") == ["1049511700"]:
                return _fake_urlopen_response(json.dumps({"code": "OK", "data": [records[0]]}))
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": records}))
        if path == "/v1/send-ledger/recent":
            return _fake_urlopen_response(
                json.dumps(
                    {
                        "code": "OK",
                        "data": {
                            "recent": True,
                            "from_bot_id": query.get("from_bot_id", [""])[0],
                            "conversation_id": query.get("conversation_id", [""])[0],
                            "content_hash": "sha256:reply-hash-a",
                            "window_seconds": 60,
                        },
                    }
                )
            )
        raise AssertionError(f"unhandled runtime call: {path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        list_response = client.get(
            "/api/dashboard/send-ledger",
            params={
                "from_bot_id": "1049511700",
                "q": "2365524513",
                "page": 1,
                "page_size": 10,
            },
        )
        assert list_response.status_code == 200
        payload = list_response.json()
        assert payload["total"] == 1
        item = payload["items"][0]
        assert item["from_bot_id"] == "1049511700"
        assert item["conversation_id"] == "2365524513"
        assert item["short_hash"] == "sha256:reply"
        assert payload["status"]["runtime_available"] is True

        recent_response = client.get(
            "/api/dashboard/send-ledger/recent",
            params={
                "from_bot_id": "1049511700",
                "conversation_id": "2365524513",
                "content": "帮我生成一个古装美女",
                "window_seconds": 60,
            },
        )
        assert recent_response.status_code == 200
        recent = recent_response.json()
        assert recent["recent"] is True
        assert recent["content_hash"] == "sha256:reply-hash-a"

    assert any(query.get("from_bot_id") == ["1049511700"] for query in seen_queries)


def test_send_ledger_panel_assets_are_exposed(monkeypatch, tmp_path) -> None:
    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        if parsed.path.startswith("/v1/send-ledger/"):
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
            if item["id"] == "send_ledger"
        }
        js_response = client.get("/plugins/send_ledger/dashboard_panel.js")
        css_response = client.get("/plugins/send_ledger/dashboard_panel.css")

    assert plugin_panels["send_ledger"][0]["name"] == "dashboard_panel"
    assert plugin_panels["send_ledger"][0]["has_css"] is True
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
