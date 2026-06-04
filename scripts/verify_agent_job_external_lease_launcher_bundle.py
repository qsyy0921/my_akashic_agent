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

DESIRED_OWNER = "python_ai_worker_with_nats_result_ack"


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


def _resolve_script_path(repo_root: Path, script_path: str) -> Path:
    raw = str(script_path or "").strip()
    if not raw:
        raise RuntimeError("launcher bundle did not include script_path")
    normalized = raw.replace("/", "\\")
    if normalized.startswith(".\\"):
        return (repo_root / normalized[2:]).resolve()
    if normalized.startswith("./"):
        return (repo_root / normalized[2:]).resolve()
    candidate = Path(normalized)
    if candidate.is_absolute():
        return candidate
    return (repo_root / normalized).resolve()


def _required_input_map(
    bundle: dict[str, Any],
    *,
    queue_dsn: str,
) -> dict[str, str]:
    values: dict[str, str] = {}
    for item in list(bundle.get("required_external_inputs") or []):
        parameter = str(item.get("parameter") or "").strip()
        required = bool(item.get("required"))
        if not parameter:
            continue
        if parameter == "QueueDSN":
            values[parameter] = queue_dsn
            continue
        if required:
            raise RuntimeError(
                f"launcher bundle requires unsupported external input parameter {parameter!r}"
            )
    return values


def _build_launcher_arguments_from_bundle(
    *,
    repo_root: Path,
    bundle: dict[str, Any],
    runtime_addr: str,
    runtime_state_dir: Path,
    shadow_audit_path: Path,
    queue_dsn: str,
) -> list[str]:
    script_path = _resolve_script_path(repo_root, str(bundle.get("script_path") or ""))
    supplied_inputs = _required_input_map(bundle, queue_dsn=queue_dsn)
    launcher_parameters = dict(bundle.get("launcher_parameters") or {})
    missing_required: list[str] = []
    for item in list(bundle.get("required_external_inputs") or []):
        parameter = str(item.get("parameter") or "").strip()
        if parameter and bool(item.get("required")) and parameter not in supplied_inputs:
            missing_required.append(parameter)
    if missing_required:
        raise RuntimeError(
            "launcher bundle is missing required external inputs: "
            + ", ".join(sorted(missing_required))
        )

    args = [
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
    ]
    for parameter in sorted(launcher_parameters):
        args.extend([f"-{parameter}", str(launcher_parameters[parameter])])
    for parameter in sorted(supplied_inputs):
        args.extend([f"-{parameter}", supplied_inputs[parameter]])
    args.extend(["-StartupTimeoutSeconds", "90"])
    return args


