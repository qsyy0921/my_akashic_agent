from __future__ import annotations

import math
from collections import defaultdict
from typing import Any

from group_memory.models import RetrievalResult, RetrievalTrace, SearchHit
from group_memory.text import lexical_similarity, tokenize


_RRF_K = 60


class HybridGroupRag:
    """Deterministic high-standard retrieval pipeline.

    This implements the non-LLM parts of a modern RAG stack:
    structured filtering, lexical retrieval, semantic-ish retrieval, evidence
    retrieval, RRF fusion, lightweight reranking, and evidence packing.
    """

    def __init__(self, store: Any) -> None:
        self._store = store

    def retrieve(
        self,
        query: str,
        *,
        group_id: str | None = None,
        status: str | None = None,
        limit: int = 10,
        evidence_limit: int = 3,
    ) -> RetrievalResult:
        clean_query = str(query or "").strip()
        rows = self._store.list_strategies(
            group_id=group_id,
            status=status,
            limit=1000,
        )
        channels = {
            "structured": _structured_rank(clean_query, rows),
            "lexical": _lexical_rank(clean_query, rows),
            "semantic": _semantic_rank(clean_query, rows),
            "evidence": self._evidence_rank(clean_query, rows),
        }
        fused = _rrf_fuse(channels)
        hits: list[SearchHit] = []
        for strategy_id, base_score in fused:
            row = next((r for r in rows if str(r.get("id")) == strategy_id), None)
            if row is None:
                continue
            evidence = self._store.list_evidence(strategy_id, limit=evidence_limit)
            rerank_score = _rerank_score(clean_query, row, evidence)
            channel_names = [name for name, ranking in channels.items() if strategy_id in ranking]
            hits.append(
                _to_hit(
                    row=row,
                    score=base_score + rerank_score,
                    channels=channel_names,
                    evidence=evidence,
                    evidence_count=self._store.evidence_count(strategy_id),
                )
            )
        hits.sort(key=lambda h: h.score, reverse=True)
        safe_limit = max(1, min(int(limit), 50))
        trace = RetrievalTrace(
            query=clean_query,
            group_id=group_id or "",
            status=status or "",
            channels={name: len(ranking) for name, ranking in channels.items()},
            fused_count=len(fused),
        )
        return RetrievalResult(hits=hits[:safe_limit], trace=trace)

    def _evidence_rank(self, query: str, rows: list[dict[str, Any]]) -> dict[str, float]:
        ranked: dict[str, float] = {}
        for row in rows:
            strategy_id = str(row.get("id") or "")
            evidence = self._store.list_evidence(strategy_id, limit=20)
            score = 0.0
            for item in evidence:
                quote = str(item.get("quote") or "")
                score = max(score, lexical_similarity(query, quote))
                score += _token_overlap(query, quote) * 0.25
            if score > 0:
                ranked[strategy_id] = score
        return dict(sorted(ranked.items(), key=lambda item: item[1], reverse=True))


def _structured_rank(query: str, rows: list[dict[str, Any]]) -> dict[str, float]:
    tokens = set(tokenize(query))
    ranked: dict[str, float] = {}
    for row in rows:
        fields = [
            str(row.get("topic") or ""),
            str(row.get("category") or ""),
            str(row.get("game") or ""),
            str(row.get("version_hint") or ""),
            str(row.get("status") or ""),
        ]
        score = 0.0
        for field in fields:
            score = max(score, _token_overlap_tokens(tokens, field))
        topic = str(row.get("topic") or "").lower()
        if topic and topic in query.lower():
            score += 0.35
        if score > 0:
            ranked[str(row["id"])] = score
    return dict(sorted(ranked.items(), key=lambda item: item[1], reverse=True))


def _lexical_rank(query: str, rows: list[dict[str, Any]]) -> dict[str, float]:
    q_tokens = tokenize(query)
    ranked: dict[str, float] = {}
    for row in rows:
        text = _row_text(row)
        score = _bm25ish(q_tokens, tokenize(text), len(tokenize(text)))
        if score > 0:
            ranked[str(row["id"])] = score
    return dict(sorted(ranked.items(), key=lambda item: item[1], reverse=True))


