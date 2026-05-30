from __future__ import annotations

import asyncio
import logging
import re
from pathlib import Path
from typing import Iterable

from group_memory.evolution import StrategyEvolutionEngine
from group_memory.extractor import RuleBasedGameStrategyExtractor
from group_memory.models import IngestStats, ObservedMessage
from group_memory.rag import HybridGroupRag
from group_memory.sources import GroupMessageSource, SessionGroupMessageSource
from group_memory.store import GroupMemoryStore
from session.store import SessionStore

logger = logging.getLogger(__name__)

_QQ_GROUP_PREFIX = "qq:gqq:"


class GroupMemoryService:
    def __init__(
        self,
        *,
        session_store: SessionStore,
        memory_store: GroupMemoryStore,
        message_source: GroupMessageSource | None = None,
        batch_max_messages: int = 80,
    ) -> None:
        self._sessions = session_store
        self._source = message_source or SessionGroupMessageSource(session_store)
        self._store = memory_store
        self._extractor = RuleBasedGameStrategyExtractor()
        self._evolution = StrategyEvolutionEngine(memory_store)
        self._rag = HybridGroupRag(memory_store)
        self._batch_max_messages = max(1, int(batch_max_messages))

    @classmethod
    def from_workspace(
        cls,
        workspace: str | Path,
        *,
        session_store: SessionStore,
        message_source: GroupMessageSource | None = None,
        batch_max_messages: int = 80,
    ) -> "GroupMemoryService":
        return cls(
            session_store=session_store,
            memory_store=GroupMemoryStore(Path(workspace) / "group_memory.db"),
            message_source=message_source,
            batch_max_messages=batch_max_messages,
        )

    def ingest_group(self, group_id: str) -> IngestStats:
        session_key = f"{_QQ_GROUP_PREFIX}{group_id}"
        return self.ingest_session(session_key=session_key, group_id=group_id)

    def ingest_session(self, *, session_key: str, group_id: str | None = None) -> IngestStats:
        group = group_id or _group_id_from_session_key(session_key)
        if not group:
            return IngestStats(session_key=session_key, group_id="", cursor=-1)
        cursor = self._store.get_cursor(session_key)
        new_rows = self._source.fetch_new_messages(
            session_key=session_key,
            group_id=group,
            after_seq=cursor,
            limit=self._batch_max_messages,
        )
        if not new_rows:
            return IngestStats(session_key=session_key, group_id=group, cursor=cursor)
        all_new_max_seq = max(int(r.get("seq", cursor)) for r in new_rows)

        messages = [_to_observed_message(row, group) for row in new_rows]
        messages = [m for m in messages if m is not None]
        scanned = len(messages)
        if not messages:
            self._store.set_cursor(session_key, all_new_max_seq)
            return IngestStats(
                session_key=session_key,
                group_id=group,
                scanned=0,
                cursor=all_new_max_seq,
            )
        added = reinforced = merged = conflicted = candidates_count = 0
        max_seq = cursor
        for start in range(0, len(messages), self._batch_max_messages):
            batch = messages[start : start + self._batch_max_messages]
            if not batch:
                continue
            candidates = self._extractor.extract(batch)
            candidates_count += len(candidates)
            for candidate in candidates:
                op = self._evolution.apply(candidate)
                if op == "ADD":
                    added += 1
                elif op == "MERGE":
                    merged += 1
                elif op == "CONFLICT":
                    conflicted += 1
                else:
                    reinforced += 1
            max_seq = max(max_seq, *(m.seq for m in batch))

        max_seq = max(max_seq, all_new_max_seq)
        self._store.set_cursor(session_key, max_seq)
        return IngestStats(
            session_key=session_key,
            group_id=group,
            scanned=scanned,
            candidates=candidates_count,
            added=added,
            reinforced=reinforced,
            merged=merged,
            conflicted=conflicted,
            cursor=max_seq,
        )

    def search(
        self,
        query: str,
        *,
        group_id: str | None = None,
        status: str | None = None,
        limit: int = 10,
    ):
        return self.retrieve(
            query,
            group_id=group_id,
            status=status,
            limit=limit,
        ).hits

    def retrieve(
        self,
        query: str,
        *,
        group_id: str | None = None,
        status: str | None = None,
        limit: int = 10,
        evidence_limit: int = 3,
    ):
        return self._rag.retrieve(
            query,
            group_id=group_id,
            status=status,
            limit=limit,
            evidence_limit=evidence_limit,
        )

    def list_strategies(
        self,
        *,
        group_id: str | None = None,
        status: str | None = None,
        limit: int = 50,
    ) -> list[dict]:
        return self._store.list_strategies(group_id=group_id, status=status, limit=limit)

    def list_evidence(self, strategy_id: str, *, limit: int = 20) -> list[dict]:
        return self._store.list_evidence(strategy_id, limit=limit)


class GroupMemoryLoop:
    def __init__(
        self,
        *,
        service: GroupMemoryService,
        group_ids: Iterable[str],
        interval_seconds: float = 60.0,
    ) -> None:
        self._service = service
        self._group_ids = [str(g).strip() for g in group_ids if str(g).strip()]
        self._interval = max(5.0, float(interval_seconds))
        self._stopped = asyncio.Event()

    async def run(self) -> None:
        logger.info("[group_memory] loop started groups=%s", self._group_ids)
        try:
            while not self._stopped.is_set():
                for group_id in self._group_ids:
                    try:
                        stats = self._service.ingest_group(group_id)
                        if stats.scanned or stats.candidates:
                            logger.info("[group_memory] ingested %s", stats)
                    except Exception:
                        logger.exception("[group_memory] ingest failed group_id=%s", group_id)
                try:
                    await asyncio.wait_for(self._stopped.wait(), timeout=self._interval)
                except asyncio.TimeoutError:
                    continue
        finally:
            logger.info("[group_memory] loop stopped")

    def stop(self) -> None:
        self._stopped.set()


def _group_id_from_session_key(session_key: str) -> str:
    if session_key.startswith(_QQ_GROUP_PREFIX):
        return session_key[len(_QQ_GROUP_PREFIX) :]
    return ""


def _to_observed_message(row: dict, group_id: str) -> ObservedMessage | None:
    if str(row.get("role") or "") != "user":
        return None
    content = str(row.get("content") or "").strip()
    if not content:
        return None
    sender_id = str(row.get("sender_id") or "").strip()
    if not sender_id:
        sender_id = _sender_from_observed_content(content)
    return ObservedMessage(
        session_key=str(row.get("session_key") or ""),
        seq=int(row.get("seq") or 0),
        message_id=str(row.get("id") or ""),
        group_id=group_id,
        sender_id=sender_id or "unknown",
        content=content,
        timestamp=str(row.get("timestamp") or row.get("ts") or ""),
    )


def _sender_from_observed_content(content: str) -> str:
    match = re.match(r"^\[QQ群\s+[^|]+\|\s*([^\]]+)\]", content)
    return match.group(1).strip() if match else ""
