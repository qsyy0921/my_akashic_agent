from __future__ import annotations

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import uuid
from contextlib import suppress
from dataclasses import asdict, dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any
from urllib.parse import quote

import httpx

REPO_ROOT = Path(__file__).resolve().parents[1]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from agent.config_models import AgentRuntimeIntegrationConfig
from agent.scheduler import ScheduledJob, SchedulerService


def _utcnow() -> datetime:
    return datetime.now(timezone.utc)


def _find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _find_go_exe() -> str:
    go_exe = shutil.which("go")
    if go_exe:
        return go_exe
    local_app_data = os.environ.get("LOCALAPPDATA", "")
    candidate = Path(local_app_data) / "Programs" / "Go" / "bin" / "go.exe"
    if candidate.exists():
        return str(candidate)
    raise RuntimeError("go executable not found in PATH or LOCALAPPDATA Programs\\Go\\bin")


def _job_to_payload(job: ScheduledJob) -> dict[str, Any]:
    payload = asdict(job)
    payload["fire_at"] = job.fire_at.isoformat()
    payload["created_at"] = job.created_at.isoformat()
    return payload


def _read_state_jobs(state_file: Path) -> list[dict[str, Any]]:
    if not state_file.exists():
        return []
    data = json.loads(state_file.read_text(encoding="utf-8"))
    if isinstance(data, list):
        return [item for item in data if isinstance(item, dict)]
    if isinstance(data, dict):
        jobs = data.get("jobs", [])
        if isinstance(jobs, list):
            return [item for item in jobs if isinstance(item, dict)]
    raise RuntimeError(f"unexpected scheduler state payload in {state_file}")


def _state_job_ids(state_file: Path) -> list[str]:
    job_ids: list[str] = []
    for item in _read_state_jobs(state_file):
        value = item.get("id")
        if value is None:
            value = item.get("ID")
        job_ids.append(str(value or ""))
    return job_ids


def _lease_items(payload: dict[str, Any]) -> list[dict[str, Any]]:
    leases = payload.get("leases", [])
    if not isinstance(leases, list):
        return []
    return [item for item in leases if isinstance(item, dict)]


def _job_items(items: list[dict[str, Any]], prefix: str) -> list[dict[str, Any]]:
    return [item for item in items if str(item.get("id", "")).startswith(prefix)]


@dataclass
class TempRuntimeHandle:
    process: subprocess.Popen[str]
    base_url: str
    state_dir: Path
    state_file: Path
    stdout_log: Path
    stderr_log: Path

    def stop(self) -> None:
        if self.process.poll() is None:
            with suppress(Exception):
                self.process.terminate()
            try:
                self.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                with suppress(Exception):
                    self.process.kill()
                with suppress(Exception):
                    self.process.wait(timeout=5)


