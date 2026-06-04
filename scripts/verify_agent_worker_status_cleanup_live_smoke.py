from __future__ import annotations

import argparse
import importlib.util
import json
import sys
import uuid
from contextlib import suppress
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any


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


def _item_value(item: dict[str, Any] | None, *keys: str) -> Any:
    if not isinstance(item, dict):
        return None
    for key in keys:
        if key in item and item[key] is not None:
            return item[key]
    return None


def _parse_iso_datetime(value: Any) -> datetime | None:
    text = str(value or "").strip()
    if not text:
        return None
    if text.endswith("Z"):
        text = text[:-1] + "+00:00"
    with suppress(ValueError):
        return datetime.fromisoformat(text).astimezone(timezone.utc)
    return None


def _status_items(payload: dict[str, Any]) -> list[dict[str, Any]]:
    return [item for item in payload.get("workers", []) if isinstance(item, dict)]


def _cleanup_payload(
    *,
    worker_id: str = "",
    instance_id: str = "",
    timestamp: datetime,
    stale_after_seconds: int = 60,
) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "timestamp": timestamp.astimezone(timezone.utc).isoformat(),
        "stale_after_seconds": stale_after_seconds,
    }
    if worker_id:
        payload["worker_id"] = worker_id
    if instance_id:
        payload["instance_id"] = instance_id
    return payload


def run_worker_status_cleanup_smoke(
    client: Any,
    *,
    state_file: Path,
    stale_worker_id: str,
    active_worker_id: str,
    now: datetime,
) -> dict[str, Any]:
    stale_instance_id = f"{stale_worker_id}:44444:{uuid.uuid4().hex[:8]}"
    active_instance_id = f"{active_worker_id}:55555:{uuid.uuid4().hex[:8]}"

    stale_response = client.report_status(
        HELPERS._make_report_payload(
            worker_id=stale_worker_id,
            instance_id=stale_instance_id,
            timestamp=now - timedelta(minutes=2),
        )
    )
    active_response = client.report_status(
        HELPERS._make_report_payload(
            worker_id=active_worker_id,
            instance_id=active_instance_id,
            timestamp=now,
        )
    )
    before_listing = client.request_json(
        "GET",
        "/v1/agent-worker-statuses?stale_after_seconds=60",
    )
    before_items = _status_items(before_listing)
    stale_before = HELPERS._status_item(before_items, stale_worker_id)
    active_before = HELPERS._status_item(before_items, active_worker_id)

    cleanup_response = client.request(
        "POST",
        "/v1/agent-worker-statuses/cleanup-stale",
        json_body=_cleanup_payload(timestamp=now),
    )
    cleanup_payload = cleanup_response.json()
    cleanup_data = cleanup_payload.get("data", cleanup_payload)
    if not isinstance(cleanup_data, dict):
        cleanup_data = {"items": cleanup_data}

    after_listing = client.request_json(
        "GET",
        "/v1/agent-worker-statuses?stale_after_seconds=60",
    )
    after_items = _status_items(after_listing)
    stale_after = HELPERS._status_item(after_items, stale_worker_id)
    active_after = HELPERS._status_item(after_items, active_worker_id)
    state_items = HELPERS._read_state_items(state_file)
    stale_state = HELPERS._status_item(state_items, stale_worker_id)
    active_state = HELPERS._status_item(state_items, active_worker_id)

    checks = {
        "stale_report_accepted": stale_response.status_code == 202,
        "active_report_accepted": active_response.status_code == 202,
        "pre_cleanup_stale_visible": stale_before is not None
        and bool(_item_value(stale_before, "stale", "Stale")) is True,
        "pre_cleanup_active_visible": active_before is not None
        and str(_item_value(active_before, "instance_id", "InstanceID") or "") == active_instance_id,
        "cleanup_request_succeeded": cleanup_response.status_code == 200,
        "cleanup_deleted_one_stale_worker": int(cleanup_data.get("totals", {}).get("deleted", 0)) == 1,
        "cleanup_removed_stale_from_api": stale_after is None,
        "cleanup_preserved_active_worker": active_after is not None
        and str(_item_value(active_after, "instance_id", "InstanceID") or "") == active_instance_id,
        "cleanup_removed_stale_from_state_file": stale_state is None,
        "cleanup_preserved_active_state_file_record": active_state is not None
        and str(_item_value(active_state, "instance_id", "InstanceID") or "") == active_instance_id,
    }
    overall_ok = all(bool(value) for value in checks.values())

    return {
        "stale_worker_id": stale_worker_id,
        "active_worker_id": active_worker_id,
        "stale_instance_id": stale_instance_id,
        "active_instance_id": active_instance_id,
        "status_codes": {
            "stale_report": stale_response.status_code,
            "active_report": active_response.status_code,
            "cleanup": cleanup_response.status_code,
        },
        "cleanup_totals": cleanup_data.get("totals", {}),
        "state_file": str(state_file),
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if overall_ok else "verification_incomplete",
            "category": (
                "worker_status_cleanup_live_verified"
                if overall_ok
                else "worker_status_cleanup_verification_incomplete"
            ),
            "reason": (
                "temp Go runtime removed only stale agent-worker-status records and preserved active worker state."
                if overall_ok
                else "at least one worker-status cleanup smoke check failed"
            ),
        },
    }


