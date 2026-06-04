from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_worker_control_executor_boundary.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_worker_control_executor_boundary", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_result_marks_plan_only_live_verified():
    module = _load_module()
    result = module._build_result(
        runtime_base_url="http://127.0.0.1:8780",
        runtime_overview={
            "summary": {
                "agent_job_capacity_ready": True,
                "agent_job_capacity_reason": "agent_job_capacity_ready",
                "agent_job_capacity_blockers": 0,
                "agent_job_capacity_max_pending": 3,
                "agent_job_priority_ready": True,
                "agent_job_priority_reason": "agent_job_priority_ready",
                "agent_job_priority_blockers": 0,
                "agent_job_priority_max_priority_score": 20,
            }
        },
        runtime_workers={
            "totals": {"workers": 2, "enabled": 2, "running": 2},
            "workers": [
                {"name": "outbox_delivery_worker", "kind": "outbox_delivery"},
                {"name": "knowledge_job_planner", "kind": "knowledge_job_planner"},
            ],
        },
        capacity_plan={
            "ready": True,
            "reason": "agent_job_capacity_ready",
            "blockers": [],
            "notes": [
                "read-only capacity plan; no job lease, worker startup, queue acknowledgement, retry, config mutation, or AI execution is performed"
            ],
        },
        priority_plan={
            "ready": True,
            "reason": "agent_job_priority_ready",
            "blockers": [],
            "summary": {"max_priority_score": 20},
            "attributes": {
                "priority_control": "manual_only",
                "worker_control_owner": "python",
            },
            "notes": [
                "read-only priority plan; no job creation, lease, queue acknowledgement, retry, config mutation, worker startup, or AI execution is performed"
            ],
        },
        control_mutation_policy={
            "allowed": True,
            "reason": "control_mutation_policy_listed",
            "intents": [
                {"target_kind": "agent_job_capacity", "actions": ["apply", "rollback"]},
                {"target_kind": "agent_job_priority", "actions": ["apply", "rollback"]},
            ],
            "notes": ["policy query only; no runtime configuration is changed"],
            "side_effect": "none",
        },
    )

    assert all(result["checks"].values())
    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "go_control_plane_live_verified_without_worker_control_executor"
    )


def test_build_result_marks_failure_when_priority_control_drifted():
    module = _load_module()
    result = module._build_result(
        runtime_base_url="http://127.0.0.1:8780",
        runtime_overview={
            "summary": {
                "agent_job_capacity_ready": True,
                "agent_job_capacity_reason": "agent_job_capacity_ready",
                "agent_job_capacity_blockers": 0,
                "agent_job_capacity_max_pending": 3,
                "agent_job_priority_ready": True,
                "agent_job_priority_reason": "agent_job_priority_ready",
                "agent_job_priority_blockers": 0,
                "agent_job_priority_max_priority_score": 20,
            }
        },
        runtime_workers={
            "totals": {"workers": 2, "enabled": 2, "running": 2},
            "workers": [{"name": "priority_executor", "kind": "priority_executor"}],
        },
        capacity_plan={
            "ready": True,
            "reason": "agent_job_capacity_ready",
            "blockers": [],
            "notes": [
                "read-only capacity plan; no job lease, worker startup, queue acknowledgement, retry, config mutation, or AI execution is performed"
            ],
        },
        priority_plan={
            "ready": True,
            "reason": "agent_job_priority_ready",
            "blockers": [],
            "summary": {"max_priority_score": 20},
            "attributes": {
                "priority_control": "automatic",
                "worker_control_owner": "go",
            },
            "notes": [
                "priority executor now mutates worker state"
            ],
        },
        control_mutation_policy={
            "allowed": True,
            "reason": "control_mutation_policy_listed",
            "intents": [
                {"target_kind": "agent_job_capacity", "actions": ["apply", "rollback"]},
                {"target_kind": "agent_job_priority", "actions": ["apply", "rollback"]},
            ],
            "notes": ["policy query only; no runtime configuration is changed"],
            "side_effect": "none",
        },
    )

    assert result["checks"]["priority_plan_is_manual_only"] is False
    assert result["checks"]["priority_plan_execution_owner_is_python"] is False
    assert result["checks"]["no_runtime_worker_control_executor_present"] is False
    assert result["conclusion"]["status"] == "verification_failed"
    assert (
        result["conclusion"]["category"]
        == "worker_control_executor_boundary_mismatch_detected"
    )
