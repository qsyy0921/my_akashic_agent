from __future__ import annotations

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time
from contextlib import suppress
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import httpx


REPO_ROOT = Path(__file__).resolve().parents[1]


def _find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _find_go_exe() -> str:
    go_exe = shutil.which("go")
    if go_exe:
        return go_exe
    local_app_data = os.environ.get("LOCALAPPDATA", "")
    candidate = Path(local_app_data) / "Programs" / "Go" / "bin" / "go.exe"
    if candidate.exists():
        return str(candidate)
    raise RuntimeError("go executable not found in PATH or LOCALAPPDATA Programs\\Go\\bin")


def _read_state_counts(state_file: Path) -> dict[str, int]:
    if not state_file.exists():
        return {
            "deliveries": 0,
            "seen_items": 0,
            "rejection_cooldowns": 0,
            "context_only": 0,
            "session_marks": 0,
            "global_marks": 0,
            "anyaction_quotas": 0,
            "drift_skills": 0,
            "drift_recent_runs": 0,
            "tick_logs": 0,
            "tick_steps": 0,
        }
    payload = json.loads(state_file.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected proactive state payload in {state_file}")
    return {
        "deliveries": len(payload.get("deliveries", []) or []),
        "seen_items": len(payload.get("seen_items", []) or []),
        "rejection_cooldowns": len(payload.get("rejection_cooldowns", []) or []),
        "context_only": len(payload.get("context_only", []) or []),
        "session_marks": len(payload.get("session_marks", []) or []),
        "global_marks": len(payload.get("global_marks", []) or []),
        "anyaction_quotas": len(payload.get("anyaction_quotas", []) or []),
        "drift_skills": len(payload.get("drift_skills", []) or []),
        "drift_recent_runs": len(payload.get("drift_recent_runs", []) or []),
        "tick_logs": len(payload.get("tick_logs", []) or []),
        "tick_steps": len(payload.get("tick_steps", []) or []),
    }


def _build_result(
    *,
    base_url: str,
    state_dir: Path,
    state_file: Path,
    stdout_log: Path,
    stderr_log: Path,
    delivery_seen_cleanup_smoke: dict[str, Any],
    anyaction_drift_bg_context_smoke: dict[str, Any],
    tick_log_smoke: dict[str, Any],
) -> dict[str, Any]:
    checks: dict[str, bool] = {}
    for group in (
        delivery_seen_cleanup_smoke.get("checks", {}),
        anyaction_drift_bg_context_smoke.get("checks", {}),
        tick_log_smoke.get("checks", {}),
    ):
        for key, value in group.items():
            checks[key] = bool(value)

    overall_ok = all(checks.values())
    return {
        "base_url": base_url,
        "state_dir": str(state_dir),
        "state_file": str(state_file),
        "stdout_log": str(stdout_log),
        "stderr_log": str(stderr_log),
        "delivery_seen_cleanup_smoke": delivery_seen_cleanup_smoke,
        "anyaction_drift_bg_context_smoke": anyaction_drift_bg_context_smoke,
        "tick_log_smoke": tick_log_smoke,
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if overall_ok else "verification_incomplete",
            "category": (
                "go_proactive_state_flow_live_verified"
                if overall_ok
                else "proactive_state_flow_verification_incomplete"
            ),
            "reason": (
                "temp Go runtime verified proactive deliveries, anyaction quota, seen/rejection cleanup, context-only, drift, bg-context, and tick-log state transitions without triggering platform sends or Python AI."
                if overall_ok
                else "at least one proactive state-flow live smoke check failed"
            ),
        },
    }


@dataclass
class TempRuntimeHandle:
    process: subprocess.Popen[str]
    base_url: str
    state_dir: Path
    state_file: Path
    stdout_log: Path
    stderr_log: Path

    def stop(self) -> None:
        if self.process.poll() is None:
            with suppress(Exception):
                self.process.terminate()
            try:
                self.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                with suppress(Exception):
                    self.process.kill()
                with suppress(Exception):
                    self.process.wait(timeout=5)


