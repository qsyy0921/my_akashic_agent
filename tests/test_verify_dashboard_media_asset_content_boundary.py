from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_dashboard_media_asset_content_boundary.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_dashboard_media_asset_content_boundary", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def _sample_runtime_payload():
    detail = {
        "items": [
            {
                "asset_id": "asset:qq:image:3219982:x:1",
                "name": "qq-image.jpg",
                "kind": "image",
                "content_status": "forbidden",
                "content_reason": "media_asset_content_forbidden",
                "content_endpoint": "/v1/media-assets/asset:qq:image:3219982:x:1/content",
                "content_access_plan_endpoint": "/v1/media-assets/content-access-plan?asset_id=asset%3Aqq%3Aimage%3A3219982%3Ax%3A1",
                "content_recovery_plan_endpoint": "/v1/media-assets/content-recovery-plan?asset_id=asset%3Aqq%3Aimage%3A3219982%3Ax%3A1",
                "content_recovery_preflight_endpoint": "/v1/media-assets/content-recovery/preflight?asset_id=asset%3Aqq%3Aimage%3A3219982%3Ax%3A1",
            }
        ],
        "totals": {
            "assets": 1,
            "ready": 0,
            "forbidden": 1,
            "unavailable": 0,
            "disabled": 0,
            "error": 0,
        },
        "notes": ["read-only"],
        "side_effect": "none",
    }
    return {
        "data": {
            "summary": {
                "media_asset_content_assets": 1,
                "media_asset_content_ready": 0,
                "media_asset_content_forbidden": 1,
                "media_asset_content_unavailable": 0,
                "media_asset_content_disabled": 0,
                "media_asset_content_error": 0,
            },
            "media_asset_content_diagnostics": detail,
            "cards": [
                {
                    "id": "media_asset_content",
                    "label": "Media Asset Content",
                    "value": "0/1",
                    "status": "danger",
                    "detail": {"media_asset_content_diagnostics": detail},
                }
            ],
        }
    }


def _sample_preflight_payload():
    return {
        "code": "OK",
        "data": {
            "ready": False,
            "reason": "approval_not_ready",
            "blockers": ["approval_not_found"],
            "target_kind": "media_asset_content",
            "target_id": "asset:qq:image:3219982:x:1",
            "action": "recover_content",
            "operator_id": "qsyy",
            "approval_id": "approval-missing",
            "asset_id": "asset:qq:image:3219982:x:1",
            "recovery_needed": True,
            "executor_scope": "operator_runtime_config",
            "plan": {
                "ready": False,
                "reason": "media_asset_content_recovery_fix_content_roots",
                "runtime_path": "/v1/media-assets/content-recovery-plan?asset_id=asset%3Aqq%3Aimage%3A3219982%3Ax%3A1",
                "dashboard_path": "/api/dashboard/media-assets/content-recovery-plan?asset_id=asset%3Aqq%3Aimage%3A3219982%3Ax%3A1",
                "content_url": "/api/dashboard/media-assets/content?asset_id=asset%3Aqq%3Aimage%3A3219982%3Ax%3A1",
                "side_effect": "none",
            },
            "control_preflight": {
                "ready": False,
                "reason": "approval_not_ready",
                "side_effect": "none",
            },
            "notes": ["read-only"],
            "side_effect": "none",
        },
    }


def test_build_verification_marks_live_parity_when_dashboard_matches_runtime():
    module = _load_module()
    runtime_payload = _sample_runtime_payload()
    dashboard_payload = runtime_payload["data"].copy()
    preflight_payload = _sample_preflight_payload()

    verification = module.build_dashboard_media_asset_content_boundary_verification(
        runtime_overview=runtime_payload,
        dashboard_overview=dashboard_payload,
        runtime_preflight=preflight_payload,
        dashboard_preflight=preflight_payload["data"],
    )

    assert all(verification["checks"].values())
    assert verification["sample_asset_id"] == "asset:qq:image:3219982:x:1"
    assert (
        verification["dashboard_runtime_overview"]["media_asset_content_diagnostics"][
            "items"
        ][0]["content_recovery_preflight_endpoint"]
        == "/v1/media-assets/content-recovery/preflight?asset_id=asset%3Aqq%3Aimage%3A3219982%3Ax%3A1"
    )


def test_build_verification_detects_dashboard_preflight_proxy_drift():
    module = _load_module()
    runtime_payload = _sample_runtime_payload()
    dashboard_payload = runtime_payload["data"].copy()
    runtime_preflight = _sample_preflight_payload()
    dashboard_preflight = {
        **runtime_preflight["data"],
        "approval_id": "approval-other",
    }

    verification = module.build_dashboard_media_asset_content_boundary_verification(
        runtime_overview=runtime_payload,
        dashboard_overview=dashboard_payload,
        runtime_preflight=runtime_preflight,
        dashboard_preflight=dashboard_preflight,
    )

    assert (
        verification["checks"][
            "dashboard_media_asset_content_preflight_proxy_matches_runtime"
        ]
        is False
    )


def test_build_result_marks_status_variants(tmp_path):
    module = _load_module()
    verification = {
        "checks": {"dashboard_media_asset_content_preflight_proxy_matches_runtime": True}
    }

    live_result = module._build_result(
        repo_root=tmp_path,
        runtime_base_url="http://127.0.0.1:8780",
        dashboard_base_url="http://127.0.0.1:2236",
        verification=verification,
    )
    assert live_result["conclusion"]["status"] == "live_verified"
    assert (
        live_result["conclusion"]["category"]
        == "dashboard_media_asset_content_boundary_live_verified"
    )

    failed_result = module._build_result(
        repo_root=tmp_path,
        runtime_base_url="http://127.0.0.1:8780",
        dashboard_base_url="http://127.0.0.1:2236",
        verification={
            "checks": {
                "dashboard_media_asset_content_preflight_proxy_matches_runtime": False
            }
        },
    )
    assert failed_result["conclusion"]["status"] == "verification_failed"
    assert (
        failed_result["conclusion"]["category"]
        == "dashboard_media_asset_content_boundary_verification_failed"
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
        == "dashboard_media_asset_content_boundary_error"
    )