def _semantic_rank(query: str, rows: list[dict[str, Any]]) -> dict[str, float]:
    ranked: dict[str, float] = {}
    for row in rows:
        score = lexical_similarity(query, _row_text(row))
        if score > 0:
            ranked[str(row["id"])] = score
    return dict(sorted(ranked.items(), key=lambda item: item[1], reverse=True))


def _rrf_fuse(channels: dict[str, dict[str, float]]) -> list[tuple[str, float]]:
    fused: dict[str, float] = defaultdict(float)
    for ranking in channels.values():
        ordered = list(ranking.items())
        for rank, (strategy_id, score) in enumerate(ordered, start=1):
            fused[strategy_id] += (1.0 / (_RRF_K + rank)) * (1.0 + min(score, 1.0))
    return sorted(fused.items(), key=lambda item: item[1], reverse=True)


def _rerank_score(query: str, row: dict[str, Any], evidence: list[dict[str, Any]]) -> float:
    text_score = lexical_similarity(query, _row_text(row))
    evidence_score = max(
        [lexical_similarity(query, str(item.get("quote") or "")) for item in evidence]
        or [0.0]
    )
    confidence = float(row.get("confidence") or 0.0)
    reinforcement = min(int(row.get("reinforcement") or 1), 8) / 8
    status_bonus = {
        "validated": 0.08,
        "disputed": 0.03,
        "needs_review": 0.02,
        "pending": 0.0,
        "outdated": -0.08,
    }.get(str(row.get("status") or ""), 0.0)
    return (
        0.18 * text_score
        + 0.12 * evidence_score
        + 0.05 * confidence
        + 0.04 * reinforcement
        + status_bonus
    )


def _to_hit(
    *,
    row: dict[str, Any],
    score: float,
    channels: list[str],
    evidence: list[dict[str, Any]],
    evidence_count: int,
) -> SearchHit:
    return SearchHit(
        id=str(row["id"]),
        group_id=str(row["group_id"]),
        game=str(row["game"]),
        topic=str(row["topic"]),
        category=str(row["category"]),
        summary=str(row["summary"]),
        status=str(row["status"]),
        confidence=float(row.get("confidence") or 0.0),
        reinforcement=int(row.get("reinforcement") or 1),
        version_hint=str(row.get("version_hint") or ""),
        score=float(score),
        evidence_count=evidence_count,
        channels=channels,
        evidence=[
            {
                "message_id": str(item.get("message_id") or ""),
                "sender_id": str(item.get("sender_id") or ""),
                "message_time": str(item.get("message_time") or ""),
                "quote": str(item.get("quote") or ""),
            }
            for item in evidence
        ],
    )


def _row_text(row: dict[str, Any]) -> str:
    return " ".join(
        str(row.get(key, "") or "")
        for key in ("game", "topic", "category", "summary", "version_hint", "search_text")
    )


def _token_overlap(query: str, text: str) -> float:
    return _token_overlap_tokens(set(tokenize(query)), text)


def _token_overlap_tokens(query_tokens: set[str], text: str) -> float:
    if not query_tokens:
        return 0.0
    text_tokens = set(tokenize(text))
    return len(query_tokens & text_tokens) / len(query_tokens)


def _bm25ish(query_tokens: list[str], doc_tokens: list[str], doc_len: int) -> float:
    if not query_tokens or not doc_tokens:
        return 0.0
    tf: dict[str, int] = defaultdict(int)
    for token in doc_tokens:
        tf[token] += 1
    score = 0.0
    avgdl = 24.0
    k1 = 1.4
    b = 0.75
    for token in query_tokens:
        freq = tf.get(token, 0)
        if not freq:
            continue
        idf = 1.0
        denom = freq + k1 * (1 - b + b * (doc_len / avgdl))
        score += idf * (freq * (k1 + 1)) / denom
    return score / math.sqrt(max(1, len(query_tokens)))
