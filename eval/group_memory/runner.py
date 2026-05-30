from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from group_memory import GroupMemoryService, GroupMemoryStore
from session.store import SessionStore


def run_group_memory_fixture_eval(
    *,
    fixture_path: str | Path,
    workspace: str | Path,
    min_top1_accuracy: float = 1.0,
    min_evidence_coverage: float = 1.0,
) -> dict[str, Any]:
    fixture_path = Path(fixture_path)
    workspace = Path(workspace)
    workspace.mkdir(parents=True, exist_ok=True)
    fixture = json.loads(fixture_path.read_text(encoding="utf-8"))
    group_id = str(fixture["group_id"])
    session_key = f"qq:gqq:{group_id}"

    session_store = SessionStore(workspace / "sessions.db")
    memory_store = GroupMemoryStore(workspace / "group_memory.db")
    try:
        _seed(session_store, fixture, session_key)
        service = GroupMemoryService(
            session_store=session_store,
            memory_store=memory_store,
            batch_max_messages=20,
        )
        stats = service.ingest_group(group_id)
        results = []
        correct = 0
        evidence_ok = 0
        for question in fixture.get("questions", []):
            retrieval = service.retrieve(str(question["query"]), group_id=group_id, limit=3)
            top = retrieval.hits[0] if retrieval.hits else None
            expected_status = str(question.get("expected_status") or "")
            expected_topic = str(question.get("expected_topic") or "").lower()
            expected_evidence_count = int(question.get("expected_evidence_count") or 0)
            ok = bool(top) and (
                (not expected_status or top.status == expected_status)
                and (not expected_topic or expected_topic in top.topic.lower())
            )
            ev_ok = bool(top) and (
                expected_evidence_count <= 0 or top.evidence_count >= expected_evidence_count
            )
            correct += int(ok)
            evidence_ok += int(ev_ok)
            results.append(
                {
                    "query": question["query"],
                    "ok": ok,
                    "evidence_ok": ev_ok,
                    "trace": retrieval.trace.__dict__,
                    "top": top.__dict__ if top else None,
                }
            )

        question_count = len(results)
        top1_accuracy = correct / question_count if question_count else 0.0
        evidence_coverage = evidence_ok / question_count if question_count else 0.0
        passed = (
            top1_accuracy >= float(min_top1_accuracy)
            and evidence_coverage >= float(min_evidence_coverage)
        )
        return {
            "fixture": str(fixture_path),
            "license": fixture.get("license", ""),
            "stats": stats.__dict__,
            "questions": question_count,
            "top1_accuracy": top1_accuracy,
            "evidence_coverage": evidence_coverage,
            "min_top1_accuracy": float(min_top1_accuracy),
            "min_evidence_coverage": float(min_evidence_coverage),
            "passed": passed,
            "results": results,
        }
    finally:
        session_store.close()
        memory_store.close()


def _seed(store: SessionStore, fixture: dict[str, Any], session_key: str) -> None:
    group_id = str(fixture["group_id"])
    store.upsert_session(
        session_key,
        created_at="2026-01-01T00:00:00+00:00",
        updated_at="2026-01-01T00:00:00+00:00",
        last_consolidated=0,
        metadata={"chat_type": "group", "group_id": group_id, "observe_only": True},
    )
    existing = store.count_messages(session_key)
    if existing:
        return
    for seq, message in enumerate(fixture["messages"]):
        sender_id = str(message["sender_id"])
        content = f"[QQ群 {group_id} | {sender_id}] {message['content']}"
        store.insert_message(
            session_key,
            role="user",
            content=content,
            ts=f"2026-01-01T00:00:{seq:02d}+00:00",
            seq=seq,
            extra={
                "chat_type": "group",
                "group_id": group_id,
                "sender_id": sender_id,
                "observe_only": True,
            },
        )
