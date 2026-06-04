from __future__ import annotations

import importlib.util
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from urllib.parse import unquote

import httpx


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_scheduler_runtime_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_scheduler_runtime_live_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class FakeSchedulerRuntime:
    def __init__(self, state_file: Path) -> None:
        self.state_file = state_file
        self.jobs: dict[str, dict] = {}
        self.leases: dict[str, dict] = {}
        self.requests: list[tuple[str, str]] = []
        self._write_state()

    def _write_state(self) -> None:
        payload = {
            "version": "2026-05-31.schedulerjobstore.v1",
            "jobs": list(self.jobs.values()),
        }
        self.state_file.parent.mkdir(parents=True, exist_ok=True)
        self.state_file.write_text(
            json.dumps(payload, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )

    def _list_jobs(self) -> list[dict]:
        return sorted(self.jobs.values(), key=lambda item: str(item["id"]))

    def _list_leases(self, now: datetime) -> dict:
        items = []
        active = 0
        expired = 0
        for lease in self.leases.values():
            expires_at = datetime.fromisoformat(str(lease["expires_at"]))
            is_active = expires_at > now
            if is_active:
                active += 1
            else:
                expired += 1
            items.append(
                {
                    "job_id": lease["job_id"],
                    "holder_id": lease["holder_id"],
                    "lease_token_present": True,
                    "active": is_active,
                    "expires_at": lease["expires_at"],
                }
            )
        return {
            "leases": sorted(items, key=lambda item: item["job_id"]),
            "totals": {"leases": len(items), "active": active, "expired": expired},
            "notes": ["side_effect=runtime_state_only"],
        }

    def _diagnostics(self, now: datetime) -> dict:
        overdue = 0
        recent = []
        for job in self._list_jobs():
            fire_at = datetime.fromisoformat(str(job["fire_at"]))
            if fire_at <= now:
                overdue += 1
            recent.append(
                {
                    "id": job["id"],
                    "fire_at": job["fire_at"],
                    "trigger": job["trigger"],
                    "tier": job["tier"],
                    "status": "overdue" if fire_at <= now else "future",
                    "run_count": int(job.get("run_count") or 0),
                    "enabled": bool(job.get("enabled", True)),
                }
            )
        return {
            "sampled_jobs": len(self.jobs),
            "overdue_jobs": overdue,
            "due_soon_jobs": 0,
            "recent": recent[:10],
        }

    def handler(self, request: httpx.Request) -> httpx.Response:
        self.requests.append((request.method, request.url.path))
        if request.url.path == "/v1/scheduler/jobs" and request.method == "GET":
            return httpx.Response(200, json={"code": "OK", "data": self._list_jobs()})
        if request.url.path == "/v1/scheduler/jobs/upsert" and request.method == "POST":
            body = json.loads(request.read().decode("utf-8"))
            job = dict(body["job"])
            self.jobs[str(job["id"])] = job
            self._write_state()
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": job["id"],
                        "created": True,
                        "deleted": False,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        if request.url.path.startswith("/v1/scheduler/jobs/") and request.method == "DELETE":
            job_id = unquote(request.url.path.split("/")[-1])
            found = self.jobs.pop(job_id, None) is not None
            self._write_state()
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": job_id,
                        "found": found,
                        "deleted": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        if request.url.path == "/v1/scheduler/leases" and request.method == "GET":
            return httpx.Response(
                200,
                json={"code": "OK", "data": self._list_leases(datetime.now(timezone.utc))},
            )
        if request.url.path == "/v1/scheduler/leases/acquire" and request.method == "POST":
            body = json.loads(request.read().decode("utf-8"))
            job_id = str(body["job_id"])
            token = f"lease-{job_id}"
            expires_at = (datetime.now(timezone.utc) + timedelta(seconds=body["ttl_seconds"])).isoformat()
            self.leases[job_id] = {
                "job_id": job_id,
                "holder_id": body["holder_id"],
                "lease_token": token,
                "expires_at": expires_at,
            }
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": job_id,
                        "holder_id": body["holder_id"],
                        "lease_token": token,
                        "lease_token_present": True,
                        "active": True,
                        "acquired": True,
                        "expires_at": expires_at,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        if request.url.path.startswith("/v1/scheduler/jobs/") and request.url.path.endswith("/complete"):
            body = json.loads(request.read().decode("utf-8"))
            job_id = unquote(request.url.path.split("/")[-2])
            lease = self.leases.get(job_id)
            assert lease is not None
            assert lease["holder_id"] == body["holder_id"]
            assert lease["lease_token"] == body["lease_token"]
            deleted = body["action"] == "delete"
            if deleted:
                self.jobs.pop(job_id, None)
            else:
                self.jobs[job_id] = dict(body["job"])
            self.leases.pop(job_id, None)
            self._write_state()
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": job_id,
                        "action": body["action"],
                        "deleted": deleted,
                        "lease_released": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        if request.url.path == "/v1/scheduler/diagnostics" and request.method == "GET":
            timestamp = datetime.fromisoformat(request.url.params["timestamp"])
            return httpx.Response(
                200,
                json={"code": "OK", "data": self._diagnostics(timestamp)},
            )
        raise AssertionError(f"unexpected request: {request.method} {request.url}")


def test_run_crud_smoke_returns_expected_evidence(tmp_path):
    module = _load_module()
    runtime = FakeSchedulerRuntime(tmp_path / "scheduler-jobs.json")
    client = module.SchedulerRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )
    now = datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    result = module.run_crud_smoke(
        client,
        state_file=runtime.state_file,
        prefix="scheduler-live-smoke-test",
        now=now,
    )

    assert all(result["checks"].values())
    assert result["job_id"].startswith("scheduler-live-smoke-test-")
    assert ("POST", "/v1/scheduler/jobs/upsert") in runtime.requests
    assert any(method == "DELETE" for method, _ in runtime.requests)


def test_run_completion_smoke_returns_expected_evidence(tmp_path):
    module = _load_module()
    runtime = FakeSchedulerRuntime(tmp_path / "scheduler-jobs.json")
    client = module.SchedulerRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )
    now = datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    result = module.run_completion_smoke(
        client,
        state_file=runtime.state_file,
        prefix="scheduler-live-smoke-test",
        now=now,
    )

    assert all(result["checks"].values())
    assert result["delete_response"]["lease_released"] is True
    assert result["reschedule_response"]["lease_released"] is True
    complete_paths = [
        path for method, path in runtime.requests if method == "POST" and path.endswith("/complete")
    ]
    assert len(complete_paths) == 2


def test_run_recovery_smoke_returns_expected_evidence(tmp_path):
    module = _load_module()
    runtime = FakeSchedulerRuntime(tmp_path / "scheduler-jobs.json")
    transport = httpx.MockTransport(runtime.handler)
    client = module.SchedulerRuntimeClient(
        "http://agent-runtime.test",
        transport=transport,
    )
    now = datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    result = module.run_recovery_smoke(
        client,
        base_url="http://agent-runtime.test",
        state_file=runtime.state_file,
        prefix="scheduler-live-smoke-test",
        now=now,
        transport=transport,
    )

    assert all(result["checks"].values())
    assert result["diagnostics_after_recovery"]["overdue_jobs"] == 0
    assert ("GET", "/v1/scheduler/jobs") in runtime.requests
    assert ("POST", "/v1/scheduler/jobs/upsert") in runtime.requests
