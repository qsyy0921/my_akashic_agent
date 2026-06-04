from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_dashboard_control_audit_boundary.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_dashboard_control_audit_boundary", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def _sample_runtime_payload():
    return {
        "data": {
            "summary": {
                "operator_approvals_total": 1,
                "operator_approvals_active": 1,
                "control_mutations_total": 2,
                "control_mutations_planned": 1,
                "control_mutation_policy_allowed": True,
                "control_mutation_policy_reason": "control_mutation_policy_listed",
                "control_mutation_policy_targets": 2,
                "control_mutation_policy_actions": 4,
            },
            "operator_approvals": {
                "approvals": [],
                "totals": {
                    "approvals": 1,
                    "active": 1,
                    "approved": 1,
                    "rejected": 0,
                    "revoked": 0,
                },
                "side_effect": "runtime_state_only",
            },
            "control_mutations": {
                "mutations": [],
                "totals": {
                    "mutations": 2,
                    "planned": 1,
                    "applied": 1,
                    "failed": 0,
                    "rolled_back": 0,
                },
                "side_effect": "runtime_state_only",
            },
            "control_mutation_policy": {
                "allowed": True,
                "reason": "control_mutation_policy_listed",
                "side_effect": "none",
                "intents": [
                    {
                        "target_kind": "outbound_cutover",
                        "actions": ["rollback", "enable"],
                    },
                    {
                        "target_kind": "agent_job_capacity",
                        "actions": ["rollback", "apply"],
                    },
                ],
            },
            "cards": [
                {
                    "id": "control_audit",
                    "label": "Control Audit",
                    "value": "1/2",
                    "status": "warn",
                    "detail": {
                        "operator_approvals": {
                            "totals": {
                                "approvals": 1,
                                "active": 1,
                                "approved": 1,
                                "rejected": 0,
                                "revoked": 0,
                            },
                            "side_effect": "runtime_state_only",
                        },
                        "control_mutations": {
                            "totals": {
                                "mutations": 2,
                                "planned": 1,
                                "applied": 1,
                                "failed": 0,
                                "rolled_back": 0,
                            },
                            "side_effect": "runtime_state_only",
                        },
                    },
                },
                {
                    "id": "control_mutation_policy",
                    "label": "Control Mutation Policy",
                    "value": "2/4",
                    "status": "ok",
                    "detail": {
                        "control_mutation_policy": {
                            "allowed": True,
                            "reason": "control_mutation_policy_listed",
                            "side_effect": "none",
                            "intents": [
                                {
                                    "target_kind": "agent_job_capacity",
                                    "actions": ["apply", "rollback"],
                                },
                                {
                                    "target_kind": "outbound_cutover",
                                    "actions": ["enable", "rollback"],
                                },
                            ],
                        }
                    },
                },
            ],
        }
    }


def test_build_verification_marks_live_parity_when_dashboard_matches_runtime():
    module = _load_module()
    runtime_payload = _sample_runtime_payload()
    dashboard_payload = runtime_payload["data"].copy()

    verification = module.build_dashboard_control_audit_boundary_verification(
        runtime_overview=runtime_payload,
        dashboard_overview=dashboard_payload,
    )

    assert all(verification["checks"].values())
    assert (
        verification["dashboard_runtime_overview"]["control_mutation_policy"]["intents"][0][
            "target_kind"
        ]
        == "agent_job_capacity"
    )
    assert (
        verification["dashboard_runtime_overview"]["control_audit_card"]["value"]
        == "1/2"
    )


def test_build_verification_detects_dashboard_policy_drift():
    module = _load_module()
    runtime_payload = _sample_runtime_payload()
    dashboard_payload = runtime_payload["data"].copy()
    dashboard_payload["control_mutation_policy"] = {
        **dashboard_payload["control_mutation_policy"],
        "intents": [
            {
                "target_kind": "outbound_cutover",
                "actions": ["enable"],
            }
        ],
    }

    verification = module.build_dashboard_control_audit_boundary_verification(
        runtime_overview=runtime_payload,
        dashboard_overview=dashboard_payload,
    )

    assert (
        verification["checks"]["dashboard_control_mutation_policy_matches_runtime"]
        is False
    )


def test_build_result_marks_status_variants(tmp_path):
    module = _load_module()
    verification = {"checks": {"dashboard_control_audit_card_matches_runtime": True}}

    live_result = module._build_result(
        repo_root=tmp_path,
        runtime_base_url="http://127.0.0.1:8780",
        dashboard_base_url="http://127.0.0.1:2236",
        verification=verification,
    )
    assert live_result["conclusion"]["status"] == "live_verified"
    assert (
        live_result["conclusion"]["category"]
        == "dashboard_control_audit_boundary_live_verified"
    )

    failed_result = module._build_result(
        repo_root=tmp_path,
        runtime_base_url="http://127.0.0.1:8780",
        dashboard_base_url="http://127.0.0.1:2236",
        verification={"checks": {"dashboard_control_audit_card_matches_runtime": False}},
    )
    assert failed_result["conclusion"]["status"] == "verification_failed"
    assert (
        failed_result["conclusion"]["category"]
        == "dashboard_control_audit_boundary_verification_failed"
    )

    error_result = module._build_result(
        repo_root=tmp_path,
        runtime_base_url="http://127.0.0.1:8780",
        dashboard_base_url="http://127.0.0.1:2236",
        error="dashboard runtime-overview unavailable",
    )
    assert error_result["conclusion"]["status"] == "error"
    assert (
        error_result["conclusion"]["category"]
        == "dashboard_control_audit_boundary_error"
    )
