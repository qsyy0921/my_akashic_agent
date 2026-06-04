from __future__ import annotations

import importlib.util
import json
import sys
from pathlib import Path

import httpx


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_worker_status_startup_prune_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_worker_status_startup_prune_live_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class FakeStartupPruneRuntime:
    def __init__(self) -> None:
        self.requests: list[tuple[str, str]] = []

    def handler(self, request: httpx.Request) -> httpx.Response:
        self.requests.append((request.method, request.url.path))
        if request.url.path == "/v1/agent-worker-statuses" and request.method == "GET":
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "workers": [
                            {
                                "worker_id": "worker-active",
                                "instance_id": "worker-active:22222:active",
                                "worker_type": "knowledge",
                                "status": "running",
                                "stale": False,
                            }
                        ],
                        "totals": {"workers": 1, "stale": 0},
                        "side_effect": "none",
                    },
                },
            )
        if request.url.path == "/v1/runtime-overview" and request.method == "GET":
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {"summary": {"agent_workers_stale": 0}},
                },
            )
        raise AssertionError(f"unexpected request: {request.method} {request.url}")


def test_verify_startup_prune_state_returns_expected_evidence(tmp_path):
    module = _load_module()
    state_file = tmp_path / "agent-worker-statuses.json"
    state_file.write_text(
        json.dumps(
            {
                "version": "2026-06-03.agentworkerstatusstore.v1",
                "workers": [
                    {
                        "WorkerID": "worker-active",
                        "InstanceID": "worker-active:22222:active",
                        "WorkerType": "knowledge",
                        "Status": "running",
                        "UpdatedAt": "2026-06-03T12:00:00+00:00",
                        "LeaseUntil": "2026-06-03T12:02:00+00:00",
                    }
                ],
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    runtime = FakeStartupPruneRuntime()
    client = module.HELPERS.WorkerStatusRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )

    result = module.verify_startup_prune_state(
        client,
        state_file=state_file,
        stale_worker_id="worker-stale",
        active_worker_id="worker-active",
    )

    assert all(result["checks"].values())
    assert result["workers_totals"]["workers"] == 1
    assert result["runtime_overview_summary"]["agent_workers_stale"] == 0
    assert ("GET", "/v1/agent-worker-statuses") in runtime.requests
    assert ("GET", "/v1/runtime-overview") in runtime.requests
