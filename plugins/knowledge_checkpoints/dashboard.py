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


class KnowledgeCheckpointsDashboardReader:
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

    def list_checkpoints(
        self,
        *,
        page: int = 1,
        page_size: int = 25,
        prefix: str = "",
        q: str = "",
        group_id: str = "",
        dataset_id: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        safe_page = max(1, page)
        safe_page_size = max(1, min(page_size, _MAX_PAGE_SIZE))
        requested_limit = min(max(safe_page * safe_page_size, safe_page_size), _DEFAULT_LIST_LIMIT)
        status_meta: dict[str, Any] = {
            "runtime_url": self.runtime_base_url,
            "runtime_available": False,
        }

        items, error = self._read_runtime_checkpoints(
            limit=requested_limit,
            prefix=prefix,
        )
        if error:
            status_meta["runtime_error"] = error
            items = []
        else:
            status_meta["runtime_available"] = True

        items = [
            item
            for item in items
            if _matches_checkpoint_filters(
                item,
                q=q,
                group_id=group_id,
                dataset_id=dataset_id,
            )
        ]
        sort_key = sort_by if sort_by else "updated_at"
        reverse = str(sort_order).lower() != "asc"
        items.sort(
            key=lambda item: (_sort_value(item, sort_key), str(item.get("checkpoint_id") or "")),
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

    def get_checkpoint(self, checkpoint_id: str) -> dict[str, Any] | None:
        if not self.runtime_base_url:
            raise HTTPException(status_code=503, detail="agent-runtime 未配置")
        encoded = urllib.parse.quote(checkpoint_id, safe="")
        url = f"{self.runtime_base_url}/v1/knowledge-checkpoints/{encoded}"
        data, error = self._request_json(url)
        if error:
            raise HTTPException(status_code=502, detail=error)
        if not isinstance(data, Mapping):
            raise HTTPException(status_code=502, detail="runtime checkpoint response is not an object")
        return _normalize_checkpoint(data)

    def _read_runtime_checkpoints(
        self,
        *,
        limit: int,
        prefix: str,
    ) -> tuple[list[dict[str, Any]], str | None]:
        if not self.runtime_base_url:
            return [], "runtime base url is empty"
        params = {"limit": max(1, limit)}
        if prefix:
            params["prefix"] = prefix
        url = f"{self.runtime_base_url}/v1/knowledge-checkpoints?{urllib.parse.urlencode(params)}"
        data, error = self._request_json(url)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], "runtime /v1/knowledge-checkpoints response is not a list"
        return [_normalize_checkpoint(item) for item in data if isinstance(item, Mapping)], None

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
    reader = KnowledgeCheckpointsDashboardReader(workspace)

    @app.get("/api/dashboard/knowledge-checkpoints")
    def list_knowledge_checkpoints(
        page: int = Query(1, ge=1),
        page_size: int = Query(25, ge=1, le=_MAX_PAGE_SIZE),
        prefix: str = "",
        q: str = "",
        group_id: str = "",
        dataset_id: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        return reader.list_checkpoints(
            page=page,
            page_size=page_size,
            prefix=prefix,
            q=q,
            group_id=group_id,
            dataset_id=dataset_id,
            sort_by=sort_by,
            sort_order=sort_order,
        )

    @app.get("/api/dashboard/knowledge-checkpoints/{checkpoint_id:path}")
    def get_knowledge_checkpoint(checkpoint_id: str) -> dict[str, Any]:
        item = reader.get_checkpoint(checkpoint_id)
        if item is None:
            raise HTTPException(status_code=404, detail="knowledge checkpoint 不存在")
        return item


def _normalize_checkpoint(item: Mapping[str, Any]) -> dict[str, Any]:
    metadata = _mapping_or_empty(item.get("metadata"))
    checkpoint_id = _text(item.get("checkpoint_id"))
    parsed = _parse_checkpoint_id(checkpoint_id)
    group_id = _text(metadata.get("group_id")) or parsed.get("group_id", "")
    dataset_id = _text(metadata.get("dataset_id")) or parsed.get("dataset_id", "")
    source = _text(metadata.get("source")) or parsed.get("source", "")
    return {
        "checkpoint_id": checkpoint_id,
        "cursor": _int_value(item.get("cursor"), fallback=-1),
        "updated_at": _text(item.get("updated_at")),
        "metadata": metadata,
        "source": source,
        "group_id": group_id,
        "dataset_id": dataset_id,
        "kind": parsed.get("kind", ""),
        "target": parsed.get("target", ""),
    }


def _parse_checkpoint_id(checkpoint_id: str) -> dict[str, str]:
    parts = checkpoint_id.split(":")
    if len(parts) >= 4 and parts[0] == "ragflow" and parts[1] == "qq":
        return {
            "kind": "ragflow",
            "source": "qq",
            "group_id": parts[2],
            "dataset_id": ":".join(parts[3:]),
            "target": ":".join(parts[3:]),
        }
    if len(parts) >= 3:
        return {
            "kind": parts[0],
            "source": parts[1],
            "target": ":".join(parts[2:]),
        }
    return {"kind": parts[0] if parts else ""}


def _matches_checkpoint_filters(
    item: Mapping[str, Any],
    *,
    q: str,
    group_id: str,
    dataset_id: str,
) -> bool:
    if group_id and group_id != _text(item.get("group_id")):
        return False
    if dataset_id and dataset_id != _text(item.get("dataset_id")):
        return False
    query = str(q).strip().lower()
    if not query:
        return True
    haystack = "\n".join(
        [
            _text(item.get("checkpoint_id")),
            _text(item.get("source")),
            _text(item.get("group_id")),
            _text(item.get("dataset_id")),
            _text(item.get("kind")),
            json.dumps(item.get("metadata") or {}, ensure_ascii=False),
        ]
    ).lower()
    return query in haystack


def _sort_value(item: Mapping[str, Any], key: str) -> object:
    if key == "cursor":
        return _int_value(item.get("cursor"), fallback=-1)
    if key == "updated_at":
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
