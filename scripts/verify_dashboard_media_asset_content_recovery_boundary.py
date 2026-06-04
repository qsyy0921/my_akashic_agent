from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urljoin
from urllib.request import Request, urlopen


REPO_ROOT = Path(__file__).resolve().parents[1]


def _utcnow_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _request_json(base_url: str, path: str) -> dict[str, Any]:
    request = Request(urljoin(base_url.rstrip("/") + "/", path.lstrip("/")))
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


def _normalize_audit(item: Any) -> dict[str, Any]:
    if not isinstance(item, dict):
        return {}
    return {
        "mutation_id": item.get("mutation_id"),
        "target_kind": item.get("target_kind"),
        "target_id": item.get("target_id"),
        "action": item.get("action"),
        "status": item.get("status"),
        "operator_id": item.get("operator_id"),
        "approval_id": item.get("approval_id"),
        "reason": item.get("reason"),
        "created_at": item.get("created_at"),
    }


def _normalize_summary(view: dict[str, Any]) -> dict[str, Any]:
    summary = _mapping(view.get("summary"))
    return {
        "media_asset_content_recovery_ready": summary.get(
            "media_asset_content_recovery_ready"
        ),
        "media_asset_content_recovery_reason": summary.get(
            "media_asset_content_recovery_reason"
        ),
        "media_asset_content_recovery_applied": summary.get(
            "media_asset_content_recovery_applied"
        ),
        "media_asset_content_recovery_failed": summary.get(
            "media_asset_content_recovery_failed"
        ),
        "media_asset_content_recovery_recent_audits": summary.get(
            "media_asset_content_recovery_recent_audits"
        ),
    }


def _normalize_media_recovery(view: dict[str, Any]) -> dict[str, Any]:
    detail = _mapping(view.get("media_asset_content_recovery"))
    totals = _mapping(detail.get("totals"))
    endpoints = _mapping(detail.get("endpoints"))
    return {
        "ready": detail.get("ready"),
        "reason": detail.get("reason"),
        "totals": {
            "audits": _int_value(totals.get("audits")),
            "planned": _int_value(totals.get("planned")),
            "applied": _int_value(totals.get("applied")),
            "failed": _int_value(totals.get("failed")),
            "rolled_back": _int_value(totals.get("rolled_back")),
        },
        "recent_audits": [
            _normalize_audit(item)
            for item in _list(detail.get("recent_audits"))
            if isinstance(item, dict)
        ],
        "endpoints": {
            "plan": endpoints.get("plan"),
            "preflight": endpoints.get("preflight"),
            "recovery": endpoints.get("recovery"),
        },
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
        "media_asset_content_recovery": _normalize_media_recovery(
            {"media_asset_content_recovery": detail.get("media_asset_content_recovery")}
        ),
    }


def build_dashboard_media_asset_content_recovery_boundary_verification(
    runtime_overview: dict[str, Any],
    dashboard_overview: dict[str, Any],
) -> dict[str, Any]:
    runtime_data = _data(runtime_overview)
    dashboard_data = dashboard_overview

    runtime_summary = _normalize_summary(runtime_data)
    dashboard_summary = _normalize_summary(dashboard_data)
    runtime_detail = _normalize_media_recovery(runtime_data)
    dashboard_detail = _normalize_media_recovery(dashboard_data)
    runtime_card = _normalize_card(
        _find_card(_list(runtime_data.get("cards")), "media_asset_content_recovery")
    )
    dashboard_card = _normalize_card(
        _find_card(_list(dashboard_data.get("cards")), "media_asset_content_recovery")
    )

    checks = {
        "runtime_overview_exposes_media_asset_content_recovery_card": bool(
            runtime_card.get("id")
        ),
        "runtime_overview_exposes_media_asset_content_recovery_detail": bool(
            runtime_detail.get("reason")
        ),
        "dashboard_overview_exposes_media_asset_content_recovery_card": bool(
            dashboard_card.get("id")
        ),
        "dashboard_overview_exposes_media_asset_content_recovery_detail": bool(
            dashboard_detail.get("reason")
        ),
        "dashboard_summary_matches_runtime_media_asset_content_recovery": (
            dashboard_summary == runtime_summary
        ),
        "dashboard_media_asset_content_recovery_detail_matches_runtime": (
            dashboard_detail == runtime_detail
        ),
        "dashboard_media_asset_content_recovery_card_matches_runtime": (
            dashboard_card == runtime_card
        ),
        "media_asset_content_recovery_boundary_is_read_only": (
            runtime_detail.get("side_effect") == "none"
            and dashboard_detail.get("side_effect") == "none"
            and any("read-only" in note.lower() for note in runtime_detail.get("notes", []))
            and runtime_detail.get("endpoints", {}).get("plan")
            == "/v1/media-assets/content-recovery-plan"
            and runtime_detail.get("endpoints", {}).get("preflight")
            == "/v1/media-assets/content-recovery/preflight"
            and runtime_detail.get("endpoints", {}).get("recovery")
            == "/v1/media-assets/content-recovery"
        ),
    }

    return {
        "runtime_overview": {
            "summary": runtime_summary,
            "media_asset_content_recovery": runtime_detail,
            "media_asset_content_recovery_card": runtime_card,
        },
        "dashboard_runtime_overview": {
            "summary": dashboard_summary,
            "media_asset_content_recovery": dashboard_detail,
            "media_asset_content_recovery_card": dashboard_card,
        },
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
                "category": "dashboard_media_asset_content_recovery_boundary_error",
                "reason": "dashboard media asset content recovery boundary verifier could not collect current-turn runtime or dashboard evidence.",
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
                "dashboard_media_asset_content_recovery_boundary_live_verified"
                if all_checks_passed
                else "dashboard_media_asset_content_recovery_boundary_verification_failed"
            ),
            "reason": (
                "Dashboard runtime-overview preserves the Go media asset content recovery summary, card, and detail without invoking recovery side effects on the current turn."
                if all_checks_passed
                else "At least one media asset content recovery dashboard parity check failed on the current turn."
            ),
        },
    }


def run_dashboard_media_asset_content_recovery_boundary_verifier(
    *,
    repo_root: Path,
    runtime_base_url: str,
    dashboard_base_url: str,
) -> dict[str, Any]:
    try:
        runtime_overview = _request_json(runtime_base_url, "/v1/runtime-overview")
        dashboard_overview = _request_json(
            dashboard_base_url, "/api/dashboard/runtime-overview"
        )
        verification = build_dashboard_media_asset_content_recovery_boundary_verification(
            runtime_overview=runtime_overview,
            dashboard_overview=dashboard_overview,
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

    result = run_dashboard_media_asset_content_recovery_boundary_verifier(
        repo_root=Path(args.repo_root).resolve(),
        runtime_base_url=args.runtime_base_url,
        dashboard_base_url=args.dashboard_base_url,
    )
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
