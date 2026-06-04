from __future__ import annotations

import importlib.util
import json
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_proactive_runtime_flow_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_proactive_runtime_flow_live_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_read_state_counts_handles_missing_file(tmp_path):
    module = _load_module()

    counts = module._read_state_counts(tmp_path / "missing.json")

    assert counts["deliveries"] == 0
    assert counts["anyaction_quotas"] == 0
    assert counts["tick_steps"] == 0


def test_read_state_counts_summarizes_expected_keys(tmp_path):
    module = _load_module()
    state_file = tmp_path / "proactive-state.json"
    state_file.write_text(
        json.dumps(
            {
                "deliveries": [{"session_key": "telegram:1"}],
                "seen_items": [{"item_id": "item-a"}],
                "rejection_cooldowns": [{"item_id": "item-b"}],
                "context_only": [{"session_key": "telegram:1"}],
                "session_marks": [{"key": "drift_last_at"}],
                "global_marks": [{"key": "bg_context_last_main_at"}],
                "anyaction_quotas": [{"quota_key": "default"}],
                "drift_skills": [{"skill_name": "explore-curiosity"}],
                "drift_recent_runs": [{"skill_name": "explore-curiosity"}],
                "tick_logs": [{"tick_id": "tick-1"}],
                "tick_steps": [{"tick_id": "tick-1", "step_index": 1}],
            }
        ),
        encoding="utf-8",
    )

    counts = module._read_state_counts(state_file)

    assert counts == {
        "deliveries": 1,
        "seen_items": 1,
        "rejection_cooldowns": 1,
        "context_only": 1,
        "session_marks": 1,
        "global_marks": 1,
        "anyaction_quotas": 1,
        "drift_skills": 1,
        "drift_recent_runs": 1,
        "tick_logs": 1,
        "tick_steps": 1,
    }


def test_build_result_marks_live_verified_when_all_checks_pass():
    module = _load_module()

    result = module._build_result(
        base_url="http://127.0.0.1:9999",
        state_dir=Path("/tmp/proactive-state"),
        state_file=Path("/tmp/proactive-state/proactive-state.json"),
        stdout_log=Path("/tmp/proactive-state/stdout.log"),
        stderr_log=Path("/tmp/proactive-state/stderr.log"),
        delivery_seen_cleanup_smoke={"checks": {"a": True}},
        anyaction_drift_bg_context_smoke={"checks": {"b": True}},
        tick_log_smoke={"checks": {"c": True}},
    )

    assert result["conclusion"]["status"] == "live_verified"
    assert result["conclusion"]["category"] == "go_proactive_state_flow_live_verified"
    assert result["checks"] == {"a": True, "b": True, "c": True}


def test_build_result_marks_incomplete_when_any_check_fails():
    module = _load_module()

    result = module._build_result(
        base_url="http://127.0.0.1:9999",
        state_dir=Path("/tmp/proactive-state"),
        state_file=Path("/tmp/proactive-state/proactive-state.json"),
        stdout_log=Path("/tmp/proactive-state/stdout.log"),
        stderr_log=Path("/tmp/proactive-state/stderr.log"),
        delivery_seen_cleanup_smoke={"checks": {"a": True}},
        anyaction_drift_bg_context_smoke={"checks": {"b": False}},
        tick_log_smoke={"checks": {"c": True}},
    )

    assert result["conclusion"]["status"] == "verification_incomplete"
    assert result["conclusion"]["category"] == "proactive_state_flow_verification_incomplete"
    assert result["checks"]["b"] is False
