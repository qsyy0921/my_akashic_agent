from __future__ import annotations

import argparse
import importlib.util
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any
from urllib.parse import quote


REPO_ROOT = Path(__file__).resolve().parents[1]
HELPERS_PATH = Path(__file__).resolve().with_name(
    "verify_agent_worker_status_fencing_live_smoke.py"
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
    HELPERS_PATH,
)


def _utcnow() -> datetime:
    return datetime.now(timezone.utc)


def _request_payload(
    client: Any,
    method: str,
    path: str,
    *,
    body: Any | None = None,
) -> dict[str, Any]:
    response = client.request(method, path, json_body=body)
    response.raise_for_status()
    payload = response.json()
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected non-object response for {method} {path}")
    data = payload.get("data", payload)
    if not isinstance(data, dict):
        raise RuntimeError(f"unexpected non-object data payload for {method} {path}")
    return {
        "status_code": response.status_code,
        "payload": payload,
        "data": data,
    }


def _read_state_items(path: Path, key: str) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected state payload in {path}")
    items = payload.get(key, [])
    if not isinstance(items, list):
        return []
    return [item for item in items if isinstance(item, dict)]


def _find_card(cards: list[dict[str, Any]], card_id: str) -> dict[str, Any] | None:
    for item in cards:
        if str(item.get("id")) == card_id:
            return item
    return None


def _find_intent(intents: list[dict[str, Any]], target_kind: str) -> dict[str, Any] | None:
    for item in intents:
        if str(item.get("target_kind")) == target_kind:
            return item
    return None


def _value(item: dict[str, Any], snake_name: str, title_name: str) -> Any:
    if snake_name in item:
        return item.get(snake_name)
    return item.get(title_name)


