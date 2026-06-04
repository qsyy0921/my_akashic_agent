from __future__ import annotations

import argparse
import json
import sys
from typing import Any

import httpx


def _request_json(base_url: str, path: str) -> dict[str, Any]:
    with httpx.Client(timeout=30.0, trust_env=True) as client:
        response = client.get(base_url.rstrip("/") + path)
    response.raise_for_status()
    payload = response.json()
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected non-object response for {path}")
    data = payload.get("data", payload)
    if not isinstance(data, dict):
        raise RuntimeError(f"unexpected non-object data payload for {path}")
    return data


def _find_intent(intents: list[dict[str, Any]], target_kind: str) -> dict[str, Any] | None:
    for item in intents:
        if str(item.get("target_kind")) == target_kind:
            return item
    return None


def _contains_all(texts: list[str], expected_fragments: list[str]) -> bool:
    haystack = " ".join(texts).lower()
    return all(fragment.lower() in haystack for fragment in expected_fragments)


def _build_result(
    *,
    runtime_base_url: str,
    runtime_overview: dict[str, Any],
    runtime_workers: dict[str, Any],
    capacity_plan: dict[str, Any],
    priority_plan: dict[str, Any],
    control_mutation_policy: dict[str, Any],
) -> dict[str, Any]:
    summary = runtime_overview.get("summary", {})
    workers = [
        item for item in runtime_workers.get("workers", []) if isinstance(item, dict)
    ]
    intents = [
        item for item in control_mutation_policy.get("intents", []) if isinstance(item, dict)
    ]
    capacity_intent = _find_intent(intents, "agent_job_capacity")
    priority_intent = _find_intent(intents, "agent_job_priority")

    plan_only_worker_candidates = [
        item
        for item in workers
        if any(
            token in str(item.get("name", "")).lower()
            or token in str(item.get("kind", "")).lower()
            for token in ("autoscal", "priority", "concurrency", "capacity")
        )
    ]

    capacity_notes = [str(item) for item in capacity_plan.get("notes", [])]
    priority_notes = [str(item) for item in priority_plan.get("notes", [])]

    checks = {
        "capacity_plan_ready_matches_summary": capacity_plan.get("ready")
        == summary.get("agent_job_capacity_ready"),
        "capacity_plan_reason_matches_summary": capacity_plan.get("reason")
        == summary.get("agent_job_capacity_reason"),
        "capacity_plan_blockers_match_summary": len(capacity_plan.get("blockers", []))
        == summary.get("agent_job_capacity_blockers"),
        "priority_plan_ready_matches_summary": priority_plan.get("ready")
        == summary.get("agent_job_priority_ready"),
        "priority_plan_reason_matches_summary": priority_plan.get("reason")
        == summary.get("agent_job_priority_reason"),
        "priority_plan_blockers_match_summary": len(priority_plan.get("blockers", []))
        == summary.get("agent_job_priority_blockers"),
        "priority_plan_max_score_matches_summary": priority_plan.get("summary", {}).get(
            "max_priority_score"
        )
        == summary.get("agent_job_priority_max_priority_score"),
        "capacity_plan_is_read_only": _contains_all(
            capacity_notes,
            [
                "read-only capacity plan",
                "no job lease",
                "ai execution",
            ],
        ),
        "priority_plan_is_manual_only": str(
            priority_plan.get("attributes", {}).get("priority_control", "")
        ).lower()
        == "manual_only",
        "priority_plan_execution_owner_is_python": str(
            priority_plan.get("attributes", {}).get("worker_control_owner", "")
        ).lower()
        == "python",
        "capacity_intent_present_in_policy": capacity_intent is not None,
        "priority_intent_present_in_policy": priority_intent is not None,
        "capacity_intent_actions_allow_apply_and_rollback": capacity_intent is not None
        and sorted(str(item) for item in capacity_intent.get("actions", []))
        == ["apply", "rollback"],
        "priority_intent_actions_allow_apply_and_rollback": priority_intent is not None
        and sorted(str(item) for item in priority_intent.get("actions", []))
        == ["apply", "rollback"],
        "no_runtime_worker_control_executor_present": len(plan_only_worker_candidates) == 0,
    }
    all_checks_passed = all(bool(value) for value in checks.values())
    executor_present = len(plan_only_worker_candidates) > 0

    if all_checks_passed and not executor_present:
        status = "live_verified"
        category = "go_control_plane_live_verified_without_worker_control_executor"
        reason = (
            "Agent-job capacity and priority plans are live, read-only, and policy-bound; "
            "runtime workers still do not expose an autoscaling, concurrency, or priority executor."
        )
    elif all_checks_passed:
        status = "live_verified"
        category = "go_worker_control_executor_present"
        reason = (
            "Agent-job capacity and priority plans are live, and runtime workers now advertise "
            "a worker-control executor that should be reviewed before claiming plan-only status."
        )
    else:
        status = "verification_failed"
        category = "worker_control_executor_boundary_mismatch_detected"
        reason = (
            "At least one capacity/priority/policy/runtime-worker parity check failed, so the "
            "worker-control executor boundary is not proven for the current turn."
        )

    return {
        "runtime_base_url": runtime_base_url,
        "runtime_overview_summary": {
            "agent_job_capacity_ready": summary.get("agent_job_capacity_ready"),
            "agent_job_capacity_reason": summary.get("agent_job_capacity_reason"),
            "agent_job_capacity_blockers": summary.get("agent_job_capacity_blockers"),
            "agent_job_capacity_max_pending": summary.get("agent_job_capacity_max_pending"),
            "agent_job_priority_ready": summary.get("agent_job_priority_ready"),
            "agent_job_priority_reason": summary.get("agent_job_priority_reason"),
            "agent_job_priority_blockers": summary.get("agent_job_priority_blockers"),
            "agent_job_priority_max_priority_score": summary.get(
                "agent_job_priority_max_priority_score"
            ),
        },
        "capacity_plan": capacity_plan,
        "priority_plan": priority_plan,
        "control_mutation_policy": {
            "allowed": control_mutation_policy.get("allowed"),
            "reason": control_mutation_policy.get("reason"),
            "intents": intents,
            "notes": control_mutation_policy.get("notes"),
            "side_effect": control_mutation_policy.get("side_effect"),
        },
        "runtime_workers": {
            "totals": runtime_workers.get("totals"),
            "worker_control_executor_candidates": plan_only_worker_candidates,
            "workers": workers,
        },
        "checks": checks,
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
    }


def run_worker_control_executor_boundary_verifier(runtime_base_url: str) -> dict[str, Any]:
    runtime_overview = _request_json(runtime_base_url, "/v1/runtime-overview")
    runtime_workers = _request_json(runtime_base_url, "/v1/runtime-workers")
    capacity_plan = _request_json(runtime_base_url, "/v1/agent-job-capacity/plan")
    priority_plan = _request_json(runtime_base_url, "/v1/agent-job-priority/plan")
    control_mutation_policy = _request_json(runtime_base_url, "/v1/control-mutations/policy")
    return _build_result(
        runtime_base_url=runtime_base_url,
        runtime_overview=runtime_overview,
        runtime_workers=runtime_workers,
        capacity_plan=capacity_plan,
        priority_plan=priority_plan,
        control_mutation_policy=control_mutation_policy,
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--runtime-base-url", default="http://127.0.0.1:8780")
    args = parser.parse_args(argv)
    result = run_worker_control_executor_boundary_verifier(args.runtime_base_url)
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
