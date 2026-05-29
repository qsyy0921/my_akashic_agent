from __future__ import annotations

import json
from pathlib import Path
from types import SimpleNamespace

from fastapi.testclient import TestClient

from bootstrap.dashboard_api import create_dashboard_app


class _MemoryAdmin:
    def describe(self):
        return SimpleNamespace(name="default")

    def close(self) -> None:
        return None


def test_shadow_audit_dashboard_reads_jsonl_fallback(tmp_path, monkeypatch) -> None:
    monkeypatch.setenv("AKASHIC_SHADOW_GATEWAY_URL", "http://127.0.0.1:1")
    _write_shadow_line(
        tmp_path / "shadow" / "session.jsonl",
        {
            "schema_version": "2026-05-30.v1",
            "kind": "MessageEnvelope",
            "event_id": "qq:2365524513:group:27234224:msg-1",
            "platform": "qq",
            "account_id": "2365524513",
            "conversation_id": "27234224",
            "conversation_type": "group",
            "sender": {"id": "1049511700", "kind": "human"},
            "content": "hardware photo",
            "attachments": [
                {
                    "id": "asset:1",
                    "kind": "image",
                    "url": "file:///E:/agent/akashic/.tmp/photo.png",
                    "mime_type": "image/png",
                    "name": "photo.png",
                }
            ],
            "timestamp": "2026-05-30T00:40:00+08:00",
            "metadata": {"observe_only": "true"},
        },
    )

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        response = client.get(
            "/api/dashboard/shadow-audit/observed",
            params={"source": "jsonl", "q": "hardware"},
        )

    assert response.status_code == 200
    payload = response.json()
    assert payload["total"] == 1
    item = payload["items"][0]
    assert item["event_id"] == "qq:2365524513:group:27234224:msg-1"
    assert item["attachment_count"] == 1
    assert item["attachments"][0]["name"] == "photo.png"
    assert item["metadata"]["observe_only"] == "true"


def test_shadow_audit_plugin_assets_are_exposed(tmp_path) -> None:
    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        plugins = client.get("/api/dashboard/plugins").json()
        panels = {
            item["id"]: item["panels"]
            for item in plugins
            if item["id"] == "shadow_audit"
        }
        js_response = client.get("/plugins/shadow_audit/dashboard_panel.js")
        css_response = client.get("/plugins/shadow_audit/dashboard_panel.css")

    assert panels["shadow_audit"][0]["name"] == "dashboard_panel"
    assert panels["shadow_audit"][0]["has_css"] is True
    assert js_response.status_code == 200
    assert css_response.status_code == 200


def _write_shadow_line(path: Path, payload: dict[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False) + "\n", encoding="utf-8")
