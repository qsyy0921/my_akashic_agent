from __future__ import annotations

import re

from group_memory.models import ObservedMessage, StrategyCandidate
from group_memory.text import compact_quote


_STRATEGY_MARKERS = (
    "攻略",
    "打法",
    "机制",
    "配装",
    "出装",
    "装备",
    "词条",
    "build",
    "boss",
    "p1",
    "p2",
    "一阶段",
    "二阶段",
    "三阶段",
    "小怪",
    "打断",
    "奶",
    "输出",
    "路线",
    "优先",
    "先",
    "别贪",
    "别打",
    "版本",
)

_SUPPORT_MARKERS = ("确实", "验证", "实测", "稳", "有效", "对的", "没问题")
_CONFLICT_MARKERS = (
    "不对",
    "不是",
    "别",
    "不要",
    "无效",
    "过期",
    "改了",
    "重测",
    "削了",
)


class RuleBasedGameStrategyExtractor:
    """Offline-safe extractor for first-pass group strategy memory.

    The production path can replace this with an LLM extractor later. This rule
    layer is intentionally deterministic so ingestion and RAG can be tested
    without network or model calls.
    """

    def extract(self, messages: list[ObservedMessage]) -> list[StrategyCandidate]:
        candidates: list[StrategyCandidate] = []
        for message in messages:
            text = _strip_observe_prefix(message.content)
            if not _looks_like_strategy(text):
                continue
            summary = compact_quote(text)
            topic = _infer_topic(text)
            category = _infer_category(text)
            status = "pending"
            confidence = 0.56
            lower = text.lower()
            if any(marker in lower for marker in _SUPPORT_MARKERS):
                confidence = 0.68
            if any(marker in lower for marker in _CONFLICT_MARKERS):
                status = "needs_review"
                confidence = 0.52
            candidates.append(
                StrategyCandidate(
                    group_id=message.group_id,
                    game="unknown",
                    topic=topic,
                    category=category,
                    summary=summary,
                    status=status,
                    confidence=confidence,
                    version_hint=_infer_version_hint(text),
                    entities=_infer_entities(text),
                    evidence=[message],
                )
            )
        return candidates


def _strip_observe_prefix(content: str) -> str:
    text = str(content or "").strip()
    return re.sub(r"^\[QQ群\s+[^\]]+\]\s*", "", text).strip()


def _looks_like_strategy(text: str) -> bool:
    normalized = str(text or "").strip().lower()
    if len(normalized) < 6:
        return False
    marker_hits = sum(1 for marker in _STRATEGY_MARKERS if marker in normalized)
    if marker_hits >= 1 and any(
        verb in normalized
        for verb in ("先", "优先", "打", "清", "带", "换", "堆", "走", "躲", "断")
    ):
        return True
    return marker_hits >= 2


def _infer_topic(text: str) -> str:
    patterns = [
        r"(boss\s*[a-z0-9_-]*)",
        r"([A-Za-z0-9_+-]+\s*(?:boss|Boss))",
        r"([\u4e00-\u9fffA-Za-z0-9_+-]{1,16}(?:二阶段|一阶段|三阶段|P1|P2|p1|p2))",
        r"([\u4e00-\u9fffA-Za-z0-9_+-]{1,16}(?:配装|出装|打法|路线|机制))",
    ]
    for pattern in patterns:
        match = re.search(pattern, text, re.IGNORECASE)
        if match:
            return match.group(1).strip()
    compact = re.sub(r"\s+", " ", text).strip()
    return compact[:24] or "未命名攻略"


def _infer_category(text: str) -> str:
    lower = text.lower()
    if "boss" in lower or "阶段" in lower or "小怪" in lower or "机制" in lower:
        return "boss"
    if "配装" in lower or "出装" in lower or "装备" in lower or "build" in lower:
        return "build"
    if "路线" in lower or "走" in lower:
        return "route"
    if "版本" in lower or "改了" in lower or "过期" in lower:
        return "version"
    if "http://" in lower or "https://" in lower or "wiki" in lower:
        return "resource"
    return "strategy"


def _infer_version_hint(text: str) -> str:
    match = re.search(r"(?:v|版本)\s*([0-9][0-9A-Za-z_.-]*)", text, re.IGNORECASE)
    if match:
        return match.group(0).strip()
    if "新版本" in text or "版本" in text:
        return "version_mentioned"
    return ""


def _infer_entities(text: str) -> list[str]:
    entities: list[str] = []
    for pattern in (r"boss\s*[a-z0-9_-]*", r"[Pp][123]", r"[一二三]阶段"):
        for match in re.finditer(pattern, text, re.IGNORECASE):
            value = match.group(0).strip()
            if value and value not in entities:
                entities.append(value)
    return entities[:8]
