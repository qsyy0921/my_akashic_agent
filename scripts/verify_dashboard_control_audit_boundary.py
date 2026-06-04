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
    except HTTPError as exc:  # pragma: no cover - exercised via wrapper error path
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code} for {path}: {body}") from exc
    except URLError as exc:  # pragma: no cover - exercised via wrapper error path
        raise RuntimeError(f"request failed for {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected non-object response for {path}")
    return payload


def _runtime_data(runtime_overview: dict[str, Any]) -> dict[str, Any]:
    data = runtime_overview.get("data", runtime_overview)
    if not isinstance(data, dict):
        raise RuntimeError("runtime overview data payload must be an object")
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


def _normalize_policy(policy: dict[str, Any]) -> dict[str, Any]:
    intents: list[dict[str, Any]] = []
    for item in _list(policy.get("intents")):
        if not isinstance(item, dict):
            continue
        target_kind = str(item.get("target_kind", ""))
        actions = sorted(str(action) for action in _list(item.get("actions")))
        intents.append({"target_kind": target_kind, "actions": actions})
    intents.sort(key=lambda item: item["target_kind"])
    return {
        "allowed": policy.get("allowed"),
        "reason": policy.get("reason"),
        "side_effect": policy.get("side_effect"),
        "intents": intents,
    }


def _normalize_approval_totals(detail: dict[str, Any]) -> dict[str, Any]:
    totals = _mapping(detail.get("totals"))
    return {
        "approvals": totals.get("approvals"),
        "active": totals.get("active"),
        "approved": totals.get("approved"),
        "rejected": totals.get("rejected"),
        "revoked": totals.get("revoked"),
        "side_effect": detail.get("side_effect"),
    }


def _normalize_mutation_totals(detail: dict[str, Any]) -> dict[str, Any]:
    totals = _mapping(detail.get("totals"))
    return {
        "mutations": totals.get("mutations"),
        "planned": totals.get("planned"),
        "applied": totals.get("applied"),
        "failed": totals.get("failed"),
        "rolled_back": totals.get("rolled_back"),
        "side_effect": detail.get("side_effect"),
    }


def _normalize_control_audit_card(card: dict[str, Any]) -> dict[str, Any]:
    detail = _mapping(card.get("detail"))
    approvals = _normalize_approval_totals(_mapping(detail.get("operator_approvals")))
    mutations = _normalize_mutation_totals(_mapping(detail.get("control_mutations")))
    return {
        "id": card.get("id"),
        "label": card.get("label"),
        "value": card.get("value"),
        "status": card.get("status"),
        "operator_approvals": approvals,
        "control_mutations": mutations,
    }


def _normalize_control_audit_detail(runtime_or_dashboard: dict[str, Any]) -> dict[str, Any]:
    return {
        "operator_approvals": _normalize_approval_totals(
            _mapping(runtime_or_dashboard.get("operator_approvals"))
        ),
        "control_mutations": _normalize_mutation_totals(
            _mapping(runtime_or_dashboard.get("control_mutations"))
        ),
    }


def _normalize_summary(runtime_or_dashboard: dict[str, Any]) -> dict[str, Any]:
    summary = _mapping(runtime_or_dashboard.get("summary"))
    return {
        "operator_approvals_total": summary.get("operator_approvals_total"),
        "operator_approvals_active": summary.get("operator_approvals_active"),
        "control_mutations_total": summary.get("control_mutations_total"),
        "control_mutations_planned": summary.get("control_mutations_planned"),
        "control_mutation_policy_allowed": summary.get("control_mutation_policy_allowed"),
        "control_mutation_policy_reason": summary.get("control_mutation_policy_reason"),
        "control_mutation_policy_targets": summary.get("control_mutation_policy_targets"),
        "control_mutation_policy_actions": summary.get("control_mutation_policy_actions"),
    }


