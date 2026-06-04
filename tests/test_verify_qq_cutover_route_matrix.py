from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_qq_cutover_route_matrix.py"
)
GOAL_VERIFIER_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify-go-migration-goal.ps1"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_qq_cutover_route_matrix", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_result_marks_live_verified_for_current_partial_cutover():
    module = _load_module()
    result = module._build_result(
        runtime_base_url="http://127.0.0.1:8780",
        runtime_config={
            "runtime": {"bot_ids": ["1049511700", "2365524513"]},
            "delivery": {
                "qq_group_send_enabled": False,
                "telegram_token_configured": False,
            },
        },
        queue_backend={
            "provider": "local",
            "mode": "local_state_store",
            "outbox_execution_owner": "go_local_outbox_worker",
            "outbox_execution_scope": "account_conversation_kind_gated",
            "recommended_first_backend": "nats_jetstream",
            "outbox_allowed_kinds": ["text"],
            "outbox_allowed_kinds_by_account": {"2365524513": ["text", "file"]},
            "outbox_allowed_kinds_by_account_conversation_type": {
                "1049511700": {"private": ["text", "file"]}
            },
        },
        outbound_cutover_readiness={
            "ready": True,
            "execution_ready": True,
            "execution_owner": "go_local_outbox_worker",
            "reason": "outbound_cutover_ready",
            "blockers": [],
        },
        queue_topology={
            "work_kinds": [
                {
                    "work_kind": "outbox_delivery",
                    "execution_owner": "go_local_outbox_worker",
                    "ack_owner": "go_state_store_api",
                }
            ]
        },
        runtime_overview={
            "summary": {
                "queue_outbox_execution_owner": "go_local_outbox_worker",
                "outbound_cutover_plan_current_owner": "go_local_outbox_worker",
                "outbound_cutover_plan_ready": True,
            }
        },
        primary_account_id="1049511700",
        secondary_account_id="2365524513",
    )

    assert all(result["checks"].values())
    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "qq_partial_go_cutover_route_matrix_live_verified"
    )
    assert {
        (item["account_id"], item["conversation_type"], item["kind"])
        for item in result["route_matrix"]["currently_sendable_routes"]
    } == {
        ("*", "private", "text"),
        ("2365524513", "private", "text"),
        ("2365524513", "private", "file"),
        ("1049511700", "private", "text"),
        ("1049511700", "private", "file"),
    }


def test_build_result_marks_failure_when_first_account_group_file_enters_scope():
    module = _load_module()
    result = module._build_result(
        runtime_base_url="http://127.0.0.1:8780",
        runtime_config={
            "runtime": {"bot_ids": ["1049511700", "2365524513"]},
            "delivery": {
                "qq_group_send_enabled": False,
                "telegram_token_configured": False,
            },
        },
        queue_backend={
            "provider": "local",
            "mode": "local_state_store",
            "outbox_execution_owner": "go_local_outbox_worker",
            "outbox_execution_scope": "account_conversation_kind_gated",
            "recommended_first_backend": "nats_jetstream",
            "outbox_allowed_kinds": ["text"],
            "outbox_allowed_kinds_by_account": {
                "1049511700": ["text", "file"],
                "2365524513": ["text", "file"],
            },
            "outbox_allowed_kinds_by_account_conversation_type": {
                "1049511700": {"private": ["text", "file"]}
            },
        },
        outbound_cutover_readiness={
            "ready": True,
            "execution_ready": True,
            "execution_owner": "go_local_outbox_worker",
            "reason": "outbound_cutover_ready",
            "blockers": [],
        },
        queue_topology={
            "work_kinds": [
                {
                    "work_kind": "outbox_delivery",
                    "execution_owner": "go_local_outbox_worker",
                    "ack_owner": "go_state_store_api",
                }
            ]
        },
        runtime_overview={
            "summary": {
                "queue_outbox_execution_owner": "go_local_outbox_worker",
                "outbound_cutover_plan_current_owner": "go_local_outbox_worker",
                "outbound_cutover_plan_ready": True,
            }
        },
        primary_account_id="1049511700",
        secondary_account_id="2365524513",
    )

    assert result["checks"]["first_account_group_file_not_in_go_scope"] is False
    assert result["conclusion"]["status"] == "verification_failed"
    assert (
        result["conclusion"]["category"]
        == "qq_cutover_route_matrix_mismatch_detected"
    )


def test_goal_verifier_wires_qq_cutover_route_matrix():
    script_text = GOAL_VERIFIER_PATH.read_text(encoding="utf-8")

    assert "verify-qq-cutover-route-matrix.ps1" in script_text
    assert "qqCutoverRouteMatrixVerification" in script_text
    assert "qq_cutover_route_matrix = $qqCutoverRouteMatrixVerification" in script_text
    assert "route_matrix = $qqCutoverRouteMatrixVerification.route_matrix" in script_text
