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
HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_worker_status_fencing_live_smoke.py"
)


def _load_helpers():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_worker_status_fencing_live_smoke",
        HELPERS_PATH,
    )
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load helper module from {HELPERS_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


HELPERS = _load_helpers()


def _status_items(payload: dict[str, Any]) -> list[dict[str, Any]]:
    return [item for item in payload.get("receivers", []) if isinstance(item, dict)]


def _item_value(item: dict[str, Any] | None, *keys: str) -> Any:
    if not isinstance(item, dict):
        return None
    for key in keys:
        if key in item and item[key] is not None:
            return item[key]
    return None


def _receiver_status_item(items: list[dict[str, Any]], receiver_id: str) -> dict[str, Any] | None:
    for item in items:
        if str(_item_value(item, "receiver_id", "ReceiverID") or "") == receiver_id:
            return item
    return None


def _read_state_items(state_file: Path) -> list[dict[str, Any]]:
    if not state_file.exists():
        return []
    payload = json.loads(state_file.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected receiver-status state payload in {state_file}")
    items = payload.get("receivers", [])
    if not isinstance(items, list):
        return []
    return [item for item in items if isinstance(item, dict)]


def _make_report_payload(
    *,
    kind: str,
    channel_name: str,
    account_id: str,
    status: str,
    timestamp: datetime,
    source: str = "python_channel",
) -> dict[str, Any]:
    return {
        "kind": kind,
        "channel_name": channel_name,
        "account_id": account_id,
        "status": status,
        "source": source,
        "timestamp": timestamp.astimezone(timezone.utc).isoformat(),
    }


def _cleanup_payload(
    *,
    receiver_id: str = "",
    timestamp: datetime,
    stale_after_seconds: int = 120,
) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "timestamp": timestamp.astimezone(timezone.utc).isoformat(),
        "stale_after_seconds": stale_after_seconds,
    }
    if receiver_id:
        payload["receiver_id"] = receiver_id
    return payload


def run_receiver_status_cleanup_smoke(
    client: Any,
    *,
    state_file: Path,
    stale_receiver_id: str,
    active_receiver_id: str,
    now: datetime,
) -> dict[str, Any]:
    stale_response = client.request(
        "POST",
        "/v1/receiver-statuses/report",
        json_body=_make_report_payload(
            kind="qq",
            channel_name="qq",
            account_id="1049511700",
            status="connected",
            timestamp=now - timedelta(minutes=5),
        ),
    )
    active_response = client.request(
        "POST",
        "/v1/receiver-statuses/report",
        json_body=_make_report_payload(
            kind="telegram",
            channel_name="telegram",
            account_id="7689386159",
            status="connected",
            timestamp=now,
        ),
    )
    before_listing = client.request_json("GET", "/v1/receiver-statuses")
    before_items = _status_items(before_listing)
    stale_before = _receiver_status_item(before_items, stale_receiver_id)
    active_before = _receiver_status_item(before_items, active_receiver_id)

    cleanup_response = client.request(
        "POST",
        "/v1/receiver-statuses/cleanup-stale",
        json_body=_cleanup_payload(timestamp=now),
    )
    cleanup_payload = cleanup_response.json()
    cleanup_data = cleanup_payload.get("data", cleanup_payload)
    if not isinstance(cleanup_data, dict):
        cleanup_data = {"items": cleanup_data}

    after_listing = client.request_json("GET", "/v1/receiver-statuses")
    after_items = _status_items(after_listing)
    stale_after = _receiver_status_item(after_items, stale_receiver_id)
    active_after = _receiver_status_item(after_items, active_receiver_id)
    state_items = _read_state_items(state_file)
    stale_state = _receiver_status_item(state_items, stale_receiver_id)
    active_state = _receiver_status_item(state_items, active_receiver_id)

    checks = {
        "stale_report_accepted": stale_response.status_code == 200,
        "active_report_accepted": active_response.status_code == 200,
        "pre_cleanup_stale_visible": stale_before is not None
        and str(_item_value(stale_before, "status", "Status") or "") == "stopped"
        and str(_item_value(stale_before, "reason", "Reason") or "") == "heartbeat_stale",
        "pre_cleanup_active_visible": active_before is not None
        and str(_item_value(active_before, "status", "Status") or "") == "connected",
        "cleanup_request_succeeded": cleanup_response.status_code == 200,
        "cleanup_deleted_one_stale_receiver": int(cleanup_data.get("totals", {}).get("deleted", 0)) == 1,
        "cleanup_removed_stale_from_api": stale_after is None,
        "cleanup_preserved_active_receiver": active_after is not None
        and str(_item_value(active_after, "status", "Status") or "") == "connected",
        "cleanup_removed_stale_from_state_file": stale_state is None,
        "cleanup_preserved_active_state_file_record": active_state is not None
        and str(_item_value(active_state, "ReceiverID", "receiver_id") or "") == active_receiver_id,
    }
    overall_ok = all(bool(value) for value in checks.values())
    return {
        "stale_receiver_id": stale_receiver_id,
        "active_receiver_id": active_receiver_id,
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
                "receiver_status_cleanup_live_verified"
                if overall_ok
                else "receiver_status_cleanup_verification_incomplete"
            ),
            "reason": (
                "temp Go runtime removed only stale receiver-status records and preserved active receiver state."
                if overall_ok
                else "at least one receiver-status cleanup smoke check failed"
            ),
        },
    }


