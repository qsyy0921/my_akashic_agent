from __future__ import annotations

import argparse
import importlib.util
import json
import shutil
import subprocess
import sys
import tempfile
import time
import uuid
from contextlib import suppress
from dataclasses import dataclass
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
PREFLIGHT_HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_job_external_lease_cutover_preflight.py"
)


def _load_module(module_name: str, path: Path):
    spec = importlib.util.spec_from_file_location(module_name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load helper module from {path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


PREFLIGHT = _load_module(
    "verify_agent_job_external_lease_cutover_preflight",
    PREFLIGHT_HELPERS_PATH,
)


@dataclass
class LauncherRuntimeHandle:
    pid: int
    base_url: str
    runtime_root: Path
    state_dir: Path
    stdout_log: Path
    stderr_log: Path
    metadata: dict[str, Any]

    def stop(self) -> None:
        taskkill_exe = shutil.which("taskkill.exe") or shutil.which("taskkill")
        if not taskkill_exe:
            return
        with suppress(Exception):
            subprocess.run(
                [taskkill_exe, "/PID", str(self.pid), "/T", "/F"],
                capture_output=True,
                text=True,
                timeout=20,
                encoding="utf-8",
                errors="replace",
            )


def _find_powershell_exe() -> str:
    powershell_exe = (
        shutil.which("powershell.exe")
        or shutil.which("powershell")
        or shutil.which("pwsh.exe")
        or shutil.which("pwsh")
    )
    if not powershell_exe:
        raise RuntimeError("powershell executable not found")
    return powershell_exe


def _build_launcher_arguments(
    *,
    script_path: Path,
    runtime_addr: str,
    runtime_state_dir: Path,
    shadow_audit_path: Path,
    queue_dsn: str,
) -> list[str]:
    return [
        _find_powershell_exe(),
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        str(script_path),
        "-Foreground",
        "-RuntimeAddr",
        runtime_addr,
        "-RuntimeStateDir",
        str(runtime_state_dir),
        "-ShadowAuditPath",
        str(shadow_audit_path),
        "-QueueBackend",
        "nats",
        "-QueueMode",
        "external_lease",
        "-QueueDSN",
        queue_dsn,
        "-QueueExternalLeaseCutover",
        "true",
        "-QueueDualReadSmokePassed",
        "true",
        "-QueueStateLeaseWorkersDisabled",
        "true",
        "-QueueExternalLeaseAgentJobEnabled",
        "true",
        "-QueueAgentJobDuplicateSmokePassed",
        "true",
        "-QueueAgentJobFlowSmokePassed",
        "true",
        "-AgentJobStrictLeaseToken",
        "true",
        "-StartupTimeoutSeconds",
        "90",
        "-OutboxDeliveryAllowedKinds",
        "text",
        "-OutboxDeliveryAllowedKindsByAccount",
        "2365524513=text|file",
        "-OutboxDeliveryAllowedKindsByAccountConversationType",
        "1049511700/private=text|file",
    ]


def _runtime_config_env_index(items: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    for item in items:
        key = str(item.get("key") or "")
        if key:
            result[key] = item
    return result


def _post_blocked_private_image_outbound(client: Any, *, timestamp: str) -> str:
    event_id = f"outbox:external-lease:gated:image:{uuid.uuid4().hex[:8]}"
    PREFLIGHT._request_json(
        client,
        "POST",
        "/v1/outbound",
        body={
            "event_id": event_id,
            "channel": {
                "platform": "qq",
                "account_id": "1049511700",
                "conversation_id": "2365524513",
                "conversation_type": "private",
            },
            "content": "blocked image route should remain queued",
            "attachments": [
                {
                    "kind": "image",
                    "url": "https://example.com/external-lease-gated-image.png",
                    "mime_type": "image/png",
                    "name": "external-lease-gated-image.png",
                }
            ],
            "timestamp": timestamp,
            "metadata": {
                "source": "verify_agent_job_external_lease_launcher_preflight"
            },
        },
    )
    return event_id


def _start_runtime_via_launcher(repo_root: Path, *, queue_dsn: str) -> LauncherRuntimeHandle:
    runtime_root = Path(tempfile.mkdtemp(prefix="agent-job-launcher-preflight-"))
    state_dir = runtime_root / "state"
    state_dir.mkdir(parents=True, exist_ok=True)
    shadow_audit_path = runtime_root / "shadow-runtime-audit.jsonl"
    port = PREFLIGHT.HELPERS._find_free_port()
    runtime_addr = f"127.0.0.1:{port}"
    log_prefix = f"agent-job-launcher-preflight-{uuid.uuid4().hex[:8]}"
    stdout_log = repo_root / "logs" / f"{log_prefix}.out.log"
    stderr_log = repo_root / "logs" / f"{log_prefix}.err.log"
    stdout_log.parent.mkdir(parents=True, exist_ok=True)
    script_path = repo_root / "scripts" / "start-agent-runtime.ps1"
    args = _build_launcher_arguments(
        script_path=script_path,
        runtime_addr=runtime_addr,
        runtime_state_dir=state_dir,
        shadow_audit_path=shadow_audit_path,
        queue_dsn=queue_dsn,
    )
    stdout_handle = stdout_log.open("w", encoding="utf-8")
    stderr_handle = stderr_log.open("w", encoding="utf-8")
    process = subprocess.Popen(
        args,
        cwd=str(repo_root),
        stdout=stdout_handle,
        stderr=stderr_handle,
        text=True,
    )
    base_url = f"http://127.0.0.1:{port}"
    deadline = time.time() + 90
    last_error = ""
    while time.time() < deadline:
        if process.poll() is not None:
            break
        try:
            with PREFLIGHT.HELPERS.httpx.Client(timeout=2.0, trust_env=True) as client:
                response = client.get(base_url + "/healthz")
            if response.status_code == 200:
                stdout_handle.close()
                stderr_handle.close()
                payload = {
                    "pid": process.pid,
                    "runtime_addr": runtime_addr,
                    "runtime_state_dir": str(state_dir),
                    "shadow_audit_path": str(shadow_audit_path),
                    "health_url": base_url + "/healthz",
                    "stdout_log": str(stdout_log),
                    "stderr_log": str(stderr_log),
                    "launcher_mode": "foreground",
                }
                return LauncherRuntimeHandle(
                    pid=process.pid,
                    base_url=base_url,
                    runtime_root=runtime_root,
                    state_dir=state_dir,
                    stdout_log=stdout_log,
                    stderr_log=stderr_log,
                    metadata=payload,
                )
        except Exception as exc:  # pragma: no cover - startup wait
            last_error = str(exc)
        time.sleep(0.5)
    stdout_handle.close()
    stderr_handle.close()
    stderr_tail = ""
    if stderr_log.exists():
        stderr_tail = "\n".join(stderr_log.read_text(encoding="utf-8").splitlines()[-20:])
    if process.poll() is None:
        with suppress(Exception):
            process.terminate()
        with suppress(Exception):
            process.wait(timeout=10)
    raise RuntimeError(
        f"start-agent-runtime foreground launch did not become healthy at {base_url}/healthz; "
        f"last_error={last_error!r}\n{stderr_tail}"
    )


def _run_launcher_preflight_smoke(repo_root: Path) -> dict[str, Any]:
    nats = PREFLIGHT.start_temp_nats(repo_root)
    runtime = None
    try:
        runtime = _start_runtime_via_launcher(repo_root, queue_dsn=nats.dsn)
        client = PREFLIGHT.HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
        runtime_config = PREFLIGHT._request_json(client, "GET", "/v1/runtime-config")
        snapshot = PREFLIGHT._collect_preflight_snapshot(client)
        missing_approval_preflight = PREFLIGHT._request_json(
            client,
            "GET",
            "/v1/agent-job-external-lease/preflight?desired_execution_owner=python_ai_worker_with_nats_result_ack&operator_id=qsyy",
        )
        approval = PREFLIGHT._record_operator_approval(
            client,
            target_kind="agent_job_external_lease",
            target_id="python_ai_worker_with_nats_result_ack",
            operator_id="qsyy",
            timestamp=PREFLIGHT._utcnow(),
        )
        approved_preflight = PREFLIGHT._request_json(
            client,
            "GET",
            "/v1/agent-job-external-lease/preflight?desired_execution_owner=python_ai_worker_with_nats_result_ack&operator_id=qsyy&approval_id="
            + approval["approval_id"],
        )
        gated_event_id = _post_blocked_private_image_outbound(
            client, timestamp=PREFLIGHT._utcnow().isoformat()
        )
        time.sleep(2.0)
        gated_delivery = PREFLIGHT._request_json(
            client, "GET", f"/v1/outbox/{gated_event_id}"
        )
        env_index = _runtime_config_env_index(list(runtime_config.get("environment") or []))
        required_flags = [
            "AKASHIC_QUEUE_BACKEND",
            "AKASHIC_QUEUE_DSN",
            "AKASHIC_QUEUE_MODE",
            "AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER",
            "AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED",
            "AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED",
            "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED",
            "AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED",
            "AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED",
            "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN",
            "AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS",
            "AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT",
            "AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE",
        ]
        agent_job_topology = next(
            (
                item
                for item in list(snapshot["queue_topology"].get("work_kinds") or [])
                if isinstance(item, dict) and item.get("work_kind") == "agent_job"
            ),
            {},
        )
        summary = snapshot["runtime_overview"].get("summary") or {}
        checks = {
            "launcher_reported_runtime_addr": runtime.metadata.get("runtime_addr")
            == f"127.0.0.1:{runtime.base_url.rsplit(':', 1)[1]}",
            "launcher_reported_runtime_state_dir": Path(str(runtime.metadata.get("runtime_state_dir"))) == runtime.state_dir,
            "required_flags_visible_in_runtime_config": all(
                bool(env_index.get(key, {}).get("present")) for key in required_flags
            ),
            "readiness_ready_from_launcher_flags": snapshot["readiness"].get("ready") is True,
            "plan_ready_from_launcher_flags": snapshot["plan"].get("ready") is True
            and snapshot["plan"].get("decision") == "ready",
            "queue_backend_promoted_to_nats": snapshot["queue_backend"].get("provider")
            == "nats_jetstream",
            "queue_backend_preserves_outbox_gate_scope": snapshot["queue_backend"].get(
                "outbox_execution_scope"
            )
            == "account_conversation_kind_gated",
            "queue_topology_ack_owner_promoted": agent_job_topology.get("ack_owner")
            == "nats_external_lease_result_ack",
            "queue_topology_execution_owner_promoted": agent_job_topology.get("execution_owner")
            == "python_ai_worker_with_nats_result_ack",
            "missing_approval_without_approval_id": missing_approval_preflight.get("ready") is False
            and missing_approval_preflight.get("reason") == "missing_approval_id",
            "approved_preflight_ready": approved_preflight.get("ready") is True
            and approved_preflight.get("reason") == "agent_job_external_lease_preflight_ready",
            "runtime_overview_summary_promoted": summary.get("agent_job_external_lease_execution_owner")
            == "python_ai_worker_with_nats_result_ack",
            "blocked_image_route_remains_queued": gated_delivery.get("status") == "queued"
            and int(gated_delivery.get("attempts") or 0) == 0
            and not str(gated_delivery.get("lease_owner") or "").strip(),
        }
        return {
            "launcher": runtime.metadata,
            "runtime_root": str(runtime.runtime_root),
            "nats": {
                "dsn": nats.dsn,
                "container_name": nats.container_name,
                "port": nats.port,
            },
            "runtime_config": runtime_config,
            "snapshot": snapshot,
            "missing_approval_preflight": missing_approval_preflight,
            "approval": approval,
            "approved_preflight": approved_preflight,
            "gated_delivery": gated_delivery,
            "checks": checks,
        }
    finally:
        if runtime is not None:
            runtime.stop()
        nats.stop(repo_root)


def _build_result(repo_root: Path, verification: dict[str, Any] | None, error: str | None = None) -> dict[str, Any]:
    checks = dict((verification or {}).get("checks") or {})
    if error:
        status = "error"
        category = "agent_job_external_lease_launcher_preflight_error"
        reason = error
    elif checks and all(bool(value) for value in checks.values()):
        status = "live_verified"
        category = "repo_owned_launcher_external_lease_preflight_live_verified"
        reason = (
            "start-agent-runtime.ps1 launched a temp runtime with explicit external-lease "
            "result-ack flags, and the runtime exposed the promoted ownership and "
            "approval-bound preflight path from missing_approval_id to preflight_ready."
        )
    else:
        status = "verification_failed"
        category = "agent_job_external_lease_launcher_preflight_failed"
        reason = "at least one launcher-driven external-lease preflight check failed"
    result: dict[str, Any] = {
        "generated_at": PREFLIGHT._utcnow().isoformat(),
        "repo_root": str(repo_root),
        "verification_scope": "agent_job_external_lease_launcher_preflight",
        "verification": verification,
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
    }
    if error:
        result["error"] = error
    return result


def run_agent_job_external_lease_launcher_preflight(repo_root: Path) -> dict[str, Any]:
    verification = None
    try:
        verification = _run_launcher_preflight_smoke(repo_root)
        return _build_result(repo_root, verification)
    except Exception as exc:  # pragma: no cover - defensive live path
        return _build_result(repo_root, verification, error=str(exc))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_agent_job_external_lease_launcher_preflight(
        Path(args.repo_root).resolve()
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
