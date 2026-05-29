from __future__ import annotations

import re


_CJK = r"\u4e00-\u9fff"
_TOKEN_RE = re.compile(rf"[a-zA-Z0-9_+.-]+|[{_CJK}]{{2,}}")

_STOPWORDS = {
    "这个",
    "那个",
    "怎么",
    "一下",
    "感觉",
    "可以",
    "不是",
    "就是",
    "今天",
    "明天",
}


def normalize_text(text: str) -> str:
    return re.sub(r"\s+", " ", str(text or "").strip().lower())


def tokenize(text: str) -> list[str]:
    raw = [m.group(0).lower() for m in _TOKEN_RE.finditer(text or "")]
    tokens: list[str] = []
    for token in raw:
        if token in _STOPWORDS:
            continue
        if len(token) == 1 and not token.isdigit():
            continue
        tokens.append(token)
    return tokens


def lexical_similarity(a: str, b: str) -> float:
    a_tokens = set(tokenize(a))
    b_tokens = set(tokenize(b))
    if not a_tokens or not b_tokens:
        return 0.0
    overlap = len(a_tokens & b_tokens)
    union = len(a_tokens | b_tokens)
    score = overlap / union if union else 0.0
    a_norm = normalize_text(a)
    b_norm = normalize_text(b)
    if a_norm and b_norm and (a_norm in b_norm or b_norm in a_norm):
        score = max(score, 0.72)
    return score


def compact_quote(text: str, limit: int = 180) -> str:
    normalized = re.sub(r"\s+", " ", str(text or "")).strip()
    if len(normalized) <= limit:
        return normalized
    return normalized[: max(0, limit - 1)].rstrip() + "…"
