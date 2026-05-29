from __future__ import annotations

from group_memory.models import StrategyCandidate
from group_memory.store import GroupMemoryStore


_CONFLICT_WORDS = ("不对", "不是", "别", "不要", "无效", "过期", "改了", "重测", "削了")
_SUPPORT_WORDS = ("确实", "验证", "实测", "稳", "有效", "对的", "没问题")


class StrategyEvolutionEngine:
    def __init__(self, store: GroupMemoryStore) -> None:
        self._store = store

    def apply(self, candidate: StrategyCandidate) -> str:
        matches = self._store.find_similar(candidate)
        if not matches:
            self._store.add_strategy(candidate)
            return "ADD"

        row, score = matches[0]
        strategy_id = str(row["id"])
        old_summary = str(row["summary"] or "")
        new_text = candidate.summary
        operation = _decide_operation(candidate, old_summary, score)
        evidence_ids = [e.message_id for e in candidate.evidence]
        for evidence in candidate.evidence:
            self._store.add_evidence(strategy_id, evidence, weight=1.0)

        if operation == "CONFLICT":
            self._store.update_strategy(
                strategy_id,
                status="disputed",
                confidence=max(0.35, min(float(row["confidence"] or 0.0), 0.65)),
                reinforcement_delta=1,
            )
            reason = "candidate conflicts with existing strategy"
        elif operation == "MERGE":
            merged = _merge_summary(old_summary, new_text)
            evidence_count = self._store.evidence_count(strategy_id)
            status = "validated" if evidence_count >= 2 else str(row["status"] or "pending")
            confidence = min(0.95, max(float(row["confidence"] or 0.0), candidate.confidence) + 0.08)
            self._store.update_strategy(
                strategy_id,
                summary=merged,
                status=status,
                confidence=confidence,
                version_hint=candidate.version_hint or str(row["version_hint"] or ""),
                reinforcement_delta=1,
            )
            reason = "candidate adds complementary details"
        else:
            evidence_count = self._store.evidence_count(strategy_id)
            status = "validated" if evidence_count >= 2 else str(row["status"] or "pending")
            if str(row["status"]) == "disputed" and _has_support_signal(new_text):
                status = "needs_review"
            confidence = min(0.95, float(row["confidence"] or 0.0) + 0.06)
            self._store.update_strategy(
                strategy_id,
                status=status,
                confidence=confidence,
                reinforcement_delta=1,
            )
            reason = "candidate supports existing strategy"

        self._store.add_log(
            strategy_id,
            operation,
            old_summary,
            new_text,
            reason,
            evidence_ids,
        )
        return operation


def _decide_operation(candidate: StrategyCandidate, old_summary: str, score: float) -> str:
    text = candidate.summary
    if any(word in text for word in _CONFLICT_WORDS):
        return "CONFLICT"
    if score < 0.52 and len(set(text) - set(old_summary)) > 12:
        return "MERGE"
    if len(text) > len(old_summary) + 18 and text not in old_summary:
        return "MERGE"
    return "REINFORCE"


def _has_support_signal(text: str) -> bool:
    return any(word in text for word in _SUPPORT_WORDS)


def _merge_summary(old: str, new: str) -> str:
    old_clean = old.strip()
    new_clean = new.strip()
    if not old_clean:
        return new_clean
    if not new_clean or new_clean in old_clean:
        return old_clean
    if old_clean in new_clean:
        return new_clean
    return f"{old_clean}；补充：{new_clean}"
