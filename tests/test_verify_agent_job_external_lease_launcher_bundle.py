from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_agent_job_external_lease_launcher_bundle.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_agent_job_external_lease_launcher_bundle", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_launcher_arguments_from_bundle_uses_bundle_contract(
    tmp_path, monkeypatch
):
    module = _load_module()
    monkeypatch.setattr(module, "_find_powershell_exe", lambda: "powershell.exe")

    args = module._build_launcher_arguments_from_bundle(
        repo_root=tmp_path,
        bundle={
            "script_path": ".\\scripts\\start-agent-runtime.ps1",
            "launcher_parameters": {
                "QueueBackend": "nats_jetstream",
                "QueueMode": "external_lease",
                "QueueExternalLeaseCutover": "true",
                "AgentJobStrictLeaseToken": "true",
            },
            "required_external_inputs": [
                {
                    "name": "NATS DSN",
                    "parameter": "QueueDSN",
                    "environment_key": "AKASHIC_QUEUE_DSN",
                    "required": True,
                    "secret": True,
                }
            ],
        },
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
        str(tmp_path / "scripts" / "start-agent-runtime.ps1"),
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
        "nats_jetstream",
        "-QueueMode",
        "external_lease",
        "-QueueExternalLeaseCutover",
        "true",
        "-AgentJobStrictLeaseToken",
        "true",
        "-QueueDSN",
        "nats://127.0.0.1:14222",
    ):
        assert expected in args


def test_build_result_marks_live_verified_when_all_bundle_checks_pass(tmp_path):
    module = _load_module()

    result = module._build_result(
        repo_root=tmp_path,
        blocked_bundle={"checks": {"bundle_endpoint_reachable": True}},
        promoted_bundle={"checks": {"launcher_bundle_ready_after_worker": True}},
    )

    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "repo_owned_launcher_external_lease_bundle_live_verified"
    )
