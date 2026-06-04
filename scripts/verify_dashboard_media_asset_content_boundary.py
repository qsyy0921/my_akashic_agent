from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urljoin
from urllib.request import Request, urlopen


REPO_ROOT = Path(__file__).resolve().parents[1]
_PRECHECK_OPERATOR_ID = "qsyy"
_PRECHECK_APPROVAL_ID = "approval-missing"
_OVERVIEW_QUERY = {
    "limit": "5",
    "event_limit": "5",
    "stale_after_seconds": "300",
}


def _utcnow_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _request_json(
    base_url: str,
    path: str,
    *,
    query: dict[str, str] | None = None,
) -> dict[str, Any]:
    url = urljoin(base_url.rstrip("/") + "/", path.lstrip("/"))
    if query:
        url = f"{url}?{urlencode(query)}"
    request = Request(url)
    try:
        with urlopen(request, timeout=30) as response:
            payload = json.loads(response.read().decode("utf-8"))
    except HTTPError as exc:  # pragma: no cover - exercised via CLI path
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code} for {path}: {body}") from exc
    except URLError as exc:  # pragma: no cover - exercised via CLI path
        raise RuntimeError(f"request failed for {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected non-object response for {path}")
    return payload


def _data(payload: dict[str, Any]) -> dict[str, Any]:
    data = payload.get("data", payload)
    if not isinstance(data, dict):
        raise RuntimeError("payload data must be an object")
    return data


