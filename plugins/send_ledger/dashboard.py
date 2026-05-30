from __future__ import annotations

import json
import os
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Mapping

from fastapi import FastAPI, HTTPException, Query

_DEFAULT_RUNTIME_BASE_URL = "http://127.0.0.1:8780"
_DEFAULT_LIST_LIMIT = 200
_MAX_PAGE_SIZE = 100


class SendLedgerDashboardReader:
    def __init__(
        self,
        workspace: Path,
        *,
        runtime_base_url: str | None = None,
        request_timeout_seconds: float = 0.5,
    ) -> None:
        self.workspace = workspace
        self.runtime_base_url = _clean_base_url(
            runtime_base_url
            or os.environ.get("AKASHIC_AGENT_RUNTIME_URL", "")
            or os.environ.get("AKASHIC_RUNTIME_BASE_URL", "")
            or os.environ.get("AKASHIC_GATEWAY_BASE_URL", "")
            or os.environ.get("AKASHIC_AGENT_GATEWAY_URL", "")
            or _DEFAULT_RUNTIME_BASE_URL
        )
        self.request_timeout_seconds = max(0.05, float(request_timeout_seconds))

    def list_records(
        self,
        *,
        page: int = 1,
        page_size: int = 25,
        from_bot_id: str = "",
        conversation_id: str = "",
        content_hash: str = "",
        q: str = "",
        sort_by: str = "timestamp",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        safe_page = max(1, page)
        safe_page_size = max(1, min(page_size, _MAX_PAGE_SIZE))
        requested_limit = min(max(safe_page * safe_page_size, safe_page_size), _DEFAULT_LIST_LIMIT)
        status_meta: dict[str, Any] = {
            "runtime_url": self.runtime_base_url,
            "runtime_available": False,
        }

        records, error = self._read_runtime_records(
            limit=requested_limit,
            from_bot_id=from_bot_id,
            conversation_id=conversation_id,
            content_hash=content_hash,
        )
        if error:
            status_meta["runtime_error"] = error
            records = []
        else:
            status_meta["runtime_available"] = True

        records = [item for item in records if _matches_record_query(item, q=q)]
        sort_key = sort_by if sort_by else "timestamp"
        reverse = str(sort_order).lower() != "asc"
        records.sort(
            key=lambda item: (_sort_value(item, sort_key), str(item.get("content_hash") or "")),
            reverse=reverse,
        )
        total = len(records)
        start = (safe_page - 1) * safe_page_size
        return {
            "items": records[start : start + safe_page_size],
            "total": total,
            "page": safe_page,
            "page_size": safe_page_size,
            "status": status_meta,
        }

    def check_recent(
        self,
        *,
        from_bot_id: str,
        conversation_id: str,
        content: str = "",
        content_hash: str = "",
        window_seconds: int = 60,
    ) -> dict[str, Any]:
        if not self.runtime_base_url:
            raise HTTPException(status_code=503, detail="agent-runtime 未配置")
        params = {
            "from_bot_id": from_bot_id,
            "conversation_id": conversation_id,
            "window_seconds": max(1, min(int(window_seconds), 86400)),
        }
        if content_hash:
            params["content_hash"] = content_hash
        if content:
            params["content"] = content
        url = f"{self.runtime_base_url}/v1/send-ledger/recent?{urllib.parse.urlencode(params)}"
        data, error = self._request_json(url)
        if error:
            raise HTTPException(status_code=502, detail=error)
        if not isinstance(data, Mapping):
            raise HTTPException(status_code=502, detail="runtime recent response is not an object")
        item = _normalize_recent(data)
        item["runtime_url"] = self.runtime_base_url
        return item

    def _read_runtime_records(
        self,
        *,
        limit: int,
        from_bot_id: str,
        conversation_id: str,
        content_hash: str,
    ) -> tuple[list[dict[str, Any]], str | None]:
        if not self.runtime_base_url:
            return [], "runtime base url is empty"
        params = {"limit": max(1, limit)}
        if from_bot_id:
            params["from_bot_id"] = from_bot_id
        if conversation_id:
            params["conversation_id"] = conversation_id
        if content_hash:
            params["content_hash"] = content_hash
        url = f"{self.runtime_base_url}/v1/send-ledger/records?{urllib.parse.urlencode(params)}"
        data, error = self._request_json(url)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], "runtime /v1/send-ledger/records response is not a list"
        return [_normalize_record(item) for item in data if isinstance(item, Mapping)], None

    def _request_json(
        self,
        url: str,
    ) -> tuple[dict[str, Any] | list[Any] | None, str | None]:
        request = urllib.request.Request(url, method="GET")
        try:
            with urllib.request.urlopen(request, timeout=self.request_timeout_seconds) as response:
                raw = response.read().decode("utf-8")
        except (TimeoutError, OSError, urllib.error.URLError) as exc:
            return None, str(exc)
        if not raw.strip():
            return {}, None
        try:
            payload_obj = json.loads(raw)
        except json.JSONDecodeError as exc:
            return None, str(exc)

        if isinstance(payload_obj, dict) and "code" in payload_obj:
            code = payload_obj.get("code")
            if code not in ("OK", "0", 0, None):
                return None, str(payload_obj.get("message") or payload_obj)
            return payload_obj.get("data", payload_obj), None
        return payload_obj, None


