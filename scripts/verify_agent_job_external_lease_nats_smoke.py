from __future__ import annotations

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import time
import uuid
from contextlib import suppress
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[1]
NATS_IMAGE = "nats:2-alpine"
SELECTED_TESTS = [
    "TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck",
    "TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow",
]
RUN_PATTERN = "TestExternalLeaseNATSSmoke(AgentJobDuplicateTerminalAck|AgentJobPendingRunningSucceededFlow)"


def _utc_timestamp() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _find_go_exe() -> str | None:
    go_exe = shutil.which("go")
    if go_exe:
        return go_exe
    local_app_data = os.environ.get("LOCALAPPDATA", "")
    candidate = Path(local_app_data) / "Programs" / "Go" / "bin" / "go.exe"
    if candidate.exists():
        return str(candidate)
    return None


def _find_docker_exe() -> str | None:
    docker_exe = shutil.which("docker")
    if docker_exe:
        return docker_exe
    docker_cli = Path(os.environ.get("ProgramFiles", r"C:\Program Files")) / "Docker" / "Docker" / "resources" / "bin" / "docker.exe"
    if docker_cli.exists():
        return str(docker_cli)
    return None


def _run_command(
    args: list[str],
    *,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    timeout_seconds: int = 120,
) -> dict[str, Any]:
    completed = subprocess.run(
        args,
        cwd=str(cwd) if cwd is not None else None,
        env=env,
        capture_output=True,
        text=True,
        timeout=timeout_seconds,
        encoding="utf-8",
        errors="replace",
    )
    return {
        "args": args,
        "cwd": str(cwd) if cwd is not None else None,
        "returncode": completed.returncode,
        "stdout": completed.stdout,
        "stderr": completed.stderr,
    }


def _wait_for_port(host: str, port: int, *, timeout_seconds: float = 20.0) -> bool:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        try:
            with socket.create_connection((host, port), timeout=1.0):
                return True
        except OSError:
            time.sleep(0.25)
    return False


def _trim_lines(text: str, *, limit: int = 80) -> list[str]:
    lines = [line.rstrip() for line in text.splitlines() if line.rstrip()]
    if len(lines) <= limit:
        return lines
    return lines[-limit:]


def _parse_go_test_json(stdout: str) -> dict[str, Any]:
    tests: dict[str, dict[str, Any]] = {}
    package_action: str | None = None
    package_elapsed: float | None = None

    for raw_line in stdout.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not isinstance(event, dict):
            continue

        test_name = event.get("Test")
        action = event.get("Action")
        output = event.get("Output")
        elapsed = event.get("Elapsed")

        if isinstance(test_name, str) and test_name:
            item = tests.setdefault(
                test_name,
                {
                    "action": None,
                    "elapsed_seconds": None,
                    "output_tail": [],
                },
            )
            if isinstance(output, str):
                lines = [entry for entry in output.splitlines() if entry.strip()]
                if lines:
                    item["output_tail"].extend(lines)
                    item["output_tail"] = item["output_tail"][-20:]
            if action in {"pass", "fail", "skip"}:
                item["action"] = action
            if isinstance(elapsed, (int, float)):
                item["elapsed_seconds"] = float(elapsed)
            continue

        if event.get("Package") and action in {"pass", "fail"}:
            package_action = str(action)
            if isinstance(elapsed, (int, float)):
                package_elapsed = float(elapsed)

    for name in SELECTED_TESTS:
        tests.setdefault(
            name,
            {
                "action": "missing",
                "elapsed_seconds": None,
                "output_tail": [],
            },
        )

    return {
        "tests": tests,
        "package_action": package_action,
        "package_elapsed_seconds": package_elapsed,
    }


