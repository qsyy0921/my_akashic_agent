from __future__ import annotations

import argparse
import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import time
import uuid
from contextlib import suppress
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
FENCING_HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_worker_status_fencing_live_smoke.py"
)
NATS_HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_job_external_lease_nats_smoke.py"
)


def _load_module(module_name: str, path: Path):
    spec = importlib.util.spec_from_file_location(module_name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load helper module from {path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


HELPERS = _load_module(
    "verify_agent_worker_status_fencing_live_smoke",
    FENCING_HELPERS_PATH,
)
NATS_HELPERS = _load_module(
    "verify_agent_job_external_lease_nats_smoke",
    NATS_HELPERS_PATH,
)


def _utcnow() -> datetime:
    return datetime.now(timezone.utc)


def _trim_lines(text: str, *, limit: int = 80) -> list[str]:
    lines = [line.rstrip() for line in text.splitlines() if line.rstrip()]
    if len(lines) <= limit:
        return lines
    return lines[-limit:]


@dataclass
class TempRuntimeHandle:
    process: subprocess.Popen[str]
    base_url: str
    runtime_root: Path
    state_dir: Path
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


@dataclass
class TempNATSHandle:
    docker_exe: str
    container_name: str
    port: int

    @property
    def dsn(self) -> str:
        return f"nats://127.0.0.1:{self.port}"

    def stop(self, repo_root: Path) -> None:
        with suppress(Exception):
            subprocess.run(
                [self.docker_exe, "rm", "-f", self.container_name],
                cwd=str(repo_root),
                capture_output=True,
                text=True,
                timeout=30,
                encoding="utf-8",
                errors="replace",
            )


def start_temp_nats(repo_root: Path) -> TempNATSHandle:
    docker_exe = NATS_HELPERS._find_docker_exe()
    if not docker_exe:
        raise RuntimeError("docker executable not found")
    image_pull = NATS_HELPERS._run_command(
        [docker_exe, "pull", NATS_HELPERS.NATS_IMAGE],
        cwd=repo_root,
        timeout_seconds=180,
    )
    if image_pull["returncode"] != 0:
        raise RuntimeError(
            "docker pull failed for temporary NATS image:\n"
            + "\n".join(_trim_lines(image_pull["stderr"], limit=40))
        )
    port = HELPERS._find_free_port()
    container_name = f"akashic-agentjob-preflight-{uuid.uuid4().hex[:8]}"
    run_result = NATS_HELPERS._run_command(
        [
            docker_exe,
            "run",
            "-d",
            "--rm",
            "--name",
            container_name,
            "-p",
            f"127.0.0.1:{port}:4222",
            NATS_HELPERS.NATS_IMAGE,
            "-js",
        ],
        cwd=repo_root,
        timeout_seconds=30,
    )
    if run_result["returncode"] != 0:
        raise RuntimeError(
            "docker run failed for temporary NATS container:\n"
            + "\n".join(_trim_lines(run_result["stderr"], limit=40))
        )
    if not NATS_HELPERS._wait_for_port("127.0.0.1", port, timeout_seconds=20.0):
        logs = NATS_HELPERS._run_command(
            [docker_exe, "logs", container_name],
            cwd=repo_root,
            timeout_seconds=15,
        )
        raise RuntimeError(
            "temporary NATS container did not become reachable:\n"
            + "\n".join(_trim_lines(logs["stdout"], limit=40))
        )
    return TempNATSHandle(docker_exe=docker_exe, container_name=container_name, port=port)


def start_temp_runtime(
    repo_root: Path,
    *,
    env_overrides: dict[str, str],
    env_remove: list[str] | None = None,
    prefix: str,
) -> TempRuntimeHandle:
    port = HELPERS._find_free_port()
    runtime_root = Path(tempfile.mkdtemp(prefix=prefix))
    state_dir = runtime_root / "state"
    state_dir.mkdir(parents=True, exist_ok=True)
    stdout_log = runtime_root / "agent-runtime.stdout.log"
    stderr_log = runtime_root / "agent-runtime.stderr.log"
    stdout_handle = stdout_log.open("w", encoding="utf-8")
    stderr_handle = stderr_log.open("w", encoding="utf-8")
    env = os.environ.copy()
    env["AKASHIC_RUNTIME_ADDR"] = f"127.0.0.1:{port}"
    env["AKASHIC_RUNTIME_STATE_DIR"] = str(state_dir)
    for key in env_remove or []:
        env.pop(key, None)
    env.update(env_overrides)
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
            with HELPERS.httpx.Client(timeout=2.0, trust_env=True) as client:
                response = client.get(base_url + "/healthz")
            if response.status_code == 200:
                stdout_handle.close()
                stderr_handle.close()
                return TempRuntimeHandle(
                    process=process,
                    base_url=base_url,
                    runtime_root=runtime_root,
                    state_dir=state_dir,
                    stdout_log=stdout_log,
                    stderr_log=stderr_log,
                )
        except Exception as exc:  # pragma: no cover - startup wait
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


def _request_json(client: Any, method: str, path: str, *, body: Any | None = None) -> dict[str, Any]:
    return client.request_json(method, path, json_body=body)


def _record_operator_approval(
    client: Any,
    *,
    target_kind: str,
    target_id: str,
    operator_id: str,
    timestamp: datetime,
) -> dict[str, Any]:
    return _request_json(
        client,
        "POST",
        "/v1/operator-approvals",
        body={
            "target_kind": target_kind,
            "target_id": target_id,
            "decision": "approved",
            "operator_id": operator_id,
            "timestamp": timestamp.astimezone(timezone.utc).isoformat(),
            "metadata": {"source": "verify_agent_job_external_lease_cutover_preflight"},
        },
    )


def _post_job(client: Any, *, job_type: str, timestamp: datetime) -> dict[str, Any]:
    job_id = f"{job_type}:verify:{uuid.uuid4().hex[:8]}"
    body = {
        "job_id": job_id,
        "job_type": job_type,
        "agent_id": "verify-agent-job-external-lease-preflight",
        "route": {
            "platform": "qq",
            "account_id": "1049511700",
            "conversation_id": "3219982",
            "conversation_type": "group",
        },
        "source_event_ids": [f"qq:gqq:3219982:{uuid.uuid4().hex[:8]}"],
        "payload": {"source": "verify_preflight"},
        "dedupe_key": f"verify:{job_type}:{uuid.uuid4().hex[:8]}",
        "max_attempts": 2,
        "timestamp": timestamp.astimezone(timezone.utc).isoformat(),
        "metadata": {"source": "verify_agent_job_external_lease_cutover_preflight"},
    }
    return _request_json(client, "POST", "/v1/jobs", body=body)


def _report_knowledge_worker(client: Any, *, timestamp: datetime) -> dict[str, Any]:
    worker_id = f"verify-knowledge-worker:{uuid.uuid4().hex[:8]}"
    instance_id = f"{worker_id}:44444:{uuid.uuid4().hex[:8]}"
    response = client.report_status(
        HELPERS._make_report_payload(
            worker_id=worker_id,
            instance_id=instance_id,
            timestamp=timestamp,
        )
    )
    return {
        "worker_id": worker_id,
        "instance_id": instance_id,
        "status_code": response.status_code,
    }


def _collect_preflight_snapshot(client: Any) -> dict[str, Any]:
    queue_backend = _request_json(client, "GET", "/v1/queue-backend")
    queue_topology = _request_json(client, "GET", "/v1/queue-topology")
    readiness = _request_json(client, "GET", "/v1/agent-job-external-lease/readiness")
    plan = _request_json(client, "GET", "/v1/agent-job-external-lease/plan")
    runtime_overview = _request_json(client, "GET", "/v1/runtime-overview")
    workers = _request_json(client, "GET", "/v1/agent-worker-statuses")
    jobs = _request_json(client, "GET", "/v1/jobs?type=group_memory_extract&limit=20")
    return {
        "queue_backend": queue_backend,
        "queue_topology": queue_topology,
        "readiness": readiness,
        "plan": plan,
        "runtime_overview": runtime_overview,
        "workers": workers,
        "jobs": jobs,
    }


def _config_blocked_scenario(repo_root: Path) -> dict[str, Any]:
    runtime = start_temp_runtime(
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
        prefix="agent-job-cutover-config-blocked-",
    )
    client = HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
    try:
        snapshot = _collect_preflight_snapshot(client)
        endpoint_preflight = _request_json(
            client,
            "GET",
            "/v1/agent-job-external-lease/preflight?desired_execution_owner=python_ai_worker_with_nats_result_ack&operator_id=qsyy",
        )
        readiness = snapshot["readiness"]
        queue_backend = snapshot["queue_backend"]
        checks = {
            "readiness_blocked": readiness.get("ready") is False,
            "configuration_blocker_present": "external_lease_not_configured"
            in list(readiness.get("blockers") or []),
            "strict_token_blocker_present": "strict_lease_token_disabled"
            in list(readiness.get("blockers") or []),
            "queue_provider_is_local": queue_backend.get("provider") == "local",
            "current_owner_is_state_store": queue_backend.get("agent_job_execution_owner")
            == "python_ai_worker_state_store_lease",
            "approval_bound_preflight_blocks_on_plan_first": endpoint_preflight.get("ready") is False
            and endpoint_preflight.get("reason") == "agent_job_external_lease_plan_not_ready",
        }
        return {
            "base_url": runtime.base_url,
            "runtime_root": str(runtime.runtime_root),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "snapshot": snapshot,
            "endpoint_preflight": endpoint_preflight,
            "checks": checks,
        }
    finally:
        runtime.stop()


def _preflight_ready_scenario(repo_root: Path) -> dict[str, Any]:
    nats = start_temp_nats(repo_root)
    runtime = None
    try:
        runtime = start_temp_runtime(
            repo_root,
            env_overrides={
                "AKASHIC_QUEUE_BACKEND": "nats",
                "AKASHIC_QUEUE_MODE": "external_lease",
                "AKASHIC_QUEUE_DSN": nats.dsn,
                "AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER": "true",
                "AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED": "true",
                "AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED": "true",
                "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED": "true",
                "AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED": "true",
                "AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED": "true",
                "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN": "true",
                "AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED": "false",
            },
            env_remove=[
                "AKASHIC_ONEBOT_WS_URLS",
                "AKASHIC_ONEBOT_ACCESS_TOKEN",
                "AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT",
                "AKASHIC_BOT_IDS",
                "AKASHIC_QQ_GROUP_SEND_ENABLED",
                "AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED",
                "AKASHIC_TELEGRAM_BOT_TOKEN",
                "TELEGRAM_BOT_TOKEN",
            ],
            prefix="agent-job-cutover-preflight-",
        )
        client = HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
        created_job = _post_job(
            client,
            job_type="group_memory_extract",
            timestamp=_utcnow() - timedelta(minutes=31),
        )
        before_worker = _collect_preflight_snapshot(client)
        worker_report = _report_knowledge_worker(client, timestamp=_utcnow())
        after_worker = _collect_preflight_snapshot(client)
        missing_approval_preflight = _request_json(
            client,
            "GET",
            "/v1/agent-job-external-lease/preflight?desired_execution_owner=python_ai_worker_with_nats_result_ack&operator_id=qsyy",
        )
        approval = _record_operator_approval(
            client,
            target_kind="agent_job_external_lease",
            target_id="python_ai_worker_with_nats_result_ack",
            operator_id="qsyy",
            timestamp=_utcnow(),
        )
        approved_preflight = _request_json(
            client,
            "GET",
            "/v1/agent-job-external-lease/preflight?desired_execution_owner=python_ai_worker_with_nats_result_ack&operator_id=qsyy&approval_id="
            + approval["approval_id"],
        )
        before_readiness = before_worker["readiness"]
        after_readiness = after_worker["readiness"]
        before_plan = before_worker["plan"]
        after_plan = after_worker["plan"]
        after_queue_backend = after_worker["queue_backend"]
        after_queue_topology = after_worker["queue_topology"]
        after_summary = after_worker["runtime_overview"].get("summary") or {}
        agent_job_topology = next(
            (
                item
                for item in list(after_queue_topology.get("work_kinds") or [])
                if isinstance(item, dict) and item.get("work_kind") == "agent_job"
            ),
            {},
        )
        checks = {
            "queue_backend_nats_selected": after_queue_backend.get("provider")
            == "nats_jetstream",
            "external_lease_base_ready": bool(
                ((after_queue_backend.get("external_lease") or {}).get("allow_execution"))
            ),
            "agent_job_owner_promoted_to_nats_result_ack": after_queue_backend.get(
                "agent_job_execution_owner"
            )
            == "python_ai_worker_with_nats_result_ack",
            "queue_topology_ack_owner_promoted": agent_job_topology.get("ack_owner")
            == "nats_external_lease_result_ack",
            "queue_topology_execution_owner_promoted": agent_job_topology.get(
                "execution_owner"
            )
            == "python_ai_worker_with_nats_result_ack",
            "worker_coverage_blocks_without_worker": before_readiness.get("ready") is False
            and "agent_job_worker_coverage_blocked"
            in list(before_readiness.get("blockers") or []),
            "worker_status_report_accepted": worker_report["status_code"] == 202,
            "readiness_ready_after_worker": after_readiness.get("ready") is True,
            "plan_ready_after_worker": after_plan.get("ready") is True
            and after_plan.get("decision") == "ready",
            "approval_bound_preflight_requires_approval": missing_approval_preflight.get("ready") is False
            and missing_approval_preflight.get("reason") == "missing_approval_id",
            "approval_bound_preflight_ready_after_approval": approved_preflight.get("ready") is True
            and approved_preflight.get("reason") == "agent_job_external_lease_preflight_ready",
            "runtime_overview_summary_ready_after_worker": (
                after_summary.get("agent_job_external_lease_ready") is True
                and after_summary.get("agent_job_external_lease_execution_owner")
                == "python_ai_worker_with_nats_result_ack"
            ),
        }
        return {
            "base_url": runtime.base_url,
            "runtime_root": str(runtime.runtime_root),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "nats": {
                "dsn": nats.dsn,
                "container_name": nats.container_name,
                "port": nats.port,
            },
            "created_job": created_job,
            "worker_report": worker_report,
            "approval": approval,
            "before_worker": before_worker,
            "after_worker": after_worker,
            "missing_approval_preflight": missing_approval_preflight,
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
    config_blocked: dict[str, Any] | None,
    preflight_ready: dict[str, Any] | None,
    error: str | None = None,
) -> dict[str, Any]:
    config_checks = dict((config_blocked or {}).get("checks") or {})
    preflight_checks = dict((preflight_ready or {}).get("checks") or {})
    all_checks = list(config_checks.values()) + list(preflight_checks.values())
    if error:
        status = "error"
        category = "agent_job_external_lease_cutover_preflight_error"
        reason = error
    elif all_checks and all(bool(value) for value in all_checks):
        status = "live_verified"
        category = "repo_owned_temp_runtime_cutover_preflight_live_verified"
        reason = (
            "temp runtimes proved the current configuration gate remains blocked "
            "without external-lease flags, and that NATS plus explicit result-ack "
            "flags and active worker coverage promote agent_job ownership and readiness."
        )
    else:
        status = "verification_failed"
        category = "agent_job_external_lease_cutover_preflight_failed"
        reason = "at least one isolated external-lease cutover preflight check failed"
    return {
        "generated_at": _utcnow().isoformat(),
        "repo_root": str(repo_root),
        "verification_scope": "agent_job_external_lease_result_ack_cutover_preflight",
        "config_blocked_scenario": config_blocked,
        "preflight_ready_scenario": preflight_ready,
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
        **({"error": error} if error else {}),
    }


def run_agent_job_external_lease_cutover_preflight(repo_root: Path) -> dict[str, Any]:
    config_blocked = None
    preflight_ready = None
    try:
        config_blocked = _config_blocked_scenario(repo_root)
        preflight_ready = _preflight_ready_scenario(repo_root)
        return _build_result(
            repo_root=repo_root,
            config_blocked=config_blocked,
            preflight_ready=preflight_ready,
        )
    except Exception as exc:  # pragma: no cover - defensive live path
        return _build_result(
            repo_root=repo_root,
            config_blocked=config_blocked,
            preflight_ready=preflight_ready,
            error=str(exc),
        )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_agent_job_external_lease_cutover_preflight(
        Path(args.repo_root).resolve()
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
