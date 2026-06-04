from __future__ import annotations

import argparse
import json
import os
import sqlite3
import sys
import tempfile
import warnings
from contextlib import contextmanager
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[1]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

warnings.filterwarnings(
    "ignore",
    message=r"Using `httpx` with `starlette\.testclient` is deprecated; install `httpx2` instead\.",
)

from fastapi.testclient import TestClient

from bootstrap.dashboard_api import create_dashboard_app
from memory2.store import MemoryStore2
from plugins.default_memory.engine import DefaultMemoryEngine

from scripts.verify_proactive_runtime_flow_live_smoke import (
    ProactiveRuntimeClient,
    TempRuntimeHandle,
    start_temp_runtime,
)

class _DashboardMemoryAdmin:
    def __init__(self, workspace: Path) -> None:
        self._store = MemoryStore2(workspace / "memory" / "memory2.db")

    def describe(self) -> Any:
        return DefaultMemoryEngine.DESCRIPTOR

    def keyword_match_procedures(self, action_tokens: list[str]) -> Any:
        return self._store.keyword_match_procedures(action_tokens)

    def list_events_by_time_range(self, time_start: Any, time_end: Any, *, limit: int = 200) -> Any:
        return self._store.list_events_by_time_range(time_start, time_end, limit=limit)

    def list_items_for_dashboard(self, **kwargs: Any) -> Any:
        return self._store.list_items_for_dashboard(**kwargs)

    def get_item_for_dashboard(self, item_id: str, *, include_embedding: bool = False) -> Any:
        return self._store.get_item_for_dashboard(
            item_id, include_embedding=include_embedding
        )

    def update_item_for_dashboard(self, item_id: str, **kwargs: Any) -> Any:
        return self._store.update_item_for_dashboard(item_id, **kwargs)

    def delete_item(self, item_id: str) -> bool:
        return self._store.delete_item(item_id)

    def delete_items_batch(self, ids: list[str]) -> int:
        return self._store.delete_items_batch(ids)

    def find_similar_items_for_dashboard(self, item_id: str, **kwargs: Any) -> Any:
        return self._store.find_similar_items_for_dashboard(item_id, **kwargs)

    def close(self) -> None:
        self._store.close()


@contextmanager
def _temporary_env(updates: dict[str, str]) -> Any:
    original: dict[str, str | None] = {}
    for key, value in updates.items():
        original[key] = os.environ.get(key)
        os.environ[key] = value
    try:
        yield
    finally:
        for key, previous in original.items():
            if previous is None:
                os.environ.pop(key, None)
            else:
                os.environ[key] = previous


def _utcnow_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _count_tick_logs(db_path: Path) -> int:
    if not db_path.exists():
        return 0
    conn = sqlite3.connect(str(db_path))
    try:
        row = conn.execute(
            "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tick_log'"
        ).fetchone()
        if row is None or int(row[0] or 0) <= 0:
            return 0
        count_row = conn.execute("SELECT COUNT(*) FROM tick_log").fetchone()
        return int(count_row[0] or 0) if count_row is not None else 0
    finally:
        conn.close()


def _seed_runtime_tick_log(client: ProactiveRuntimeClient, tick_id: str) -> dict[str, Any]:
    start = client.request_json(
        "POST",
        "/v1/proactive/tick-logs/start",
        json_body={
            "tick_id": tick_id,
            "session_key": "runtime:room",
            "started_at": "2026-05-31T10:00:00Z",
        },
    )
    step = client.request_json(
        "POST",
        "/v1/proactive/tick-steps",
        json_body={
            "tick_id": tick_id,
            "step_index": 1,
            "phase": "loop",
            "tool_name": "message_push",
            "tool_call_id": "call-runtime",
            "tool_args": {"message": "runtime hello"},
            "tool_result_text": '{"ok":true}',
            "interesting_ids_after": ["runtime:feed:1"],
            "cited_ids_after": ["runtime:feed:1"],
            "terminal_action_after": "reply",
            "final_message_after": "runtime hello",
        },
    )
    finish = client.request_json(
        "POST",
        "/v1/proactive/tick-logs/finish",
        json_body={
            "tick_id": tick_id,
            "session_key": "runtime:room",
            "started_at": "2026-05-31T10:00:00Z",
            "finished_at": "2026-05-31T10:00:02Z",
            "terminal_action": "reply",
            "steps_taken": 1,
            "alert_count": 1,
            "content_count": 1,
            "context_count": 0,
            "interesting_ids": ["runtime:feed:1"],
            "discarded_ids": [],
            "cited_ids": ["runtime:feed:1"],
            "drift_entered": False,
            "final_message": "runtime hello",
        },
    )
    return {"start": start, "step": step, "finish": finish}