def _mapping(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


def _list(value: Any) -> list[Any]:
    return value if isinstance(value, list) else []


def _string_list(value: Any) -> list[str]:
    return [str(item) for item in _list(value)]


def _find_card(cards: list[Any], card_id: str) -> dict[str, Any]:
    for item in cards:
        if isinstance(item, dict) and str(item.get("id")) == card_id:
            return item
    return {}


def _int_value(value: Any) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return 0


def _normalize_summary(view: dict[str, Any]) -> dict[str, Any]:
    summary = _mapping(view.get("summary"))
    return {
        "media_asset_content_assets": summary.get("media_asset_content_assets"),
        "media_asset_content_ready": summary.get("media_asset_content_ready"),
        "media_asset_content_forbidden": summary.get("media_asset_content_forbidden"),
        "media_asset_content_unavailable": summary.get(
            "media_asset_content_unavailable"
        ),
        "media_asset_content_disabled": summary.get("media_asset_content_disabled"),
        "media_asset_content_error": summary.get("media_asset_content_error"),
    }


def _normalize_media_asset_item(item: Any) -> dict[str, Any]:
    if not isinstance(item, dict):
        return {}
    return {
        "asset_id": item.get("asset_id"),
        "name": item.get("name"),
        "kind": item.get("kind"),
        "content_status": item.get("content_status"),
        "content_reason": item.get("content_reason"),
        "content_endpoint": item.get("content_endpoint"),
        "content_access_plan_endpoint": item.get("content_access_plan_endpoint"),
        "content_recovery_plan_endpoint": item.get("content_recovery_plan_endpoint"),
        "content_recovery_preflight_endpoint": item.get(
            "content_recovery_preflight_endpoint"
        ),
    }


def _normalize_media_asset_content(view: dict[str, Any]) -> dict[str, Any]:
    detail = _mapping(view.get("media_asset_content_diagnostics"))
    totals = _mapping(detail.get("totals"))
    return {
        "totals": {
            "assets": _int_value(totals.get("assets")),
            "ready": _int_value(totals.get("ready")),
            "forbidden": _int_value(totals.get("forbidden")),
            "unavailable": _int_value(totals.get("unavailable")),
            "disabled": _int_value(totals.get("disabled")),
            "error": _int_value(totals.get("error")),
        },
        "items": [
            _normalize_media_asset_item(item)
            for item in _list(detail.get("items"))
            if isinstance(item, dict)
        ],
        "notes": _string_list(detail.get("notes")),
        "side_effect": detail.get("side_effect"),
    }


def _normalize_card(card: dict[str, Any]) -> dict[str, Any]:
    detail = _mapping(card.get("detail"))
    return {
        "id": card.get("id"),
        "label": card.get("label"),
        "value": card.get("value"),
        "status": card.get("status"),
        "media_asset_content_diagnostics": _normalize_media_asset_content(
            {
                "media_asset_content_diagnostics": detail.get(
                    "media_asset_content_diagnostics"
                )
            }
        ),
    }


def _normalize_preflight(view: dict[str, Any]) -> dict[str, Any]:
    plan = _mapping(view.get("plan"))
    control_preflight = _mapping(view.get("control_preflight"))
    return {
        "ready": view.get("ready"),
        "reason": view.get("reason"),
        "blockers": _string_list(view.get("blockers")),
        "target_kind": view.get("target_kind"),
        "target_id": view.get("target_id"),
        "action": view.get("action"),
        "operator_id": view.get("operator_id"),
        "approval_id": view.get("approval_id"),
        "asset_id": view.get("asset_id"),
        "recovery_needed": view.get("recovery_needed"),
        "executor_scope": view.get("executor_scope"),
        "plan": {
            "ready": plan.get("ready"),
            "reason": plan.get("reason"),
            "runtime_path": plan.get("runtime_path"),
            "dashboard_path": plan.get("dashboard_path"),
            "content_url": plan.get("content_url"),
            "side_effect": plan.get("side_effect"),
        },
        "control_preflight": {
            "ready": control_preflight.get("ready"),
            "reason": control_preflight.get("reason"),
            "side_effect": control_preflight.get("side_effect"),
        },
        "notes": _string_list(view.get("notes")),
        "side_effect": view.get("side_effect"),
    }


def _sample_asset(items: list[dict[str, Any]]) -> dict[str, Any]:
    for item in items:
        if item.get("asset_id") and item.get("content_recovery_preflight_endpoint"):
            return item
    return items[0] if items else {}


def build_dashboard_media_asset_content_boundary_verification(
    runtime_overview: dict[str, Any],
    dashboard_overview: dict[str, Any],
    runtime_preflight: dict[str, Any],
    dashboard_preflight: dict[str, Any],
) -> dict[str, Any]:
    runtime_data = _data(runtime_overview)
    dashboard_data = dashboard_overview

    runtime_summary = _normalize_summary(runtime_data)
    dashboard_summary = _normalize_summary(dashboard_data)
    runtime_detail = _normalize_media_asset_content(runtime_data)
    dashboard_detail = _normalize_media_asset_content(dashboard_data)
    runtime_card = _normalize_card(
        _find_card(_list(runtime_data.get("cards")), "media_asset_content")
    )
    dashboard_card = _normalize_card(
        _find_card(_list(dashboard_data.get("cards")), "media_asset_content")
    )
    runtime_sample_asset = _sample_asset(runtime_detail["items"])
    dashboard_sample_asset = _sample_asset(dashboard_detail["items"])
    runtime_preflight_data = _normalize_preflight(_data(runtime_preflight))
    dashboard_preflight_data = _normalize_preflight(_data(dashboard_preflight))

    checks = {
        "runtime_overview_exposes_media_asset_content_card": bool(runtime_card.get("id")),
        "runtime_overview_exposes_media_asset_content_detail": bool(
            runtime_detail["totals"]["assets"] or runtime_detail["items"]
        ),
        "dashboard_overview_exposes_media_asset_content_card": bool(
            dashboard_card.get("id")
        ),
        "dashboard_overview_exposes_media_asset_content_detail": bool(
            dashboard_detail["totals"]["assets"] or dashboard_detail["items"]
        ),
        "dashboard_summary_matches_runtime_media_asset_content": (
            dashboard_summary == runtime_summary
        ),
        "dashboard_media_asset_content_detail_matches_runtime": (
            dashboard_detail == runtime_detail
        ),
        "dashboard_media_asset_content_card_matches_runtime": (
            dashboard_card == runtime_card
        ),
        "dashboard_sample_media_asset_matches_runtime": (
            dashboard_sample_asset == runtime_sample_asset
        ),
        "dashboard_media_asset_content_preflight_proxy_matches_runtime": (
            dashboard_preflight_data == runtime_preflight_data
        ),
        "media_asset_content_boundary_is_read_only": (
            runtime_detail.get("side_effect") == "none"
            and dashboard_detail.get("side_effect") == "none"
            and runtime_preflight_data.get("side_effect") == "none"
            and dashboard_preflight_data.get("side_effect") == "none"
        ),
    }

    return {
        "sample_asset_id": runtime_sample_asset.get("asset_id"),
        "runtime_overview": {
            "summary": runtime_summary,
            "media_asset_content_diagnostics": runtime_detail,
            "media_asset_content_card": runtime_card,
        },
        "dashboard_runtime_overview": {
            "summary": dashboard_summary,
            "media_asset_content_diagnostics": dashboard_detail,
            "media_asset_content_card": dashboard_card,
        },
        "runtime_preflight": runtime_preflight_data,
        "dashboard_preflight": dashboard_preflight_data,
        "checks": checks,
    }


def _build_result(
    *,
    repo_root: Path,
    runtime_base_url: str,
    dashboard_base_url: str,
    verification: dict[str, Any] | None = None,
    error: str | None = None,
) -> dict[str, Any]:
    if error is not None:
        return {
            "generated_at": _utcnow_iso(),
            "repo_root": str(repo_root),
            "runtime_base_url": runtime_base_url,
            "dashboard_base_url": dashboard_base_url,
            "error": error,
            "conclusion": {
                "status": "error",
                "category": "dashboard_media_asset_content_boundary_error",
                "reason": "dashboard media asset content boundary verifier could not collect current-turn runtime or dashboard evidence.",
            },
        }

    assert verification is not None
    all_checks_passed = all(bool(value) for value in verification.get("checks", {}).values())
    return {
        "generated_at": _utcnow_iso(),
        "repo_root": str(repo_root),
        "runtime_base_url": runtime_base_url,
        "dashboard_base_url": dashboard_base_url,
        **verification,
        "conclusion": {
            "status": "live_verified" if all_checks_passed else "verification_failed",
            "category": (
                "dashboard_media_asset_content_boundary_live_verified"
                if all_checks_passed
                else "dashboard_media_asset_content_boundary_verification_failed"
            ),
            "reason": (
                "Dashboard runtime-overview preserves the Go media asset content summary, card, detail, and recovery-preflight proxy boundary without invoking recovery side effects on the current turn."
                if all_checks_passed
                else "At least one media asset content dashboard parity or preflight proxy check failed on the current turn."
            ),
        },
    }


def run_dashboard_media_asset_content_boundary_verifier(
    *,
    repo_root: Path,
    runtime_base_url: str,
    dashboard_base_url: str,
) -> dict[str, Any]:
    try:
        runtime_overview = _request_json(
            runtime_base_url,
            "/v1/runtime-overview",
            query=_OVERVIEW_QUERY,
        )
        dashboard_overview = _request_json(
            dashboard_base_url,
            "/api/dashboard/runtime-overview",
            query=_OVERVIEW_QUERY,
        )
        runtime_data = _data(runtime_overview)
        runtime_detail = _normalize_media_asset_content(runtime_data)
        sample_asset = _sample_asset(runtime_detail["items"])
        sample_asset_id = str(sample_asset.get("asset_id") or "").strip()
        if not sample_asset_id:
            raise RuntimeError("runtime overview did not expose a media asset content sample")
        preflight_query = {
            "asset_id": sample_asset_id,
            "operator_id": _PRECHECK_OPERATOR_ID,
            "approval_id": _PRECHECK_APPROVAL_ID,
        }
        runtime_preflight = _request_json(
            runtime_base_url,
            "/v1/media-assets/content-recovery/preflight",
            query=preflight_query,
        )
        dashboard_preflight = _request_json(
            dashboard_base_url,
            "/api/dashboard/media-assets/content-recovery/preflight",
            query=preflight_query,
        )
        verification = build_dashboard_media_asset_content_boundary_verification(
            runtime_overview=runtime_overview,
            dashboard_overview=dashboard_overview,
            runtime_preflight=runtime_preflight,
            dashboard_preflight=dashboard_preflight,
        )
        return _build_result(
            repo_root=repo_root,
            runtime_base_url=runtime_base_url,
            dashboard_base_url=dashboard_base_url,
            verification=verification,
        )
    except Exception as exc:  # pragma: no cover - exercised via CLI path
        return _build_result(
            repo_root=repo_root,
            runtime_base_url=runtime_base_url,
            dashboard_base_url=dashboard_base_url,
            error=str(exc),
        )


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    parser.add_argument("--runtime-base-url", default="http://127.0.0.1:8780")
    parser.add_argument("--dashboard-base-url", default="http://127.0.0.1:2236")
    args = parser.parse_args()

    result = run_dashboard_media_asset_content_boundary_verifier(
        repo_root=Path(args.repo_root).resolve(),
        runtime_base_url=args.runtime_base_url,
        dashboard_base_url=args.dashboard_base_url,
    )
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