class ProactiveRuntimeClient:
    def __init__(self, base_url: str, *, timeout: float = 30.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def request_json(
        self,
        method: str,
        path: str,
        *,
        json_body: Any | None = None,
        params: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        with httpx.Client(timeout=self.timeout, trust_env=True) as client:
            response = client.request(
                method.upper(),
                self.base_url + path,
                json=json_body,
                params=params,
                headers={"Content-Type": "application/json"},
            )
        response.raise_for_status()
        payload = response.json()
        if not isinstance(payload, dict):
            raise RuntimeError(f"unexpected non-object response for {method} {path}")
        code = payload.get("code")
        if code not in (None, "OK", 0):
            raise RuntimeError(str(payload.get("message") or payload))
        data = payload.get("data", payload)
        if isinstance(data, dict):
            return data
        return {"items": data}


def run_delivery_seen_cleanup_smoke(
    client: ProactiveRuntimeClient, *, state_file: Path
) -> dict[str, Any]:
    client.request_json(
        "POST",
        "/v1/proactive/deliveries",
        json_body={
            "session_key": "telegram:1",
            "delivery_key": "delivery-a",
            "timestamp": "2026-05-30T10:00:00Z",
        },
    )
    duplicate = client.request_json(
        "GET",
        "/v1/proactive/deliveries/duplicate",
        params={
            "session_key": "telegram:1",
            "delivery_key": "delivery-a",
            "window_hours": 24,
            "timestamp": "2026-05-30T11:00:00Z",
        },
    )
    delivery_count = client.request_json(
        "GET",
        "/v1/proactive/deliveries/count",
        params={
            "session_key": "telegram:1",
            "window_hours": 24,
            "timestamp": "2026-05-30T11:00:00Z",
        },
    )
    client.request_json(
        "POST",
        "/v1/proactive/seen-items",
        json_body={
            "entries": [{"source_key": "mcp:news:feed", "item_id": "item-a"}],
            "timestamp": "2026-05-30T10:00:00Z",
        },
    )
    seen = client.request_json(
        "GET",
        "/v1/proactive/seen-items/seen",
        params={
            "source_key": "mcp:news:other",
            "item_id": "item-a",
            "ttl_hours": 24,
            "timestamp": "2026-05-30T11:00:00Z",
        },
    )
    client.request_json(
        "POST",
        "/v1/proactive/rejection-cooldowns",
        json_body={
            "entries": [{"source_key": "qq:group:1", "item_id": "item-b"}],
            "hours": 2,
            "timestamp": "2026-05-30T10:00:00Z",
        },
    )
    cooled = client.request_json(
        "GET",
        "/v1/proactive/rejection-cooldowns/cooled",
        params={
            "source_key": "qq:group:1",
            "item_id": "item-b",
            "ttl_hours": 2,
            "timestamp": "2026-05-30T11:00:00Z",
        },
    )
    client.request_json(
        "POST",
        "/v1/proactive/context-only",
        json_body={
            "session_key": "telegram:1",
            "timestamp": "2026-05-30T11:30:00Z",
        },
    )
    context_last = client.request_json(
        "GET",
        "/v1/proactive/context-only/last",
        params={"session_key": "telegram:1"},
    )
    context_count = client.request_json(
        "GET",
        "/v1/proactive/context-only/count",
        params={
            "session_key": "telegram:1",
            "window_hours": 24,
            "timestamp": "2026-05-30T12:00:00Z",
        },
    )
    state_counts_before_cleanup = _read_state_counts(state_file)
    cleanup = client.request_json(
        "POST",
        "/v1/proactive/cleanup",
        json_body={
            "seen_ttl_hours": 1,
            "delivery_ttl_hours": 1,
            "context_only_ttl_hours": 1,
            "rejection_cooldown_ttl_hours": 1,
            "timestamp": "2026-05-30T12:31:00Z",
        },
    )
    state_counts_after_cleanup = _read_state_counts(state_file)

    checks = {
        "delivery_duplicate_detected": duplicate.get("duplicate") is True,
        "delivery_count_visible": int(delivery_count.get("count") or 0) == 1,
        "seen_item_normalized_hit_visible": (
            seen.get("seen") is True
            and str(seen.get("source_key")) == "mcp:news"
            and str(seen.get("item_id")) == "item-a"
        ),
        "rejection_cooldown_visible": cooled.get("cooled") is True,
        "context_only_last_visible": context_last.get("found") is True,
        "context_only_count_visible": int(context_count.get("count") or 0) == 1,
        "cleanup_removed_expected_records": (
            int(cleanup.get("removed_deliveries") or 0) == 1
            and int(cleanup.get("removed_seen_items") or 0) == 1
            and int(cleanup.get("removed_context_only") or 0) == 1
            and int(cleanup.get("removed_rejection_cooldowns") or 0) == 1
        ),
        "state_file_persisted_before_cleanup": (
            state_counts_before_cleanup["deliveries"] == 1
            and state_counts_before_cleanup["seen_items"] == 1
            and state_counts_before_cleanup["rejection_cooldowns"] == 1
            and state_counts_before_cleanup["context_only"] == 1
        ),
        "state_file_cleanup_removed_expected_records": (
            state_counts_after_cleanup["deliveries"] == 0
            and state_counts_after_cleanup["seen_items"] == 0
            and state_counts_after_cleanup["rejection_cooldowns"] == 0
            and state_counts_after_cleanup["context_only"] == 0
        ),
    }

    return {
        "duplicate": duplicate,
        "delivery_count": delivery_count,
        "seen": seen,
        "cooled": cooled,
        "context_last": context_last,
        "context_count": context_count,
        "cleanup": cleanup,
        "state_counts_before_cleanup": state_counts_before_cleanup,
        "state_counts_after_cleanup": state_counts_after_cleanup,
        "checks": checks,
    }


def run_anyaction_drift_bg_context_smoke(
    client: ProactiveRuntimeClient, *, state_file: Path
) -> dict[str, Any]:
    quota_before = client.request_json(
        "GET",
        "/v1/proactive/anyaction/quota",
        params={
            "reset_hour": 12,
            "timezone": "Asia/Shanghai",
            "timestamp": "2026-05-30T03:00:00Z",
        },
    )
    quota_after_action = client.request_json(
        "POST",
        "/v1/proactive/anyaction/actions",
        json_body={
            "reset_hour": 12,
            "timezone": "Asia/Shanghai",
            "timestamp": "2026-05-30T03:01:00Z",
        },
    )
    quota_after = client.request_json(
        "GET",
        "/v1/proactive/anyaction/quota",
        params={
            "reset_hour": 12,
            "timezone": "Asia/Shanghai",
            "timestamp": "2026-05-30T03:02:00Z",
        },
    )
    client.request_json(
        "POST",
        "/v1/proactive/drift-runs",
        json_body={
            "session_key": "telegram:1",
            "timestamp": "2026-05-30T12:00:00Z",
        },
    )
    drift_last = client.request_json(
        "GET",
        "/v1/proactive/drift-runs/last",
        params={"session_key": "telegram:1"},
    )
    drift_finish = client.request_json(
        "POST",
        "/v1/proactive/drift/finish",
        json_body={
            "skill_used": "explore-curiosity",
            "one_line": "整理攻略线索",
            "next": "继续核验",
            "message_result": "sent",
            "note": "游戏群",
            "timestamp": "2026-05-30T12:05:00Z",
        },
    )
    drift_summary = client.request_json(
        "GET",
        "/v1/proactive/drift/summary",
        params={"limit": 10},
    )
    drift_skill = client.request_json(
        "GET",
        "/v1/proactive/drift/skills/explore-curiosity",
    )
    client.request_json(
        "POST",
        "/v1/proactive/bg-context/main",
        json_body={"timestamp": "2026-05-30T12:30:00Z"},
    )
    bg_context_last = client.request_json(
        "GET",
        "/v1/proactive/bg-context/main/last",
    )
    state_counts = _read_state_counts(state_file)

    recent_runs = drift_summary.get("recent_runs", [])
    checks = {
        "anyaction_quota_window_visible_before_action": (
            str(quota_before.get("window_key")) == "2026-05-29@12@Asia/Shanghai"
        ),
        "anyaction_quota_incremented": (
            int(quota_after_action.get("used") or 0) == 1
            and int(quota_after.get("used") or 0) == 1
            and quota_after.get("found") is True
        ),
        "drift_last_marker_visible": (
            drift_last.get("found") is True
            and str(drift_last.get("key")) == "drift_last_at"
        ),
        "drift_finish_recorded_skill_state": (
            int(drift_finish.get("skill_state", {}).get("run_count") or 0) == 1
            and str(drift_finish.get("skill_state", {}).get("skill_name"))
            == "explore-curiosity"
        ),
        "drift_summary_visible": (
            isinstance(recent_runs, list)
            and any(str(item.get("skill")) == "explore-curiosity" for item in recent_runs)
        ),
        "drift_skill_state_visible": (
            drift_skill.get("found") is True
            and str(drift_skill.get("skill_name")) == "explore-curiosity"
        ),
        "bg_context_marker_visible": (
            bg_context_last.get("found") is True
            and str(bg_context_last.get("key")) == "bg_context_last_main_at"
        ),
        "state_file_persisted_anyaction_and_drift": (
            state_counts["anyaction_quotas"] == 1
            and state_counts["drift_skills"] == 1
            and state_counts["drift_recent_runs"] == 1
            and state_counts["session_marks"] >= 1
            and state_counts["global_marks"] == 1
        ),
    }

    return {
        "quota_before": quota_before,
        "quota_after_action": quota_after_action,
        "quota_after": quota_after,
        "drift_last": drift_last,
        "drift_finish": drift_finish,
        "drift_summary": drift_summary,
        "drift_skill": drift_skill,
        "bg_context_last": bg_context_last,
        "state_counts": state_counts,
        "checks": checks,
    }


def run_tick_log_smoke(client: ProactiveRuntimeClient, *, state_file: Path) -> dict[str, Any]:
    start = client.request_json(
        "POST",
        "/v1/proactive/tick-logs/start",
        json_body={
            "tick_id": "tick-1",
            "session_key": "telegram:1",
            "started_at": "2026-05-31T10:00:00Z",
        },
    )
    step = client.request_json(
        "POST",
        "/v1/proactive/tick-steps",
        json_body={
            "tick_id": "tick-1",
            "step_index": 1,
            "phase": "loop",
            "tool_name": "message_push",
            "tool_call_id": "call-1",
            "tool_args": {"message": "hello"},
            "tool_result_text": '{"ok":true}',
            "interesting_ids_after": ["feed:1"],
            "final_message_after": "hello",
        },
    )
    finish = client.request_json(
        "POST",
        "/v1/proactive/tick-logs/finish",
        json_body={
            "tick_id": "tick-1",
            "session_key": "telegram:1",
            "started_at": "2026-05-31T10:00:00Z",
            "finished_at": "2026-05-31T10:00:01Z",
            "terminal_action": "reply",
            "steps_taken": 1,
            "content_count": 1,
            "interesting_ids": ["feed:1"],
            "cited_ids": ["feed:1"],
            "final_message": "hello",
        },
    )
    tick_list = client.request_json(
        "GET",
        "/v1/proactive/tick-logs",
        params={"terminal_action": "reply"},
    )
    tick_detail = client.request_json(
        "GET",
        "/v1/proactive/tick-logs/tick-1",
    )
    tick_steps = client.request_json(
        "GET",
        "/v1/proactive/tick-logs/tick-1/steps",
    )
    state_counts = _read_state_counts(state_file)

    tick_items = tick_list.get("items", [])
    step_items = tick_steps.get("items", [])
    checks = {
        "tick_log_start_accepted": str(start.get("tick_id")) == "tick-1",
        "tick_step_recorded": (
            str(step.get("tick_id")) == "tick-1"
            and str(step.get("tool_name")) == "message_push"
        ),
        "tick_log_finish_recorded": (
            str(finish.get("tick_id")) == "tick-1"
            and str(finish.get("final_message")) == "hello"
        ),
        "tick_log_list_visible": (
            int(tick_list.get("total") or 0) == 1
            and any(str(item.get("tick_id")) == "tick-1" for item in tick_items)
        ),
        "tick_log_detail_visible": (
            tick_detail.get("found") is True
            and str(tick_detail.get("final_message")) == "hello"
        ),
        "tick_log_steps_visible": (
            int(tick_steps.get("total") or 0) == 1
            and any(str(item.get("tool_name")) == "message_push" for item in step_items)
        ),
        "state_file_persisted_tick_logs": (
            state_counts["tick_logs"] == 1 and state_counts["tick_steps"] == 1
        ),
    }

    return {
        "start": start,
        "step": step,
        "finish": finish,
        "tick_list": tick_list,
        "tick_detail": tick_detail,
        "tick_steps": tick_steps,
        "state_counts": state_counts,
        "checks": checks,
    }


def start_temp_runtime(repo_root: Path) -> TempRuntimeHandle:
    port = _find_free_port()
    runtime_root = Path(tempfile.mkdtemp(prefix="proactive-runtime-live-smoke-"))
    state_dir = runtime_root / "state"
    state_dir.mkdir(parents=True, exist_ok=True)
    stdout_log = runtime_root / "agent-runtime.stdout.log"
    stderr_log = runtime_root / "agent-runtime.stderr.log"
    stdout_handle = stdout_log.open("w", encoding="utf-8")
    stderr_handle = stderr_log.open("w", encoding="utf-8")
    env = os.environ.copy()
    env["AKASHIC_RUNTIME_ADDR"] = f"127.0.0.1:{port}"
    env["AKASHIC_RUNTIME_STATE_DIR"] = str(state_dir)
    for key in (
        "AKASHIC_ONEBOT_WS_URLS",
        "AKASHIC_ONEBOT_ACCESS_TOKEN",
        "AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT",
        "AKASHIC_BOT_IDS",
        "AKASHIC_QQ_GROUP_SEND_ENABLED",
        "AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED",
        "AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED",
        "AKASHIC_TELEGRAM_BOT_TOKEN",
        "TELEGRAM_BOT_TOKEN",
    ):
        env.pop(key, None)
    process = subprocess.Popen(
        [_find_go_exe(), "run", "./cmd/agent-runtime"],
        cwd=str(repo_root / "services" / "agent-runtime"),
        env=env,
        stdout=stdout_handle,
        stderr=stderr_handle,
        text=True,
    )
    base_url = f"http://127.0.0.1:{port}"
    deadline = time.time() + 60
    last_error = ""
    while time.time() < deadline:
        if process.poll() is not None:
            break
        try:
            with httpx.Client(timeout=2.0, trust_env=True) as client:
                response = client.get(base_url + "/healthz")
            if response.status_code == 200:
                stdout_handle.close()
                stderr_handle.close()
                return TempRuntimeHandle(
                    process=process,
                    base_url=base_url,
                    state_dir=state_dir,
                    state_file=state_dir / "proactive-state.json",
                    stdout_log=stdout_log,
                    stderr_log=stderr_log,
                )
        except Exception as exc:  # pragma: no cover - best effort startup wait
            last_error = str(exc)
        time.sleep(0.5)
    stdout_handle.close()
    stderr_handle.close()
    stderr_tail = ""
    if stderr_log.exists():
        stderr_tail = "\n".join(stderr_log.read_text(encoding="utf-8").splitlines()[-20:])
    process.kill()
    process.wait(timeout=5)
    raise RuntimeError(
        f"temp agent-runtime did not become healthy at {base_url}/healthz. "
        f"last_error={last_error!r}\n{stderr_tail}"
    )


def run_proactive_runtime_flow_live_smoke(repo_root: Path) -> dict[str, Any]:
    runtime = start_temp_runtime(repo_root)
    client = ProactiveRuntimeClient(runtime.base_url)
    try:
        delivery_seen_cleanup_smoke = run_delivery_seen_cleanup_smoke(
            client, state_file=runtime.state_file
        )
        anyaction_drift_bg_context_smoke = run_anyaction_drift_bg_context_smoke(
            client, state_file=runtime.state_file
        )
        tick_log_smoke = run_tick_log_smoke(client, state_file=runtime.state_file)
        return _build_result(
            base_url=runtime.base_url,
            state_dir=runtime.state_dir,
            state_file=runtime.state_file,
            stdout_log=runtime.stdout_log,
            stderr_log=runtime.stderr_log,
            delivery_seen_cleanup_smoke=delivery_seen_cleanup_smoke,
            anyaction_drift_bg_context_smoke=anyaction_drift_bg_context_smoke,
            tick_log_smoke=tick_log_smoke,
        )
    finally:
        runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_proactive_runtime_flow_live_smoke(Path(args.repo_root).resolve())
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
