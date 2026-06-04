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


def _normalize_totals(value: Any) -> dict[str, int]:
    totals = _mapping(value)
    fields = (
        "targets",
        "enabled",
        "ready",
        "warning",
        "blocked",
        "lagging",
        "high_pressure",
        "receiver_connected",
        "configured_rag_datasets",
        "configured_rag_dataset_not_started",
        "rag_datasets",
        "rag_dataset_blocked",
        "rag_dataset_warning",
        "rag_dataset_index_ready",
        "rag_dataset_index_missing_snapshot",
        "rag_dataset_index_empty",
        "rag_dataset_index_lagging",
        "rag_dataset_ingest_snapshots",
        "rag_checkpoints",
        "memory_checkpoints",
        "group_memory_pending",
        "rag_ingest_pending",
        "stale_checkpoints",
        "stale_active_leases",
        "expired_active_leases",
        "stalled",
        "stagnant",
        "muted",
    )
    return {field: _int_value(totals.get(field)) for field in fields}


def _normalize_notes(value: Any) -> list[str]:
    return [str(item) for item in _list(value)]


def _normalize_pipeline_view(view: dict[str, Any]) -> dict[str, Any]:
    pipelines = [item for item in _list(view.get("pipelines")) if isinstance(item, dict)]
    return {
        "totals": _normalize_totals(view.get("totals")),
        "notes": _normalize_notes(view.get("notes")),
        "side_effect": str(view.get("side_effect") or ""),
        "targets": [str(item.get("target_id") or "") for item in pipelines],
        "pipeline_count": len(pipelines),
    }


def _boundary_view(view: dict[str, Any]) -> dict[str, Any]:
    totals = _mapping(view.get("totals"))
    return {
        "totals": {
            key: totals.get(key)
            for key in (
                "targets",
                "enabled",
                "ready",
                "warning",
                "blocked",
                "lagging",
                "receiver_connected",
                "configured_rag_datasets",
                "configured_rag_dataset_not_started",
                "rag_datasets",
                "rag_dataset_blocked",
                "rag_dataset_warning",
                "rag_dataset_index_ready",
                "rag_dataset_index_missing_snapshot",
                "rag_dataset_index_empty",
                "rag_dataset_index_lagging",
                "rag_dataset_ingest_snapshots",
                "rag_checkpoints",
                "memory_checkpoints",
            )
        },
        "notes": list(view.get("notes") or []),
        "side_effect": view.get("side_effect"),
        "targets": list(view.get("targets") or []),
        "pipeline_count": view.get("pipeline_count"),
    }


def _expected_card_value(totals: dict[str, int]) -> str:
    return f"{totals['ready']}/{totals['targets']}"


def _expected_card_status(totals: dict[str, int]) -> str:
    if totals["targets"] <= 0:
        return "muted"
    if totals["blocked"] > 0:
        return "danger"
    if totals["warning"] > 0:
        return "warn"
    return "ok"


def _normalize_card(card: dict[str, Any]) -> dict[str, Any]:
    detail = _mapping(card.get("detail"))
    view = _mapping(detail.get("knowledge_pipelines"))
    return {
        "id": card.get("id"),
        "label": card.get("label"),
        "value": card.get("value"),
        "status": card.get("status"),
        "knowledge_pipelines": _normalize_pipeline_view(view),
    }