def _runtime_config_env_index(items: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    for item in items:
        key = str(item.get("key") or "")
        if key:
            result[key] = item
    return result


def _start_runtime_via_launcher_bundle(
    repo_root: Path,
    *,
    bundle: dict[str, Any],
    queue_dsn: str,
) -> LauncherRuntimeHandle:
    runtime_root = Path(tempfile.mkdtemp(prefix="agent-job-launcher-bundle-"))
    state_dir = runtime_root / "state"
    state_dir.mkdir(parents=True, exist_ok=True)
    shadow_audit_path = runtime_root / "shadow-runtime-audit.jsonl"
    port = PREFLIGHT.HELPERS._find_free_port()
    runtime_addr = f"127.0.0.1:{port}"
    log_prefix = f"agent-job-launcher-bundle-{uuid.uuid4().hex[:8]}"
    stdout_log = repo_root / "logs" / f"{log_prefix}.out.log"
    stderr_log = repo_root / "logs" / f"{log_prefix}.err.log"
    stdout_log.parent.mkdir(parents=True, exist_ok=True)
    args = _build_launcher_arguments_from_bundle(
        repo_root=repo_root,
        bundle=bundle,
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
                    "launcher_mode": "foreground_bundle",
                    "script_path": bundle.get("script_path"),
                    "launcher_parameters": bundle.get("launcher_parameters") or {},
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
        f"launcher-bundle foreground launch did not become healthy at {base_url}/healthz; "
        f"last_error={last_error!r}\n{stderr_tail}"
    )


def _fetch_launcher_bundle(client: Any) -> dict[str, Any]:
    return PREFLIGHT._request_json(
        client,
        "GET",
        "/v1/agent-job-external-lease/launcher-bundle?desired_execution_owner="
        + DESIRED_OWNER,
    )


def _blocked_bundle_scenario(repo_root: Path) -> dict[str, Any]:
    runtime = PREFLIGHT.start_temp_runtime(
        repo_root,
        env_overrides={},
        env_remove=[
            "AKASHIC_QUEUE_BACKEND",
            "AKASHIC_QUEUE_MODE",
            "AKASHIC_QUEUE_DSN",
            "AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER",
            "AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED",
            "AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED",
            "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED",
            "AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED",
            "AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED",
            "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN",
            "AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED",
            "AKASHIC_ONEBOT_WS_URLS",
            "AKASHIC_ONEBOT_ACCESS_TOKEN",
            "AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT",
            "AKASHIC_BOT_IDS",
            "AKASHIC_QQ_GROUP_SEND_ENABLED",
            "AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED",
            "AKASHIC_TELEGRAM_BOT_TOKEN",
            "TELEGRAM_BOT_TOKEN",
        ],
        prefix="agent-job-launcher-bundle-blocked-",
    )
    client = PREFLIGHT.HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
    try:
        bundle = _fetch_launcher_bundle(client)
        parameters = dict(bundle.get("launcher_parameters") or {})
        required_inputs = {
            str(item.get("parameter") or ""): item
            for item in list(bundle.get("required_external_inputs") or [])
            if str(item.get("parameter") or "")
        }
        checks = {
            "bundle_endpoint_reachable": True,
            "bundle_reports_blocked_reason": bundle.get("ready") is False
            and bundle.get("reason")
            == "agent_job_external_lease_launcher_bundle_blocked",
            "bundle_exposes_script_path": str(bundle.get("script_path") or "")
            == ".\\scripts\\start-agent-runtime.ps1",
            "bundle_includes_canonical_flags": all(
                parameters.get(key) == value
                for key, value in {
                    "QueueBackend": "nats_jetstream",
                    "QueueMode": "external_lease",
                    "QueueExternalLeaseCutover": "true",
                    "QueueDualReadSmokePassed": "true",
                    "QueueStateLeaseWorkersDisabled": "true",
                    "QueueExternalLeaseAgentJobEnabled": "true",
                    "QueueAgentJobDuplicateSmokePassed": "true",
                    "QueueAgentJobFlowSmokePassed": "true",
                    "AgentJobStrictLeaseToken": "true",
                }.items()
            ),
            "bundle_requires_queue_dsn": (
                "QueueDSN" in required_inputs
                and bool(required_inputs["QueueDSN"].get("required"))
                and bool(required_inputs["QueueDSN"].get("secret"))
            ),
            "bundle_verification_steps_exposed": len(
                list(bundle.get("verification_steps") or [])
            )
            >= 5,
        }
        return {
            "base_url": runtime.base_url,
            "runtime_root": str(runtime.runtime_root),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "bundle": bundle,
            "checks": checks,
        }
    finally:
        runtime.stop()


def _promoted_bundle_scenario(
    repo_root: Path,
    *,
    blocked_bundle: dict[str, Any],
) -> dict[str, Any]:
    nats = PREFLIGHT.start_temp_nats(repo_root)
    runtime = None
    try:
        runtime = _start_runtime_via_launcher_bundle(
            repo_root,
            bundle=blocked_bundle,
            queue_dsn=nats.dsn,
        )
        client = PREFLIGHT.HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
        runtime_config = PREFLIGHT._request_json(client, "GET", "/v1/runtime-config")
        created_job = PREFLIGHT._post_job(
            client,
            job_type="group_memory_extract",
            timestamp=PREFLIGHT._utcnow() - PREFLIGHT.timedelta(minutes=31),
        )
        before_worker = PREFLIGHT._collect_preflight_snapshot(client)
        before_worker_bundle = _fetch_launcher_bundle(client)
        worker_report = PREFLIGHT._report_knowledge_worker(
            client,
            timestamp=PREFLIGHT._utcnow(),
        )
        after_worker = PREFLIGHT._collect_preflight_snapshot(client)
        ready_bundle = _fetch_launcher_bundle(client)
        missing_approval_preflight = PREFLIGHT._request_json(
            client,
            "GET",
            "/v1/agent-job-external-lease/preflight?desired_execution_owner="
            + DESIRED_OWNER
            + "&operator_id=qsyy",
        )
        approval = PREFLIGHT._record_operator_approval(
            client,
            target_kind="agent_job_external_lease",
            target_id=DESIRED_OWNER,
            operator_id="qsyy",
            timestamp=PREFLIGHT._utcnow(),
        )
        approved_preflight = PREFLIGHT._request_json(
            client,
            "GET",
            "/v1/agent-job-external-lease/preflight?desired_execution_owner="
            + DESIRED_OWNER
            + "&operator_id=qsyy&approval_id="
            + approval["approval_id"],
        )
        env_index = _runtime_config_env_index(list(runtime_config.get("environment") or []))
        required_env_keys = [
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
        ]
        agent_job_topology = next(
            (
                item
                for item in list(after_worker["queue_topology"].get("work_kinds") or [])
                if isinstance(item, dict) and item.get("work_kind") == "agent_job"
            ),
            {},
        )
        summary = after_worker["runtime_overview"].get("summary") or {}
        checks = {
            "launcher_reported_runtime_addr": runtime.metadata.get("runtime_addr")
            == f"127.0.0.1:{runtime.base_url.rsplit(':', 1)[1]}",
            "required_bundle_env_visible_in_runtime_config": all(
                bool(env_index.get(key, {}).get("present")) for key in required_env_keys
            ),
            "queue_backend_promoted_to_nats": after_worker["queue_backend"].get("provider")
            == "nats_jetstream",
            "queue_topology_ack_owner_promoted": agent_job_topology.get("ack_owner")
            == "nats_external_lease_result_ack",
            "queue_topology_execution_owner_promoted": agent_job_topology.get(
                "execution_owner"
            )
            == DESIRED_OWNER,
            "launcher_bundle_blocks_before_worker_coverage": before_worker_bundle.get(
                "ready"
            )
            is False
            and "agent_job_worker_coverage_blocked"
            in list((before_worker_bundle.get("plan") or {}).get("blockers") or []),
            "worker_status_report_accepted": worker_report["status_code"] == 202,
            "launcher_bundle_ready_after_worker": ready_bundle.get("ready") is True
            and ready_bundle.get("reason")
            == "agent_job_external_lease_launcher_bundle_ready",
            "approval_bound_preflight_requires_approval": missing_approval_preflight.get(
                "ready"
            )
            is False
            and missing_approval_preflight.get("reason") == "missing_approval_id",
            "approval_bound_preflight_ready_after_approval": approved_preflight.get(
                "ready"
            )
            is True
            and approved_preflight.get("reason")
            == "agent_job_external_lease_preflight_ready",
            "runtime_overview_summary_promoted": summary.get(
                "agent_job_external_lease_execution_owner"
            )
            == DESIRED_OWNER
            and summary.get("agent_job_external_lease_ready") is True,
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
            "created_job": created_job,
            "before_worker": before_worker,
            "before_worker_bundle": before_worker_bundle,
            "worker_report": worker_report,
            "after_worker": after_worker,
            "ready_bundle": ready_bundle,
            "missing_approval_preflight": missing_approval_preflight,
            "approval": approval,
            "approved_preflight": approved_preflight,
            "checks": checks,
        }
    finally:
        if runtime is not None:
            runtime.stop()
        nats.stop(repo_root)


def _build_result(
    *,
    repo_root: Path,
    blocked_bundle: dict[str, Any] | None,
    promoted_bundle: dict[str, Any] | None,
    error: str | None = None,
) -> dict[str, Any]:
    blocked_checks = dict((blocked_bundle or {}).get("checks") or {})
    promoted_checks = dict((promoted_bundle or {}).get("checks") or {})
    all_checks = list(blocked_checks.values()) + list(promoted_checks.values())
    if error:
        status = "error"
        category = "agent_job_external_lease_launcher_bundle_error"
        reason = error
    elif all_checks and all(bool(value) for value in all_checks):
        status = "live_verified"
        category = "repo_owned_launcher_external_lease_bundle_live_verified"
        reason = (
            "The read-only launcher bundle endpoint exposed the canonical external-lease "
            "result-ack startup contract, and a temp runtime launched from that bundle "
            "reached the expected NATS ownership and approval-bound preflight states."
        )
    else:
        status = "verification_failed"
        category = "agent_job_external_lease_launcher_bundle_failed"
        reason = "at least one launcher-bundle external-lease check failed"
    return {
        "generated_at": PREFLIGHT._utcnow().isoformat(),
        "repo_root": str(repo_root),
        "verification_scope": "agent_job_external_lease_launcher_bundle",
        "blocked_bundle_scenario": blocked_bundle,
        "promoted_bundle_scenario": promoted_bundle,
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
        **({"error": error} if error else {}),
    }


def run_agent_job_external_lease_launcher_bundle(repo_root: Path) -> dict[str, Any]:
    blocked_bundle = None
    promoted_bundle = None
    try:
        blocked_bundle = _blocked_bundle_scenario(repo_root)
        promoted_bundle = _promoted_bundle_scenario(
            repo_root,
            blocked_bundle=blocked_bundle["bundle"],
        )
        return _build_result(
            repo_root=repo_root,
            blocked_bundle=blocked_bundle,
            promoted_bundle=promoted_bundle,
        )
    except Exception as exc:  # pragma: no cover - defensive live path
        return _build_result(
            repo_root=repo_root,
            blocked_bundle=blocked_bundle,
            promoted_bundle=promoted_bundle,
            error=str(exc),
        )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_agent_job_external_lease_launcher_bundle(
        Path(args.repo_root).resolve()
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
