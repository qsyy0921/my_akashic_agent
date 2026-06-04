from __future__ import annotations

import sqlite3
from pathlib import Path

from scripts.verify_dashboard_proactive_tick_logs_boundary import (
    _build_result,
    _count_tick_logs,
)


class _Runtime:
    def __init__(self, tmp_path: Path) -> None:
        self.base_url = "http://127.0.0.1:18888"
        self.state_dir = tmp_path / "runtime-state"
        self.stdout_log = tmp_path / "runtime.stdout.log"
        self.stderr_log = tmp_path / "runtime.stderr.log"


def test_count_tick_logs_returns_zero_for_missing_db(tmp_path: Path) -> None:
    assert _count_tick_logs(tmp_path / "missing.db") == 0


def test_count_tick_logs_reads_existing_rows(tmp_path: Path) -> None:
    db_path = tmp_path / "proactive.db"
    conn = sqlite3.connect(db_path)
    try:
        conn.execute(
            "CREATE TABLE tick_log (id INTEGER PRIMARY KEY AUTOINCREMENT, tick_id TEXT)"
        )
        conn.execute("INSERT INTO tick_log(tick_id) VALUES ('tick-1')")
        conn.execute("INSERT INTO tick_log(tick_id) VALUES ('tick-2')")
        conn.commit()
    finally:
        conn.close()

    assert _count_tick_logs(db_path) == 2


def test_build_result_marks_live_verified_when_all_checks_pass(tmp_path: Path) -> None:
    runtime = _Runtime(tmp_path)
    result = _build_result(
        repo_root=tmp_path,
        runtime=runtime,
        workspace=tmp_path / "workspace",
        sqlite_tick_logs_before=0,
        sqlite_tick_logs_after=0,
        runtime_seed={"finish": {"tick_id": "runtime-fallback-tick", "final_message": "runtime hello"}},
        dashboard_list={"total": 1, "items": [{"tick_id": "runtime-fallback-tick"}]},
        dashboard_detail={"tick_id": "runtime-fallback-tick", "final_message": "runtime hello"},
        dashboard_steps={"total": 1, "items": [{"tool_name": "message_push"}]},
    )

    assert result["checks"]["dashboard_tick_log_list_falls_back_to_runtime"] is True
    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "dashboard_proactive_tick_logs_runtime_fallback_live_verified"
    )


def test_build_result_marks_incomplete_when_any_check_fails(tmp_path: Path) -> None:
    runtime = _Runtime(tmp_path)
    result = _build_result(
        repo_root=tmp_path,
        runtime=runtime,
        workspace=tmp_path / "workspace",
        sqlite_tick_logs_before=1,
        sqlite_tick_logs_after=1,
        runtime_seed={"finish": {"tick_id": "runtime-fallback-tick", "final_message": "runtime hello"}},
        dashboard_list={"total": 0, "items": []},
        dashboard_detail={},
        dashboard_steps={"total": 0, "items": []},
    )

    assert result["checks"]["sqlite_tick_logs_empty_before_dashboard_reads"] is False
    assert result["conclusion"]["status"] == "verification_incomplete"
    assert (
        result["conclusion"]["category"]
        == "dashboard_proactive_tick_logs_runtime_fallback_incomplete"
    )