def run_control_audit_boundary_smoke(
    client: Any,
    *,
    state_dir: Path,
    now: datetime,
) -> dict[str, Any]:
    approvals_file = state_dir / "operator-approvals.json"
    mutations_file = state_dir / "control-mutations.json"
    approved_time = now
    rejected_time = now + timedelta(minutes=1)
    planned_time = now + timedelta(minutes=2)
    failed_time = now + timedelta(minutes=3)

    approved_response = _request_payload(
        client,
        "POST",
        "/v1/operator-approvals",
        body={
            "target_kind": "outbound_cutover",
            "target_id": "cutover-a",
            "decision": "approved",
            "operator_id": "qsyy",
            "timestamp": approved_time.isoformat(),
            "expires_at": (approved_time + timedelta(hours=4)).isoformat(),
            "metadata": {"source": "verify_control_audit_boundary"},
        },
    )
    approved_approval = approved_response["data"]
    approved_id = str(approved_approval.get("approval_id"))

    rejected_response = _request_payload(
        client,
        "POST",
        "/v1/operator-approvals",
        body={
            "target_kind": "outbound_cutover",
            "target_id": "cutover-b",
            "decision": "rejected",
            "operator_id": "qsyy",
            "timestamp": rejected_time.isoformat(),
            "reason": "manual rejection for verification",
            "metadata": {"source": "verify_control_audit_boundary"},
        },
    )
    rejected_approval = rejected_response["data"]
    rejected_id = str(rejected_approval.get("approval_id"))

    approvals_list = _request_payload(
        client,
        "GET",
        "/v1/operator-approvals?target_kind=outbound_cutover&limit=10",
    )
    approval_check = _request_payload(
        client,
        "GET",
        (
            "/v1/operator-approvals/check?"
            f"approval_id={quote(approved_id)}&target_kind=outbound_cutover&target_id=cutover-a"
        ),
    )
    rejected_check = _request_payload(
        client,
        "GET",
        (
            "/v1/operator-approvals/check?"
            f"approval_id={quote(rejected_id)}&target_kind=outbound_cutover&target_id=cutover-b"
        ),
    )
    missing_check = _request_payload(
        client,
        "GET",
        "/v1/operator-approvals/check?target_kind=outbound_cutover&target_id=missing",
    )

    preflight_ready = _request_payload(
        client,
        "GET",
        (
            "/v1/control-mutations/preflight?"
            "target_kind=outbound_cutover&target_id=cutover-a&action=enable&operator_id=qsyy"
            f"&approval_id={quote(approved_id)}"
        ),
    )
    preflight_missing = _request_payload(
        client,
        "GET",
        (
            "/v1/control-mutations/preflight?"
            "target_kind=outbound_cutover&target_id=cutover-a&action=enable&operator_id=qsyy"
            "&approval_id=missing"
        ),
    )
    preflight_unsupported_action = _request_payload(
        client,
        "GET",
        (
            "/v1/control-mutations/preflight?"
            "target_kind=agent_job_capacity&target_id=capacity-a&action=enable&operator_id=qsyy"
            f"&approval_id={quote(approved_id)}"
        ),
    )

    policy_all = _request_payload(client, "GET", "/v1/control-mutations/policy")
    policy_filtered = _request_payload(
        client,
        "GET",
        "/v1/control-mutations/policy?target_kind=model_provider_config",
    )

    planned_mutation = _request_payload(
        client,
        "POST",
        "/v1/control-mutations",
        body={
            "target_kind": "outbound_cutover",
            "target_id": "cutover-a",
            "action": "enable",
            "status": "planned",
            "operator_id": "qsyy",
            "approval_id": approved_id,
            "timestamp": planned_time.isoformat(),
            "metadata": {"source": "verify_control_audit_boundary"},
        },
    )
    failed_mutation = _request_payload(
        client,
        "POST",
        "/v1/control-mutations",
        body={
            "target_kind": "outbound_cutover",
            "target_id": "cutover-a",
            "action": "enable",
            "status": "failed",
            "operator_id": "qsyy",
            "approval_id": approved_id,
            "timestamp": failed_time.isoformat(),
            "reason": "simulated verification failure",
            "metadata": {"source": "verify_control_audit_boundary"},
        },
    )
    mutations_list = _request_payload(
        client,
        "GET",
        f"/v1/control-mutations?target_kind=outbound_cutover&approval_id={quote(approved_id)}&limit=10",
    )

    runtime_overview = _request_payload(client, "GET", "/v1/runtime-overview")

    approvals_state = _read_state_items(approvals_file, "approvals")
    mutations_state = _read_state_items(mutations_file, "audits")

    policy_view = policy_all["data"]
    policy_intents = [
        item for item in list(policy_view.get("intents") or []) if isinstance(item, dict)
    ]
    outbound_intent = _find_intent(policy_intents, "outbound_cutover")
    capacity_intent = _find_intent(policy_intents, "agent_job_capacity")

    overview_data = runtime_overview["data"]
    summary = overview_data.get("summary") or {}
    cards = [
        item for item in list(overview_data.get("cards") or []) if isinstance(item, dict)
    ]
    policy_card = _find_card(cards, "control_mutation_policy")
    audit_card = _find_card(cards, "control_audit")
    overview_approvals = overview_data.get("operator_approvals") or {}
    overview_mutations = overview_data.get("control_mutations") or {}
    overview_policy = overview_data.get("control_mutation_policy") or {}

    checks = {
        "policy_lists_allowlist": (
            policy_all["status_code"] == 200
            and policy_view.get("allowed") is True
            and policy_view.get("reason") == "control_mutation_policy_listed"
            and policy_view.get("side_effect") == "none"
            and outbound_intent is not None
            and sorted(str(item) for item in outbound_intent.get("actions", []))
            == ["enable", "rollback"]
            and capacity_intent is not None
            and sorted(str(item) for item in capacity_intent.get("actions", []))
            == ["apply", "rollback"]
        ),
        "policy_filtered_blocks_unsupported_target": (
            policy_filtered["status_code"] == 200
            and policy_filtered["data"].get("allowed") is False
            and policy_filtered["data"].get("reason")
            == "unsupported_control_mutation_target"
            and "unsupported_control_mutation_target"
            in list(policy_filtered["data"].get("blockers") or [])
            and policy_filtered["data"].get("side_effect") == "none"
        ),
        "operator_approvals_recorded_with_expected_status": (
            approved_response["status_code"] == 202
            and rejected_response["status_code"] == 202
            and approved_approval.get("decision") == "approved"
            and approved_approval.get("active") is True
            and rejected_approval.get("decision") == "rejected"
            and rejected_approval.get("active") is False
        ),
        "operator_approvals_list_totals_match": (
            approvals_list["status_code"] == 200
            and approvals_list["data"].get("totals", {}).get("approvals") == 2
            and approvals_list["data"].get("totals", {}).get("active") == 1
            and approvals_list["data"].get("totals", {}).get("approved") == 1
            and approvals_list["data"].get("totals", {}).get("rejected") == 1
            and approvals_list["data"].get("side_effect") == "runtime_state_only"
        ),
        "approval_check_active_approval_ready": (
            approval_check["status_code"] == 200
            and approval_check["data"].get("approved") is True
            and approval_check["data"].get("reason") == "approval_active"
            and approval_check["data"].get("side_effect") == "none"
        ),
        "approval_check_rejected_blocks": (
            rejected_check["status_code"] == 200
            and rejected_check["data"].get("approved") is False
            and rejected_check["data"].get("reason") == "approval_not_approved"
            and "approval_not_approved"
            in list(rejected_check["data"].get("blockers") or [])
            and rejected_check["data"].get("side_effect") == "none"
        ),
        "approval_check_missing_blocks": (
            missing_check["status_code"] == 200
            and missing_check["data"].get("approved") is False
            and missing_check["data"].get("reason") == "approval_not_found"
            and "approval_not_found"
            in list(missing_check["data"].get("blockers") or [])
            and missing_check["data"].get("side_effect") == "none"
        ),
        "preflight_ready_returns_planned_audit": (
            preflight_ready["status_code"] == 200
            and preflight_ready["data"].get("ready") is True
            and preflight_ready["data"].get("reason") == "approval_active"
            and preflight_ready["data"].get("suggested_audit", {}).get("status")
            == "planned"
            and preflight_ready["data"].get("side_effect") == "none"
        ),
        "preflight_missing_blocks": (
            preflight_missing["status_code"] == 200
            and preflight_missing["data"].get("ready") is False
            and preflight_missing["data"].get("reason") == "approval_not_ready"
            and "approval_not_found"
            in list(preflight_missing["data"].get("blockers") or [])
            and preflight_missing["data"].get("side_effect") == "none"
        ),
        "preflight_unsupported_action_blocks": (
            preflight_unsupported_action["status_code"] == 200
            and preflight_unsupported_action["data"].get("ready") is False
            and preflight_unsupported_action["data"].get("reason")
            == "unsupported_control_mutation_action"
            and "unsupported_control_mutation_action"
            in list(preflight_unsupported_action["data"].get("blockers") or [])
            and sorted(
                str(item)
                for item in list(
                    preflight_unsupported_action["data"].get("supported_actions") or []
                )
            )
            == ["apply", "rollback"]
            and preflight_unsupported_action["data"].get("side_effect") == "none"
        ),
        "control_mutations_recorded_with_expected_status": (
            planned_mutation["status_code"] == 202
            and failed_mutation["status_code"] == 202
            and planned_mutation["data"].get("status") == "planned"
            and failed_mutation["data"].get("status") == "failed"
        ),
        "control_mutations_list_totals_match": (
            mutations_list["status_code"] == 200
            and mutations_list["data"].get("totals", {}).get("mutations") == 2
            and mutations_list["data"].get("totals", {}).get("planned") == 1
            and mutations_list["data"].get("totals", {}).get("failed") == 1
            and mutations_list["data"].get("side_effect") == "runtime_state_only"
        ),
        "approval_store_persisted_records": (
            len(approvals_state) == 2
            and any(str(_value(item, "approval_id", "ApprovalID")) == approved_id for item in approvals_state)
            and any(str(_value(item, "approval_id", "ApprovalID")) == rejected_id for item in approvals_state)
        ),
        "control_mutation_store_persisted_records": (
            len(mutations_state) == 2
            and any(str(_value(item, "status", "Status")) == "planned" for item in mutations_state)
            and any(str(_value(item, "status", "Status")) == "failed" for item in mutations_state)
        ),
        "runtime_overview_summary_matches_control_audit": (
            summary.get("control_mutation_policy_allowed") is True
            and summary.get("control_mutation_policy_reason")
            == "control_mutation_policy_listed"
            and summary.get("control_mutation_policy_targets") == len(policy_intents)
            and summary.get("control_mutation_policy_actions")
            == sum(len(list(item.get("actions") or [])) for item in policy_intents)
            and summary.get("operator_approvals_total") == 2
            and summary.get("operator_approvals_active") == 1
            and summary.get("operator_approvals_approved") == 1
            and summary.get("operator_approvals_rejected") == 1
            and summary.get("control_mutations_total") == 2
            and summary.get("control_mutations_planned") == 1
            and summary.get("control_mutations_failed") == 1
        ),
        "runtime_overview_cards_match_control_audit": (
            policy_card is not None
            and str(policy_card.get("value")) == f"{len(policy_intents)}/{sum(len(list(item.get('actions') or [])) for item in policy_intents)}"
            and str(policy_card.get("status")) == "ok"
            and audit_card is not None
            and str(audit_card.get("value")) == "1/2"
            and str(audit_card.get("status")) == "warn"
        ),
        "runtime_overview_detail_matches_control_audit": (
            overview_policy.get("allowed") is True
            and overview_policy.get("reason") == "control_mutation_policy_listed"
            and overview_policy.get("side_effect") == "none"
            and overview_approvals.get("totals", {}).get("approvals") == 2
            and overview_approvals.get("totals", {}).get("active") == 1
            and overview_approvals.get("side_effect") == "runtime_state_only"
            and overview_mutations.get("totals", {}).get("mutations") == 2
            and overview_mutations.get("totals", {}).get("failed") == 1
            and overview_mutations.get("side_effect") == "runtime_state_only"
        ),
    }
    overall_ok = all(bool(value) for value in checks.values())

    return {
        "state_dir": str(state_dir),
        "operator_approvals_file": str(approvals_file),
        "control_mutations_file": str(mutations_file),
        "policy": {
            "all": policy_all,
            "filtered": policy_filtered,
        },
        "operator_approvals": {
            "approved_record": approved_response,
            "rejected_record": rejected_response,
            "list": approvals_list,
            "active_check": approval_check,
            "rejected_check": rejected_check,
            "missing_check": missing_check,
            "persisted": approvals_state,
        },
        "control_mutations": {
            "preflight_ready": preflight_ready,
            "preflight_missing": preflight_missing,
            "preflight_unsupported_action": preflight_unsupported_action,
            "planned_record": planned_mutation,
            "failed_record": failed_mutation,
            "list": mutations_list,
            "persisted": mutations_state,
        },
        "runtime_overview": runtime_overview,
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if overall_ok else "verification_incomplete",
            "category": (
                "control_audit_boundary_live_verified"
                if overall_ok
                else "control_audit_boundary_verification_incomplete"
            ),
            "reason": (
                "temp Go runtime persisted operator approvals and control mutations, enforced approval-bound preflight, and kept runtime-overview policy/audit summaries aligned with the ledger state."
                if overall_ok
                else "at least one control-audit boundary check failed"
            ),
        },
    }