def inspect_live_runtime_worker_cleanup_state(
    client: Any,
    *,
    stale_after_seconds: int = 180,
) -> dict[str, Any]:
    workers_payload = client.request_json("GET", "/v1/agent-worker-statuses")
    overview_payload = client.request_json("GET", "/v1/runtime-overview")
    workers = _status_items(workers_payload)
    stale_workers = [
        {
            "worker_id": str(_item_value(item, "worker_id", "WorkerID") or ""),
            "instance_id": str(_item_value(item, "instance_id", "InstanceID") or ""),
            "worker_type": str(_item_value(item, "worker_type", "WorkerType") or ""),
            "status": str(_item_value(item, "status", "Status") or ""),
            "updated_at": _item_value(item, "updated_at", "UpdatedAt"),
            "last_error": _item_value(item, "last_error", "LastError"),
        }
        for item in workers
        if bool(_item_value(item, "stale", "Stale")) is True
    ]
    summary = overview_payload.get("summary", {}) if isinstance(overview_payload, dict) else {}
    return {
        "workers_total": int(workers_payload.get("totals", {}).get("workers", len(workers))),
        "workers_stale": int(workers_payload.get("totals", {}).get("stale", len(stale_workers))),
        "runtime_overview_agent_workers_stale": int(summary.get("agent_workers_stale", 0)),
        "stale_workers": stale_workers,
        "stale_after_seconds": stale_after_seconds,
    }


def cleanup_live_runtime_stale_workers(
    client: Any,
    *,
    stale_after_seconds: int = 180,
) -> dict[str, Any]:
    before = inspect_live_runtime_worker_cleanup_state(
        client,
        stale_after_seconds=stale_after_seconds,
    )
    cleanup_results: list[dict[str, Any]] = []
    for item in before["stale_workers"]:
        response = client.request_json(
            "POST",
            "/v1/agent-worker-statuses/cleanup-stale",
            json_body=_cleanup_payload(
                worker_id=str(item.get("worker_id") or ""),
                instance_id=str(item.get("instance_id") or ""),
                timestamp=HELPERS._utcnow(),
                stale_after_seconds=stale_after_seconds,
            ),
        )
        cleanup_results.append(
            {
                "worker_id": item.get("worker_id"),
                "instance_id": item.get("instance_id"),
                "totals": response.get("totals", {}),
            }
        )
    after = inspect_live_runtime_worker_cleanup_state(
        client,
        stale_after_seconds=stale_after_seconds,
    )
    checks = {
        "cleanup_not_required_or_attempted": bool(before["workers_stale"] == 0 or cleanup_results),
        "live_runtime_stale_count_not_increased": after["workers_stale"] <= before["workers_stale"],
        "live_runtime_stale_records_removed": (
            before["workers_stale"] == 0 or after["workers_stale"] < before["workers_stale"]
        ),
    }
    overall_ok = all(bool(value) for value in checks.values())
    return {
        "before": before,
        "cleanup_results": cleanup_results,
        "after": after,
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if overall_ok else "verification_incomplete",
            "category": (
                "worker_status_cleanup_live_runtime_applied"
                if overall_ok
                else "worker_status_cleanup_live_runtime_verification_incomplete"
            ),
            "reason": (
                "current runtime stale agent-worker-status records were removed through the new cleanup endpoint."
                if overall_ok
                else "current runtime stale worker cleanup did not reduce stale records as expected"
            ),
        },
    }


