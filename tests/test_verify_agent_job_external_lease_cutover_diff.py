from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_job_external_lease_cutover_diff.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_job_external_lease_cutover_diff", SCRIPT_PATH
    )
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_result_marks_live_verified_when_all_checks_pass():
    module = _load_module()
    result = module._build_result(
        repo_root=Path("E:/agent/my-akashic_agent"),
        live_runtime={"checks": {"live_runtime_reports_blocked": True}},
        promoted_runtime={"checks": {"promoted_runtime_cutover_diff_ready_after_worker": True}},
    )
    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "repo_owned_agent_job_external_lease_cutover_diff_live_verified"
    )


def test_build_result_marks_failure_when_any_check_fails():
    module = _load_module()
    result = module._build_result(
        repo_root=Path("E:/agent/my-akashic_agent"),
        live_runtime={"checks": {"live_runtime_reports_blocked": True}},
        promoted_runtime={"checks": {"promoted_runtime_cutover_diff_ready_after_worker": False}},
    )
    assert result["conclusion"]["status"] == "verification_failed"
    assert result["conclusion"]["category"] == "agent_job_external_lease_cutover_diff_failed"
