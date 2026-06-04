from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_job_external_lease_nats_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_job_external_lease_nats_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_parse_go_test_json_tracks_selected_tests():
    module = _load_module()
    stdout = """
{"Time":"2026-06-03T01:00:00Z","Action":"run","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck"}
{"Time":"2026-06-03T01:00:01Z","Action":"output","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck","Output":"duplicate ok\\n"}
{"Time":"2026-06-03T01:00:02Z","Action":"pass","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck","Elapsed":2.0}
{"Time":"2026-06-03T01:00:03Z","Action":"run","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow"}
{"Time":"2026-06-03T01:00:04Z","Action":"output","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow","Output":"flow ok\\n"}
{"Time":"2026-06-03T01:00:05Z","Action":"pass","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow","Elapsed":1.5}
{"Time":"2026-06-03T01:00:05Z","Action":"pass","Package":"smoke","Elapsed":3.5}
""".strip()

    parsed = module._parse_go_test_json(stdout)

    assert parsed["package_action"] == "pass"
    assert parsed["tests"][module.SELECTED_TESTS[0]]["action"] == "pass"
    assert parsed["tests"][module.SELECTED_TESTS[1]]["action"] == "pass"
    assert parsed["tests"][module.SELECTED_TESTS[1]]["output_tail"] == ["flow ok"]


def test_build_result_marks_live_verified_when_both_smokes_pass(tmp_path):
    module = _load_module()
    go_test_result = {
        "args": ["go", "test"],
        "cwd": str(tmp_path),
        "returncode": 0,
        "stdout": """
{"Action":"pass","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck","Elapsed":2.0}
{"Action":"pass","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow","Elapsed":1.0}
{"Action":"pass","Package":"smoke","Elapsed":3.0}
""".strip(),
        "stderr": "",
    }

    result = module._build_result(
        repo_root=tmp_path,
        docker_exe="docker",
        go_exe="go",
        port=4222,
        image_pull=None,
        container_name="tmp-nats",
        nats_ready=True,
        go_test_result=go_test_result,
        nats_logs=None,
    )

    assert result["checks"]["duplicate_terminal_ack_passed"] is True
    assert result["checks"]["pending_running_succeeded_flow_passed"] is True
    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "repo_owned_temp_nats_smoke_live_verified"
    )


def test_build_result_marks_failure_when_flow_smoke_fails(tmp_path):
    module = _load_module()
    go_test_result = {
        "args": ["go", "test"],
        "cwd": str(tmp_path),
        "returncode": 1,
        "stdout": """
{"Action":"pass","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck","Elapsed":2.0}
{"Action":"output","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow","Output":"expected running notification to nack while waiting for result\\n"}
{"Action":"fail","Package":"smoke","Test":"TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow","Elapsed":0.1}
{"Action":"fail","Package":"smoke","Elapsed":2.1}
""".strip(),
        "stderr": "",
    }

    result = module._build_result(
        repo_root=tmp_path,
        docker_exe="docker",
        go_exe="go",
        port=4222,
        image_pull=None,
        container_name="tmp-nats",
        nats_ready=True,
        go_test_result=go_test_result,
        nats_logs={"stdout": "nats ready"},
    )

    assert result["checks"]["duplicate_terminal_ack_passed"] is True
    assert result["checks"]["pending_running_succeeded_flow_passed"] is False
    assert result["conclusion"]["status"] == "verification_failed"
    assert (
        result["go_test"]["selected_tests"][1]["output_tail"]
        == ["expected running notification to nack while waiting for result"]
    )
