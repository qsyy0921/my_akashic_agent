from __future__ import annotations

import json
import os
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Mapping, Sequence, cast

from fastapi import FastAPI, Query


_DEFAULT_GATEWAY_BASE_URL = "http://127.0.0.1:8780"
_MAX_PAGE_SIZE = 100
_MAX_GATEWAY_LIMIT = 200


class ShadowAuditDashboardReader:
    def __init__(
        self,
        workspace: Path,
        *,
        gateway_base_url: str | None = None,
        request_timeout_seconds: float = 0.35,
    ) -> None:
        self.workspace = workspace
        self.gateway_base_url = _clean_base_url(
            gateway_base_url
            or os.environ.get("AKASHIC_SHADOW_GATEWAY_URL", "")
            or os.environ.get("AKASHIC_GATEWAY_BASE_URL", "")
            or _DEFAULT_GATEWAY_BASE_URL
        )
        self.request_timeout_seconds = max(0.05, request_timeout_seconds)

    def list_observed(
        self,
        *,
        page: int = 1,
        page_size: int = 25,
        source: str = "auto",
        q: str = "",
        platform: str = "",
        account_id: str = "",
        conversation_id: str = "",
        conversation_type: str = "",
        sort_by: str = "timestamp",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        safe_page = max(1, page)
        safe_size = max(1, min(page_size, _MAX_PAGE_SIZE))
        source_mode = source if source in {"auto", "gateway", "jsonl"} else "auto"
        requested_limit = min(max(safe_page * safe_size, safe_size), _MAX_GATEWAY_LIMIT)
        status: dict[str, Any] = {
            "gateway_url": self.gateway_base_url,
            "gateway_available": False,
            "jsonl_paths": [str(path) for path in self._jsonl_paths()],
        }

        records: list[dict[str, Any]] = []
        if source_mode in {"auto", "gateway"}:
            gateway_records, error = self._read_gateway(limit=requested_limit)
            if error:
                status["gateway_error"] = error
            else:
                status["gateway_available"] = True
                records = gateway_records

        if source_mode == "jsonl" or (source_mode == "auto" and not records):
            records = self._read_jsonl()

        records = [
            item
            for item in records
            if _matches_filters(
                item,
                q=q,
                platform=platform,
                account_id=account_id,
                conversation_id=conversation_id,
                conversation_type=conversation_type,
            )
        ]
        records.sort(
            key=lambda item: (
                _sort_value(item, sort_by),
                str(item.get("event_id") or ""),
            ),
            reverse=sort_order != "asc",
        )
        total = len(records)
        start = (safe_page - 1) * safe_size
        return {
            "items": records[start : start + safe_size],
            "total": total,
            "page": safe_page,
            "page_size": safe_size,
            "source": source_mode,
            "status": status,
        }

    def _read_gateway(self, *, limit: int) -> tuple[list[dict[str, Any]], str | None]:
        if not self.gateway_base_url:
            return [], "gateway base url is empty"
        query = urllib.parse.urlencode({"limit": limit})
        url = f"{self.gateway_base_url}/v1/shadow/observed?{query}"
        try:
            with urllib.request.urlopen(url, timeout=self.request_timeout_seconds) as resp:
                payload = json.loads(resp.read().decode("utf-8"))
        except (OSError, TimeoutError, urllib.error.URLError, json.JSONDecodeError) as exc:
            return [], str(exc)
        data = payload.get("data") if isinstance(payload, Mapping) else None
        if not isinstance(data, list):
            return [], "gateway response does not contain a data list"
        return [
            _normalize_record(cast(Mapping[str, Any], item), source="gateway")
            for item in data
            if isinstance(item, Mapping)
        ], None

    def _read_jsonl(self) -> list[dict[str, Any]]:
        records: list[dict[str, Any]] = []
        seen: set[str] = set()
        for path in self._jsonl_paths():
            if not path.is_file():
                continue
            for line_index, line in enumerate(_tail_lines(path, limit=500)):
                try:
                    raw = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if not isinstance(raw, Mapping):
                    continue
                item = _normalize_record(raw, source="jsonl")
                dedupe_key = str(
                    item.get("event_id")
                    or f"{path}:{line_index}:{item.get('timestamp')}:{item.get('sender_id')}"
                )
                if dedupe_key and dedupe_key in seen:
                    continue
                if dedupe_key:
                    seen.add(dedupe_key)
                records.append(item)
        return records

    def _jsonl_paths(self) -> list[Path]:
        return [
            self.workspace / "shadow" / "session.jsonl",
            self.workspace / "shadow" / "inbound.jsonl",
        ]


def register(app: FastAPI, plugin_dir: Path, workspace: Path) -> None:
    _ = plugin_dir
    reader = ShadowAuditDashboardReader(workspace)

    @app.get("/api/dashboard/shadow-audit/observed")
    def list_shadow_observed(
        page: int = Query(1, ge=1),
        page_size: int = Query(25, ge=1, le=_MAX_PAGE_SIZE),
        source: str = "auto",
        q: str = "",
        platform: str = "",
        account_id: str = "",
        conversation_id: str = "",
        conversation_type: str = "",
        sort_by: str = "timestamp",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        return reader.list_observed(
            page=page,
            page_size=page_size,
            source=source,
            q=q,
            platform=platform,
            account_id=account_id,
            conversation_id=conversation_id,
            conversation_type=conversation_type,
            sort_by=sort_by,
            sort_order=sort_order,
        )


def _normalize_record(record: Mapping[str, Any], *, source: str) -> dict[str, Any]:
    metadata = _mapping_or_empty(record.get("metadata"))
    sender = _mapping_or_empty(record.get("sender"))
    channel = _mapping_or_empty(record.get("channel"))
    attachments = [
        cast(dict[str, Any], dict(item))
        for item in _sequence_or_empty(record.get("attachments"))
        if isinstance(item, Mapping)
    ]
    platform = _text(record.get("platform") or channel.get("platform") or channel.get("kind"))
    account_id = _text(record.get("account_id") or channel.get("account_id"))
    conversation_id = _text(record.get("conversation_id") or channel.get("conversation_id"))
    conversation_type = _text(
        record.get("conversation_type") or channel.get("conversation_type")
    )
    sender_id = _text(record.get("sender_id") or sender.get("id") or metadata.get("sender_id"))
    return {
        "event_id": _text(record.get("event_id")),
        "platform": platform,
        "account_id": account_id,
        "conversation_id": conversation_id,
        "conversation_type": conversation_type,
        "sender_id": sender_id,
        "content": _text(record.get("content")),
        "timestamp": _text(record.get("timestamp")),
        "attachment_count": _int_value(
            record.get("attachment_count"),
            fallback=len(attachments),
        ),
        "decision_action": _text(record.get("decision_action")),
        "decision_reason": _text(record.get("decision_reason")),
        "attachments": attachments,
        "metadata": metadata,
        "source": source,
    }


def _matches_filters(
    item: Mapping[str, Any],
    *,
    q: str,
    platform: str,
    account_id: str,
    conversation_id: str,
    conversation_type: str,
) -> bool:
    query = q.strip().lower()
    if query:
        haystack = "\n".join(
            [
                _text(item.get("event_id")),
                _text(item.get("content")),
                _text(item.get("sender_id")),
                json.dumps(item.get("metadata") or {}, ensure_ascii=False),
            ]
        ).lower()
        if query not in haystack:
            return False
    for key, expected in {
        "platform": platform,
        "account_id": account_id,
        "conversation_id": conversation_id,
        "conversation_type": conversation_type,
    }.items():
        if expected and _text(item.get(key)) != expected:
            return False
    return True


def _sort_value(item: Mapping[str, Any], key: str) -> object:
    if key == "timestamp":
        return _parse_timestamp(_text(item.get("timestamp")))
    if key == "attachment_count":
        return _int_value(item.get("attachment_count"), fallback=0)
    return _text(item.get(key))


def _parse_timestamp(value: str) -> datetime:
    try:
        parsed = datetime.fromisoformat(value)
        if parsed.tzinfo is None:
            return parsed.replace(tzinfo=timezone.utc)
        return parsed.astimezone(timezone.utc)
    except ValueError:
        return datetime.min.replace(tzinfo=timezone.utc)


def _tail_lines(path: Path, *, limit: int) -> list[str]:
    lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
    return lines[-limit:]


def _mapping_or_empty(value: object) -> dict[str, Any]:
    return dict(cast(Mapping[str, Any], value)) if isinstance(value, Mapping) else {}


def _sequence_or_empty(value: object) -> Sequence[object]:
    return cast(Sequence[object], value) if isinstance(value, list | tuple) else []


def _text(value: object) -> str:
    return str(value or "")


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


def _clean_base_url(value: str) -> str:
    return value.strip().rstrip("/")
