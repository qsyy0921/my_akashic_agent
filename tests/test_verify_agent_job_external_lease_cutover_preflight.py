from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_job_external_lease_cutover_preflight.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_job_external_lease_cutover_preflight", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_result_marks_live_verified_when_both_scenarios_pass(tmp_path):
    module = _load_module()
    config_blocked = {
        "checks": {
            "readiness_blocked": True,
            "configuration_blocker_present": True,
            "strict_token_blocker_present": True,
            "queue_provider_is_local": True,
            "current_owner_is_state_store": True,
            "approval_bound_preflight_blocks_on_plan_first": True,
        }
    }
    preflight_ready = {
        "checks": {
            "queue_backend_nats_selected": True,
            "external_lease_base_ready": True,
            "agent_job_owner_promoted_to_nats_result_ack": True,
            "queue_topology_ack_owner_promoted": True,
            "queue_topology_execution_owner_promoted": True,
            "worker_coverage_blocks_without_worker": True,
            "worker_status_report_accepted": True,
            "readiness_ready_after_worker": True,
            "plan_ready_after_worker": True,
            "approval_bound_preflight_requires_approval": True,
            "approval_bound_preflight_ready_after_approval": True,
            "runtime_overview_summary_ready_after_worker": True,
        }
    }

    result = module._build_result(
        repo_root=tmp_path,
        config_blocked=config_blocked,
        preflight_ready=preflight_ready,
    )

    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "repo_owned_temp_runtime_cutover_preflight_live_verified"
    )


def test_build_result_marks_failure_when_preflight_check_fails(tmp_path):
    module = _load_module()
    config_blocked = {
        "checks": {
            "readiness_blocked": True,
            "configuration_blocker_present": True,
            "strict_token_blocker_present": True,
            "queue_provider_is_local": True,
            "current_owner_is_state_store": True,
            "approval_bound_preflight_blocks_on_plan_first": True,
        }
    }
    preflight_ready = {
        "checks": {
            "queue_backend_nats_selected": True,
            "external_lease_base_ready": True,
            "agent_job_owner_promoted_to_nats_result_ack": False,
            "approval_bound_preflight_requires_approval": True,
            "approval_bound_preflight_ready_after_approval": True,
        }
    }

    result = module._build_result(
        repo_root=tmp_path,
        config_blocked=config_blocked,
        preflight_ready=preflight_ready,
    )

    assert result["conclusion"]["status"] == "verification_failed"
    assert (
        result["conclusion"]["category"]
        == "agent_job_external_lease_cutover_preflight_failed"
    )


def test_build_result_marks_error_when_runtime_setup_raises(tmp_path):
    module = _load_module()
    result = module._build_result(
        repo_root=tmp_path,
        config_blocked=None,
        preflight_ready=None,
        error="docker executable not found",
    )

    assert result["conclusion"]["status"] == "error"
    assert (
        result["conclusion"]["category"]
        == "agent_job_external_lease_cutover_preflight_error"
    )
    assert result["error"] == "docker executable not found"
