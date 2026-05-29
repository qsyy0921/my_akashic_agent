from __future__ import annotations

import json
from pathlib import Path
from types import SimpleNamespace
from urllib.parse import urlparse

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
    assert item["attachments"][0]["content_url"] == (
        "/api/dashboard/media-assets/content?asset_id=asset%3A1"
    )
    assert item["metadata"]["observe_only"] == "true"


def test_dashboard_media_asset_content_proxy_uses_agent_runtime(
    tmp_path,
    monkeypatch,
) -> None:
    calls: list[tuple[str, float | None]] = []

    def _fake_urlopen(url, timeout=None):  # type: ignore[no-untyped-def]
        target = str(url)
        parsed = urlparse(target)
        calls.append((target, timeout))
        assert parsed.scheme == "http"
        assert parsed.netloc == "agent-runtime.local"
        assert parsed.path == "/v1/media-assets/asset%3Aqq%3A1/content"
        return _fake_bytes_response(
            b"image-bytes",
            {
                "Content-Type": "image/png",
                "Content-Disposition": 'inline; filename="photo.png"',
            },
        )

    monkeypatch.setenv("AKASHIC_AGENT_RUNTIME_URL", "http://agent-runtime.local")
    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        response = client.get(
            "/api/dashboard/media-assets/content",
            params={"asset_id": "asset:qq:1"},
        )

    assert response.status_code == 200
    assert response.content == b"image-bytes"
    assert response.headers["content-type"].startswith("image/png")
    assert response.headers["content-disposition"] == 'inline; filename="photo.png"'
    assert calls


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


def _fake_bytes_response(payload: bytes, headers: dict[str, str]):
    class _Resp:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return None

        def read(self):
            return payload

    resp = _Resp()
    resp.headers = headers
    return resp
