from __future__ import annotations

import json
from pathlib import Path

import pytest

from agent.tools.group_memory import (
    IngestGroupMemoryTool,
    RecallGroupMemoryTool,
    ShowGroupStrategyEvidenceTool,
)
from group_memory import GroupMemoryService, GroupMemoryStore
from session.store import SessionStore


def _load_fixture() -> dict:
    path = Path(__file__).parent / "fixtures" / "group_memory_open_strategy_dataset.json"
    return json.loads(path.read_text(encoding="utf-8"))


def _seed_group_session(store: SessionStore, fixture: dict) -> str:
    group_id = str(fixture["group_id"])
    session_key = f"qq:gqq:{group_id}"
    store.upsert_session(
        session_key,
        created_at="2026-01-01T00:00:00+00:00",
        updated_at="2026-01-01T00:00:00+00:00",
        last_consolidated=0,
        metadata={"chat_type": "group", "group_id": group_id, "observe_only": True},
    )
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
    return session_key


def _build_service(tmp_path) -> tuple[GroupMemoryService, SessionStore, dict]:
    fixture = _load_fixture()
    session_store = SessionStore(tmp_path / "sessions.db")
    _seed_group_session(session_store, fixture)
    service = GroupMemoryService(
        session_store=session_store,
        memory_store=GroupMemoryStore(tmp_path / "group_memory.db"),
        batch_max_messages=10,
    )
    return service, session_store, fixture


class _FakeGroupMessageSource:
    def __init__(self, rows: list[dict]) -> None:
        self.calls: list[dict] = []
        self._rows = rows

    def fetch_new_messages(
        self,
        *,
        session_key: str,
        group_id: str,
        after_seq: int,
        limit: int,
    ) -> list[dict]:
        self.calls.append(
            {
                "session_key": session_key,
                "group_id": group_id,
                "after_seq": after_seq,
                "limit": limit,
            }
        )
        return [row for row in self._rows if int(row["seq"]) > after_seq][:limit]


def test_group_memory_ingests_open_strategy_fixture_and_evolves(tmp_path):
    service, _session_store, fixture = _build_service(tmp_path)
    group_id = str(fixture["group_id"])

    stats = service.ingest_group(group_id)

    assert stats.scanned == 4
    assert stats.candidates == 4
    assert stats.added == 2
    assert stats.reinforced == 1
    assert stats.conflicted == 1

    strategies = service.list_strategies(group_id=group_id)
    assert len(strategies) == 2
    boss = next(s for s in strategies if str(s["category"]) == "boss")
    assert boss["status"] == "disputed"
    assert boss["reinforcement"] == 3
    assert service.list_evidence(str(boss["id"]))

    second = service.ingest_group(group_id)
    assert second.scanned == 0
    assert second.cursor == stats.cursor


def test_group_memory_can_ingest_from_injected_message_source(tmp_path):
    fixture = _load_fixture()
    group_id = str(fixture["group_id"])
    session_key = f"qq:gqq:{group_id}"
    rows = []
    for seq, message in enumerate(fixture["messages"]):
        sender_id = str(message["sender_id"])
        rows.append(
            {
                "id": f"{session_key}:{seq}",
                "session_key": session_key,
                "seq": seq,
                "role": "user",
                "content": f"[QQ群 {group_id} | {sender_id}] {message['content']}",
                "timestamp": f"2026-01-01T00:00:{seq:02d}+00:00",
                "sender_id": sender_id,
            }
        )
    source = _FakeGroupMessageSource(rows)
    service = GroupMemoryService(
        session_store=SessionStore(tmp_path / "sessions.db"),
        memory_store=GroupMemoryStore(tmp_path / "group_memory.db"),
        message_source=source,
        batch_max_messages=10,
    )

    stats = service.ingest_group(group_id)
    second = service.ingest_group(group_id)

    assert stats.scanned == 4
    assert stats.candidates == 4
    assert second.scanned == 0
    assert source.calls[0]["after_seq"] == -1
    assert source.calls[1]["after_seq"] == stats.cursor


def test_group_memory_rag_search_returns_strategy_and_evidence(tmp_path):
    service, _session_store, fixture = _build_service(tmp_path)
    group_id = str(fixture["group_id"])
    service.ingest_group(group_id)

    hits = service.search("Boss A 二阶段怎么打", group_id=group_id, limit=3)

    assert hits
    assert hits[0].category == "boss"
    assert hits[0].status == "disputed"
    assert hits[0].evidence_count == 3
    assert "左边小怪" in hits[0].summary
    assert {"structured", "lexical", "semantic", "evidence"} & set(hits[0].channels)
    assert hits[0].evidence
    assert any("左边小怪" in str(e["quote"]) for e in hits[0].evidence)


def test_group_memory_retrieve_returns_trace_and_fused_evidence_pack(tmp_path):
    service, _session_store, fixture = _build_service(tmp_path)
    group_id = str(fixture["group_id"])
    service.ingest_group(group_id)

    result = service.retrieve("Boss A 小怪刷新改了么", group_id=group_id, limit=2)

    assert result.trace.fused_count >= 1
    assert result.trace.channels["evidence"] >= 1
    assert result.hits[0].status == "disputed"
    assert result.hits[0].evidence_count == 3
    assert any("新版本" in str(item["quote"]) for item in result.hits[0].evidence)


def test_group_memory_retrieve_abstains_on_unrelated_query(tmp_path):
    service, _session_store, fixture = _build_service(tmp_path)
    group_id = str(fixture["group_id"])
    service.ingest_group(group_id)

    result = service.retrieve("完全无关的钓鱼天气烹饪问题", group_id=group_id, limit=3)

    assert result.hits == []
    assert result.trace.fused_count == 0


@pytest.mark.asyncio
async def test_group_memory_tools_support_ingest_recall_and_evidence(tmp_path):
    service, _session_store, fixture = _build_service(tmp_path)
    group_id = str(fixture["group_id"])

    ingest_payload = json.loads(
        await IngestGroupMemoryTool(service).execute(group_id=group_id)
    )
    assert ingest_payload["candidates"] == 4

    recall_payload = json.loads(
        await RecallGroupMemoryTool(service).execute(
            query="Build B 配装怎么选",
            group_id=group_id,
            limit=5,
        )
    )
    assert recall_payload["count"] >= 1
    build_item = recall_payload["items"][0]
    assert build_item["category"] == "build"
    assert recall_payload["trace"]["channels"]["lexical"] >= 1
    assert build_item["retrieval_channels"]
    assert build_item["evidence"]

    evidence_payload = json.loads(
        await ShowGroupStrategyEvidenceTool(service).execute(
            strategy_id=build_item["id"],
            limit=5,
        )
    )
    assert evidence_payload["count"] == 1
    assert evidence_payload["evidence"][0]["message_id"].startswith(f"qq:gqq:{group_id}:")
