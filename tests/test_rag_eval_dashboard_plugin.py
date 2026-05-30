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


def test_rag_eval_dashboard_plugin_summarizes_quality_gate_jobs(
    monkeypatch,
    tmp_path,
) -> None:
    jobs = [
        {
            "job_id": "rag_eval:qq:27234224:passed",
            "job_type": "rag_eval",
            "agent_id": "eval-worker",
            "status": "succeeded",
            "attempts": 1,
            "max_attempts": 1,
            "result": {
                "questions": "2",
                "top1_accuracy": "1.0",
                "evidence_coverage": "1.0",
                "min_top1_accuracy": "0.9",
                "min_evidence_coverage": "0.9",
                "passed": "true",
                "results": json.dumps(
                    [
                        {
                            "question": "显卡怎么选",
                            "top1_hit": True,
                            "evidence_hit": True,
                        }
                    ],
                    ensure_ascii=False,
                ),
            },
            "payload": {"fixture": "tests/fixtures/group_memory_open_strategy_dataset.json"},
            "metadata": {},
            "created_at": "2026-05-30T09:00:00Z",
            "updated_at": "2026-05-30T09:01:00Z",
        },
        {
            "job_id": "rag_eval:qq:27234224:failed-quality",
            "job_type": "rag_eval",
            "agent_id": "eval-worker",
            "status": "succeeded",
            "attempts": 1,
            "max_attempts": 1,
            "result": {
                "questions": "2",
                "top1_accuracy": "0.5",
                "evidence_coverage": "1.0",
                "min_top1_accuracy": "1.0",
                "min_evidence_coverage": "1.0",
                "passed": "false",
            },
            "payload": {},
            "metadata": {},
            "created_at": "2026-05-30T09:10:00Z",
            "updated_at": "2026-05-30T09:11:00Z",
        },
        {
            "job_id": "rag_eval:infra-failed",
            "job_type": "rag_eval",
            "agent_id": "eval-worker",
            "status": "failed",
            "attempts": 2,
            "max_attempts": 2,
            "error_message": "fixture missing",
            "result": {},
            "payload": {},
            "metadata": {},
            "created_at": "2026-05-30T09:20:00Z",
            "updated_at": "2026-05-30T09:21:00Z",
        },
        {
            "job_id": "rag_eval:running",
            "job_type": "rag_eval",
            "agent_id": "eval-worker",
            "status": "running",
            "attempts": 1,
            "max_attempts": 2,
            "lease_owner": "eval-worker",
            "result": {},
            "payload": {},
            "metadata": {},
            "created_at": "2026-05-30T09:30:00Z",
            "updated_at": "2026-05-30T09:31:00Z",
        },
    ]

    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        query = parse_qs(parsed.query)
        assert parsed.path == "/v1/jobs"
        assert query["type"] == ["rag_eval"]
        assert query["limit"] == ["10"]
        return _fake_urlopen_response(json.dumps({"code": "OK", "data": jobs}))

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        response = client.get(
            "/api/dashboard/rag-eval",
            params={"page_size": 10},
        )
        failed_response = client.get(
            "/api/dashboard/rag-eval",
            params={"page_size": 10, "quality_status": "failed_quality"},
        )

    assert response.status_code == 200
    payload = response.json()
    assert payload["total"] == 4
    assert payload["summary"]["completed"] == 2
    assert payload["summary"]["passed"] == 1
    assert payload["summary"]["failed_quality"] == 1
    assert payload["summary"]["infra_failed"] == 1
    assert payload["summary"]["active"] == 1
    assert payload["summary"]["pass_rate"] == 0.5
    assert payload["summary"]["average_top1_accuracy"] == 0.75
    assert len(payload["trend"]) == 2
    latest = payload["items"][0]
    assert latest["job_id"] == "rag_eval:running"
    first_completed = next(
        item for item in payload["items"] if item["job_id"] == "rag_eval:qq:27234224:passed"
    )
    assert first_completed["results"][0]["question"] == "显卡怎么选"
    assert failed_response.status_code == 200
    assert failed_response.json()["total"] == 1
    assert failed_response.json()["items"][0]["quality_status"] == "failed_quality"


def test_rag_eval_panel_assets_are_exposed(monkeypatch, tmp_path) -> None:
    def _fake_urlopen(request, timeout=None):  # type: ignore[no-untyped-def]
        target = request.full_url if hasattr(request, "full_url") else str(request)
        parsed = urlparse(target)
        if parsed.path.startswith("/v1/"):
            return _fake_urlopen_response(json.dumps({"code": "OK", "data": []}))
        raise AssertionError(f"unhandled request: {parsed.path}")

    monkeypatch.setattr("urllib.request.urlopen", _fake_urlopen)

    with TestClient(create_dashboard_app(tmp_path, memory_admin=_MemoryAdmin())) as client:
        plugins = client.get("/api/dashboard/plugins").json()
        plugin_panels = {
            item["id"]: item["panels"]
            for item in plugins
            if item["id"] == "rag_eval"
        }
        js_response = client.get("/plugins/rag_eval/dashboard_panel.js")
        css_response = client.get("/plugins/rag_eval/dashboard_panel.css")

    assert plugin_panels["rag_eval"][0]["name"] == "dashboard_panel"
    assert plugin_panels["rag_eval"][0]["has_css"] is True
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
