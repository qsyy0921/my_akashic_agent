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
    / "verify_agent_worker_status_cleanup_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_worker_status_cleanup_live_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class FakeCleanupRuntime:
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

    def _is_stale(self, item: dict, stale_after_seconds: int = 60) -> bool:
        updated_at = datetime.fromisoformat(str(item["UpdatedAt"])).astimezone(timezone.utc)
        return item["Status"] in {"starting", "idle", "running"} and updated_at <= (
            self.now - timedelta(seconds=stale_after_seconds)
        )

    def _view_item(self, item: dict, stale_after_seconds: int = 60) -> dict:
        stale = self._is_stale(item, stale_after_seconds=stale_after_seconds)
        return {
            "worker_id": item["WorkerID"],
            "instance_id": item["InstanceID"],
            "worker_type": item["WorkerType"],
            "status": "stopped" if stale else item["Status"],
            "updated_at": item["UpdatedAt"],
            "lease_until": item["LeaseUntil"],
            "lease_active": not stale,
            "stale": stale,
            "last_error": (
                "last agent worker heartbeat exceeded stale threshold" if stale else ""
            ),
        }

    @property
    def now(self) -> datetime:
        return datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    def handler(self, request: httpx.Request) -> httpx.Response:
        self.requests.append((request.method, request.url.path))
        if request.url.path == "/v1/agent-worker-statuses" and request.method == "GET":
            items = [self._view_item(item) for item in self.records.values()]
            stale_total = sum(1 for item in items if item["stale"])
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "workers": sorted(items, key=lambda item: item["worker_id"]),
                        "totals": {"workers": len(items), "stale": stale_total},
                        "side_effect": "none",
                    },
                },
            )
        if request.url.path == "/v1/runtime-overview" and request.method == "GET":
            stale_total = sum(1 for item in self.records.values() if self._is_stale(item))
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {"summary": {"agent_workers_stale": stale_total}},
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
                    "data": {"worker_id": worker_id, "accepted": True},
                },
            )
        if request.url.path == "/v1/agent-worker-statuses/cleanup-stale" and request.method == "POST":
            body = json.loads(request.read().decode("utf-8"))
            stale_after_seconds = int(body.get("stale_after_seconds") or 60)
            worker_id = str(body.get("worker_id") or "")
            instance_id = str(body.get("instance_id") or "")
            deleted = []
            for key, item in list(self.records.items()):
                if worker_id and key != worker_id:
                    continue
                if instance_id and str(item["InstanceID"]) != instance_id:
                    continue
                if not self._is_stale(item, stale_after_seconds=stale_after_seconds):
                    continue
                deleted.append(self._view_item(item, stale_after_seconds=stale_after_seconds))
                del self.records[key]
            self._write_state()
            remaining = [self._view_item(item) for item in self.records.values()]
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "deleted": deleted,
                        "remaining": remaining,
                        "totals": {
                            "deleted": len(deleted),
                            "remaining": len(remaining),
                            "remaining_stale": sum(1 for item in remaining if item["stale"]),
                        },
                        "side_effect": "runtime_state_only",
                    },
                },
            )
        raise AssertionError(f"unexpected request: {request.method} {request.url}")


def test_run_worker_status_cleanup_smoke_returns_expected_evidence(tmp_path):
    module = _load_module()
    runtime = FakeCleanupRuntime(tmp_path / "agent-worker-statuses.json")
    client = module.HELPERS.WorkerStatusRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )
    now = datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    result = module.run_worker_status_cleanup_smoke(
        client,
        state_file=runtime.state_file,
        stale_worker_id="worker-stale",
        active_worker_id="worker-active",
        now=now,
    )

    assert all(result["checks"].values())
    assert result["status_codes"]["cleanup"] == 200
    assert result["cleanup_totals"]["deleted"] == 1
    assert ("POST", "/v1/agent-worker-statuses/cleanup-stale") in runtime.requests


def test_inspect_live_runtime_worker_cleanup_state_detects_stale_workers(tmp_path):
    module = _load_module()
    runtime = FakeCleanupRuntime(tmp_path / "agent-worker-statuses.json")
    stale_time = runtime.now - timedelta(minutes=2)
    active_time = runtime.now
    runtime.records = {
        "worker-stale": {
            "WorkerID": "worker-stale",
            "InstanceID": "instance-stale",
            "WorkerType": "knowledge",
            "Status": "running",
            "UpdatedAt": stale_time.isoformat(),
            "LeaseUntil": (stale_time + timedelta(seconds=120)).isoformat(),
        },
        "worker-active": {
            "WorkerID": "worker-active",
            "InstanceID": "instance-active",
            "WorkerType": "knowledge",
            "Status": "running",
            "UpdatedAt": active_time.isoformat(),
            "LeaseUntil": (active_time + timedelta(seconds=120)).isoformat(),
        },
    }
    runtime._write_state()
    client = module.HELPERS.WorkerStatusRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )

    state = module.inspect_live_runtime_worker_cleanup_state(client)

    assert state["workers_total"] == 2
    assert state["workers_stale"] == 1
    assert state["runtime_overview_agent_workers_stale"] == 1
    assert state["stale_workers"][0]["worker_id"] == "worker-stale"