class SchedulerRuntimeClient:
    def __init__(
        self,
        base_url: str,
        *,
        timeout: float = 30.0,
        transport: httpx.BaseTransport | None = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.transport = transport

    def request_json(
        self, method: str, path: str, *, json_body: Any | None = None
    ) -> dict[str, Any]:
        with httpx.Client(
            timeout=self.timeout,
            transport=self.transport,
            trust_env=True,
        ) as client:
            response = client.request(
                method.upper(),
                self.base_url + path,
                json=json_body,
                headers={"Content-Type": "application/json"},
            )
        response.raise_for_status()
        payload = response.json()
        if not isinstance(payload, dict):
            raise RuntimeError(f"unexpected non-object response for {method} {path}")
        code = payload.get("code")
        if code not in (None, "OK", 0):
            raise RuntimeError(str(payload.get("message") or payload))
        data = payload.get("data", payload)
        if isinstance(data, dict):
            return data
        return {"items": data}

    def list_jobs(self) -> list[dict[str, Any]]:
        data = self.request_json("GET", "/v1/scheduler/jobs")
        items = data.get("items", data)
        if not isinstance(items, list):
            raise RuntimeError("scheduler jobs response is not a list")
        return [item for item in items if isinstance(item, dict)]

    def list_leases(self) -> dict[str, Any]:
        data = self.request_json("GET", "/v1/scheduler/leases")
        if not isinstance(data, dict):
            raise RuntimeError("scheduler leases response is not an object")
        return data

    def diagnostics(self, now: datetime) -> dict[str, Any]:
        timestamp = quote(now.astimezone(timezone.utc).isoformat(), safe="")
        data = self.request_json(
            "GET",
            f"/v1/scheduler/diagnostics?timestamp={timestamp}&due_soon_seconds=120",
        )
        if not isinstance(data, dict):
            raise RuntimeError("scheduler diagnostics response is not an object")
        return data

    def upsert_job(self, job: ScheduledJob, *, source: str) -> dict[str, Any]:
        return self.request_json(
            "POST",
            "/v1/scheduler/jobs/upsert",
            json_body={"job": _job_to_payload(job), "source": source},
        )

    def delete_job(self, job_id: str, *, source: str) -> dict[str, Any]:
        encoded_job_id = quote(job_id, safe="")
        return self.request_json(
            "DELETE",
            f"/v1/scheduler/jobs/{encoded_job_id}?source={quote(source, safe='')}",
        )

    def acquire_lease(
        self,
        job_id: str,
        *,
        holder_id: str,
        ttl_seconds: int,
        metadata: dict[str, str],
    ) -> dict[str, Any]:
        return self.request_json(
            "POST",
            "/v1/scheduler/leases/acquire",
            json_body={
                "job_id": job_id,
                "holder_id": holder_id,
                "ttl_seconds": ttl_seconds,
                "metadata": metadata,
            },
        )

    def complete_job(
        self,
        job_id: str,
        *,
        source: str,
        holder_id: str,
        lease_token: str,
        action: str,
        job: ScheduledJob | None = None,
    ) -> dict[str, Any]:
        body: dict[str, Any] = {
            "source": source,
            "holder_id": holder_id,
            "lease_token": lease_token,
            "action": action,
        }
        if job is not None:
            body["job"] = _job_to_payload(job)
        encoded_job_id = quote(job_id, safe="")
        return self.request_json(
            "POST",
            f"/v1/scheduler/jobs/{encoded_job_id}/complete",
            json_body=body,
        )


def _make_job(
    *,
    job_id: str,
    trigger: str,
    tier: str,
    fire_at: datetime,
    channel: str = "telegram",
    chat_id: str = "smoke",
    interval_seconds: int | None = None,
    message: str | None = "scheduler runtime smoke",
    prompt: str | None = None,
    name: str | None = None,
    run_count: int = 0,
) -> ScheduledJob:
    return ScheduledJob(
        trigger=trigger,
        tier=tier,
        fire_at=fire_at,
        channel=channel,
        chat_id=chat_id,
        interval_seconds=interval_seconds,
        cron_expr=None,
        message=message,
        prompt=prompt,
        name=name,
        timezone="UTC",
        created_at=_utcnow(),
        run_count=run_count,
        enabled=True,
        id=job_id,
    )


def run_crud_smoke(
    client: SchedulerRuntimeClient,
    *,
    state_file: Path,
    prefix: str,
    now: datetime,
) -> dict[str, Any]:
    job_id = f"{prefix}-crud"
    source = "verify_scheduler_runtime_live_smoke"
    job = _make_job(
        job_id=job_id,
        trigger="at",
        tier="instant",
        fire_at=now + timedelta(days=1),
        message="scheduler crud smoke",
        name="scheduler-crud-smoke",
    )
    upsert_response = client.upsert_job(job, source=source)
    jobs_after_upsert = client.list_jobs()
    state_ids_after_upsert = _state_job_ids(state_file)
    delete_response = client.delete_job(job_id, source=source)
    jobs_after_delete = client.list_jobs()
    state_ids_after_delete = _state_job_ids(state_file)

    checks = {
        "job_visible_via_get_after_upsert": any(item.get("id") == job_id for item in jobs_after_upsert),
        "job_persisted_to_state_file_after_upsert": job_id in state_ids_after_upsert,
        "job_removed_via_delete": all(item.get("id") != job_id for item in jobs_after_delete),
        "job_removed_from_state_file_after_delete": job_id not in state_ids_after_delete,
    }

    return {
        "job_id": job_id,
        "upsert_response": upsert_response,
        "delete_response": delete_response,
        "checks": checks,
    }


def run_completion_smoke(
    client: SchedulerRuntimeClient,
    *,
    state_file: Path,
    prefix: str,
    now: datetime,
) -> dict[str, Any]:
    source = "verify_scheduler_runtime_live_smoke"
    holder_id = f"scheduler:verify-runtime-live-smoke:{uuid.uuid4().hex[:12]}"

    delete_job_id = f"{prefix}-complete-delete"
    delete_job = _make_job(
        job_id=delete_job_id,
        trigger="after",
        tier="instant",
        fire_at=now + timedelta(hours=12),
        message="scheduler complete delete smoke",
        name="scheduler-complete-delete-smoke",
    )
    client.upsert_job(delete_job, source=source)
    delete_lease = client.acquire_lease(
        delete_job_id,
        holder_id=holder_id,
        ttl_seconds=300,
        metadata={"source": source, "smoke": "delete"},
    )
    delete_response = client.complete_job(
        delete_job_id,
        source=source,
        holder_id=holder_id,
        lease_token=str(delete_lease.get("lease_token") or ""),
        action="delete",
    )
    jobs_after_delete = client.list_jobs()
    leases_after_delete = client.list_leases()

    reschedule_job_id = f"{prefix}-complete-reschedule"
    base_job = _make_job(
        job_id=reschedule_job_id,
        trigger="every",
        tier="instant",
        fire_at=now + timedelta(minutes=30),
        interval_seconds=1800,
        message="scheduler complete reschedule smoke",
        name="scheduler-complete-reschedule-smoke",
    )
    client.upsert_job(base_job, source=source)
    reschedule_lease = client.acquire_lease(
        reschedule_job_id,
        holder_id=holder_id,
        ttl_seconds=300,
        metadata={"source": source, "smoke": "reschedule"},
    )
    rescheduled_job = _make_job(
        job_id=reschedule_job_id,
        trigger="every",
        tier="instant",
        fire_at=base_job.fire_at + timedelta(seconds=base_job.interval_seconds or 0),
        interval_seconds=base_job.interval_seconds,
        message=base_job.message,
        name=base_job.name,
        run_count=base_job.run_count + 1,
    )
    reschedule_response = client.complete_job(
        reschedule_job_id,
        source=source,
        holder_id=holder_id,
        lease_token=str(reschedule_lease.get("lease_token") or ""),
        action="reschedule",
        job=rescheduled_job,
    )
    jobs_after_reschedule = client.list_jobs()
    leases_after_reschedule = client.list_leases()
    state_ids_after_reschedule = _state_job_ids(state_file)

    rescheduled_entry = next(
        (item for item in jobs_after_reschedule if item.get("id") == reschedule_job_id),
        None,
    )
    delete_lease_items = _lease_items(leases_after_delete)
    reschedule_lease_items = _lease_items(leases_after_reschedule)
    checks = {
        "delete_completion_removed_job": all(item.get("id") != delete_job_id for item in jobs_after_delete),
        "delete_completion_released_lease": all(item.get("job_id") != delete_job_id for item in delete_lease_items),
        "reschedule_completion_updated_job": (
            rescheduled_entry is not None
            and datetime.fromisoformat(str(rescheduled_entry.get("fire_at"))).astimezone(timezone.utc)
            >= rescheduled_job.fire_at
            and int(rescheduled_entry.get("run_count") or 0) >= rescheduled_job.run_count
            and reschedule_job_id in state_ids_after_reschedule
        ),
        "reschedule_completion_released_lease": all(
            item.get("job_id") != reschedule_job_id for item in reschedule_lease_items
        ),
    }
    cleanup_response = client.delete_job(reschedule_job_id, source=source)

    return {
        "delete_job_id": delete_job_id,
        "reschedule_job_id": reschedule_job_id,
        "delete_response": delete_response,
        "reschedule_response": reschedule_response,
        "cleanup_response": cleanup_response,
        "checks": checks,
    }


def run_recovery_smoke(
    client: SchedulerRuntimeClient,
    *,
    base_url: str,
    state_file: Path,
    prefix: str,
    now: datetime,
    transport: httpx.BaseTransport | None = None,
) -> dict[str, Any]:
    source = "verify_scheduler_runtime_live_smoke"
    recurring_job_id = f"{prefix}-recovery-recurring"
    expired_job_id = f"{prefix}-recovery-expired"
    future_job_id = f"{prefix}-recovery-future"

    recurring_job = _make_job(
        job_id=recurring_job_id,
        trigger="every",
        tier="instant",
        fire_at=now - timedelta(hours=3),
        interval_seconds=3600,
        message="scheduler recovery recurring smoke",
        name="scheduler-recovery-recurring-smoke",
    )
    expired_job = _make_job(
        job_id=expired_job_id,
        trigger="after",
        tier="instant",
        fire_at=now - timedelta(seconds=400),
        message="scheduler recovery expired smoke",
        name="scheduler-recovery-expired-smoke",
    )
    future_job = _make_job(
        job_id=future_job_id,
        trigger="at",
        tier="instant",
        fire_at=now + timedelta(hours=1),
        message="scheduler recovery future smoke",
        name="scheduler-recovery-future-smoke",
    )

    client.upsert_job(recurring_job, source=source)
    client.upsert_job(expired_job, source=source)
    client.upsert_job(future_job, source=source)

    local_store_path = state_file.parent / "python-scheduler-recovery-mirror.json"
    scheduler = SchedulerService(
        store_path=local_store_path,
        push_tool=object(),
        agent_loop=None,
        _now_fn=lambda: now,
        runtime_config=AgentRuntimeIntegrationConfig(
            enabled=True,
            base_url=base_url,
            worker_id="verify-scheduler-runtime-live-smoke",
        ),
        runtime_transport=transport,
    )
    scheduler.load_and_recover()

    runtime_jobs_after_recovery = client.list_jobs()
    diagnostics_after_recovery = client.diagnostics(now)
    state_ids_after_recovery = _state_job_ids(state_file)
    local_store_ids = []
    if local_store_path.exists():
        local_store_ids = _state_job_ids(local_store_path)

    recurring_entry = next(
        (item for item in runtime_jobs_after_recovery if item.get("id") == recurring_job_id),
        None,
    )
    checks = {
        "recurring_job_advanced_to_future": (
            recurring_entry is not None
            and datetime.fromisoformat(str(recurring_entry.get("fire_at"))).astimezone(timezone.utc) > now
        ),
        "expired_job_deleted_from_runtime_store": expired_job_id not in state_ids_after_recovery,
        "expired_job_absent_from_runtime_jobs": all(
            item.get("id") != expired_job_id for item in runtime_jobs_after_recovery
        ),
        "future_job_preserved": future_job_id in state_ids_after_recovery and future_job_id in local_store_ids,
        "local_recovery_mirror_contains_expected_jobs": set(local_store_ids) == {recurring_job_id, future_job_id},
        "diagnostics_no_overdue_jobs_after_recovery": int(diagnostics_after_recovery.get("overdue_jobs") or 0) == 0,
    }

    return {
        "recurring_job_id": recurring_job_id,
        "expired_job_id": expired_job_id,
        "future_job_id": future_job_id,
        "diagnostics_after_recovery": diagnostics_after_recovery,
        "local_store_path": str(local_store_path),
        "checks": checks,
    }


def start_temp_runtime(repo_root: Path) -> TempRuntimeHandle:
    port = _find_free_port()
    runtime_root = Path(tempfile.mkdtemp(prefix="scheduler-runtime-live-smoke-"))
    state_dir = runtime_root / "state"
    state_dir.mkdir(parents=True, exist_ok=True)
    stdout_log = runtime_root / "agent-runtime.stdout.log"
    stderr_log = runtime_root / "agent-runtime.stderr.log"
    stdout_handle = stdout_log.open("w", encoding="utf-8")
    stderr_handle = stderr_log.open("w", encoding="utf-8")
    env = os.environ.copy()
    env["AKASHIC_RUNTIME_ADDR"] = f"127.0.0.1:{port}"
    env["AKASHIC_RUNTIME_STATE_DIR"] = str(state_dir)
    for key in (
        "AKASHIC_ONEBOT_WS_URLS",
        "AKASHIC_ONEBOT_ACCESS_TOKEN",
        "AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT",
        "AKASHIC_BOT_IDS",
        "AKASHIC_QQ_GROUP_SEND_ENABLED",
        "AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED",
        "AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED",
        "AKASHIC_TELEGRAM_BOT_TOKEN",
        "TELEGRAM_BOT_TOKEN",
    ):
        env.pop(key, None)
    process = subprocess.Popen(
        [_find_go_exe(), "run", "./cmd/agent-runtime"],
        cwd=str(repo_root / "services" / "agent-runtime"),
        env=env,
        stdout=stdout_handle,
        stderr=stderr_handle,
        text=True,
    )
    base_url = f"http://127.0.0.1:{port}"
    deadline = time.time() + 60
    last_error = ""
    while time.time() < deadline:
        if process.poll() is not None:
            break
        try:
            with httpx.Client(timeout=2.0, trust_env=True) as client:
                response = client.get(base_url + "/healthz")
            if response.status_code == 200:
                stdout_handle.close()
                stderr_handle.close()
                return TempRuntimeHandle(
                    process=process,
                    base_url=base_url,
                    state_dir=state_dir,
                    state_file=state_dir / "scheduler-jobs.json",
                    stdout_log=stdout_log,
                    stderr_log=stderr_log,
                )
        except Exception as exc:  # pragma: no cover - best effort startup wait
            last_error = str(exc)
        time.sleep(0.5)
    stdout_handle.close()
    stderr_handle.close()
    stderr_tail = ""
    if stderr_log.exists():
        stderr_tail = "\n".join(stderr_log.read_text(encoding="utf-8").splitlines()[-20:])
    process.kill()
    process.wait(timeout=5)
    raise RuntimeError(
        f"temp agent-runtime did not become healthy at {base_url}/healthz. "
        f"last_error={last_error!r}\n{stderr_tail}"
    )


def run_scheduler_runtime_live_smoke(repo_root: Path) -> dict[str, Any]:
    runtime = start_temp_runtime(repo_root)
    client = SchedulerRuntimeClient(runtime.base_url)
    now = _utcnow()
    prefix = f"scheduler-live-smoke-{uuid.uuid4().hex[:8]}"
    try:
        crud = run_crud_smoke(client, state_file=runtime.state_file, prefix=prefix, now=now)
        completion = run_completion_smoke(
            client, state_file=runtime.state_file, prefix=prefix, now=now
        )
        recovery = run_recovery_smoke(
            client,
            base_url=runtime.base_url,
            state_file=runtime.state_file,
            prefix=prefix,
            now=now,
        )
        all_checks = {}
        for group in (crud.get("checks", {}), completion.get("checks", {}), recovery.get("checks", {})):
            all_checks.update(group)
        overall_ok = all(bool(value) for value in all_checks.values())
        return {
            "base_url": runtime.base_url,
            "state_dir": str(runtime.state_dir),
            "state_file": str(runtime.state_file),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "crud_smoke": crud,
            "completion_smoke": completion,
            "recovery_smoke": recovery,
            "checks": all_checks,
            "conclusion": {
                "status": "live_verified" if overall_ok else "verification_incomplete",
                "category": (
                    "go_control_plane_mutation_and_recovery_live_verified"
                    if overall_ok
                    else "scheduler_mutation_or_recovery_verification_incomplete"
                ),
                "reason": (
                    "temp Go runtime verified scheduler CRUD persistence, lease-guarded completion mutation, and startup recovery reconciliation without triggering platform sends."
                    if overall_ok
                    else "at least one scheduler CRUD/completion/recovery smoke check failed"
                ),
            },
        }
    finally:
        runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_scheduler_runtime_live_smoke(Path(args.repo_root).resolve())
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
