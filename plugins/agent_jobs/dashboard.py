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

_DEFAULT_GATEWAY_BASE_URL = "http://127.0.0.1:8780"
_DEFAULT_LIST_LIMIT = 200
_MAX_PAGE_SIZE = 100


class AgentJobsDashboardReader:
    def __init__(
        self,
        workspace: Path,
        *,
        gateway_base_url: str | None = None,
        request_timeout_seconds: float = 0.5,
    ) -> None:
        self.workspace = workspace
        self.gateway_base_url = _clean_base_url(
            gateway_base_url
            or os.environ.get("AKASHIC_AGENT_RUNTIME_URL", "")
            or os.environ.get("AKASHIC_RUNTIME_BASE_URL", "")
            or os.environ.get("AKASHIC_GATEWAY_BASE_URL", "")
            or os.environ.get("AKASHIC_AGENT_GATEWAY_URL", "")
            or _DEFAULT_GATEWAY_BASE_URL
        )
        self.request_timeout_seconds = max(0.05, float(request_timeout_seconds))

    def list_jobs(
        self,
        *,
        page: int = 1,
        page_size: int = 25,
        job_type: str = "",
        status: str = "",
        q: str = "",
        route_account_id: str = "",
        route_conversation_id: str = "",
        route_kind: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        safe_page = max(1, page)
        safe_page_size = max(1, min(page_size, _MAX_PAGE_SIZE))
        requested_limit = min(max(safe_page * safe_page_size, safe_page_size), _DEFAULT_LIST_LIMIT)
        status_meta: dict[str, Any] = {
            "gateway_url": self.gateway_base_url,
            "gateway_available": False,
        }

        items: list[dict[str, Any]] = []
        items, error = self._read_gateway_jobs(
            limit=requested_limit,
            job_type=job_type,
            status=status,
        )
        if error:
            status_meta["gateway_error"] = error
        else:
            status_meta["gateway_available"] = True

        items = [
            item
            for item in items
            if _matches_job_filters(
                item,
                q=q,
                route_account_id=route_account_id,
                route_conversation_id=route_conversation_id,
                route_kind=route_kind,
            )
        ]
        sort_key = sort_by if sort_by else "updated_at"
        reverse = str(sort_order).lower() != "asc"
        items.sort(
            key=lambda item: (_sort_value(item, sort_key), str(item.get("job_id") or "")),
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

    def get_job(self, job_id: str) -> dict[str, Any] | None:
        job, error = self._get_gateway_job(job_id)
        if error:
            raise HTTPException(status_code=502, detail=error)
        return job

    def list_job_events(
        self,
        job_id: str,
        *,
        limit: int = 50,
        event_type: str = "",
    ) -> dict[str, Any]:
        items, error = self._read_gateway_job_events(
            job_id=job_id,
            limit=limit,
            event_type=event_type,
        )
        status_meta: dict[str, Any] = {
            "gateway_url": self.gateway_base_url,
            "gateway_available": error is None,
        }
        if error:
            status_meta["gateway_error"] = error
            items = []
        return {
            "items": items,
            "total": len(items),
            "job_id": job_id,
            "status": status_meta,
        }

    def retry_job(self, job_id: str) -> dict[str, Any]:
        job, error = self._post_job_action(job_id, "retry")
        if error:
            raise HTTPException(status_code=502, detail=error)
        return job

    def cancel_job(self, job_id: str) -> dict[str, Any]:
        job, error = self._post_job_action(job_id, "cancel")
        if error:
            raise HTTPException(status_code=502, detail=error)
        return job

    def recover_expired_jobs(self, *, limit: int = 50) -> dict[str, Any]:
        if not self.gateway_base_url:
            raise HTTPException(status_code=503, detail="agent-runtime 未配置")
        url = f"{self.gateway_base_url}/v1/jobs/recover-expired"
        data, error = self._request_json(
            url,
            method="POST",
            body={"limit": max(1, min(int(limit or 50), _DEFAULT_LIST_LIMIT))},
        )
        if error:
            raise HTTPException(status_code=502, detail=error)
        if not isinstance(data, Mapping):
            raise HTTPException(
                status_code=502,
                detail="gateway recover-expired response is not an object",
            )
        return _normalize_agent_job_recovery(data)

    def _read_gateway_jobs(
        self,
        *,
        limit: int,
        job_type: str,
        status: str,
    ) -> tuple[list[dict[str, Any]], str | None]:
        if not self.gateway_base_url:
            return [], "gateway base url is empty"
        params = {"limit": max(1, limit)}
        if job_type:
            params["type"] = job_type
        if status:
            params["status"] = status
        url = f"{self.gateway_base_url}/v1/jobs?{urllib.parse.urlencode(params)}"
        data, error = self._request_json(url)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], "gateway /v1/jobs response is not a list"
        return [
            _normalize_agent_job(item) for item in data if isinstance(item, Mapping)
        ], None

    def _get_gateway_job(self, job_id: str) -> tuple[dict[str, Any] | None, str | None]:
        url = f"{self.gateway_base_url}/v1/jobs/{job_id}"
        data, error = self._request_json(url)
        if error:
            return None, error
        if not isinstance(data, Mapping):
            return None, "gateway /v1/jobs/{job_id} response is not an object"
        return _normalize_agent_job(data), None

    def _read_gateway_job_events(
        self,
        *,
        job_id: str,
        limit: int,
        event_type: str,
    ) -> tuple[list[dict[str, Any]], str | None]:
        if not self.gateway_base_url:
            return [], "gateway base url is empty"
        params = {
            "job_id": job_id,
            "limit": max(1, min(int(limit or 50), _DEFAULT_LIST_LIMIT)),
        }
        if event_type:
            params["event"] = event_type
        url = f"{self.gateway_base_url}/v1/job-events?{urllib.parse.urlencode(params)}"
        data, error = self._request_json(url)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], "gateway /v1/job-events response is not a list"
        return [
            _normalize_agent_job_event(item)
            for item in data
            if isinstance(item, Mapping)
        ], None

    def _post_job_action(
        self,
        job_id: str,
        action: str,
    ) -> tuple[dict[str, Any], str | None]:
        url = f"{self.gateway_base_url}/v1/jobs/{job_id}/{action}"
        data, error = self._request_json(url, method="POST", body={})
        if error:
            return {}, error
        if not isinstance(data, Mapping):
            return {}, "gateway job action response is not an object"
        return _normalize_agent_job(data), None

    def _request_json(
        self,
        url: str,
        *,
        method: str = "GET",
        body: dict[str, Any] | None = None,
    ) -> tuple[dict[str, Any] | list[Any] | None, str | None]:
        headers = {"Content-Type": "application/json"}
        payload = json.dumps(body or {}).encode("utf-8")
        request = urllib.request.Request(
            url,
            method=method,
            data=payload if method == "POST" else None,
            headers=headers if method == "POST" else {},
        )
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
    reader = AgentJobsDashboardReader(workspace)

    @app.get("/api/dashboard/agent-jobs")
    def list_agent_jobs(
        page: int = Query(1, ge=1),
        page_size: int = Query(25, ge=1, le=_MAX_PAGE_SIZE),
        job_type: str = "",
        status: str = "",
        q: str = "",
        route_account_id: str = "",
        route_conversation_id: str = "",
        route_kind: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        return reader.list_jobs(
            page=page,
            page_size=page_size,
            job_type=job_type,
            status=status,
            q=q,
            route_account_id=route_account_id,
            route_conversation_id=route_conversation_id,
            route_kind=route_kind,
            sort_by=sort_by,
            sort_order=sort_order,
        )

    @app.get("/api/dashboard/agent-jobs/{job_id:path}/events")
    def list_agent_job_events(
        job_id: str,
        limit: int = Query(50, ge=1, le=_DEFAULT_LIST_LIMIT),
        event_type: str = "",
    ) -> dict[str, Any]:
        return reader.list_job_events(
            job_id,
            limit=limit,
            event_type=event_type,
        )

    @app.get("/api/dashboard/agent-jobs/{job_id:path}")
    def get_agent_job(job_id: str) -> dict[str, Any]:
        item = reader.get_job(job_id)
        if item is None:
            raise HTTPException(status_code=404, detail="agent job 不存在")
        return item

    @app.post("/api/dashboard/agent-jobs/recover-expired")
    def recover_expired_agent_jobs(
        payload: Mapping[str, Any] | None = None,
    ) -> dict[str, Any]:
        body = dict(payload or {})
        return reader.recover_expired_jobs(
            limit=_int_value(body.get("limit"), fallback=50),
        )

    @app.post("/api/dashboard/agent-jobs/{job_id:path}/retry")
    def retry_agent_job(job_id: str) -> dict[str, Any]:
        return reader.retry_job(job_id)

    @app.post("/api/dashboard/agent-jobs/{job_id:path}/cancel")
    def cancel_agent_job(job_id: str) -> dict[str, Any]:
        return reader.cancel_job(job_id)


def _normalize_agent_job(item: Mapping[str, Any]) -> dict[str, Any]:
    route = _mapping_or_empty(item.get("route"))
    return {
        "job_id": _text(item.get("job_id")),
        "job_type": _text(item.get("job_type")),
        "agent_id": _text(item.get("agent_id")),
        "route": route,
        "route_kind": _text(route.get("kind")),
        "route_account_id": _text(route.get("account_id")),
        "route_conversation_id": _text(route.get("conversation_id")),
        "route_conversation_type": _text(route.get("conversation_type")),
        "source_event_ids": list(_string_sequence(item.get("source_event_ids"))),
        "source_asset_ids": list(_string_sequence(item.get("source_asset_ids"))),
        "payload": _mapping_or_empty(item.get("payload")),
        "status": _text(item.get("status") or "unknown"),
        "attempts": _int_value(item.get("attempts"), fallback=0),
        "max_attempts": _int_value(item.get("max_attempts"), fallback=0),
        "lease_owner": _text(item.get("lease_owner")),
        "lease_token_present": bool(_text(item.get("lease_token"))),
        "lease_expires_at": _text(item.get("lease_expires_at")),
        "result": _mapping_or_empty(item.get("result")),
        "error_message": _text(item.get("error_message")),
        "metadata": _mapping_or_empty(item.get("metadata")),
        "created_at": _text(item.get("created_at")),
        "updated_at": _text(item.get("updated_at")),
    }


def _normalize_agent_job_recovery(item: Mapping[str, Any]) -> dict[str, Any]:
    items = item.get("items")
    if not isinstance(items, list):
        items = []
    return {
        "timestamp": _text(item.get("timestamp")),
        "scanned": _int_value(item.get("scanned"), fallback=0),
        "recovered": _int_value(item.get("recovered"), fallback=0),
        "dead_lettered": _int_value(item.get("dead_lettered"), fallback=0),
        "items": [
            _normalize_agent_job_recovery_item(entry)
            for entry in items
            if isinstance(entry, Mapping)
        ],
    }


def _normalize_agent_job_recovery_item(item: Mapping[str, Any]) -> dict[str, Any]:
    job = item.get("job")
    return {
        "action": _text(item.get("action")),
        "job": _normalize_agent_job(job) if isinstance(job, Mapping) else {},
    }


def _normalize_agent_job_event(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "event_id": _text(item.get("event_id")),
        "job_id": _text(item.get("job_id")),
        "job_type": _text(item.get("job_type")),
        "event_type": _text(item.get("event_type")),
        "status": _text(item.get("status")),
        "attempt": _int_value(item.get("attempt"), fallback=0),
        "max_attempts": _int_value(item.get("max_attempts"), fallback=0),
        "lease_owner": _text(item.get("lease_owner")),
        "lease_expires_at": _text(item.get("lease_expires_at")),
        "occurred_at": _text(item.get("occurred_at") or item.get("timestamp")),
        "metadata": _mapping_or_empty(item.get("metadata")),
    }


def _matches_job_filters(
    item: Mapping[str, Any],
    *,
    q: str,
    route_account_id: str,
    route_conversation_id: str,
    route_kind: str,
) -> bool:
    if route_account_id and route_account_id != str(item.get("route_account_id") or item.get("route", {}).get("account_id", "")):
        return False
    if route_conversation_id and route_conversation_id != str(item.get("route_conversation_id") or item.get("route", {}).get("conversation_id", "")):
        return False
    if route_kind and route_kind != str(item.get("route_kind") or item.get("route", {}).get("kind", "")):
        return False
    query = str(q).strip().lower()
    if not query:
        return True
    haystack = "\n".join(
        [
            _text(item.get("job_id")),
            _text(item.get("job_type")),
            _text(item.get("agent_id")),
            _text(item.get("status")),
            _text(item.get("route_kind")),
            _text(item.get("route_account_id")),
            _text(item.get("route_conversation_id")),
            _text(item.get("route_conversation_type")),
            json.dumps(item.get("payload") or {}, ensure_ascii=False),
            json.dumps(item.get("metadata") or {}, ensure_ascii=False),
            json.dumps(item.get("result") or {}, ensure_ascii=False),
        ]
    ).lower()
    return query in haystack


def _sort_value(item: Mapping[str, Any], key: str) -> object:
    if key == "attempts":
        return _int_value(item.get("attempts"), fallback=0)
    if key == "max_attempts":
        return _int_value(item.get("max_attempts"), fallback=0)
    if key in {"created_at", "updated_at"}:
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


def _mapping_or_empty(value: object) -> dict[str, Any]:
    return cast(dict[str, Any], dict(value)) if isinstance(value, Mapping) else {}


def _string_sequence(value: object) -> list[str]:
    if not isinstance(value, list | tuple):
        return []
    return [_text(item) for item in value]


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
