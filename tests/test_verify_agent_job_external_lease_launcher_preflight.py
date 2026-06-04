from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_job_external_lease_launcher_preflight.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_job_external_lease_launcher_preflight", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_launcher_arguments_include_external_lease_cutover_flags(tmp_path, monkeypatch):
    module = _load_module()
    monkeypatch.setattr(module, "_find_powershell_exe", lambda: "powershell.exe")

    args = module._build_launcher_arguments(
        script_path=tmp_path / "start-agent-runtime.ps1",
        runtime_addr="127.0.0.1:18780",
        runtime_state_dir=tmp_path / "state",
        shadow_audit_path=tmp_path / "shadow.jsonl",
        queue_dsn="nats://127.0.0.1:14222",
    )

    assert args[:6] == [
        "powershell.exe",
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        str(tmp_path / "start-agent-runtime.ps1"),
    ]
    for expected in (
        "-Foreground",
        "-RuntimeAddr",
        "127.0.0.1:18780",
        "-RuntimeStateDir",
        str(tmp_path / "state"),
        "-ShadowAuditPath",
        str(tmp_path / "shadow.jsonl"),
        "-QueueBackend",
        "nats",
        "-QueueMode",
        "external_lease",
        "-QueueDSN",
        "nats://127.0.0.1:14222",
        "-QueueExternalLeaseCutover",
        "true",
        "-QueueDualReadSmokePassed",
        "true",
        "-QueueStateLeaseWorkersDisabled",
        "true",
        "-QueueExternalLeaseAgentJobEnabled",
        "true",
        "-QueueAgentJobDuplicateSmokePassed",
        "true",
        "-QueueAgentJobFlowSmokePassed",
        "true",
        "-AgentJobStrictLeaseToken",
        "true",
        "-OutboxDeliveryAllowedKinds",
        "text",
        "-OutboxDeliveryAllowedKindsByAccount",
        "2365524513=text|file",
        "-OutboxDeliveryAllowedKindsByAccountConversationType",
        "1049511700/private=text|file",
    ):
        assert expected in args


def test_build_result_marks_live_verified_when_all_launcher_checks_pass():
    module = _load_module()

    result = module._build_result(
        Path("E:/agent/my-akashic_agent"),
        verification={
            "checks": {
                "runtime_config_visible": True,
                "queue_backend_promoted": True,
                "preflight_ready": True,
            }
        },
    )

    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "repo_owned_launcher_external_lease_preflight_live_verified"
    )
