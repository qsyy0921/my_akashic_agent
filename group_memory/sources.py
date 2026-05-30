from __future__ import annotations

from typing import Any, Protocol

from session.store import SessionStore


class GroupMessageSource(Protocol):
    def fetch_new_messages(
        self,
        *,
        session_key: str,
        group_id: str,
        after_seq: int,
        limit: int,
    ) -> list[dict[str, Any]]:
        ...


class SessionGroupMessageSource:
    """Compatibility source backed by the Python session store."""

    def __init__(self, session_store: SessionStore) -> None:
        self._session_store = session_store

    def fetch_new_messages(
        self,
        *,
        session_key: str,
        group_id: str,
        after_seq: int,
        limit: int,
    ) -> list[dict[str, Any]]:
        del group_id
        rows = self._session_store.fetch_session_messages(session_key)
        new_rows = [r for r in rows if _row_seq(r) > after_seq]
        return new_rows[: max(1, int(limit))]


def _row_seq(row: dict[str, Any]) -> int:
    try:
        return int(row.get("seq", -1))
    except (TypeError, ValueError):
        return -1
