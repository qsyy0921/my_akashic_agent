from __future__ import annotations

import argparse
import importlib.util
import json
import sys
from contextlib import suppress
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
PREFLIGHT_HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_job_external_lease_cutover_preflight.py"
)
LAUNCHER_BUNDLE_HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_job_external_lease_launcher_bundle.py"
)
DESIRED_OWNER = "python_ai_worker_with_nats_result_ack"


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
LAUNCHER_BUNDLE = _load_module(
    "verify_agent_job_external_lease_launcher_bundle",
    LAUNCHER_BUNDLE_HELPERS_PATH,
)


def _request_cutover_diff(client: Any) -> dict[str, Any]:
    return PREFLIGHT._request_json(
        client,
        "GET",
        "/v1/agent-job-external-lease/cutover-diff?desired_execution_owner="
        + DESIRED_OWNER,
    )


def _diff_names(payload: dict[str, Any]) -> set[str]:
    return {
        str(item.get("name") or "")
        for item in list(payload.get("drift") or [])
        if str(item.get("name") or "").strip()
    }


def _live_runtime_scenario(base_url: str) -> dict[str, Any]:
    client = PREFLIGHT.HELPERS.WorkerStatusRuntimeClient(base_url)
    bundle = LAUNCHER_BUNDLE._fetch_launcher_bundle(client)
    diff = _request_cutover_diff(client)
    drift_names = _diff_names(diff)
    checks = {
        "cutover_diff_endpoint_reachable": True,
        "live_runtime_reports_blocked": diff.get("ready") is False
        and diff.get("reason") == "agent_job_external_lease_cutover_diff_blocked",
        "live_runtime_detects_expected_env_drift": {
            "AKASHIC_QUEUE_BACKEND",
            "AKASHIC_QUEUE_MODE",
            "AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER",
            "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED",
            "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN",
            "QueueDSN",
        }.issubset(drift_names),
        "live_runtime_detects_expected_owner_drift": {
            "agent_job_execution_owner",
            "agent_job_ack_owner",
            "external_lease_ready",
        }.issubset(drift_names),
        "live_runtime_current_state_matches_blocked_owner": diff.get(
            "current_execution_owner"
        )
        == "python_ai_worker_state_store_lease"
        and diff.get("current_ack_owner") == "go_state_store_api"
        and diff.get("current_queue_provider") == "local"
        and diff.get("current_queue_mode") == "local_state_store",
        "live_runtime_bundle_matches_diff_owner": bundle.get("desired_execution_owner")
        == diff.get("desired_execution_owner")
        == DESIRED_OWNER,
    }
    return {
        "base_url": base_url,
        "bundle": bundle,
        "cutover_diff": diff,
        "checks": checks,
    }


def _promoted_runtime_scenario(repo_root: Path, live_bundle: dict[str, Any]) -> dict[str, Any]:
    nats = PREFLIGHT.start_temp_nats(repo_root)
    runtime = None
    try:
        runtime = LAUNCHER_BUNDLE._start_runtime_via_launcher_bundle(
            repo_root,
            bundle=live_bundle,
            queue_dsn=nats.dsn,
        )
        client = PREFLIGHT.HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
        created_job = PREFLIGHT._post_job(
            client,
            job_type="group_memory_extract",
            timestamp=PREFLIGHT._utcnow() - PREFLIGHT.timedelta(minutes=31),
        )
        before_worker = _request_cutover_diff(client)
        worker_report = PREFLIGHT._report_knowledge_worker(
            client,
            timestamp=PREFLIGHT._utcnow(),
        )
        after_worker = _request_cutover_diff(client)
        checks = {
            "promoted_runtime_eliminates_config_drift_before_worker": len(
                list(before_worker.get("drift") or [])
            )
            == 0,
            "promoted_runtime_still_blocks_before_worker_coverage": before_worker.get(
                "ready"
            )
            is False
            and "agent_job_worker_coverage_blocked"
            in list(before_worker.get("blockers") or []),
            "worker_status_report_accepted": worker_report["status_code"] == 202,
            "promoted_runtime_cutover_diff_ready_after_worker": after_worker.get(
                "ready"
            )
            is True
            and after_worker.get("reason")
            == "agent_job_external_lease_cutover_diff_ready",
            "promoted_runtime_keeps_zero_drift_after_worker": len(
                list(after_worker.get("drift") or [])
            )
            == 0,
        }
        return {
            "launcher": runtime.metadata,
            "runtime_root": str(runtime.runtime_root),
            "nats": {
                "dsn": nats.dsn,
                "container_name": nats.container_name,
                "port": nats.port,
            },
            "created_job": created_job,
            "before_worker": before_worker,
            "worker_report": worker_report,
            "after_worker": after_worker,
            "checks": checks,
        }
    finally:
        if runtime is not None:
            runtime.stop()
        nats.stop(repo_root)


def _build_result(
    *,
    repo_root: Path,
    live_runtime: dict[str, Any] | None,
    promoted_runtime: dict[str, Any] | None,
    error: str | None = None,
) -> dict[str, Any]:
    live_checks = dict((live_runtime or {}).get("checks") or {})
    promoted_checks = dict((promoted_runtime or {}).get("checks") or {})
    all_checks = list(live_checks.values()) + list(promoted_checks.values())
    if error:
        status = "error"
        category = "agent_job_external_lease_cutover_diff_error"
        reason = error
    elif all_checks and all(bool(value) for value in all_checks):
        status = "live_verified"
        category = "repo_owned_agent_job_external_lease_cutover_diff_live_verified"
        reason = (
            "The cutover-diff endpoint exposed the exact live-runtime drift against the "
            "canonical launcher bundle, and a temp runtime launched from that same bundle "
            "eliminated config drift before worker coverage and became ready after worker coverage."
        )
    else:
        status = "verification_failed"
        category = "agent_job_external_lease_cutover_diff_failed"
        reason = "at least one cutover-diff check failed"
    return {
        "generated_at": PREFLIGHT._utcnow().isoformat(),
        "repo_root": str(repo_root),
        "verification_scope": "agent_job_external_lease_cutover_diff",
        "live_runtime_scenario": live_runtime,
        "promoted_runtime_scenario": promoted_runtime,
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
        **({"error": error} if error else {}),
    }


def run_agent_job_external_lease_cutover_diff(
    repo_root: Path,
    runtime_base_url: str,
) -> dict[str, Any]:
    live_runtime = None
    promoted_runtime = None
    try:
        live_runtime = _live_runtime_scenario(runtime_base_url)
        promoted_runtime = _promoted_runtime_scenario(
            repo_root,
            live_bundle=live_runtime["bundle"],
        )
        return _build_result(
            repo_root=repo_root,
            live_runtime=live_runtime,
            promoted_runtime=promoted_runtime,
        )
    except Exception as exc:  # pragma: no cover - defensive live path
        return _build_result(
            repo_root=repo_root,
            live_runtime=live_runtime,
            promoted_runtime=promoted_runtime,
            error=str(exc),
        )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    parser.add_argument("--runtime-base-url", default="http://127.0.0.1:8780")
    args = parser.parse_args(argv)
    result = run_agent_job_external_lease_cutover_diff(
        Path(args.repo_root).resolve(),
        str(args.runtime_base_url).rstrip("/"),
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