def build_dashboard_knowledge_rag_state_boundary_verification(
    runtime_knowledge_pipelines: dict[str, Any],
    runtime_overview: dict[str, Any],
    dashboard_overview: dict[str, Any],
) -> dict[str, Any]:
    runtime_direct = _normalize_pipeline_view(_data(runtime_knowledge_pipelines))
    runtime_data = _data(runtime_overview)
    dashboard_data = dashboard_overview

    runtime_card = _normalize_card(
        _find_card(_list(runtime_data.get("cards")), "knowledge_pipelines")
    )
    dashboard_card = _normalize_card(
        _find_card(_list(dashboard_data.get("cards")), "knowledge_pipelines")
    )
    dashboard_top_level = _normalize_pipeline_view(
        _mapping(dashboard_data.get("knowledge_pipelines"))
    )

    expected_value = _expected_card_value(runtime_direct["totals"])
    expected_status = _expected_card_status(runtime_direct["totals"])

    checks = {
        "runtime_endpoint_exposes_knowledge_pipeline_targets": runtime_direct["totals"][
            "targets"
        ]
        > 0,
        "runtime_endpoint_exposes_rag_boundary_fields": all(
            field in runtime_direct["totals"]
            for field in (
                "configured_rag_datasets",
                "rag_datasets",
                "rag_dataset_index_ready",
                "rag_dataset_index_missing_snapshot",
                "rag_dataset_index_empty",
                "rag_dataset_index_lagging",
            )
        ),
        "runtime_overview_exposes_knowledge_pipelines_card": bool(runtime_card.get("id")),
        "runtime_overview_card_matches_direct_knowledge_pipeline_state": (
            runtime_card.get("value") == expected_value
            and runtime_card.get("status") == expected_status
            and _boundary_view(runtime_card.get("knowledge_pipelines") or {})
            == _boundary_view(runtime_direct)
        ),
        "dashboard_overview_exposes_knowledge_pipelines_card": bool(
            dashboard_card.get("id")
        ),
        "dashboard_card_matches_runtime_knowledge_pipeline_state": (
            dashboard_card.get("value") == runtime_card.get("value")
            and dashboard_card.get("status") == runtime_card.get("status")
            and _boundary_view(dashboard_card.get("knowledge_pipelines") or {})
            == _boundary_view(runtime_card.get("knowledge_pipelines") or {})
        ),
        "dashboard_top_level_knowledge_pipelines_matches_runtime": (
            _boundary_view(dashboard_top_level)
            == _boundary_view(runtime_card.get("knowledge_pipelines") or {})
        ),
        "knowledge_pipeline_boundary_is_read_only": (
            runtime_direct.get("side_effect") == "none"
            and "group_level_knowledge_control_plane_view" in runtime_direct.get("notes", [])
        ),
    }

    return {
        "runtime_endpoint": runtime_direct,
        "runtime_overview_card": runtime_card,
        "dashboard_runtime_overview_card": dashboard_card,
        "dashboard_top_level": dashboard_top_level,
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
                "category": "dashboard_knowledge_rag_state_boundary_error",
                "reason": "dashboard knowledge/RAG boundary verifier could not collect current-turn runtime or dashboard evidence.",
            },
        }

    assert verification is not None
    all_checks_passed = all(bool(value) for value in verification.get("checks", {}).values())
    runtime_direct = verification["runtime_endpoint"]
    configured_rag_datasets = runtime_direct["totals"]["configured_rag_datasets"]
    category = (
        "dashboard_knowledge_rag_state_boundary_live_verified_without_configured_datasets"
        if configured_rag_datasets <= 0
        else "dashboard_knowledge_rag_state_boundary_live_verified_checkpoint_snapshot_only"
    )
    return {
        "generated_at": _utcnow_iso(),
        "repo_root": str(repo_root),
        "runtime_base_url": runtime_base_url,
        "dashboard_base_url": dashboard_base_url,
        **verification,
        "conclusion": {
            "status": "live_verified" if all_checks_passed else "verification_failed",
            "category": category if all_checks_passed else "dashboard_knowledge_rag_state_boundary_verification_failed",
            "reason": (
                "Dashboard runtime-overview preserves the Go knowledge-pipeline card and exposes the current checkpoint-derived RAG dataset/index control-plane state."
                if all_checks_passed
                else "At least one knowledge-pipeline dashboard parity check failed on the current turn."
            ),
        },
    }


def run_dashboard_knowledge_rag_state_boundary_verifier(
    *,
    repo_root: Path,
    runtime_base_url: str,
    dashboard_base_url: str,
) -> dict[str, Any]:
    try:
        runtime_knowledge_pipelines = _request_json(
            runtime_base_url,
            "/v1/knowledge-pipeline-diagnostics?limit=200&stale_after_seconds=900",
        )
        runtime_overview = _request_json(runtime_base_url, "/v1/runtime-overview")
        dashboard_overview = _request_json(
            dashboard_base_url, "/api/dashboard/runtime-overview"
        )
        verification = build_dashboard_knowledge_rag_state_boundary_verification(
            runtime_knowledge_pipelines=runtime_knowledge_pipelines,
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

    result = run_dashboard_knowledge_rag_state_boundary_verifier(
        repo_root=Path(args.repo_root).resolve(),
        runtime_base_url=args.runtime_base_url,
        dashboard_base_url=args.dashboard_base_url,
    )
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
