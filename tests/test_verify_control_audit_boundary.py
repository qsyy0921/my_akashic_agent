from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_control_audit_boundary.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_control_audit_boundary", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_result_marks_live_verified_when_all_checks_pass(tmp_path):
    module = _load_module()
    verification = {
        "checks": {
            "policy_lists_allowlist": True,
            "policy_filtered_blocks_unsupported_target": True,
            "operator_approvals_recorded_with_expected_status": True,
            "operator_approvals_list_totals_match": True,
            "approval_check_active_approval_ready": True,
            "approval_check_rejected_blocks": True,
            "approval_check_missing_blocks": True,
            "preflight_ready_returns_planned_audit": True,
            "preflight_missing_blocks": True,
            "preflight_unsupported_action_blocks": True,
            "control_mutations_recorded_with_expected_status": True,
            "control_mutations_list_totals_match": True,
            "approval_store_persisted_records": True,
            "control_mutation_store_persisted_records": True,
            "runtime_overview_summary_matches_control_audit": True,
            "runtime_overview_cards_match_control_audit": True,
            "runtime_overview_detail_matches_control_audit": True,
        }
    }

    result = module._build_result(repo_root=tmp_path, verification=verification)

    assert result["conclusion"]["status"] == "live_verified"
    assert result["conclusion"]["category"] == "control_audit_boundary_live_verified"


def test_build_result_marks_failure_when_any_check_fails(tmp_path):
    module = _load_module()
    verification = {
        "checks": {
            "policy_lists_allowlist": True,
            "operator_approvals_recorded_with_expected_status": False,
        }
    }

    result = module._build_result(repo_root=tmp_path, verification=verification)

    assert result["conclusion"]["status"] == "verification_failed"
    assert (
        result["conclusion"]["category"]
        == "control_audit_boundary_verification_failed"
    )


def test_build_result_marks_error_when_runtime_setup_fails(tmp_path):
    module = _load_module()

    result = module._build_result(
        repo_root=tmp_path,
        verification=None,
        error="temp agent-runtime did not become healthy",
    )

    assert result["conclusion"]["status"] == "error"
    assert result["conclusion"]["category"] == "control_audit_boundary_error"
    assert result["error"] == "temp agent-runtime did not become healthy"