def build_dashboard_control_audit_boundary_verification(
    runtime_overview: dict[str, Any],
    dashboard_overview: dict[str, Any],
) -> dict[str, Any]:
    runtime_data = _runtime_data(runtime_overview)
    dashboard_data = dashboard_overview

    runtime_cards = _list(runtime_data.get("cards"))
    dashboard_cards = _list(dashboard_data.get("cards"))

    runtime_control_audit_card = _normalize_control_audit_card(
        _find_card(runtime_cards, "control_audit")
    )
    dashboard_control_audit_card = _normalize_control_audit_card(
        _find_card(dashboard_cards, "control_audit")
    )
    runtime_policy_card = _mapping(_find_card(runtime_cards, "control_mutation_policy"))
    dashboard_policy_card = _mapping(_find_card(dashboard_cards, "control_mutation_policy"))

    runtime_summary = _normalize_summary(runtime_data)
    dashboard_summary = _normalize_summary(dashboard_data)
    runtime_policy = _normalize_policy(_mapping(runtime_data.get("control_mutation_policy")))
    dashboard_policy = _normalize_policy(_mapping(dashboard_data.get("control_mutation_policy")))
    runtime_audit_detail = _normalize_control_audit_detail(runtime_data)
    dashboard_audit_detail = _normalize_control_audit_detail(dashboard_data)

    checks = {
        "runtime_overview_exposes_control_audit": bool(runtime_control_audit_card.get("id")),
        "runtime_overview_exposes_control_mutation_policy": bool(
            runtime_policy_card.get("id")
        )
        and bool(runtime_policy["intents"]),
        "dashboard_overview_exposes_control_audit": bool(
            dashboard_control_audit_card.get("id")
        ),
        "dashboard_overview_exposes_control_mutation_policy": bool(
            dashboard_policy_card.get("id")
        )
        and bool(dashboard_policy["intents"]),
        "dashboard_summary_matches_runtime_control_audit": (
            dashboard_summary == runtime_summary
        ),
        "dashboard_control_audit_detail_matches_runtime": (
            dashboard_audit_detail == runtime_audit_detail
        ),
        "dashboard_control_audit_card_matches_runtime": (
            dashboard_control_audit_card == runtime_control_audit_card
        ),
        "dashboard_control_mutation_policy_matches_runtime": (
            dashboard_policy == runtime_policy
        ),
        "dashboard_control_mutation_policy_card_matches_runtime": (
            dashboard_policy_card.get("value") == runtime_policy_card.get("value")
            and dashboard_policy_card.get("status") == runtime_policy_card.get("status")
            and dashboard_policy_card.get("label") == runtime_policy_card.get("label")
        ),
    }

    return {
        "runtime_overview": {
            "summary": runtime_summary,
            "control_audit_card": runtime_control_audit_card,
            "control_audit_detail": runtime_audit_detail,
            "control_mutation_policy": runtime_policy,
            "control_mutation_policy_card": {
                "id": runtime_policy_card.get("id"),
                "label": runtime_policy_card.get("label"),
                "value": runtime_policy_card.get("value"),
                "status": runtime_policy_card.get("status"),
            },
        },
        "dashboard_runtime_overview": {
            "summary": dashboard_summary,
            "control_audit_card": dashboard_control_audit_card,
            "control_audit_detail": dashboard_audit_detail,
            "control_mutation_policy": dashboard_policy,
            "control_mutation_policy_card": {
                "id": dashboard_policy_card.get("id"),
                "label": dashboard_policy_card.get("label"),
                "value": dashboard_policy_card.get("value"),
                "status": dashboard_policy_card.get("status"),
            },
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
                "category": "dashboard_control_audit_boundary_error",
                "reason": "dashboard control-audit boundary verifier could not collect current-turn runtime or dashboard evidence.",
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
                "dashboard_control_audit_boundary_live_verified"
                if all_checks_passed
                else "dashboard_control_audit_boundary_verification_failed"
            ),
            "reason": (
                "Dashboard runtime-overview preserves control-audit and control-mutation-policy summaries, cards, and detail parity with the Go runtime overview on the current turn."
                if all_checks_passed
                else "At least one control-audit or control-mutation-policy dashboard parity check failed on the current turn."
            ),
        },
    }


def run_dashboard_control_audit_boundary_verifier(
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
        verification = build_dashboard_control_audit_boundary_verification(
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

    result = run_dashboard_control_audit_boundary_verifier(
        repo_root=Path(args.repo_root).resolve(),
        runtime_base_url=args.runtime_base_url,
        dashboard_base_url=args.dashboard_base_url,
    )
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