def _build_result(
    *,
    repo_root: Path,
    runtime: TempRuntimeHandle,
    workspace: Path,
    sqlite_tick_logs_before: int,
    sqlite_tick_logs_after: int,
    runtime_seed: dict[str, Any],
    dashboard_list: dict[str, Any],
    dashboard_detail: dict[str, Any],
    dashboard_steps: dict[str, Any],
) -> dict[str, Any]:
    list_items = dashboard_list.get("items", []) if isinstance(dashboard_list, dict) else []
    step_items = dashboard_steps.get("items", []) if isinstance(dashboard_steps, dict) else []
    checks = {
        "sqlite_tick_logs_empty_before_dashboard_reads": sqlite_tick_logs_before == 0,
        "runtime_tick_log_seeded": (
            str(runtime_seed.get("finish", {}).get("tick_id")) == "runtime-fallback-tick"
            and str(runtime_seed.get("finish", {}).get("final_message")) == "runtime hello"
        ),
        "dashboard_tick_log_list_falls_back_to_runtime": (
            int(dashboard_list.get("total") or 0) == 1
            and any(str(item.get("tick_id")) == "runtime-fallback-tick" for item in list_items)
        ),
        "dashboard_tick_log_detail_falls_back_to_runtime": (
            str(dashboard_detail.get("tick_id")) == "runtime-fallback-tick"
            and str(dashboard_detail.get("final_message")) == "runtime hello"
        ),
        "dashboard_tick_log_steps_fall_back_to_runtime": (
            int(dashboard_steps.get("total") or 0) == 1
            and any(str(item.get("tool_name")) == "message_push" for item in step_items)
        ),
        "sqlite_tick_logs_stay_empty_after_dashboard_reads": sqlite_tick_logs_after == 0,
    }
    all_passed = all(checks.values())
    return {
        "generated_at": _utcnow_iso(),
        "repo_root": str(repo_root),
        "workspace": str(workspace),
        "runtime_base_url": runtime.base_url,
        "runtime_state_dir": str(runtime.state_dir),
        "runtime_stdout_log": str(runtime.stdout_log),
        "runtime_stderr_log": str(runtime.stderr_log),
        "dashboard_proactive_db": str(workspace / "proactive.db"),
        "sqlite_tick_logs_before": sqlite_tick_logs_before,
        "sqlite_tick_logs_after": sqlite_tick_logs_after,
        "runtime_tick_seed": runtime_seed,
        "dashboard_tick_log_list": dashboard_list,
        "dashboard_tick_log_detail": dashboard_detail,
        "dashboard_tick_log_steps": dashboard_steps,
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if all_passed else "verification_incomplete",
            "category": (
                "dashboard_proactive_tick_logs_runtime_fallback_live_verified"
                if all_passed
                else "dashboard_proactive_tick_logs_runtime_fallback_incomplete"
            ),
            "reason": (
                "With SQLite tick_log mirror empty, dashboard proactive tick-log list/detail/steps still read the current Go runtime tick state without writing fallback rows."
                if all_passed
                else "At least one proactive dashboard runtime-fallback check failed."
            ),
        },
    }


def run_dashboard_proactive_tick_logs_boundary_verifier(repo_root: Path) -> dict[str, Any]:
    runtime = start_temp_runtime(repo_root)
    workspace = Path(tempfile.mkdtemp(prefix="dashboard-proactive-fallback-"))
    runtime_client = ProactiveRuntimeClient(runtime.base_url)
    tick_id = "runtime-fallback-tick"
    try:
        runtime_seed = _seed_runtime_tick_log(runtime_client, tick_id)
        sqlite_tick_logs_before = _count_tick_logs(workspace / "proactive.db")
        with _temporary_env({"AKASHIC_AGENT_RUNTIME_URL": runtime.base_url}):
            dashboard_app = create_dashboard_app(
                workspace,
                memory_admin=_DashboardMemoryAdmin(workspace),
            )
            with TestClient(dashboard_app) as client:
                dashboard_list = client.get(
                    "/api/dashboard/proactive/tick_logs",
                    params={"session_key": "runtime:room", "page_size": 25},
                )
                dashboard_detail = client.get(
                    f"/api/dashboard/proactive/tick_logs/{tick_id}"
                )
                dashboard_steps = client.get(
                    f"/api/dashboard/proactive/tick_logs/{tick_id}/steps"
                )
                list_payload = dashboard_list.json()
                detail_payload = dashboard_detail.json()
                steps_payload = dashboard_steps.json()
        sqlite_tick_logs_after = _count_tick_logs(workspace / "proactive.db")
        return _build_result(
            repo_root=repo_root,
            runtime=runtime,
            workspace=workspace,
            sqlite_tick_logs_before=sqlite_tick_logs_before,
            sqlite_tick_logs_after=sqlite_tick_logs_after,
            runtime_seed=runtime_seed,
            dashboard_list=list_payload,
            dashboard_detail=detail_payload,
            dashboard_steps=steps_payload,
        )
    finally:
        runtime.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_dashboard_proactive_tick_logs_boundary_verifier(
        Path(args.repo_root).resolve()
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