def _build_result(
    *,
    repo_root: Path,
    docker_exe: str | None,
    go_exe: str | None,
    port: int | None,
    image_pull: dict[str, Any] | None,
    container_name: str | None,
    nats_ready: bool,
    go_test_result: dict[str, Any] | None,
    nats_logs: dict[str, Any] | None,
    error: str | None = None,
) -> dict[str, Any]:
    parsed = _parse_go_test_json(go_test_result["stdout"]) if go_test_result else {
        "tests": {},
        "package_action": None,
        "package_elapsed_seconds": None,
    }
    tests = parsed["tests"]
    duplicate_passed = tests.get(SELECTED_TESTS[0], {}).get("action") == "pass"
    flow_passed = tests.get(SELECTED_TESTS[1], {}).get("action") == "pass"
    all_passed = duplicate_passed and flow_passed and (go_test_result or {}).get("returncode") == 0

    if error:
        status = "error"
        category = "repo_owned_temp_nats_smoke_error"
        reason = error
    elif not docker_exe:
        status = "verification_incomplete"
        category = "docker_not_available"
        reason = "docker executable not found; temp NATS smoke was not started"
    elif not go_exe:
        status = "verification_incomplete"
        category = "go_not_available"
        reason = "go executable not found; external lease NATS smoke tests were not run"
    elif not nats_ready:
        status = "verification_incomplete"
        category = "temp_nats_not_ready"
        reason = "temporary NATS container did not become reachable on the selected localhost port"
    elif all_passed:
        status = "live_verified"
        category = "repo_owned_temp_nats_smoke_live_verified"
        reason = "temporary NATS plus repo-owned Go smoke verified duplicate terminal ack and pending/running/succeeded result-ack flow"
    else:
        status = "verification_failed"
        category = "repo_owned_temp_nats_smoke_failed"
        reason = "at least one agent_job external lease result-ack smoke test failed under temporary NATS"

    go_test_summary = None
    if go_test_result:
        go_test_summary = {
            "command": go_test_result["args"],
            "cwd": go_test_result["cwd"],
            "returncode": go_test_result["returncode"],
            "selected_tests": [
                {
                    "name": name,
                    "action": tests.get(name, {}).get("action"),
                    "elapsed_seconds": tests.get(name, {}).get("elapsed_seconds"),
                    "output_tail": tests.get(name, {}).get("output_tail", []),
                }
                for name in SELECTED_TESTS
            ],
            "package_action": parsed["package_action"],
            "package_elapsed_seconds": parsed["package_elapsed_seconds"],
            "stdout_tail": _trim_lines(go_test_result["stdout"]),
            "stderr_tail": _trim_lines(go_test_result["stderr"]),
        }

    result = {
        "generated_at": _utc_timestamp(),
        "repo_root": str(repo_root),
        "smoke_scope": "agent_job_external_lease_result_ack",
        "tools": {
            "docker_exe": docker_exe,
            "go_exe": go_exe,
        },
        "docker": {
            "image": NATS_IMAGE,
            "container_name": container_name,
            "mapped_port": port,
            "image_pull": (
                None
                if image_pull is None
                else {
                    "command": image_pull["args"],
                    "returncode": image_pull["returncode"],
                    "stdout_tail": _trim_lines(image_pull["stdout"], limit=40),
                    "stderr_tail": _trim_lines(image_pull["stderr"], limit=40),
                }
            ),
            "nats_logs_tail": [] if nats_logs is None else _trim_lines(nats_logs.get("stdout", ""), limit=40),
        },
        "go_test": go_test_summary,
        "checks": {
            "docker_available": docker_exe is not None,
            "go_available": go_exe is not None,
            "temp_nats_ready": nats_ready,
            "duplicate_terminal_ack_passed": duplicate_passed,
            "pending_running_succeeded_flow_passed": flow_passed,
        },
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
    }
    if error:
        result["error"] = error
    return result


