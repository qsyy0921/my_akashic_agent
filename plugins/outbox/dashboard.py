from __future__ import annotations

import json
import os
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Mapping, cast

from fastapi import FastAPI, HTTPException, Query

_DEFAULT_RUNTIME_BASE_URL = "http://127.0.0.1:8780"
_DEFAULT_LIST_LIMIT = 200
_MAX_PAGE_SIZE = 100
_ACTIONS = {"dispatching", "succeeded", "failed", "retry", "lease-next"}


class OutboxDashboardReader:
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
        self.channel_by_account = _parse_key_value_csv(
            os.environ.get("AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT", "")
        )

    def list_deliveries(
        self,
        *,
        page: int = 1,
        page_size: int = 25,
        status: str = "",
        channel_kind: str = "",
        account_id: str = "",
        conversation_id: str = "",
        conversation_type: str = "",
        q: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        safe_page = max(1, page)
        safe_page_size = max(1, min(page_size, _MAX_PAGE_SIZE))
        requested_limit = min(
            max(safe_page * safe_page_size, safe_page_size), _DEFAULT_LIST_LIMIT
        )
        status_meta: dict[str, Any] = {
            "runtime_url": self.runtime_base_url,
            "runtime_available": False,
        }

        items, error = self._read_runtime_deliveries(limit=requested_limit)
        if error:
            status_meta["runtime_error"] = error
            items = []
        else:
            status_meta["runtime_available"] = True

        items = [
            item
            for item in items
            if _matches_delivery_filters(
                item,
                status=status,
                channel_kind=channel_kind,
                account_id=account_id,
                conversation_id=conversation_id,
                conversation_type=conversation_type,
                q=q,
            )
        ]
        sort_key = sort_by if sort_by else "updated_at"
        reverse = str(sort_order).lower() != "asc"
        items.sort(
            key=lambda item: (
                _sort_value(item, sort_key),
                str(item.get("event_id") or ""),
            ),
            reverse=reverse,
        )
        total = len(items)
        start = (safe_page - 1) * safe_page_size
        return {
            "items": items[start : start + safe_page_size],
            "total": total,
            "page": safe_page,
            "page_size": safe_page_size,
            "status": status_meta,
        }

    def get_delivery(self, event_id: str) -> dict[str, Any] | None:
        if not self.runtime_base_url:
            raise HTTPException(status_code=503, detail="agent-runtime 未配置")
        url = (
            f"{self.runtime_base_url}/v1/outbox/{urllib.parse.quote(event_id, safe='')}"
        )
        data, error = self._request_json(url)
        if error:
            raise HTTPException(status_code=502, detail=error)
        if not isinstance(data, Mapping):
            return None
        delivery = _normalize_delivery(data)
        delivery["dispatch_readiness"] = self._read_dispatch_readiness(event_id)
        return delivery

    def run_action(
        self,
        event_id: str,
        action: str,
        *,
        error_kind: str = "",
        error_message: str = "",
        worker_id: str = "dashboard",
        ttl_seconds: int = 300,
    ) -> dict[str, Any]:
        if action not in _ACTIONS:
            raise HTTPException(status_code=404, detail="unknown outbox action")
        if not self.runtime_base_url:
            raise HTTPException(status_code=503, detail="agent-runtime 未配置")
        body: dict[str, Any] = {}
        if action == "failed":
            body["error_kind"] = error_kind or "validation_error"
            body["error_message"] = error_message or "manual failure from dashboard"
        if action == "lease-next":
            body["worker_id"] = worker_id or "dashboard"
            body["ttl_seconds"] = ttl_seconds if ttl_seconds > 0 else 300
            url = f"{self.runtime_base_url}/v1/outbox/lease-next"
        else:
            url = f"{self.runtime_base_url}/v1/outbox/{urllib.parse.quote(event_id, safe='')}/{action}"
        data, error = self._request_json(url, method="POST", body=body)
        if error:
            raise HTTPException(status_code=502, detail=error)
        if not isinstance(data, Mapping):
            raise HTTPException(
                status_code=502,
                detail="runtime outbox action response is not an object",
            )
        return _normalize_delivery(data)

    def _read_runtime_deliveries(
        self, *, limit: int
    ) -> tuple[list[dict[str, Any]], str | None]:
        if not self.runtime_base_url:
            return [], "runtime base url is empty"
        url = f"{self.runtime_base_url}/v1/outbox?{urllib.parse.urlencode({'limit': max(1, limit)})}"
        data, error = self._request_json(url)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], "runtime /v1/outbox response is not a list"
        return [
            _normalize_delivery(item) for item in data if isinstance(item, Mapping)
        ], None

    def _read_dispatch_readiness(self, event_id: str) -> dict[str, Any]:
        if not self.runtime_base_url:
            return {
                "available": False,
                "error": "runtime base url is empty",
            }
        body: dict[str, Any] = {"event_id": event_id}
        if self.channel_by_account:
            body["channel_by_account"] = dict(self.channel_by_account)
        url = f"{self.runtime_base_url}/v1/delivery-dispatch/readiness"
        data, error = self._request_json(url, method="POST", body=body)
        if error:
            return {
                "available": False,
                "error": error,
                "side_effect": "none",
            }
        if not isinstance(data, Mapping):
            return {
                "available": False,
                "error": "runtime readiness response is not an object",
                "side_effect": "none",
            }
        readiness = _normalize_dispatch_readiness(data)
        readiness["available"] = True
        return readiness

    def _request_json(
        self,
        url: str,
        *,
        method: str = "GET",
        body: dict[str, Any] | None = None,
    ) -> tuple[dict[str, Any] | list[Any] | None, str | None]:
        headers = {"Content-Type": "application/json"}
        raw_body = json.dumps(body or {}).encode("utf-8")
        request = urllib.request.Request(
            url,
            method=method,
            data=raw_body if method == "POST" else None,
            headers=headers if method == "POST" else {},
        )
        try:
            with urllib.request.urlopen(
                request, timeout=self.request_timeout_seconds
            ) as response:
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
    reader = OutboxDashboardReader(workspace)

    @app.get("/api/dashboard/outbox")
    def list_outbox_deliveries(
        page: int = Query(1, ge=1),
        page_size: int = Query(25, ge=1, le=_MAX_PAGE_SIZE),
        status: str = "",
        channel_kind: str = "",
        account_id: str = "",
        conversation_id: str = "",
        conversation_type: str = "",
        q: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        return reader.list_deliveries(
            page=page,
            page_size=page_size,
            status=status,
            channel_kind=channel_kind,
            account_id=account_id,
            conversation_id=conversation_id,
            conversation_type=conversation_type,
            q=q,
            sort_by=sort_by,
            sort_order=sort_order,
        )

    @app.get("/api/dashboard/outbox/{event_id:path}")
    def get_outbox_delivery(event_id: str) -> dict[str, Any]:
        item = reader.get_delivery(event_id)
        if item is None:
            raise HTTPException(status_code=404, detail="outbox delivery 不存在")
        return item

    @app.post("/api/dashboard/outbox/{event_id:path}/{action}")
    def run_outbox_action(
        event_id: str, action: str, payload: Mapping[str, Any] | None = None
    ) -> dict[str, Any]:
        body = dict(payload or {})
        return reader.run_action(
            event_id,
            action,
            error_kind=_text(body.get("error_kind")),
            error_message=_text(body.get("error_message")),
            worker_id=_text(body.get("worker_id")) or "dashboard",
            ttl_seconds=_int_value(body.get("ttl_seconds"), fallback=300),
        )


def _normalize_delivery(item: Mapping[str, Any]) -> dict[str, Any]:
    channel = _mapping_or_empty(item.get("channel"))
    attachments = item.get("attachments")
    if not isinstance(attachments, list):
        attachments = []
    return {
        "event_id": _text(item.get("event_id")),
        "channel": channel,
        "channel_kind": _text(channel.get("kind")),
        "account_id": _text(channel.get("account_id")),
        "conversation_id": _text(channel.get("conversation_id")),
        "conversation_type": _text(channel.get("conversation_type")),
        "content": _text(item.get("content")),
        "attachments": [
            dict(attachment)
            for attachment in attachments
            if isinstance(attachment, Mapping)
        ],
        "status": _text(item.get("status") or "unknown"),
        "attempts": _int_value(item.get("attempts"), fallback=0),
        "max_attempts": _int_value(item.get("max_attempts"), fallback=0),
        "lease_owner": _text(item.get("lease_owner")),
        "lease_expires_at": _text(item.get("lease_expires_at")),
        "error_kind": _text(item.get("error_kind")),
        "error_message": _text(item.get("error_message")),
        "created_at": _text(item.get("created_at")),
        "updated_at": _text(item.get("updated_at")),
        "metadata": _mapping_or_empty(item.get("metadata")),
    }


def _normalize_dispatch_readiness(item: Mapping[str, Any]) -> dict[str, Any]:
    plan = _mapping_or_empty(item.get("plan"))
    missing_channels = item.get("missing_channels")
    if not isinstance(missing_channels, list):
        missing_channels = []
    steps = plan.get("steps")
    if not isinstance(steps, list):
        steps = []
    return {
        "event_id": _text(item.get("event_id")),
        "channel": _text(item.get("channel")),
        "ready": _bool_value(item.get("ready")),
        "reason": _text(item.get("reason")),
        "missing_channels": [
            _text(channel) for channel in missing_channels if _text(channel)
        ],
        "plan": {
            "event_id": _text(plan.get("event_id")),
            "channel": _text(plan.get("channel")),
            "chat_id": _text(plan.get("chat_id")),
            "step_count": _int_value(plan.get("step_count"), fallback=0),
            "steps": [dict(step) for step in steps if isinstance(step, Mapping)],
            "attributes": _mapping_or_empty(plan.get("attributes")),
        },
        "attributes": _mapping_or_empty(item.get("attributes")),
    }


def _matches_delivery_filters(
    item: Mapping[str, Any],
    *,
    status: str,
    channel_kind: str,
    account_id: str,
    conversation_id: str,
    conversation_type: str,
    q: str,
) -> bool:
    if status and status != _text(item.get("status")):
        return False
    if channel_kind and channel_kind != _text(item.get("channel_kind")):
        return False
    if account_id and account_id != _text(item.get("account_id")):
        return False
    if conversation_id and conversation_id != _text(item.get("conversation_id")):
        return False
    if conversation_type and conversation_type != _text(item.get("conversation_type")):
        return False
    query = str(q).strip().lower()
    if not query:
        return True
    haystack = "\n".join(
        [
            _text(item.get("event_id")),
            _text(item.get("channel_kind")),
            _text(item.get("account_id")),
            _text(item.get("conversation_id")),
            _text(item.get("conversation_type")),
            _text(item.get("content")),
            _text(item.get("status")),
            _text(item.get("error_kind")),
            _text(item.get("error_message")),
            json.dumps(item.get("metadata") or {}, ensure_ascii=False),
            json.dumps(item.get("attachments") or [], ensure_ascii=False),
        ]
    ).lower()
    return query in haystack


def _sort_value(item: Mapping[str, Any], key: str) -> object:
    if key in {"created_at", "updated_at"}:
        return _parse_timestamp(_text(item.get(key)) or "1970-01-01T00:00:00Z")
    if key in {"attempts", "max_attempts"}:
        return _int_value(item.get(key), fallback=0)
    return _text(item.get(key))


def _parse_timestamp(value: str) -> datetime:
    try:
        parsed = datetime.fromisoformat(value)
        if parsed.tzinfo is None:
            return parsed.replace(tzinfo=timezone.utc)
        return parsed.astimezone(timezone.utc)
    except ValueError:
        return datetime.min.replace(tzinfo=timezone.utc)


def _mapping_or_empty(value: object) -> dict[str, Any]:
    return cast(dict[str, Any], dict(value)) if isinstance(value, Mapping) else {}


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


def _bool_value(value: object) -> bool:
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        return value.strip().lower() in {"1", "true", "yes", "y", "on"}
    return bool(value)


def _text(value: object) -> str:
    return str(value or "")


def _clean_base_url(value: str) -> str:
    return value.strip().rstrip("/")


def _parse_key_value_csv(value: str) -> dict[str, str]:
    result: dict[str, str] = {}
    for part in value.split(","):
        if "=" not in part:
            continue
        key, raw = part.split("=", 1)
        key = key.strip()
        raw = raw.strip()
        if key and raw:
            result[key] = raw
    return result
