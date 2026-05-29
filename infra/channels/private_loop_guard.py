from __future__ import annotations

import re
import time
from dataclasses import dataclass

OUTBOUND_FORWARD_MARKER = "[转发消息]"
OUTBOUND_IMAGE_MARKER = "[图片]"
OUTBOUND_FILE_MARKER = "[文件]"


def _normalize_guard_text(text: str) -> str:
    return re.sub(r"\s+", " ", str(text or "")).strip()


class RecentPrivateOutboundGuard:
    """Short-lived idempotency guard for private bot-to-bot message echoes."""

    def __init__(self, *, ttl_seconds: float = 180.0) -> None:
        self._ttl_seconds = float(ttl_seconds)
        self._items: list[tuple[float, str, str, str]] = []

    def clear(self) -> None:
        self._items.clear()

    def record(self, from_bot: str, to_user: str, content: str) -> None:
        now = time.monotonic()
        self._prune(now)
        normalized = _normalize_guard_text(content)
        if normalized:
            self._items.append((now, str(from_bot), str(to_user), normalized))

    def was_recent(self, from_bot: str, to_user: str, content: str) -> bool:
        now = time.monotonic()
        self._prune(now)
        normalized = _normalize_guard_text(content)
        return any(
            bot == str(from_bot) and user == str(to_user) and text == normalized
            for _, bot, user, text in self._items
        )

    def is_echo(
        self,
        *,
        from_user: str,
        to_bot: str,
        text: str,
        has_image: bool = False,
    ) -> bool:
        content = str(text or "").strip()
        if content and self.was_recent(from_user, to_bot, content):
            return True
        if has_image and not content:
            return self.was_recent(from_user, to_bot, OUTBOUND_IMAGE_MARKER)
        return False

    def _prune(self, now: float) -> None:
        cutoff = now - self._ttl_seconds
        while self._items and self._items[0][0] < cutoff:
            self._items.pop(0)


@dataclass(frozen=True)
class BotPeerTriggerPolicy:
    """Requires explicit prefixes for configured bot peer accounts."""

    peer_ids: frozenset[str] = frozenset()
    prefixes: tuple[str, ...] = ()

    @classmethod
    def from_config(
        cls,
        *,
        peer_ids: list[str] | None = None,
        prefixes: list[str] | None = None,
    ) -> "BotPeerTriggerPolicy":
        return cls(
            peer_ids=frozenset(
                str(user_id).strip()
                for user_id in (peer_ids or [])
                if str(user_id).strip()
            ),
            prefixes=tuple(
                str(prefix).strip()
                for prefix in (prefixes or [])
                if str(prefix).strip()
            ),
        )

    def normalize_private_text(self, *, user_id: str, text: str) -> str | None:
        raw = str(text or "").strip()
        if str(user_id) not in self.peer_ids:
            return raw
        if not self.prefixes:
            return raw
        for prefix in self.prefixes:
            if raw == prefix:
                return ""
            if raw.startswith(prefix):
                return raw[len(prefix) :].lstrip(" \t:：,，")
        return None


shared_private_outbound_guard = RecentPrivateOutboundGuard()
