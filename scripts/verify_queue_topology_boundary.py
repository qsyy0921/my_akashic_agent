from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

import httpx

REPO_ROOT = Path(__file__).resolve().parents[1]


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


def _find_work_kind(items: list[dict[str, Any]], work_kind: str) -> dict[str, Any] | None:
    for item in items:
        if str(item.get("work_kind")) == work_kind:
            return item
    return None


def _build_result(
    *,
    runtime_base_url: str,
    queue_backend: dict[str, Any],
    queue_topology: dict[str, Any],
    runtime_overview: dict[str, Any],
) -> dict[str, Any]:
    work_kinds = [
        item for item in queue_topology.get("work_kinds", []) if isinstance(item, dict)
    ]
    outbox_work = _find_work_kind(work_kinds, "outbox_delivery")
    agent_job_work = _find_work_kind(work_kinds, "agent_job")
    summary = runtime_overview.get("summary", {})

    checks = {
        "provider_matches_queue_backend": queue_topology.get("provider")
        == queue_backend.get("provider"),
        "selected_provider_matches_queue_backend": queue_topology.get("selected_provider")
        == queue_backend.get("provider"),
        "recommended_provider_matches_queue_backend": queue_topology.get(
            "recommended_provider"
        )
        == queue_backend.get("recommended_first_backend"),
        "outbox_work_kind_visible": outbox_work is not None,
        "agent_job_work_kind_visible": agent_job_work is not None,
        "outbox_execution_owner_matches_queue_backend": (
            outbox_work is not None
            and outbox_work.get("execution_owner")
            == queue_backend.get("outbox_execution_owner")
        ),
        "agent_job_execution_owner_matches_queue_backend": (
            agent_job_work is not None
            and agent_job_work.get("execution_owner")
            == queue_backend.get("agent_job_execution_owner")
        ),
        "outbox_execution_owner_matches_runtime_overview_summary": (
            outbox_work is not None
            and outbox_work.get("execution_owner")
            == summary.get("queue_topology_outbox_execution_owner")
        ),
        "agent_job_execution_owner_matches_runtime_overview_summary": (
            agent_job_work is not None
            and agent_job_work.get("execution_owner")
            == summary.get("queue_topology_agent_job_execution_owner")
        ),
        "agent_job_ack_owner_matches_runtime_overview_summary": (
            agent_job_work is not None
            and agent_job_work.get("ack_owner")
            == summary.get("queue_topology_agent_job_ack_owner")
        ),
        "external_lease_ready_matches_runtime_overview_summary": (
            queue_topology.get("external_lease_ready")
            == summary.get("queue_topology_external_lease_ready")
        ),
    }
    all_checks_passed = all(bool(value) for value in checks.values())

    return {
        "runtime_base_url": runtime_base_url,
        "queue_backend": {
            "provider": queue_backend.get("provider"),
            "mode": queue_backend.get("mode"),
            "migration_phase": queue_backend.get("migration_phase"),
            "recommended_first_backend": queue_backend.get("recommended_first_backend"),
            "outbox_execution_owner": queue_backend.get("outbox_execution_owner"),
            "outbox_execution_scope": queue_backend.get("outbox_execution_scope"),
            "agent_job_execution_owner": queue_backend.get("agent_job_execution_owner"),
        },
        "queue_topology": {
            "provider": queue_topology.get("provider"),
            "mode": queue_topology.get("mode"),
            "migration_phase": queue_topology.get("migration_phase"),
            "selected_provider": queue_topology.get("selected_provider"),
            "recommended_provider": queue_topology.get("recommended_provider"),
            "external_lease_ready": queue_topology.get("external_lease_ready"),
            "nodes": queue_topology.get("nodes"),
            "edges": queue_topology.get("edges"),
            "work_kinds": work_kinds,
        },
        "runtime_overview_summary": {
            "queue_topology_nodes": summary.get("queue_topology_nodes"),
            "queue_topology_edges": summary.get("queue_topology_edges"),
            "queue_topology_work_kinds": summary.get("queue_topology_work_kinds"),
            "queue_topology_blockers": summary.get("queue_topology_blockers"),
            "queue_topology_external_lease_ready": summary.get(
                "queue_topology_external_lease_ready"
            ),
            "queue_topology_outbox_execution_owner": summary.get(
                "queue_topology_outbox_execution_owner"
            ),
            "queue_topology_agent_job_execution_owner": summary.get(
                "queue_topology_agent_job_execution_owner"
            ),
            "queue_topology_agent_job_ack_owner": summary.get(
                "queue_topology_agent_job_ack_owner"
            ),
        },
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if all_checks_passed else "verification_failed",
            "category": (
                "queue_topology_boundary_live_verified_with_runtime_overview"
                if all_checks_passed
                else "queue_topology_boundary_mismatch_detected"
            ),
            "reason": (
                "queue topology, queue backend, and runtime-overview summary agree on provider recommendation, outbox execution owner, agent_job execution owner, agent_job ack owner, and external-lease readiness."
                if all_checks_passed
                else "at least one queue topology owner or summary parity check failed."
            ),
        },
    }


def run_queue_topology_boundary_verifier(runtime_base_url: str) -> dict[str, Any]:
    queue_backend = _request_json(runtime_base_url, "/v1/queue-backend")
    queue_topology = _request_json(runtime_base_url, "/v1/queue-topology")
    runtime_overview = _request_json(runtime_base_url, "/v1/runtime-overview")
    return _build_result(
        runtime_base_url=runtime_base_url,
        queue_backend=queue_backend,
        queue_topology=queue_topology,
        runtime_overview=runtime_overview,
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--runtime-base-url", default="http://127.0.0.1:8780")
    args = parser.parse_args(argv)
    result = run_queue_topology_boundary_verifier(args.runtime_base_url)
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