def _build_result(
    *,
    repo_root: Path,
    verification: dict[str, Any] | None,
    error: str | None = None,
) -> dict[str, Any]:
    checks = dict((verification or {}).get("checks") or {})
    if error:
        status = "error"
        category = "control_audit_boundary_error"
        reason = error
    elif checks and all(bool(value) for value in checks.values()):
        status = "live_verified"
        category = "control_audit_boundary_live_verified"
        reason = (
            "temp runtime verified operator approval ledger, control mutation audit ledger, "
            "approval-bound preflight, policy allowlist, and runtime-overview control-audit "
            "summaries without introducing executor side effects."
        )
    else:
        status = "verification_failed"
        category = "control_audit_boundary_verification_failed"
        reason = "at least one control-audit boundary check failed"
    return {
        "generated_at": _utcnow().isoformat(),
        "repo_root": str(repo_root),
        "verification_scope": "control_audit_boundary",
        "verification": verification,
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
        **({"error": error} if error else {}),
    }


def run_control_audit_boundary_verifier(repo_root: Path) -> dict[str, Any]:
    runtime = None
    try:
        runtime = HELPERS.start_temp_runtime(repo_root)
        client = HELPERS.WorkerStatusRuntimeClient(runtime.base_url)
        verification = run_control_audit_boundary_smoke(
            client,
            state_dir=runtime.state_dir,
            now=_utcnow(),
        )
        result = _build_result(repo_root=repo_root, verification=verification)
        result["base_url"] = runtime.base_url
        result["stdout_log"] = str(runtime.stdout_log)
        result["stderr_log"] = str(runtime.stderr_log)
        return result
    except Exception as exc:  # pragma: no cover - defensive live path
        result = _build_result(repo_root=repo_root, verification=None, error=str(exc))
        if runtime is not None:
            result["base_url"] = runtime.base_url
            result["stdout_log"] = str(runtime.stdout_log)
            result["stderr_log"] = str(runtime.stderr_log)
        return result
    finally:
        if runtime is not None:
            runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_control_audit_boundary_verifier(Path(args.repo_root).resolve())
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
