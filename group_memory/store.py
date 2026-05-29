from __future__ import annotations

import json
import sqlite3
import threading
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from group_memory.models import SearchHit, StrategyCandidate
from group_memory.text import compact_quote, lexical_similarity, tokenize


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


class GroupMemoryStore:
    def __init__(self, path: str | Path) -> None:
        self.path = Path(path)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._db = sqlite3.connect(self.path, check_same_thread=False)
        self._db.row_factory = sqlite3.Row
        self._lock = threading.RLock()
        self._ensure_schema()

    def close(self) -> None:
        with self._lock:
            self._db.close()

    def _ensure_schema(self) -> None:
        with self._lock:
            self._db.executescript(
                """
                CREATE TABLE IF NOT EXISTS group_memory_cursors (
                    session_key TEXT PRIMARY KEY,
                    last_seq INTEGER NOT NULL DEFAULT -1,
                    updated_at TEXT NOT NULL
                );

                CREATE TABLE IF NOT EXISTS group_strategy_items (
                    id TEXT PRIMARY KEY,
                    group_id TEXT NOT NULL,
                    game TEXT NOT NULL DEFAULT 'unknown',
                    topic TEXT NOT NULL,
                    category TEXT NOT NULL,
                    summary TEXT NOT NULL,
                    status TEXT NOT NULL DEFAULT 'pending',
                    confidence REAL NOT NULL DEFAULT 0.0,
                    version_hint TEXT NOT NULL DEFAULT '',
                    reinforcement INTEGER NOT NULL DEFAULT 1,
                    search_text TEXT NOT NULL DEFAULT '',
                    created_at TEXT NOT NULL,
                    updated_at TEXT NOT NULL
                );

                CREATE INDEX IF NOT EXISTS idx_group_strategy_scope
                    ON group_strategy_items(group_id, status, category);

                CREATE TABLE IF NOT EXISTS group_strategy_evidence (
                    id TEXT PRIMARY KEY,
                    strategy_id TEXT NOT NULL,
                    session_key TEXT NOT NULL,
                    message_seq INTEGER NOT NULL,
                    message_id TEXT NOT NULL,
                    sender_id TEXT NOT NULL,
                    message_time TEXT NOT NULL,
                    quote TEXT NOT NULL,
                    weight REAL NOT NULL DEFAULT 1.0,
                    created_at TEXT NOT NULL,
                    UNIQUE(strategy_id, session_key, message_seq)
                );

                CREATE INDEX IF NOT EXISTS idx_group_strategy_evidence_strategy
                    ON group_strategy_evidence(strategy_id, message_time);

                CREATE TABLE IF NOT EXISTS group_strategy_links (
                    from_id TEXT NOT NULL,
                    to_id TEXT NOT NULL,
                    relation TEXT NOT NULL,
                    created_at TEXT NOT NULL,
                    PRIMARY KEY(from_id, to_id, relation)
                );

                CREATE TABLE IF NOT EXISTS group_evolution_logs (
                    id TEXT PRIMARY KEY,
                    strategy_id TEXT NOT NULL,
                    operation TEXT NOT NULL,
                    old_summary TEXT NOT NULL DEFAULT '',
                    new_summary TEXT NOT NULL DEFAULT '',
                    reason TEXT NOT NULL DEFAULT '',
                    evidence_json TEXT NOT NULL DEFAULT '[]',
                    created_at TEXT NOT NULL
                );
                """
            )
            self._db.commit()

    def get_cursor(self, session_key: str) -> int:
        with self._lock:
            row = self._db.execute(
                "SELECT last_seq FROM group_memory_cursors WHERE session_key=?",
                (session_key,),
            ).fetchone()
        return int(row["last_seq"]) if row else -1

    def set_cursor(self, session_key: str, last_seq: int) -> None:
        with self._lock:
            self._db.execute(
                """
                INSERT INTO group_memory_cursors(session_key, last_seq, updated_at)
                VALUES (?, ?, ?)
                ON CONFLICT(session_key) DO UPDATE SET
                    last_seq=excluded.last_seq,
                    updated_at=excluded.updated_at
                """,
                (session_key, int(last_seq), _now()),
            )
            self._db.commit()

    def add_strategy(self, candidate: StrategyCandidate) -> str:
        item_id = str(uuid.uuid4())
        now = _now()
        search_text = _build_search_text(candidate)
        with self._lock:
            self._db.execute(
                """
                INSERT INTO group_strategy_items(
                    id, group_id, game, topic, category, summary, status,
                    confidence, version_hint, reinforcement, search_text,
                    created_at, updated_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    item_id,
                    candidate.group_id,
                    candidate.game,
                    candidate.topic,
                    candidate.category,
                    candidate.summary,
                    candidate.status,
                    float(candidate.confidence),
                    candidate.version_hint,
                    1,
                    search_text,
                    now,
                    now,
                ),
            )
            self._db.commit()
        for evidence in candidate.evidence:
            self.add_evidence(item_id, evidence, weight=1.0)
        self.add_log(
            item_id,
            "ADD",
            "",
            candidate.summary,
            "new strategy candidate",
            [e.message_id for e in candidate.evidence],
        )
        return item_id

    def update_strategy(
        self,
        strategy_id: str,
        *,
        summary: str | None = None,
        status: str | None = None,
        confidence: float | None = None,
        version_hint: str | None = None,
        reinforcement_delta: int = 0,
    ) -> None:
        current = self.get_strategy(strategy_id)
        if current is None:
            return
        new_summary = summary if summary is not None else str(current["summary"])
        new_status = status if status is not None else str(current["status"])
        new_confidence = (
            float(confidence)
            if confidence is not None
            else float(current["confidence"] or 0.0)
        )
        new_version_hint = (
            version_hint if version_hint is not None else str(current["version_hint"] or "")
        )
        new_reinforcement = max(
            1, int(current["reinforcement"] or 1) + int(reinforcement_delta)
        )
        search_text = " ".join(
            [
                str(current["game"] or ""),
                str(current["topic"] or ""),
                str(current["category"] or ""),
                new_summary,
                new_status,
                new_version_hint,
            ]
        )
        with self._lock:
            self._db.execute(
                """
                UPDATE group_strategy_items
                SET summary=?, status=?, confidence=?, version_hint=?,
                    reinforcement=?, search_text=?, updated_at=?
                WHERE id=?
                """,
                (
                    new_summary,
                    new_status,
                    new_confidence,
                    new_version_hint,
                    new_reinforcement,
                    search_text,
                    _now(),
                    strategy_id,
                ),
            )
            self._db.commit()

    def add_evidence(self, strategy_id: str, message: Any, *, weight: float) -> bool:
        evidence_id = str(uuid.uuid4())
        with self._lock:
            cur = self._db.execute(
                """
                INSERT OR IGNORE INTO group_strategy_evidence(
                    id, strategy_id, session_key, message_seq, message_id,
                    sender_id, message_time, quote, weight, created_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    evidence_id,
                    strategy_id,
                    message.session_key,
                    int(message.seq),
                    message.message_id,
                    message.sender_id,
                    message.timestamp,
                    compact_quote(message.content),
                    float(weight),
                    _now(),
                ),
            )
            self._db.commit()
            return cur.rowcount > 0

    def add_log(
        self,
        strategy_id: str,
        operation: str,
        old_summary: str,
        new_summary: str,
        reason: str,
        evidence_ids: list[str],
    ) -> None:
        with self._lock:
            self._db.execute(
                """
                INSERT INTO group_evolution_logs(
                    id, strategy_id, operation, old_summary, new_summary,
                    reason, evidence_json, created_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    str(uuid.uuid4()),
                    strategy_id,
                    operation,
                    old_summary,
                    new_summary,
                    reason,
                    json.dumps(evidence_ids, ensure_ascii=False),
                    _now(),
                ),
            )
            self._db.commit()

    def get_strategy(self, strategy_id: str) -> dict[str, Any] | None:
        with self._lock:
            row = self._db.execute(
                "SELECT * FROM group_strategy_items WHERE id=?",
                (strategy_id,),
            ).fetchone()
        return dict(row) if row else None

    def list_strategies(
        self,
        *,
        group_id: str | None = None,
        status: str | None = None,
        limit: int = 50,
    ) -> list[dict[str, Any]]:
        where: list[str] = []
        params: list[Any] = []
        if group_id:
            where.append("group_id=?")
            params.append(group_id)
        if status:
            where.append("status=?")
            params.append(status)
        sql = "SELECT * FROM group_strategy_items"
        if where:
            sql += " WHERE " + " AND ".join(where)
        sql += " ORDER BY updated_at DESC LIMIT ?"
        params.append(max(1, min(int(limit), 200)))
        with self._lock:
            rows = self._db.execute(sql, tuple(params)).fetchall()
        return [dict(row) for row in rows]

    def list_evidence(self, strategy_id: str, *, limit: int = 20) -> list[dict[str, Any]]:
        with self._lock:
            rows = self._db.execute(
                """
                SELECT * FROM group_strategy_evidence
                WHERE strategy_id=?
                ORDER BY message_time ASC, message_seq ASC
                LIMIT ?
                """,
                (strategy_id, max(1, min(int(limit), 100))),
            ).fetchall()
        return [dict(row) for row in rows]

    def evidence_count(self, strategy_id: str) -> int:
        with self._lock:
            row = self._db.execute(
                "SELECT COUNT(1) AS c FROM group_strategy_evidence WHERE strategy_id=?",
                (strategy_id,),
            ).fetchone()
        return int(row["c"] if row else 0)

    def find_similar(
        self,
        candidate: StrategyCandidate,
        *,
        threshold: float = 0.34,
        limit: int = 5,
    ) -> list[tuple[dict[str, Any], float]]:
        rows = self.list_strategies(group_id=candidate.group_id, limit=500)
        scored: list[tuple[dict[str, Any], float]] = []
        candidate_text = _build_search_text(candidate)
        for row in rows:
            haystack = " ".join(
                str(row.get(k, "") or "")
                for k in ("topic", "category", "summary", "version_hint", "search_text")
            )
            score = lexical_similarity(candidate_text, haystack)
            if candidate.category == row.get("category"):
                score += 0.08
            if candidate.topic and str(row.get("topic") or ""):
                score = max(score, lexical_similarity(candidate.topic, str(row["topic"])))
            if score >= threshold:
                scored.append((row, min(score, 1.0)))
        scored.sort(key=lambda item: item[1], reverse=True)
        return scored[:limit]

    def search(
        self,
        query: str,
        *,
        group_id: str | None = None,
        status: str | None = None,
        limit: int = 10,
    ) -> list[SearchHit]:
        tokens = tokenize(query)
        rows = self.list_strategies(group_id=group_id, status=status, limit=500)
        hits: list[SearchHit] = []
        for row in rows:
            text = " ".join(
                str(row.get(k, "") or "")
                for k in ("game", "topic", "category", "summary", "version_hint", "search_text")
            )
            token_score = 0.0
            if tokens:
                token_score = sum(1 for t in tokens if t in text.lower()) / len(tokens)
            semanticish = lexical_similarity(query, text)
            reinforcement = int(row.get("reinforcement") or 1)
            confidence = float(row.get("confidence") or 0.0)
            evidence_count = self.evidence_count(str(row["id"]))
            score = (
                0.45 * token_score
                + 0.35 * semanticish
                + 0.12 * min(reinforcement, 8) / 8
                + 0.08 * confidence
            )
            if not tokens or score > 0:
                hits.append(
                    SearchHit(
                        id=str(row["id"]),
                        group_id=str(row["group_id"]),
                        game=str(row["game"]),
                        topic=str(row["topic"]),
                        category=str(row["category"]),
                        summary=str(row["summary"]),
                        status=str(row["status"]),
                        confidence=confidence,
                        reinforcement=reinforcement,
                        version_hint=str(row["version_hint"] or ""),
                        score=score,
                        evidence_count=evidence_count,
                    )
                )
        hits.sort(key=lambda h: h.score, reverse=True)
        return hits[: max(1, min(int(limit), 50))]


def _build_search_text(candidate: StrategyCandidate) -> str:
    return " ".join(
        [
            candidate.game,
            candidate.topic,
            candidate.category,
            candidate.summary,
            candidate.status,
            candidate.version_hint,
            *candidate.entities,
        ]
    )
