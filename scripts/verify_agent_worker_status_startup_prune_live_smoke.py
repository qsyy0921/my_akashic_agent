from __future__ import annotations

import argparse
import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import time
from contextlib import suppress
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

import httpx


REPO_ROOT = Path(__file__).resolve().parents[1]
FENCING_HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_worker_status_fencing_live_smoke.py"
)


def _load_fencing_helpers():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_worker_status_fencing_live_smoke",
        FENCING_HELPERS_PATH,
    )
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load helper module from {FENCING_HELPERS_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


HELPERS = _load_fencing_helpers()


def _worker_state_item(
    *,
    worker_id: str,
    instance_id: str,
    worker_type: str,
    status: str,
    updated_at: datetime,
    lease_until: datetime,
) -> dict[str, Any]:
    return {
        "WorkerID": worker_id,
        "InstanceID": instance_id,
        "WorkerType": worker_type,
        "Status": status,
        "CurrentJobID": "",
        "LastJobID": "",
        "LastError": "",
        "ProcessedTotal": 0,
        "FailedTotal": 0,
        "Source": "python",
        "Metadata": {"seeded_by": "verify_agent_worker_status_startup_prune_live_smoke"},
        "UpdatedAt": updated_at.astimezone(timezone.utc).isoformat(),
        "LeaseUntil": lease_until.astimezone(timezone.utc).isoformat(),
    }


def _write_seed_state(state_file: Path, items: list[dict[str, Any]]) -> None:
    state_file.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "version": "2026-06-03.agentworkerstatusstore.v1",
        "workers": items,
    }
    state_file.write_text(
        json.dumps(payload, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


def start_temp_runtime_with_seed(
    repo_root: Path,
    *,
    seeded_items: list[dict[str, Any]],
) -> Any:
    port = HELPERS._find_free_port()
    runtime_root = Path(tempfile.mkdtemp(prefix="agent-worker-status-startup-prune-"))
    state_dir = runtime_root / "state"
    state_dir.mkdir(parents=True, exist_ok=True)
    state_file = state_dir / "agent-worker-statuses.json"
    _write_seed_state(state_file, seeded_items)
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
        "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN",
        "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED",
    ):
        env.pop(key, None)
    process = subprocess.Popen(
        [HELPERS._find_go_exe(), "run", "./cmd/agent-runtime"],
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
                return HELPERS.TempRuntimeHandle(
                    process=process,
                    base_url=base_url,
                    state_dir=state_dir,
                    state_file=state_file,
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


def _status_item(items: list[dict[str, Any]], worker_id: str) -> dict[str, Any] | None:
    for item in items:
        item_worker_id = item.get("worker_id")
        if item_worker_id is None:
            item_worker_id = item.get("WorkerID")
        if str(item_worker_id or "") == worker_id:
            return item
    return None


def verify_startup_prune_state(
    client: Any,
    *,
    state_file: Path,
    stale_worker_id: str,
    active_worker_id: str,
) -> dict[str, Any]:
    workers_payload = client.request_json("GET", "/v1/agent-worker-statuses")
    overview_payload = client.request_json("GET", "/v1/runtime-overview")
    workers = [item for item in workers_payload.get("workers", []) if isinstance(item, dict)]
    state_items = HELPERS._read_state_items(state_file)
    stale_api = _status_item(workers, stale_worker_id)
    active_api = _status_item(workers, active_worker_id)
    stale_state = _status_item(state_items, stale_worker_id)
    active_state = _status_item(state_items, active_worker_id)
    summary = overview_payload.get("summary", {}) if isinstance(overview_payload, dict) else {}

    checks = {
        "stale_worker_removed_from_api": stale_api is None,
        "stale_worker_removed_from_state_file": stale_state is None,
        "active_worker_preserved_in_api": active_api is not None
        and str(active_api.get("worker_id") or "") == active_worker_id,
        "active_worker_preserved_in_state_file": active_state is not None
        and str(active_state.get("WorkerID") or "") == active_worker_id,
        "runtime_overview_reports_zero_stale_workers": int(summary.get("agent_workers_stale", 0)) == 0,
    }
    overall_ok = all(bool(value) for value in checks.values())
    return {
        "workers_totals": workers_payload.get("totals", {}),
        "runtime_overview_summary": summary,
        "state_file": str(state_file),
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if overall_ok else "verification_incomplete",
            "category": (
                "worker_status_startup_prune_live_verified"
                if overall_ok
                else "worker_status_startup_prune_verification_incomplete"
            ),
            "reason": (
                "temp Go runtime pruned stale agent-worker-status records during repository load and preserved active worker state."
                if overall_ok
                else "startup prune verification found stale worker residue or lost active worker state"
            ),
        },
    }


def run_agent_worker_status_startup_prune_live_smoke(repo_root: Path) -> dict[str, Any]:
    now = datetime.now(timezone.utc)
    stale_worker_id = "verify-worker-status-startup-prune-stale"
    active_worker_id = "verify-worker-status-startup-prune-active"
    runtime = start_temp_runtime_with_seed(
        repo_root,
        seeded_items=[
            _worker_state_item(
                worker_id=stale_worker_id,
                instance_id=f"{stale_worker_id}:11111:stale",
                worker_type="knowledge",
                status="running",
                updated_at=now - timedelta(minutes=10),
                lease_until=now - timedelta(minutes=8),
            ),
            _worker_state_item(
                worker_id=active_worker_id,
                instance_id=f"{active_worker_id}:22222:active",
                worker_type="knowledge",
                status="running",
                updated_at=now,
                lease_until=now + timedelta(minutes=2),
            ),
        ],
    )
    client = HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
    try:
        startup_prune = verify_startup_prune_state(
            client,
            state_file=runtime.state_file,
            stale_worker_id=stale_worker_id,
            active_worker_id=active_worker_id,
        )
        checks = dict(startup_prune.get("checks", {}))
        overall_ok = all(bool(value) for value in checks.values())
        return {
            "base_url": runtime.base_url,
            "state_dir": str(runtime.state_dir),
            "state_file": str(runtime.state_file),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "startup_prune_smoke": startup_prune,
            "checks": checks,
            "conclusion": {
                "status": "live_verified" if overall_ok else "verification_incomplete",
                "category": (
                    "worker_status_startup_prune_live_verified"
                    if overall_ok
                    else "worker_status_startup_prune_verification_incomplete"
                ),
                "reason": (
                    "temp Go runtime removed stale worker heartbeat residue during startup and kept active worker state readable."
                    if overall_ok
                    else "startup prune smoke found stale worker residue after runtime load"
                ),
            },
        }
    finally:
        runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_agent_worker_status_startup_prune_live_smoke(
        Path(args.repo_root).resolve()
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
