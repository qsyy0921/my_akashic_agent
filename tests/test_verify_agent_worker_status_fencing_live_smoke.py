from __future__ import annotations

import importlib.util
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

import httpx


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_worker_status_fencing_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_worker_status_fencing_live_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class FakeWorkerStatusRuntime:
    def __init__(self, state_file: Path) -> None:
        self.state_file = state_file
        self.records: dict[str, dict] = {}
        self.requests: list[tuple[str, str]] = []
        self._write_state()

    def _write_state(self) -> None:
        payload = {
            "version": "2026-06-03.agentworkerstatusstore.v1",
            "workers": list(self.records.values()),
        }
        self.state_file.parent.mkdir(parents=True, exist_ok=True)
        self.state_file.write_text(
            json.dumps(payload, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )

    def handler(self, request: httpx.Request) -> httpx.Response:
        self.requests.append((request.method, request.url.path))
        if request.url.path == "/v1/agent-worker-statuses" and request.method == "GET":
            items = sorted(self.records.values(), key=lambda item: str(item["worker_id"]))
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "workers": items,
                        "totals": {"workers": len(items)},
                        "side_effect": "none",
                    },
                },
            )
        if request.url.path == "/v1/agent-worker-statuses/report" and request.method == "POST":
            body = json.loads(request.read().decode("utf-8"))
            worker_id = str(body["worker_id"])
            instance_id = str(body["instance_id"])
            replace_existing = str(body.get("replace_existing_instance_id") or "")
            existing = self.records.get(worker_id)
            if existing and str(existing["instance_id"]) != instance_id:
                if replace_existing != str(existing["instance_id"]):
                    return httpx.Response(
                        409,
                        text=(
                            "agent worker status lease conflict: "
                            f"worker_id={worker_id} existing_instance_id={existing['instance_id']} "
                            "lease_until=2026-06-03T00:00:00Z"
                        ),
                    )
            self.records[worker_id] = {
                "worker_id": worker_id,
                "instance_id": instance_id,
                "worker_type": body["worker_type"],
                "status": body["status"],
                "lease_active": True,
                "updated_at": body["timestamp"],
            }
            self._write_state()
            return httpx.Response(
                202,
                json={
                    "code": "OK",
                    "data": {
                        "worker_id": worker_id,
                        "instance_id": instance_id,
                        "accepted": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        raise AssertionError(f"unexpected request: {request.method} {request.url}")


def test_run_worker_status_fencing_smoke_returns_expected_evidence(tmp_path):
    module = _load_module()
    runtime = FakeWorkerStatusRuntime(tmp_path / "agent-worker-statuses.json")
    client = module.WorkerStatusRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )
    now = datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    result = module.run_worker_status_fencing_smoke(
        client,
        state_file=runtime.state_file,
        worker_id="worker-a",
        now=now,
    )

    assert all(result["checks"].values())
    assert result["initial_status_code"] == 202
    assert result["conflict_status_code"] == 409
    assert result["takeover_status_code"] == 202
    assert "existing_instance_id=" in result["conflict_text"]
    assert ("POST", "/v1/agent-worker-statuses/report") in runtime.requests
    assert ("GET", "/v1/agent-worker-statuses") in runtime.requests


def test_make_report_payload_defaults_replace_existing_instance_id():
    module = _load_module()
    payload = module._make_report_payload(
        worker_id="worker-a",
        instance_id="worker-a:123:abc",
        timestamp=datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc),
    )

    assert payload["replace_existing_instance_id"] == ""
    assert payload["worker_id"] == "worker-a"
    assert payload["worker_type"] == "knowledge"
