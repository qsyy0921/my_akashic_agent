from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class ObservedMessage:
    session_key: str
    seq: int
    message_id: str
    group_id: str
    sender_id: str
    content: str
    timestamp: str


@dataclass(frozen=True)
class StrategyCandidate:
    group_id: str
    game: str
    topic: str
    category: str
    summary: str
    status: str
    confidence: float
    version_hint: str = ""
    entities: list[str] = field(default_factory=list)
    evidence: list[ObservedMessage] = field(default_factory=list)


@dataclass(frozen=True)
class IngestStats:
    session_key: str
    group_id: str
    scanned: int = 0
    candidates: int = 0
    added: int = 0
    reinforced: int = 0
    merged: int = 0
    conflicted: int = 0
    cursor: int = -1


@dataclass(frozen=True)
class SearchHit:
    id: str
    group_id: str
    game: str
    topic: str
    category: str
    summary: str
    status: str
    confidence: float
    reinforcement: int
    version_hint: str
    score: float
    evidence_count: int
    channels: list[str] = field(default_factory=list)
    evidence: list[dict[str, object]] = field(default_factory=list)


@dataclass(frozen=True)
class RetrievalTrace:
    query: str
    group_id: str
    status: str
    channels: dict[str, int]
    fused_count: int


@dataclass(frozen=True)
class RetrievalResult:
    hits: list[SearchHit]
    trace: RetrievalTrace
