from __future__ import annotations

import json
from urllib.parse import parse_qs, urlparse
from typing import Any

from fastapi.testclient import TestClient

from bootstrap.dashboard_api import create_dashboard_app


class _MemoryAdmin:
    def describe(self):
        class _Desc:
            name = "default"

        return _Desc()

    def close(self) -> None:
        return None


def test_agent_jobs_dashboard_plugin_reads_filtered_gateway_jobs(monkeypatch, tmp_path) -> None:
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
            "source_event_ids": ["msg-1", "msg-2"],
            "source_asset_ids": [],
            "payload": {"group_id": "27234224", "observe_only": "true"},
            "status": "pending",
            "attempts": 0,
            "max_attempts": 3,
            "lease_owner": "",
            "lease_expires_at": "",
            "result": {},
            "error_message": "",
            "metadata": {},
            "created_at": "2026-05-30T10:00:00+08:00",
            "updated_at": "2026-05-30T10:00:10+08:00",
        },
        {
            "job_id": "rag_ingest:qq:3219982:ds:1",
            "job_type": "rag_ingest",
            "agent_id": "worker-1",
            "route": {
                "kind": "qq",
                "account_id": "2365524513",
                "conversation_id": "3219982",
                "conversation_type": "group",
            },
            "source_event_ids": [],
            "source_asset_ids": ["asset-1"],
            "payload": {"group_id": "3219982", "dataset_id": "d1"},
            "status": "running",
            "attempts": 1,
            "max_attempts": 2,
            "lease_owner": "worker-1",
            "lease_expires_at": "2026-05-30T10:05:00+08:00",
            "result": {},
            "error_message": "",
            "metadata": {},
            "created_at": "2026-05-30T09:55:00+08:00",
            "updated_at": "2026-05-30T10:00:00+08:00",
        },
    ]

    def _read_jobs() -> list[dict[str, Any]]:
        return jobs

    calls: list[tuple[str, str]] = []

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        path = parsed.path
        method = getattr(request, "get_method", lambda: "GET")()
        query = parse_qs(parsed.query)

        if method == "GET" and path == "/v1/jobs":
            if query.get("type") == ["group_memory_extract"] and query.get("status") == ["pending"]:
                return _fake_urlopen_response(json.dumps({"code": "OK", "data": [_read_jobs()[0]]}))
            if "limit" in query:
                return _fake_urlopen_response(json.dumps({"code": "OK", "data": jobs}))
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": []}))
        if method == "GET" and path.startswith("/v1/jobs/"):
            job_id = path.removeprefix("/v1/jobs/")
            job = next((item for item in jobs if item["job_id"] == job_id), jobs[0])
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": job}))
        if method == "POST" and (path.endswith("/retry") or path.endswith("/cancel")):
            calls.append((method, path))
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": {"job_id": path.split("/")[-2], "status": "running"}}))
        raise AssertionError(f"unhandled gateway call: {method} {path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        list_response = client.get(
            "/api/dashboard/agent-jobs",
            params={
                "job_type": "group_memory_extract",
                "status": "pending",
                "q": "27234224",
                "page": 1,
                "page_size": 10,
            },
        )
        payload = list_response.json()
        assert list_response.status_code == 200
        assert payload["total"] == 1
        item = payload["items"][0]
        assert item["job_type"] == "group_memory_extract"
        assert item["route_account_id"] == "2365524513"

        detail = client.get("/api/dashboard/agent-jobs/group_memory_extract:qq:27234224:1")
        assert detail.status_code == 200
        detail_data = detail.json()
        assert detail_data["payload"]["group_id"] == "27234224"

        retry_response = client.post("/api/dashboard/agent-jobs/group_memory_extract:qq:27234224:1/retry")
        cancel_response = client.post("/api/dashboard/agent-jobs/group_memory_extract:qq:27234224:1/cancel")
        assert retry_response.status_code == 200
        assert cancel_response.status_code == 200
        assert retry_response.json()["status"] == "running"
        assert cancel_response.json()["status"] == "running"


def test_agent_jobs_panel_assets_are_exposed(monkeypatch, tmp_path) -> None:
    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        url = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(url)
        path = parsed.path
        if path.startswith("/v1/"):
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": []}))
        raise AssertionError("unhandled request")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        plugins = client.get("/api/dashboard/plugins").json()
        plugin_panels = {
            item["id"]: item["panels"]
            for item in plugins
            if item["id"] == "agent_jobs"
        }
        js_response = client.get("/plugins/agent_jobs/dashboard_panel.js")
        css_response = client.get("/plugins/agent_jobs/dashboard_panel.css")

    assert plugin_panels["agent_jobs"][0]["name"] == "dashboard_panel"
    assert plugin_panels["agent_jobs"][0]["has_css"] is True
    assert js_response.status_code == 200
    assert css_response.status_code == 200


def _fake_urlopen_response(payload: str):
    class _Resp:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return None

        def read(self):
            return payload.encode("utf-8")

    return _Resp()
