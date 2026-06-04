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
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import httpx


REPO_ROOT = Path(__file__).resolve().parents[1]


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


class WorkerStatusRuntimeClient:
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

    def request(
        self,
        method: str,
        path: str,
        *,
        json_body: Any | None = None,
    ) -> httpx.Response:
        with httpx.Client(
            timeout=self.timeout,
            transport=self.transport,
            trust_env=True,
        ) as client:
            return client.request(
                method.upper(),
                self.base_url + path,
                json=json_body,
                headers={"Content-Type": "application/json"},
            )

    def request_json(
        self,
        method: str,
        path: str,
        *,
        json_body: Any | None = None,
    ) -> dict[str, Any]:
        response = self.request(method, path, json_body=json_body)
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

    def report_status(self, payload: dict[str, Any]) -> httpx.Response:
        return self.request("POST", "/v1/agent-worker-statuses/report", json_body=payload)

    def list_statuses(self) -> dict[str, Any]:
        data = self.request_json("GET", "/v1/agent-worker-statuses")
        if not isinstance(data, dict):
            raise RuntimeError("agent-worker-statuses response is not an object")
        return data


def _read_state_items(state_file: Path) -> list[dict[str, Any]]:
    if not state_file.exists():
        return []
    payload = json.loads(state_file.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected worker-status state payload in {state_file}")
    items = payload.get("workers", [])
    if not isinstance(items, list):
        return []
    return [item for item in items if isinstance(item, dict)]


def _make_report_payload(
    *,
    worker_id: str,
    instance_id: str,
    timestamp: datetime,
    replace_existing_instance_id: str = "",
) -> dict[str, Any]:
    return {
        "worker_id": worker_id,
        "instance_id": instance_id,
        "replace_existing_instance_id": replace_existing_instance_id,
        "worker_type": "knowledge",
        "status": "running",
        "current_job_id": "verify-worker-status-fencing-smoke",
        "processed_total": 0,
        "failed_total": 0,
        "source": "python",
        "lease_ttl_seconds": 120,
        "timestamp": timestamp.astimezone(timezone.utc).isoformat(),
    }


def _status_item(items: list[dict[str, Any]], worker_id: str) -> dict[str, Any] | None:
    for item in items:
        item_worker_id = item.get("worker_id")
        if item_worker_id is None:
            item_worker_id = item.get("WorkerID")
        if str(item_worker_id or "") == worker_id:
            return item
    return None


def run_worker_status_fencing_smoke(
    client: WorkerStatusRuntimeClient,
    *,
    state_file: Path,
    worker_id: str,
    now: datetime,
) -> dict[str, Any]:
    instance_a = f"{worker_id}:11111:{uuid.uuid4().hex[:8]}"
    instance_b = f"{worker_id}:22222:{uuid.uuid4().hex[:8]}"

    initial_response = client.report_status(
        _make_report_payload(
            worker_id=worker_id,
            instance_id=instance_a,
            timestamp=now,
        )
    )
    list_after_initial = client.list_statuses()
    initial_items = [
        item
        for item in list_after_initial.get("workers", [])
        if isinstance(item, dict)
    ]
    initial_item = _status_item(initial_items, worker_id)

    conflict_response = client.report_status(
        _make_report_payload(
            worker_id=worker_id,
            instance_id=instance_b,
            timestamp=now,
        )
    )
    conflict_text = conflict_response.text

    takeover_response = client.report_status(
        _make_report_payload(
            worker_id=worker_id,
            instance_id=instance_b,
            timestamp=now,
            replace_existing_instance_id=instance_a,
        )
    )
    list_after_takeover = client.list_statuses()
    final_items = [
        item
        for item in list_after_takeover.get("workers", [])
        if isinstance(item, dict)
    ]
    final_item = _status_item(final_items, worker_id)
    state_items = _read_state_items(state_file)
    state_item = _status_item(state_items, worker_id)

    checks = {
        "initial_report_accepted": initial_response.status_code == 202,
        "initial_worker_record_visible": initial_item is not None
        and str(initial_item.get("instance_id") or "") == instance_a,
        "second_instance_conflict_returns_409": conflict_response.status_code == 409,
        "conflict_mentions_existing_instance_id": (
            conflict_response.status_code == 409
            and f"existing_instance_id={instance_a}" in conflict_text
        ),
        "takeover_with_matching_replace_instance_accepted": takeover_response.status_code
        == 202,
        "final_worker_record_switched_to_new_instance": final_item is not None
        and str((final_item.get("instance_id") or final_item.get("InstanceID") or "")) == instance_b,
        "state_file_switched_to_new_instance": state_item is not None
        and str((state_item.get("instance_id") or state_item.get("InstanceID") or "")) == instance_b,
    }
    overall_ok = all(bool(value) for value in checks.values())

    return {
        "worker_id": worker_id,
        "instance_a": instance_a,
        "instance_b": instance_b,
        "initial_status_code": initial_response.status_code,
        "conflict_status_code": conflict_response.status_code,
        "takeover_status_code": takeover_response.status_code,
        "conflict_text": conflict_text,
        "state_file": str(state_file),
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if overall_ok else "verification_incomplete",
            "category": (
                "controlled_worker_status_fencing_live_verified"
                if overall_ok
                else "worker_status_fencing_verification_incomplete"
            ),
            "reason": (
                "temp Go runtime rejected a second live instance for the same worker_id with HTTP 409 and only accepted takeover when replace_existing_instance_id matched the active lease holder."
                if overall_ok
                else "at least one worker-status fencing smoke check failed"
            ),
        },
    }


def start_temp_runtime(repo_root: Path) -> TempRuntimeHandle:
    port = _find_free_port()
    runtime_root = Path(tempfile.mkdtemp(prefix="agent-worker-status-fencing-"))
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
        "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN",
        "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED",
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
                    state_file=state_dir / "agent-worker-statuses.json",
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


def run_agent_worker_status_fencing_live_smoke(repo_root: Path) -> dict[str, Any]:
    runtime = start_temp_runtime(repo_root)
    client = WorkerStatusRuntimeClient(runtime.base_url)
    worker_id = f"verify-worker-status:{uuid.uuid4().hex[:8]}"
    now = _utcnow()
    try:
        fencing = run_worker_status_fencing_smoke(
            client,
            state_file=runtime.state_file,
            worker_id=worker_id,
            now=now,
        )
        checks = dict(fencing.get("checks", {}))
        overall_ok = all(bool(value) for value in checks.values())
        return {
            "base_url": runtime.base_url,
            "state_dir": str(runtime.state_dir),
            "state_file": str(runtime.state_file),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "fencing_smoke": fencing,
            "checks": checks,
            "conclusion": {
                "status": "live_verified" if overall_ok else "verification_incomplete",
                "category": (
                    "controlled_worker_status_fencing_live_verified"
                    if overall_ok
                    else "worker_status_fencing_verification_incomplete"
                ),
                "reason": (
                    "temp Go runtime enforced worker-status lease fencing with HTTP 409 for the second live instance and accepted only the explicit matching takeover request."
                    if overall_ok
                    else "at least one worker-status fencing smoke check failed"
                ),
            },
        }
    finally:
        runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_agent_worker_status_fencing_live_smoke(
        Path(args.repo_root).resolve()
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