def run_agent_worker_status_cleanup_live_smoke(
    repo_root: Path,
    *,
    runtime_base_url: str,
    apply_live_runtime_cleanup: bool,
) -> dict[str, Any]:
    runtime = HELPERS.start_temp_runtime(repo_root)
    temp_client = HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
    live_client = HELPERS.WorkerStatusRuntimeClient(runtime_base_url)
    stale_worker_id = f"verify-worker-status-cleanup-stale:{uuid.uuid4().hex[:8]}"
    active_worker_id = f"verify-worker-status-cleanup-active:{uuid.uuid4().hex[:8]}"
    now = HELPERS._utcnow()
    try:
        cleanup_smoke = run_worker_status_cleanup_smoke(
            temp_client,
            state_file=runtime.state_file,
            stale_worker_id=stale_worker_id,
            active_worker_id=active_worker_id,
            now=now,
        )
        live_runtime = (
            cleanup_live_runtime_stale_workers(live_client)
            if apply_live_runtime_cleanup
            else {
                "before": (live_before := inspect_live_runtime_worker_cleanup_state(live_client)),
                "cleanup_results": [],
                "after": None,
                "checks": {
                    "live_runtime_inspected": True,
                    "live_runtime_cleanup_not_applied": True,
                },
                "conclusion": {
                    "status": "live_verified",
                    "category": (
                        "worker_status_cleanup_live_runtime_clean"
                        if int(live_before["workers_stale"]) == 0
                        else "worker_status_cleanup_live_runtime_stale_detected"
                    ),
                    "reason": (
                        "current runtime inspection found no stale agent-worker-status records."
                        if int(live_before["workers_stale"]) == 0
                        else "current runtime inspection still shows stale agent-worker-status records; cleanup was not applied in this run."
                    ),
                },
            }
        )
        checks = dict(cleanup_smoke.get("checks", {}))
        for key, value in live_runtime.get("checks", {}).items():
            checks[f"live_{key}"] = value
        overall_ok = all(bool(value) for value in cleanup_smoke.get("checks", {}).values())
        if apply_live_runtime_cleanup:
            overall_ok = overall_ok and all(bool(value) for value in live_runtime.get("checks", {}).values())
        return {
            "runtime_base_url": runtime_base_url,
            "temp_runtime_base_url": runtime.base_url,
            "state_dir": str(runtime.state_dir),
            "state_file": str(runtime.state_file),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "cleanup_smoke": cleanup_smoke,
            "live_runtime": live_runtime,
            "checks": checks,
            "conclusion": {
                "status": "live_verified" if overall_ok else "verification_incomplete",
                "category": (
                    "worker_status_cleanup_live_verified"
                    if overall_ok
                    else "worker_status_cleanup_verification_incomplete"
                ),
                "reason": (
                    "temp Go runtime stale worker cleanup smoke passed and current runtime stale worker state was inspected."
                    if overall_ok
                    else "at least one worker-status cleanup verification check failed"
                ),
            },
        }
    finally:
        runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    parser.add_argument("--runtime-base-url", default="http://127.0.0.1:8780")
    parser.add_argument("--apply-live-runtime-cleanup", action="store_true")
    args = parser.parse_args(argv)
    result = run_agent_worker_status_cleanup_live_smoke(
        Path(args.repo_root).resolve(),
        runtime_base_url=str(args.runtime_base_url).strip(),
        apply_live_runtime_cleanup=bool(args.apply_live_runtime_cleanup),
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