def run_agent_job_external_lease_nats_smoke(repo_root: Path) -> dict[str, Any]:
    docker_exe = _find_docker_exe()
    go_exe = _find_go_exe()
    port = _find_free_port() if docker_exe else None
    container_name = None
    image_pull = None
    go_test_result = None
    nats_logs = None
    nats_ready = False

    if docker_exe is None or go_exe is None:
        return _build_result(
            repo_root=repo_root,
            docker_exe=docker_exe,
            go_exe=go_exe,
            port=port,
            image_pull=image_pull,
            container_name=container_name,
            nats_ready=nats_ready,
            go_test_result=go_test_result,
            nats_logs=nats_logs,
        )

    try:
        image_pull = _run_command([docker_exe, "pull", NATS_IMAGE], cwd=repo_root, timeout_seconds=180)
        if image_pull["returncode"] != 0:
            return _build_result(
                repo_root=repo_root,
                docker_exe=docker_exe,
                go_exe=go_exe,
                port=port,
                image_pull=image_pull,
                container_name=container_name,
                nats_ready=nats_ready,
                go_test_result=go_test_result,
                nats_logs=nats_logs,
                error="docker pull failed for temporary NATS image",
            )

        container_name = f"akashic-agentjob-nats-smoke-{uuid.uuid4().hex[:8]}"
        run_result = _run_command(
            [
                docker_exe,
                "run",
                "-d",
                "--rm",
                "--name",
                container_name,
                "-p",
                f"127.0.0.1:{port}:4222",
                NATS_IMAGE,
                "-js",
            ],
            cwd=repo_root,
            timeout_seconds=30,
        )
        if run_result["returncode"] != 0:
            return _build_result(
                repo_root=repo_root,
                docker_exe=docker_exe,
                go_exe=go_exe,
                port=port,
                image_pull=image_pull,
                container_name=container_name,
                nats_ready=nats_ready,
                go_test_result=go_test_result,
                nats_logs=nats_logs,
                error="docker run failed for temporary NATS container",
            )

        nats_ready = _wait_for_port("127.0.0.1", int(port))
        if not nats_ready:
            nats_logs = _run_command([docker_exe, "logs", container_name], cwd=repo_root, timeout_seconds=15)
            return _build_result(
                repo_root=repo_root,
                docker_exe=docker_exe,
                go_exe=go_exe,
                port=port,
                image_pull=image_pull,
                container_name=container_name,
                nats_ready=nats_ready,
                go_test_result=go_test_result,
                nats_logs=nats_logs,
            )

        env = os.environ.copy()
        env["AKASHIC_NATS_SMOKE_DSN"] = f"nats://127.0.0.1:{port}"
        go_test_result = _run_command(
            [
                go_exe,
                "test",
                "./smoke",
                "-run",
                RUN_PATTERN,
                "-count=1",
                "-json",
            ],
            cwd=repo_root / "services" / "agent-runtime",
            env=env,
            timeout_seconds=180,
        )
        if go_test_result["returncode"] != 0:
            nats_logs = _run_command([docker_exe, "logs", container_name], cwd=repo_root, timeout_seconds=15)
        return _build_result(
            repo_root=repo_root,
            docker_exe=docker_exe,
            go_exe=go_exe,
            port=port,
            image_pull=image_pull,
            container_name=container_name,
            nats_ready=nats_ready,
            go_test_result=go_test_result,
            nats_logs=nats_logs,
        )
    except subprocess.TimeoutExpired as exc:
        return _build_result(
            repo_root=repo_root,
            docker_exe=docker_exe,
            go_exe=go_exe,
            port=port,
            image_pull=image_pull,
            container_name=container_name,
            nats_ready=nats_ready,
            go_test_result=go_test_result,
            nats_logs=nats_logs,
            error=f"command timed out: {exc.cmd!r}",
        )
    except Exception as exc:  # pragma: no cover - defensive error path
        return _build_result(
            repo_root=repo_root,
            docker_exe=docker_exe,
            go_exe=go_exe,
            port=port,
            image_pull=image_pull,
            container_name=container_name,
            nats_ready=nats_ready,
            go_test_result=go_test_result,
            nats_logs=nats_logs,
            error=str(exc),
        )
    finally:
        if docker_exe and container_name:
            with suppress(Exception):
                subprocess.run(
                    [docker_exe, "rm", "-f", container_name],
                    cwd=str(repo_root),
                    capture_output=True,
                    text=True,
                    timeout=30,
                    encoding="utf-8",
                    errors="replace",
                )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_agent_job_external_lease_nats_smoke(Path(args.repo_root).resolve())
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