def register(app: FastAPI, plugin_dir: Path, workspace: Path) -> None:
    _ = plugin_dir
    reader = SendLedgerDashboardReader(workspace)

    @app.get("/api/dashboard/send-ledger")
    def list_send_ledger_records(
        page: int = Query(1, ge=1),
        page_size: int = Query(25, ge=1, le=_MAX_PAGE_SIZE),
        from_bot_id: str = "",
        conversation_id: str = "",
        content_hash: str = "",
        q: str = "",
        sort_by: str = "timestamp",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        return reader.list_records(
            page=page,
            page_size=page_size,
            from_bot_id=from_bot_id,
            conversation_id=conversation_id,
            content_hash=content_hash,
            q=q,
            sort_by=sort_by,
            sort_order=sort_order,
        )

    @app.get("/api/dashboard/send-ledger/recent")
    def check_recent_send(
        from_bot_id: str,
        conversation_id: str,
        content: str = "",
        content_hash: str = "",
        window_seconds: int = Query(60, ge=1, le=86400),
    ) -> dict[str, Any]:
        return reader.check_recent(
            from_bot_id=from_bot_id,
            conversation_id=conversation_id,
            content=content,
            content_hash=content_hash,
            window_seconds=window_seconds,
        )


def _normalize_record(item: Mapping[str, Any]) -> dict[str, Any]:
    content_hash = _text(item.get("content_hash"))
    return {
        "from_bot_id": _text(item.get("from_bot_id")),
        "conversation_id": _text(item.get("conversation_id")),
        "content_hash": content_hash,
        "short_hash": content_hash[:12],
        "timestamp": _text(item.get("timestamp")),
    }


def _normalize_recent(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "recent": bool(item.get("recent")),
        "from_bot_id": _text(item.get("from_bot_id")),
        "conversation_id": _text(item.get("conversation_id")),
        "content_hash": _text(item.get("content_hash")),
        "window_seconds": _int_value(item.get("window_seconds"), fallback=0),
    }


def _matches_record_query(item: Mapping[str, Any], *, q: str) -> bool:
    query = str(q).strip().lower()
    if not query:
        return True
    haystack = "\n".join(
        [
            _text(item.get("from_bot_id")),
            _text(item.get("conversation_id")),
            _text(item.get("content_hash")),
            _text(item.get("short_hash")),
            _text(item.get("timestamp")),
        ]
    ).lower()
    return query in haystack


def _sort_value(item: Mapping[str, Any], key: str) -> object:
    if key == "timestamp":
        return _parse_timestamp(_text(item.get(key)) or "1970-01-01T00:00:00Z")
    return _text(item.get(key))


def _parse_timestamp(value: str) -> datetime:
    try:
        parsed = datetime.fromisoformat(value)
        if parsed.tzinfo is None:
            return parsed.replace(tzinfo=timezone.utc)
        return parsed.astimezone(timezone.utc)
    except ValueError:
        return datetime.min.replace(tzinfo=timezone.utc)


def _int_value(value: object, *, fallback: int) -> int:
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    if isinstance(value, str):
        try:
            return int(value)
        except ValueError:
            return fallback
    return fallback


def _text(value: object) -> str:
    return str(value or "")


def _clean_base_url(value: str) -> str:
    return value.strip().rstrip("/")
