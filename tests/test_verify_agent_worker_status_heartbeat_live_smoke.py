from __future__ import annotations

import importlib.util
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

import httpx


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_worker_status_heartbeat_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_worker_status_heartbeat_live_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class FakeHeartbeatRuntime:
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
            items = []
            for item in self.records.values():
                items.append(
                    {
                        "worker_id": item["WorkerID"],
                        "instance_id": item["InstanceID"],
                        "worker_type": item["WorkerType"],
                        "status": item["Status"],
                        "updated_at": item["UpdatedAt"],
                        "lease_until": item["LeaseUntil"],
                        "lease_active": True,
                        "stale": False,
                    }
                )
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
            timestamp = datetime.fromisoformat(body["timestamp"]).astimezone(timezone.utc)
            lease_until = timestamp + timedelta(seconds=int(body.get("lease_ttl_seconds") or 120))
            self.records[worker_id] = {
                "WorkerID": worker_id,
                "InstanceID": str(body["instance_id"]),
                "WorkerType": body["worker_type"],
                "Status": body["status"],
                "UpdatedAt": timestamp.isoformat(),
                "LeaseUntil": lease_until.isoformat(),
            }
            self._write_state()
            return httpx.Response(
                202,
                json={
                    "code": "OK",
                    "data": {
                        "worker_id": worker_id,
                        "instance_id": body["instance_id"],
                        "accepted": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        raise AssertionError(f"unexpected request: {request.method} {request.url}")


def test_run_worker_status_heartbeat_smoke_returns_expected_evidence(tmp_path):
    module = _load_module()
    runtime = FakeHeartbeatRuntime(tmp_path / "agent-worker-statuses.json")
    client = module.HELPERS.WorkerStatusRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )
    now = datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    result = module.run_worker_status_heartbeat_smoke(
        client,
        state_file=runtime.state_file,
        worker_id="worker-heartbeat",
        now=now,
    )

    assert all(result["checks"].values())
    assert result["status_codes"] == {"first": 202, "second": 202, "third": 202}
    assert result["final_record"]["lease_active"] is True
    assert result["final_record"]["stale"] is False
    assert ("POST", "/v1/agent-worker-statuses/report") in runtime.requests
    assert ("GET", "/v1/agent-worker-statuses") in runtime.requests


def test_parse_iso_datetime_supports_z_suffix():
    module = _load_module()

    parsed = module._parse_iso_datetime("2026-06-03T12:00:00Z")

    assert parsed is not None
    assert parsed == datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)
