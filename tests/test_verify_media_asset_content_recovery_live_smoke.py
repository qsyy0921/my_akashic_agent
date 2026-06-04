from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_media_asset_content_recovery_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_media_asset_content_recovery_live_smoke", SCRIPT_PATH
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
            "preflight_requires_approval_before_download": True,
            "preflight_ready_after_approval": True,
            "dry_run_does_not_write_cache": True,
            "live_recovery_writes_cache_and_updates_registry": True,
            "content_endpoint_reads_recovered_bytes": True,
            "control_mutation_audit_recorded": True,
            "remote_source_only_downloaded_on_live_recovery": True,
        }
    }

    result = module._build_result(
        repo_root=tmp_path,
        runtime=None,
        source=None,
        verification=verification,
    )

    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "go_http_https_media_recovery_executor_live_verified"
    )


def test_build_result_marks_failure_when_any_check_fails(tmp_path):
    module = _load_module()

    result = module._build_result(
        repo_root=tmp_path,
        runtime=None,
        source=None,
        verification={"checks": {"dry_run_does_not_write_cache": False}},
    )

    assert result["conclusion"]["status"] == "verification_failed"
    assert (
        result["conclusion"]["category"]
        == "media_asset_content_recovery_executor_live_smoke_failed"
    )


def test_build_result_marks_error_when_runtime_setup_raises(tmp_path):
    module = _load_module()

    result = module._build_result(
        repo_root=tmp_path,
        runtime=None,
        source=None,
        verification=None,
        error="go executable not found",
    )

    assert result["conclusion"]["status"] == "error"
    assert (
        result["conclusion"]["category"]
        == "media_asset_content_recovery_executor_live_smoke_error"
    )
    assert result["error"] == "go executable not found"