def inspect_live_runtime_receiver_status_cleanup_state(client: Any) -> dict[str, Any]:
    receivers_payload = client.request_json("GET", "/v1/receiver-statuses")
    receivers = _status_items(receivers_payload)
    stale_receivers = [
        {
            "receiver_id": str(_item_value(item, "receiver_id", "ReceiverID") or ""),
            "kind": str(_item_value(item, "kind", "Kind") or ""),
            "channel_name": str(_item_value(item, "channel_name", "ChannelName") or ""),
            "account_id": str(_item_value(item, "account_id", "AccountID") or ""),
            "status": str(_item_value(item, "status", "Status") or ""),
            "reason": str(_item_value(item, "reason", "Reason") or ""),
            "updated_at": _item_value(item, "updated_at", "UpdatedAt"),
        }
        for item in receivers
        if str(_item_value(item, "reason", "Reason") or "") == "heartbeat_stale"
    ]
    totals = receivers_payload.get("totals", {}) if isinstance(receivers_payload, dict) else {}
    return {
        "receivers_total": int(totals.get("receivers", len(receivers))),
        "receivers_stopped": int(totals.get("stopped", 0)),
        "stale_receivers": stale_receivers,
    }


def cleanup_live_runtime_stale_receivers(client: Any) -> dict[str, Any]:
    before = inspect_live_runtime_receiver_status_cleanup_state(client)
    cleanup_results: list[dict[str, Any]] = []
    now = HELPERS._utcnow()
    for item in before["stale_receivers"]:
        receiver_id = str(item.get("receiver_id") or "")
        response = client.request_json(
            "POST",
            "/v1/receiver-statuses/cleanup-stale",
            json_body=_cleanup_payload(receiver_id=receiver_id, timestamp=now),
        )
        cleanup_results.append(
            {
                "receiver_id": receiver_id,
                "totals": response.get("totals", {}),
            }
        )
    after = inspect_live_runtime_receiver_status_cleanup_state(client)
    checks = {
        "cleanup_not_required_or_attempted": bool(before["receivers_total"] == 0 or cleanup_results or not before["stale_receivers"]),
        "live_runtime_stale_count_not_increased": len(after["stale_receivers"]) <= len(before["stale_receivers"]),
        "live_runtime_stale_records_removed": (
            len(before["stale_receivers"]) == 0 or len(after["stale_receivers"]) < len(before["stale_receivers"])
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
                "receiver_status_cleanup_live_runtime_applied"
                if overall_ok
                else "receiver_status_cleanup_live_runtime_verification_incomplete"
            ),
            "reason": (
                "current runtime stale receiver-status records were removed through the new cleanup endpoint."
                if overall_ok
                else "current runtime stale receiver cleanup did not reduce stale records as expected"
            ),
        },
    }


def run_receiver_status_cleanup_live_smoke(
    repo_root: Path,
    *,
    runtime_base_url: str,
    apply_live_runtime_cleanup: bool,
) -> dict[str, Any]:
    runtime = HELPERS.start_temp_runtime(repo_root)
    temp_client = HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
    live_client = HELPERS.WorkerStatusRuntimeClient(runtime_base_url)
    now = HELPERS._utcnow()
    stale_receiver_id = "qq:1049511700:qq"
    active_receiver_id = "telegram:7689386159:telegram"
    try:
        cleanup_smoke = run_receiver_status_cleanup_smoke(
            temp_client,
            state_file=runtime.state_dir / "receiver-statuses.json",
            stale_receiver_id=stale_receiver_id,
            active_receiver_id=active_receiver_id,
            now=now,
        )
        live_runtime = (
            cleanup_live_runtime_stale_receivers(live_client)
            if apply_live_runtime_cleanup
            else {
                "before": (live_before := inspect_live_runtime_receiver_status_cleanup_state(live_client)),
                "cleanup_results": [],
                "after": None,
                "checks": {
                    "live_runtime_inspected": True,
                    "live_runtime_cleanup_not_applied": True,
                },
                "conclusion": {
                    "status": "live_verified",
                    "category": (
                        "receiver_status_cleanup_live_runtime_clean"
                        if len(live_before["stale_receivers"]) == 0
                        else "receiver_status_cleanup_live_runtime_stale_detected"
                    ),
                    "reason": (
                        "current runtime inspection found no stale receiver-status records."
                        if len(live_before["stale_receivers"]) == 0
                        else "current runtime inspection still shows heartbeat_stale receiver-status records; cleanup was not applied in this run."
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
            "state_file": str(runtime.state_dir / "receiver-statuses.json"),
            "stdout_log": str(runtime.stdout_log),
            "stderr_log": str(runtime.stderr_log),
            "cleanup_smoke": cleanup_smoke,
            "live_runtime": live_runtime,
            "checks": checks,
            "conclusion": {
                "status": "live_verified" if overall_ok else "verification_incomplete",
                "category": (
                    "receiver_status_cleanup_live_verified"
                    if overall_ok
                    else "receiver_status_cleanup_verification_incomplete"
                ),
                "reason": (
                    "temp Go runtime stale receiver cleanup smoke passed and current runtime receiver status state was inspected."
                    if overall_ok
                    else "at least one receiver-status cleanup verification check failed"
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
    result = run_receiver_status_cleanup_live_smoke(
        Path(args.repo_root).resolve(),
        runtime_base_url=str(args.runtime_base_url).strip(),
        apply_live_runtime_cleanup=bool(args.apply_live_runtime_cleanup),
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
