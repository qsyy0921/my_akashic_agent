from __future__ import annotations

import json
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs, unquote, urlparse

from fastapi.testclient import TestClient

from bootstrap.dashboard_api import create_dashboard_app

CONTRACT_DIR = Path(__file__).resolve().parents[1] / "tests" / "fixtures" / "contracts"


class _MemoryAdmin:
    def describe(self):
        class _Desc:
            name = "default"

        return _Desc()

    def close(self) -> None:
        return None


def test_knowledge_checkpoints_dashboard_plugin_reads_runtime(monkeypatch, tmp_path) -> None:
    checkpoints = [
        {
            "checkpoint_id": "ragflow:qq:27234224:ds-main",
            "cursor": 2623,
            "updated_at": "2026-05-30T08:12:18Z",
            "metadata": {
                "source": "qq",
                "group_id": "27234224",
                "dataset_id": "ds-main",
            },
        },
        {
            "checkpoint_id": "ragflow:qq:3219982:ds-main",
            "cursor": 2706,
            "updated_at": "2026-05-30T08:12:42Z",
            "metadata": {
                "source": "qq",
                "group_id": "3219982",
                "dataset_id": "ds-main",
            },
        },
    ]
    seen_queries: list[dict[str, list[str]]] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        path = parsed.path
        query = parse_qs(parsed.query)
        seen_queries.append(query)
        if path == "/v1/knowledge-checkpoints":
            assert query.get("prefix") == ["ragflow:qq:"]
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": checkpoints}))
        if path.startswith("/v1/knowledge-checkpoints/"):
            checkpoint_id = unquote(path.removeprefix("/v1/knowledge-checkpoints/"))
            item = next(row for row in checkpoints if row["checkpoint_id"] == checkpoint_id)
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": item}))
        raise AssertionError(f"unhandled runtime call: {path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        list_response = client.get(
            "/api/dashboard/knowledge-checkpoints",
            params={
                "prefix": "ragflow:qq:",
                "group_id": "27234224",
                "page": 1,
                "page_size": 10,
            },
        )
        assert list_response.status_code == 200
        payload = list_response.json()
        assert payload["total"] == 1
        item = payload["items"][0]
        assert item["checkpoint_id"] == "ragflow:qq:27234224:ds-main"
        assert item["cursor"] == 2623
        assert item["group_id"] == "27234224"
        assert item["dataset_id"] == "ds-main"
        assert payload["status"]["runtime_available"] is True

        detail_response = client.get(
            "/api/dashboard/knowledge-checkpoints/ragflow%3Aqq%3A27234224%3Ads-main"
        )
        assert detail_response.status_code == 200
        detail = detail_response.json()
        assert detail["metadata"]["source"] == "qq"

    assert any(query.get("limit") == ["10"] for query in seen_queries)


def test_knowledge_checkpoints_panel_assets_are_exposed(monkeypatch, tmp_path) -> None:
    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        if parsed.path.startswith("/v1/knowledge-checkpoints"):
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
            if item["id"] == "knowledge_checkpoints"
        }
        js_response = client.get("/plugins/knowledge_checkpoints/dashboard_panel.js")
        css_response = client.get("/plugins/knowledge_checkpoints/dashboard_panel.css")

    assert plugin_panels["knowledge_checkpoints"][0]["name"] == "dashboard_panel"
    assert plugin_panels["knowledge_checkpoints"][0]["has_css"] is True
    assert js_response.status_code == 200
    assert css_response.status_code == 200


def test_knowledge_checkpoint_contract_fixture_exposes_dashboard_lag(
    monkeypatch,
    tmp_path,
) -> None:
    fixture = _load_contract("knowledge_checkpoint.ragflow.qq.json")
    runtime_item = {
        "checkpoint_id": fixture["checkpoint_id"],
        "cursor": fixture["cursor"],
        "updated_at": fixture["timestamp"],
        "metadata": fixture["metadata"],
    }

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        if parsed.path == "/v1/knowledge-checkpoints":
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": [runtime_item]}))
        if parsed.path.startswith("/v1/knowledge-checkpoints/"):
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": runtime_item}))
        raise AssertionError(f"unhandled runtime call: {parsed.path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        response = client.get(
            "/api/dashboard/knowledge-checkpoints",
            params={
                "prefix": "ragflow:qq:",
                "group_id": fixture["conversation_id"],
                "page_size": 10,
            },
        )
        detail_response = client.get(
            f"/api/dashboard/knowledge-checkpoints/{fixture['checkpoint_id']}"
        )

    assert response.status_code == 200
    item = response.json()["items"][0]
    assert item["checkpoint_id"] == fixture["checkpoint_id"]
    assert item["checkpoint_lag_messages"] == 17
    assert item["latest_source_seq"] == 119
    assert detail_response.status_code == 200
    assert detail_response.json()["metadata"]["ragflow_document_id"] == (
        "ragflow-doc:hardware-thread-001"
    )


def _fake_urlopen_response(payload: str):
    class _Resp:
        def __enter__(self):
            return self

        def __exit__(self, *_args: Any):
            return None

        def read(self):
            return payload.encode("utf-8")

    return _Resp()


def _load_contract(name: str) -> dict[str, Any]:
    return json.loads((CONTRACT_DIR / name).read_text(encoding="utf-8"))
