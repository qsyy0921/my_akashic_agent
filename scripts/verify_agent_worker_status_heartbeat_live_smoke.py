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


def run_worker_status_heartbeat_smoke(
    client: Any,
    *,
    state_file: Path,
    worker_id: str,
    now: datetime,
) -> dict[str, Any]:
    instance_id = f"{worker_id}:33333:{uuid.uuid4().hex[:8]}"
    first_timestamp = now
    second_timestamp = now + timedelta(seconds=5)
    third_timestamp = now + timedelta(seconds=10)

    first_response = client.report_status(
        HELPERS._make_report_payload(
            worker_id=worker_id,
            instance_id=instance_id,
            timestamp=first_timestamp,
        )
    )
    first_listing = client.list_statuses()
    first_items = [
        item
        for item in first_listing.get("workers", [])
        if isinstance(item, dict)
    ]
    first_item = HELPERS._status_item(first_items, worker_id)

    second_response = client.report_status(
        HELPERS._make_report_payload(
            worker_id=worker_id,
            instance_id=instance_id,
            timestamp=second_timestamp,
        )
    )
    second_listing = client.list_statuses()
    second_items = [
        item
        for item in second_listing.get("workers", [])
        if isinstance(item, dict)
    ]
    second_item = HELPERS._status_item(second_items, worker_id)

    third_response = client.report_status(
        HELPERS._make_report_payload(
            worker_id=worker_id,
            instance_id=instance_id,
            timestamp=third_timestamp,
        )
    )
    third_listing = client.list_statuses()
    third_items = [
        item
        for item in third_listing.get("workers", [])
        if isinstance(item, dict)
    ]
    final_item = HELPERS._status_item(third_items, worker_id)

    state_items = HELPERS._read_state_items(state_file)
    state_item = HELPERS._status_item(state_items, worker_id)

    first_updated_at = _parse_iso_datetime(_item_value(first_item, "updated_at", "UpdatedAt"))
    second_updated_at = _parse_iso_datetime(_item_value(second_item, "updated_at", "UpdatedAt"))
    final_updated_at = _parse_iso_datetime(_item_value(final_item, "updated_at", "UpdatedAt"))
    state_updated_at = _parse_iso_datetime(_item_value(state_item, "updated_at", "UpdatedAt"))

    first_lease_until = _parse_iso_datetime(_item_value(first_item, "lease_until", "LeaseUntil"))
    second_lease_until = _parse_iso_datetime(_item_value(second_item, "lease_until", "LeaseUntil"))
    final_lease_until = _parse_iso_datetime(_item_value(final_item, "lease_until", "LeaseUntil"))
    state_lease_until = _parse_iso_datetime(_item_value(state_item, "lease_until", "LeaseUntil"))

    checks = {
        "initial_report_accepted": first_response.status_code == 202,
        "renewal_report_accepted": second_response.status_code == 202,
        "second_renewal_report_accepted": third_response.status_code == 202,
        "final_worker_record_visible": final_item is not None
        and str(_item_value(final_item, "instance_id", "InstanceID") or "") == instance_id,
        "updated_at_advanced_after_renewal": (
            first_updated_at is not None
            and second_updated_at is not None
            and final_updated_at is not None
            and second_updated_at > first_updated_at
            and final_updated_at > second_updated_at
        ),
        "lease_until_advanced_after_renewal": (
            first_lease_until is not None
            and second_lease_until is not None
            and final_lease_until is not None
            and second_lease_until > first_lease_until
            and final_lease_until > second_lease_until
        ),
        "lease_active_remains_true": bool(_item_value(final_item, "lease_active", "LeaseActive")) is True,
        "stale_remains_false": bool(_item_value(final_item, "stale", "Stale")) is False,
        "state_file_updated_to_latest_heartbeat": (
            state_item is not None
            and str(_item_value(state_item, "instance_id", "InstanceID") or "") == instance_id
            and state_updated_at is not None
            and final_updated_at is not None
            and state_updated_at == final_updated_at
            and state_lease_until is not None
            and final_lease_until is not None
            and state_lease_until == final_lease_until
        ),
    }
    overall_ok = all(bool(value) for value in checks.values())

    return {
        "worker_id": worker_id,
        "instance_id": instance_id,
        "timestamps": {
            "first": first_timestamp.isoformat(),
            "second": second_timestamp.isoformat(),
            "third": third_timestamp.isoformat(),
        },
        "status_codes": {
            "first": first_response.status_code,
            "second": second_response.status_code,
            "third": third_response.status_code,
        },
        "final_record": {
            "updated_at": _item_value(final_item, "updated_at", "UpdatedAt"),
            "lease_until": _item_value(final_item, "lease_until", "LeaseUntil"),
            "lease_active": _item_value(final_item, "lease_active", "LeaseActive"),
            "stale": _item_value(final_item, "stale", "Stale"),
        },
        "state_file": str(state_file),
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if overall_ok else "verification_incomplete",
            "category": (
                "worker_status_heartbeat_renewal_live_verified"
                if overall_ok
                else "worker_status_heartbeat_renewal_verification_incomplete"
            ),
            "reason": (
                "temp Go runtime accepted repeated running heartbeats from the same worker instance and advanced updated_at and lease_until without marking the worker stale."
                if overall_ok
                else "at least one worker-status heartbeat renewal smoke check failed"
            ),
        },
    }


def run_agent_worker_status_heartbeat_live_smoke(repo_root: Path) -> dict[str, Any]:
    runtime = HELPERS.start_temp_runtime(repo_root)
    client = HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
    worker_id = f"verify-worker-status-heartbeat:{uuid.uuid4().hex[:8]}"
    now = HELPERS._utcnow()
    try:
        heartbeat = run_worker_status_heartbeat_smoke(
            client,
            state_file=runtime.state_file,
            worker_id=worker_id,
            now=now,
        )
        checks = dict(heartbeat.get("checks", {}))
        overall_ok = all(bool(value) for value in checks.values())
        return {
            "base_url": runtime.base_url,
            "state_dir": str(runtime.state_dir),
            "state_file": str(runtime.state_file),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "heartbeat_smoke": heartbeat,
            "checks": checks,
            "conclusion": {
                "status": "live_verified" if overall_ok else "verification_incomplete",
                "category": (
                    "worker_status_heartbeat_renewal_live_verified"
                    if overall_ok
                    else "worker_status_heartbeat_renewal_verification_incomplete"
                ),
                "reason": (
                    "temp Go runtime kept the same worker instance lease active across repeated running heartbeats and refreshed the persisted worker-status timestamps."
                    if overall_ok
                    else "at least one worker-status heartbeat renewal smoke check failed"
                ),
            },
        }
    finally:
        runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_agent_worker_status_heartbeat_live_smoke(
        Path(args.repo_root).resolve()
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
