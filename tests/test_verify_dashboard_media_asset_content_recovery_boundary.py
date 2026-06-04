from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_dashboard_media_asset_content_recovery_boundary.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_dashboard_media_asset_content_recovery_boundary", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def _sample_runtime_payload():
    detail = {
        "ready": True,
        "reason": "media_asset_content_recovery_audit_ready",
        "recent_audits": [
            {
                "mutation_id": "mutation-media-recovery-applied",
                "target_kind": "media_asset_content",
                "target_id": "asset:missing-a",
                "action": "recover_content",
                "status": "applied",
                "operator_id": "qsyy",
                "approval_id": "approval-recovery-a",
                "created_at": "2026-06-01T12:06:00Z",
            },
            {
                "mutation_id": "mutation-media-recovery-failed",
                "target_kind": "media_asset_content",
                "target_id": "asset:missing-b",
                "action": "recover_content",
                "status": "failed",
                "operator_id": "qsyy",
                "approval_id": "approval-recovery-b",
                "reason": "download_failed",
                "created_at": "2026-06-01T12:07:00Z",
            },
        ],
        "totals": {
            "audits": 2,
            "planned": 0,
            "applied": 1,
            "failed": 1,
            "rolled_back": 0,
        },
        "endpoints": {
            "plan": "/v1/media-assets/content-recovery-plan",
            "preflight": "/v1/media-assets/content-recovery/preflight",
            "recovery": "/v1/media-assets/content-recovery",
        },
        "notes": [
            "read-only runtime overview; does not execute media content recovery"
        ],
        "side_effect": "none",
    }
    return {
        "data": {
            "summary": {
                "media_asset_content_recovery_ready": True,
                "media_asset_content_recovery_reason": "media_asset_content_recovery_audit_ready",
                "media_asset_content_recovery_applied": 1,
                "media_asset_content_recovery_failed": 1,
                "media_asset_content_recovery_recent_audits": 2,
            },
            "media_asset_content_recovery": detail,
            "cards": [
                {
                    "id": "media_asset_content_recovery",
                    "label": "Media Content Recovery",
                    "value": "1/1",
                    "status": "danger",
                    "detail": {
                        "media_asset_content_recovery": detail,
                    },
                }
            ],
        }
    }


def test_build_verification_marks_live_parity_when_dashboard_matches_runtime():
    module = _load_module()
    runtime_payload = _sample_runtime_payload()
    dashboard_payload = runtime_payload["data"].copy()

    verification = (
        module.build_dashboard_media_asset_content_recovery_boundary_verification(
            runtime_overview=runtime_payload,
            dashboard_overview=dashboard_payload,
        )
    )

    assert all(verification["checks"].values())
    assert (
        verification["dashboard_runtime_overview"]["media_asset_content_recovery"][
            "totals"
        ]["failed"]
        == 1
    )
    assert (
        verification["dashboard_runtime_overview"]["media_asset_content_recovery_card"][
            "status"
        ]
        == "danger"
    )


def test_build_verification_detects_dashboard_summary_drift():
    module = _load_module()
    runtime_payload = _sample_runtime_payload()
    dashboard_payload = runtime_payload["data"].copy()
    dashboard_payload["summary"] = {
        **dashboard_payload["summary"],
        "media_asset_content_recovery_failed": 0,
    }

    verification = (
        module.build_dashboard_media_asset_content_recovery_boundary_verification(
            runtime_overview=runtime_payload,
            dashboard_overview=dashboard_payload,
        )
    )

    assert (
        verification["checks"][
            "dashboard_summary_matches_runtime_media_asset_content_recovery"
        ]
        is False
    )


def test_build_result_marks_status_variants(tmp_path):
    module = _load_module()
    verification = {"checks": {"dashboard_media_asset_content_recovery_card_matches_runtime": True}}

    live_result = module._build_result(
        repo_root=tmp_path,
        runtime_base_url="http://127.0.0.1:8780",
        dashboard_base_url="http://127.0.0.1:2236",
        verification=verification,
    )
    assert live_result["conclusion"]["status"] == "live_verified"
    assert (
        live_result["conclusion"]["category"]
        == "dashboard_media_asset_content_recovery_boundary_live_verified"
    )

    failed_result = module._build_result(
        repo_root=tmp_path,
        runtime_base_url="http://127.0.0.1:8780",
        dashboard_base_url="http://127.0.0.1:2236",
        verification={
            "checks": {
                "dashboard_media_asset_content_recovery_card_matches_runtime": False
            }
        },
    )
    assert failed_result["conclusion"]["status"] == "verification_failed"
    assert (
        failed_result["conclusion"]["category"]
        == "dashboard_media_asset_content_recovery_boundary_verification_failed"
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
        == "dashboard_media_asset_content_recovery_boundary_error"
    )
