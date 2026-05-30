from __future__ import annotations

import json
import os
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Mapping, cast

from fastapi import FastAPI, Query

_DEFAULT_RUNTIME_BASE_URL = "http://127.0.0.1:8780"
_DEFAULT_LIST_LIMIT = 200
_MAX_PAGE_SIZE = 100
_MAX_RUNTIME_LIMIT = 200
_INFRA_FAILURE_STATUSES = {"failed", "dead_lettered", "cancelled"}
_ACTIVE_STATUSES = {"pending", "leased", "running"}


class RagEvalDashboardReader:
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

    def list_evals(
        self,
        *,
        page: int = 1,
        page_size: int = 25,
        q: str = "",
        quality_status: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        safe_page = max(1, page)
        safe_page_size = max(1, min(page_size, _MAX_PAGE_SIZE))
        requested_limit = min(
            max(safe_page * safe_page_size, safe_page_size),
            _MAX_RUNTIME_LIMIT,
        )
        status_meta: dict[str, Any] = {
            "runtime_url": self.runtime_base_url,
            "runtime_available": False,
        }
        raw_items, error = self._read_runtime_jobs(limit=requested_limit)
        if error:
            status_meta["runtime_error"] = error
            items: list[dict[str, Any]] = []
        else:
            status_meta["runtime_available"] = True
            items = [
                _normalize_rag_eval(item)
                for item in raw_items
                if isinstance(item, Mapping)
            ]

        items = [
            item
            for item in items
            if _matches_eval_filters(item, q=q, quality_status=quality_status)
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
            "summary": _summary(items),
            "trend": _trend(items),
            "status": status_meta,
        }

    def _read_runtime_jobs(self, *, limit: int) -> tuple[list[Any], str | None]:
        if not self.runtime_base_url:
            return [], "runtime base url is empty"
        params = {"limit": max(1, min(int(limit or _DEFAULT_LIST_LIMIT), _MAX_RUNTIME_LIMIT)), "type": "rag_eval"}
        url = f"{self.runtime_base_url}/v1/jobs?{urllib.parse.urlencode(params)}"
        data, error = self._request_json(url)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], "runtime /v1/jobs response is not a list"
        return data, None

    def _request_json(
        self,
        url: str,
    ) -> tuple[dict[str, Any] | list[Any] | None, str | None]:
        request = urllib.request.Request(url, method="GET")
        try:
            with urllib.request.urlopen(
                request,
                timeout=self.request_timeout_seconds,
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
    reader = RagEvalDashboardReader(workspace)

    @app.get("/api/dashboard/rag-eval")
    def list_rag_eval_jobs(
        page: int = Query(1, ge=1),
        page_size: int = Query(25, ge=1, le=_MAX_PAGE_SIZE),
        q: str = "",
        quality_status: str = "",
        sort_by: str = "updated_at",
        sort_order: str = "desc",
    ) -> dict[str, Any]:
        return reader.list_evals(
            page=page,
            page_size=page_size,
            q=q,
            quality_status=quality_status,
            sort_by=sort_by,
            sort_order=sort_order,
        )


def _normalize_rag_eval(item: Mapping[str, Any]) -> dict[str, Any]:
    result = _parsed_result(_mapping_or_empty(item.get("result")))
    payload = _parsed_result(_mapping_or_empty(item.get("payload")))
    top1_accuracy = _float_metric(result.get("top1_accuracy"))
    evidence_coverage = _float_metric(result.get("evidence_coverage"))
    min_top1_accuracy = _float_metric(
        result.get("min_top1_accuracy"),
        payload.get("min_top1_accuracy"),
    )
    min_evidence_coverage = _float_metric(
        result.get("min_evidence_coverage"),
        payload.get("min_evidence_coverage"),
    )
    passed = _bool_metric(result.get("passed"))
    lifecycle_status = _text(item.get("status") or "unknown")
    quality_status = _quality_status(
        lifecycle_status=lifecycle_status,
        passed=passed,
        top1_accuracy=top1_accuracy,
        evidence_coverage=evidence_coverage,
    )
    return {
        "job_id": _text(item.get("job_id")),
        "job_type": _text(item.get("job_type") or "rag_eval"),
        "agent_id": _text(item.get("agent_id")),
        "lifecycle_status": lifecycle_status,
        "quality_status": quality_status,
        "passed": passed,
        "questions": _int_metric(result.get("questions")),
        "top1_accuracy": top1_accuracy,
        "evidence_coverage": evidence_coverage,
        "min_top1_accuracy": min_top1_accuracy,
        "min_evidence_coverage": min_evidence_coverage,
        "attempts": _int_value(item.get("attempts"), fallback=0),
        "max_attempts": _int_value(item.get("max_attempts"), fallback=0),
        "lease_owner": _text(item.get("lease_owner")),
        "lease_expires_at": _text(item.get("lease_expires_at")),
        "error_message": _text(item.get("error_message")),
        "payload": payload,
        "result": result,
        "results": _result_rows(result.get("results")),
        "metadata": _mapping_or_empty(item.get("metadata")),
        "created_at": _text(item.get("created_at")),
        "updated_at": _text(item.get("updated_at")),
    }


def _summary(items: list[dict[str, Any]]) -> dict[str, Any]:
    completed = [item for item in items if item.get("lifecycle_status") == "succeeded"]
    passed = [item for item in items if item.get("quality_status") == "passed"]
    failed_quality = [
        item for item in items if item.get("quality_status") == "failed_quality"
    ]
    infra_failed = [
        item for item in items if item.get("quality_status") == "infra_failed"
    ]
    active = [item for item in items if item.get("lifecycle_status") in _ACTIVE_STATUSES]
    latest_completed = completed[0] if completed else None
    metric_items = [
        item
        for item in completed
        if _float_metric(item.get("top1_accuracy")) is not None
        or _float_metric(item.get("evidence_coverage")) is not None
    ]
    return {
        "total": len(items),
        "completed": len(completed),
        "passed": len(passed),
        "failed_quality": len(failed_quality),
        "infra_failed": len(infra_failed),
        "active": len(active),
        "unknown": len(
            [item for item in items if item.get("quality_status") == "unknown"]
        ),
        "pass_rate": len(passed) / len(completed) if completed else None,
        "latest_job_id": latest_completed.get("job_id") if latest_completed else "",
        "latest_passed": latest_completed.get("passed") if latest_completed else None,
        "latest_top1_accuracy": (
            latest_completed.get("top1_accuracy") if latest_completed else None
        ),
        "latest_evidence_coverage": (
            latest_completed.get("evidence_coverage") if latest_completed else None
        ),
        "average_top1_accuracy": _average_metric(metric_items, "top1_accuracy"),
        "average_evidence_coverage": _average_metric(
            metric_items,
            "evidence_coverage",
        ),
    }


def _trend(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    completed = [
        item
        for item in items
        if item.get("lifecycle_status") == "succeeded"
        and (
            _float_metric(item.get("top1_accuracy")) is not None
            or _float_metric(item.get("evidence_coverage")) is not None
        )
    ]
    completed.sort(
        key=lambda item: (
            _parse_timestamp(_text(item.get("updated_at")))
            or datetime.min.replace(tzinfo=timezone.utc),
            _text(item.get("job_id")),
        )
    )
    return [
        {
            "job_id": item.get("job_id"),
            "updated_at": item.get("updated_at"),
            "passed": item.get("passed"),
            "top1_accuracy": item.get("top1_accuracy"),
            "evidence_coverage": item.get("evidence_coverage"),
            "questions": item.get("questions"),
        }
        for item in completed[-50:]
    ]


def _quality_status(
    *,
    lifecycle_status: str,
    passed: bool | None,
    top1_accuracy: float | None,
    evidence_coverage: float | None,
) -> str:
    if lifecycle_status in _INFRA_FAILURE_STATUSES:
        return "infra_failed"
    if lifecycle_status in _ACTIVE_STATUSES:
        return lifecycle_status
    if passed is True:
        return "passed"
    if passed is False:
        return "failed_quality"
    if top1_accuracy is not None or evidence_coverage is not None:
        return "unknown"
    return "unknown"


def _parsed_result(values: Mapping[str, Any]) -> dict[str, Any]:
    parsed: dict[str, Any] = {}
    for key, value in values.items():
        parsed[_text(key)] = _parse_jsonish(value)
    return parsed


def _parse_jsonish(value: Any) -> Any:
    if not isinstance(value, str):
        return value
    text = value.strip()
    if text == "":
        return ""
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return value


def _result_rows(value: Any) -> list[dict[str, Any]]:
    parsed = _parse_jsonish(value)
    if not isinstance(parsed, list):
        return []
    return [dict(item) for item in parsed if isinstance(item, Mapping)]


def _matches_eval_filters(
    item: Mapping[str, Any],
    *,
    q: str,
    quality_status: str,
) -> bool:
    if quality_status and quality_status != _text(item.get("quality_status")):
        return False
    query = str(q).strip().lower()
    if not query:
        return True
    haystack = "\n".join(
        [
            _text(item.get("job_id")),
            _text(item.get("agent_id")),
            _text(item.get("lifecycle_status")),
            _text(item.get("quality_status")),
            _text(item.get("error_message")),
            json.dumps(item.get("payload") or {}, ensure_ascii=False),
            json.dumps(item.get("result") or {}, ensure_ascii=False),
            json.dumps(item.get("metadata") or {}, ensure_ascii=False),
        ]
    ).lower()
    return query in haystack


def _sort_value(item: Mapping[str, Any], key: str) -> object:
    if key in {
        "questions",
        "top1_accuracy",
        "evidence_coverage",
        "min_top1_accuracy",
        "min_evidence_coverage",
    }:
        value = _float_metric(item.get(key))
        return -1.0 if value is None else value
    if key in {"attempts", "max_attempts"}:
        return _int_value(item.get(key), fallback=0)
    if key in {"created_at", "updated_at"}:
        return _parse_timestamp(_text(item.get(key))) or datetime.min.replace(
            tzinfo=timezone.utc
        )
    return _text(item.get(key))


def _average_metric(items: list[dict[str, Any]], key: str) -> float | None:
    values = [_float_metric(item.get(key)) for item in items]
    clean = [value for value in values if value is not None]
    if not clean:
        return None
    return sum(clean) / len(clean)


def _mapping_or_empty(value: object) -> dict[str, Any]:
    return cast(dict[str, Any], dict(value)) if isinstance(value, Mapping) else {}


def _int_metric(value: object) -> int | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    if isinstance(value, str):
        try:
            return int(float(value))
        except ValueError:
            return None
    return None


def _int_value(value: object, *, fallback: int) -> int:
    metric = _int_metric(value)
    return fallback if metric is None else metric


def _float_metric(*values: object) -> float | None:
    for value in values:
        if isinstance(value, bool) or value is None:
            continue
        if isinstance(value, int | float):
            return float(value)
        if isinstance(value, str):
            try:
                return float(value)
            except ValueError:
                continue
    return None


def _bool_metric(value: object) -> bool | None:
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        text = value.strip().lower()
        if text in {"true", "1", "yes", "pass", "passed"}:
            return True
        if text in {"false", "0", "no", "fail", "failed"}:
            return False
    return None


def _parse_timestamp(value: str) -> datetime | None:
    text = value.strip()
    if not text:
        return None
    try:
        parsed = datetime.fromisoformat(text.replace("Z", "+00:00"))
    except ValueError:
        return None
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def _text(value: object) -> str:
    return str(value or "")


def _clean_base_url(value: str) -> str:
    return value.strip().rstrip("/")
