from __future__ import annotations

import json
import os
import urllib.error
import urllib.parse
import urllib.request
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Mapping, cast

from fastapi import FastAPI, Query

_DEFAULT_RUNTIME_BASE_URL = "http://127.0.0.1:8780"
_DEFAULT_LIST_LIMIT = 200
_MAX_LIST_LIMIT = 200
_DEFAULT_EVENT_LIMIT = 50
_DEFAULT_STALE_AFTER_SECONDS = 15 * 60
_LEASE_STATUSES = {"leased", "running", "dispatching"}
_DEAD_LETTER_STATUS = "dead_lettered"


class RuntimeOverviewDashboardReader:
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

    def get_overview(
        self,
        *,
        limit: int = _DEFAULT_LIST_LIMIT,
        event_limit: int = _DEFAULT_EVENT_LIMIT,
        stale_after_seconds: int = _DEFAULT_STALE_AFTER_SECONDS,
    ) -> dict[str, Any]:
        safe_limit = _bounded_int(limit, default=_DEFAULT_LIST_LIMIT, maximum=_MAX_LIST_LIMIT)
        safe_event_limit = _bounded_int(
            event_limit,
            default=_DEFAULT_EVENT_LIMIT,
            maximum=_MAX_LIST_LIMIT,
        )
        safe_stale_after = _bounded_int(
            stale_after_seconds,
            default=_DEFAULT_STALE_AFTER_SECONDS,
            maximum=24 * 60 * 60,
        )

        errors: list[dict[str, str]] = []
        successful_reads = 0
        now = datetime.now(timezone.utc)

        health, error = self._read_mapping("/healthz")
        if error:
            errors.append({"endpoint": "healthz", "error": error})
        else:
            successful_reads += 1

        jobs_raw, error = self._read_list("/v1/jobs", {"limit": safe_limit})
        if error:
            errors.append({"endpoint": "jobs", "error": error})
            jobs_raw = []
        else:
            successful_reads += 1

        outbox_raw, error = self._read_list("/v1/outbox", {"limit": safe_limit})
        if error:
            errors.append({"endpoint": "outbox", "error": error})
            outbox_raw = []
        else:
            successful_reads += 1

        checkpoints_raw, error = self._read_list(
            "/v1/knowledge-checkpoints",
            {"limit": safe_limit},
        )
        if error:
            errors.append({"endpoint": "knowledge-checkpoints", "error": error})
            checkpoints_raw = []
        else:
            successful_reads += 1

        diagnostics, error = self._read_mapping(
            "/v1/knowledge-worker-diagnostics",
            {
                "limit": safe_limit,
                "stale_after_seconds": safe_stale_after,
            },
        )
        if error:
            errors.append({"endpoint": "knowledge-worker-diagnostics", "error": error})
            diagnostics = {}
        else:
            successful_reads += 1

        events_raw, error = self._read_list("/v1/job-events", {"limit": safe_event_limit})
        if error:
            errors.append({"endpoint": "job-events", "error": error})
            events_raw = []
        else:
            successful_reads += 1

        jobs = [_normalize_job(item) for item in jobs_raw if isinstance(item, Mapping)]
        outbox = [_normalize_delivery(item) for item in outbox_raw if isinstance(item, Mapping)]
        checkpoints = [
            _normalize_checkpoint(item)
            for item in checkpoints_raw
            if isinstance(item, Mapping)
        ]
        events = [
            _normalize_job_event(item)
            for item in events_raw
            if isinstance(item, Mapping)
        ]

        job_leases = [item for item in jobs if _has_active_lease(item)]
        outbox_leases = [item for item in outbox if _has_active_lease(item)]
        computed_stale = [
            item
            for item in [*jobs, *outbox]
            if _is_stale_leased_item(item, now, safe_stale_after)
        ]
        dead_jobs = [item for item in jobs if item["status"] == _DEAD_LETTER_STATUS]
        dead_outbox = [item for item in outbox if item["status"] == _DEAD_LETTER_STATUS]
        lagged_checkpoints = [
            item for item in checkpoints if _int_or_none(item.get("checkpoint_lag_messages")) is not None
        ]
        checkpoint_lag_max = max(
            [_int_value(item.get("checkpoint_lag_messages"), fallback=0) for item in lagged_checkpoints],
            default=0,
        )
        diagnostic_totals = _mapping_or_empty(diagnostics.get("totals"))
        diagnostic_stale = _int_value(diagnostic_totals.get("stale_leases"), fallback=0)
        stale_jobs_count = max(diagnostic_stale, len(computed_stale))
        rag_eval_failures = [
            item
            for item in jobs
            if item.get("job_type") == "rag_eval" and _is_failed_rag_eval(item)
        ]

        summary = {
            "jobs_total": len(jobs),
            "outbox_total": len(outbox),
            "checkpoints_total": len(checkpoints),
            "worker_leases": len(job_leases) + len(outbox_leases),
            "agent_job_leases": len(job_leases),
            "outbox_leases": len(outbox_leases),
            "stale_jobs": stale_jobs_count,
            "dead_letters": len(dead_jobs) + len(dead_outbox),
            "checkpoint_lag_max": checkpoint_lag_max,
            "job_events": len(events),
            "rag_eval_failures": len(rag_eval_failures),
        }

        cards = _overview_cards(
            health=health,
            summary=summary,
            errors=errors,
            job_leases=job_leases,
            outbox_leases=outbox_leases,
            stale_items=computed_stale,
            dead_jobs=dead_jobs,
            dead_outbox=dead_outbox,
            lagged_checkpoints=lagged_checkpoints,
            events=events,
            diagnostics=diagnostics,
            rag_eval_failures=rag_eval_failures,
        )

        return {
            "summary": summary,
            "cards": cards,
            "jobs_by_status": dict(Counter(item["status"] for item in jobs)),
            "jobs_by_type": dict(Counter(item["job_type"] for item in jobs)),
            "outbox_by_status": dict(Counter(item["status"] for item in outbox)),
            "worker_leases": {
                "agent_jobs": job_leases,
                "outbox": outbox_leases,
            },
            "stale_items": computed_stale,
            "dead_letters": {
                "agent_jobs": dead_jobs,
                "outbox": dead_outbox,
            },
            "checkpoint_lag": lagged_checkpoints,
            "recent_events": events,
            "diagnostics": diagnostics,
            "status": {
                "runtime_url": self.runtime_base_url,
                "runtime_available": successful_reads > 0,
                "health_available": bool(health),
                "partial": bool(errors) and successful_reads > 0,
                "errors": errors,
            },
        }

    def _read_list(
        self,
        path: str,
        params: Mapping[str, object] | None = None,
    ) -> tuple[list[Any], str | None]:
        data, error = self._request_json(path, params)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], f"runtime {path} response is not a list"
        return data, None

    def _read_mapping(
        self,
        path: str,
        params: Mapping[str, object] | None = None,
    ) -> tuple[dict[str, Any], str | None]:
        data, error = self._request_json(path, params)
        if error:
            return {}, error
        if not isinstance(data, Mapping):
            return {}, f"runtime {path} response is not an object"
        return dict(data), None

    def _request_json(
        self,
        path: str,
        params: Mapping[str, object] | None = None,
    ) -> tuple[dict[str, Any] | list[Any] | None, str | None]:
        if not self.runtime_base_url:
            return None, "runtime base url is empty"
        query = urllib.parse.urlencode(
            {key: value for key, value in (params or {}).items() if value is not None}
        )
        url = f"{self.runtime_base_url}{path}"
        if query:
            url = f"{url}?{query}"
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
    reader = RuntimeOverviewDashboardReader(workspace)

    @app.get("/api/dashboard/runtime-overview")
    def get_runtime_overview(
        limit: int = Query(_DEFAULT_LIST_LIMIT, ge=1, le=_MAX_LIST_LIMIT),
        event_limit: int = Query(_DEFAULT_EVENT_LIMIT, ge=1, le=_MAX_LIST_LIMIT),
        stale_after_seconds: int = Query(
            _DEFAULT_STALE_AFTER_SECONDS,
            ge=1,
            le=24 * 60 * 60,
        ),
    ) -> dict[str, Any]:
        return reader.get_overview(
            limit=limit,
            event_limit=event_limit,
            stale_after_seconds=stale_after_seconds,
        )


def _normalize_job(item: Mapping[str, Any]) -> dict[str, Any]:
    route = _mapping_or_empty(item.get("route"))
    return {
        "kind": "agent_job",
        "id": _text(item.get("job_id")),
        "job_id": _text(item.get("job_id")),
        "job_type": _text(item.get("job_type")),
        "agent_id": _text(item.get("agent_id")),
        "route": route,
        "status": _text(item.get("status") or "unknown"),
        "attempts": _int_value(item.get("attempts"), fallback=0),
        "max_attempts": _int_value(item.get("max_attempts"), fallback=0),
        "lease_owner": _text(item.get("lease_owner")),
        "lease_expires_at": _text(item.get("lease_expires_at")),
        "result": _mapping_or_empty(item.get("result")),
        "error_message": _text(item.get("error_message")),
        "metadata": _mapping_or_empty(item.get("metadata")),
        "created_at": _text(item.get("created_at")),
        "updated_at": _text(item.get("updated_at")),
    }


def _normalize_delivery(item: Mapping[str, Any]) -> dict[str, Any]:
    channel = _mapping_or_empty(item.get("channel"))
    return {
        "kind": "outbox",
        "id": _text(item.get("event_id")),
        "event_id": _text(item.get("event_id")),
        "channel": channel,
        "channel_kind": _text(channel.get("kind")),
        "account_id": _text(channel.get("account_id")),
        "conversation_id": _text(channel.get("conversation_id")),
        "conversation_type": _text(channel.get("conversation_type")),
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


def _normalize_checkpoint(item: Mapping[str, Any]) -> dict[str, Any]:
    metadata = _mapping_or_empty(item.get("metadata"))
    cursor = _int_value(item.get("cursor"), fallback=-1)
    latest_source_seq = _int_value(metadata.get("latest_source_seq"), fallback=-1)
    lag: int | None = None
    if cursor >= 0 and latest_source_seq >= 0:
        lag = max(0, latest_source_seq - cursor)
    return {
        "checkpoint_id": _text(item.get("checkpoint_id")),
        "cursor": cursor,
        "updated_at": _text(item.get("updated_at")),
        "metadata": metadata,
        "source": _text(metadata.get("source")),
        "group_id": _text(metadata.get("group_id")),
        "dataset_id": _text(metadata.get("dataset_id")),
        "latest_source_seq": latest_source_seq if latest_source_seq >= 0 else None,
        "checkpoint_lag_messages": lag,
    }


def _normalize_job_event(item: Mapping[str, Any]) -> dict[str, Any]:
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


def _overview_cards(
    *,
    health: Mapping[str, Any],
    summary: Mapping[str, Any],
    errors: list[dict[str, str]],
    job_leases: list[dict[str, Any]],
    outbox_leases: list[dict[str, Any]],
    stale_items: list[dict[str, Any]],
    dead_jobs: list[dict[str, Any]],
    dead_outbox: list[dict[str, Any]],
    lagged_checkpoints: list[dict[str, Any]],
    events: list[dict[str, Any]],
    diagnostics: Mapping[str, Any],
    rag_eval_failures: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    health_status = _text(health.get("status")) or ("degraded" if errors else "unknown")
    return [
        _card(
            "runtime_health",
            "Runtime Health",
            health_status,
            "ok" if health_status == "ok" and not errors else "warn",
            {"health": dict(health), "errors": errors},
        ),
        _card(
            "worker_leases",
            "Worker Leases",
            summary.get("worker_leases", 0),
            "ok" if not summary.get("stale_jobs") else "warn",
            {"agent_jobs": job_leases, "outbox": outbox_leases},
        ),
        _card(
            "stale_jobs",
            "Stale Jobs",
            summary.get("stale_jobs", 0),
            "danger" if summary.get("stale_jobs") else "ok",
            {"items": stale_items, "diagnostics": dict(diagnostics)},
        ),
        _card(
            "dead_letters",
            "Dead Letters",
            summary.get("dead_letters", 0),
            "danger" if summary.get("dead_letters") else "ok",
            {"agent_jobs": dead_jobs, "outbox": dead_outbox},
        ),
        _card(
            "checkpoint_lag",
            "Checkpoint Lag",
            summary.get("checkpoint_lag_max", 0),
            "warn" if summary.get("checkpoint_lag_max") else "ok",
            {"checkpoints": lagged_checkpoints},
        ),
        _card(
            "job_events",
            "Job Events",
            summary.get("job_events", 0),
            "ok" if summary.get("job_events") else "muted",
            {"recent_events": events},
        ),
        _card(
            "rag_eval_failures",
            "RAG Eval Failures",
            summary.get("rag_eval_failures", 0),
            "danger" if summary.get("rag_eval_failures") else "ok",
            {"items": rag_eval_failures},
        ),
    ]


def _card(
    card_id: str,
    label: str,
    value: object,
    status: str,
    detail: Mapping[str, Any],
) -> dict[str, Any]:
    return {
        "id": card_id,
        "label": label,
        "value": value,
        "status": status,
        "detail": dict(detail),
    }


def _has_active_lease(item: Mapping[str, Any]) -> bool:
    return bool(_text(item.get("lease_owner"))) and _text(item.get("status")) in _LEASE_STATUSES


def _is_stale_leased_item(
    item: Mapping[str, Any],
    now: datetime,
    stale_after_seconds: int,
) -> bool:
    if _text(item.get("status")) not in _LEASE_STATUSES:
        return False
    lease_expires_at = _parse_timestamp(_text(item.get("lease_expires_at")))
    if lease_expires_at is not None and now > lease_expires_at:
        return True
    updated_at = _parse_timestamp(_text(item.get("updated_at")))
    if updated_at is None:
        return False
    return (now - updated_at).total_seconds() > max(1, stale_after_seconds)


def _is_failed_rag_eval(item: Mapping[str, Any]) -> bool:
    if _text(item.get("status")) in {"failed", "cancelled", _DEAD_LETTER_STATUS}:
        return True
    result = _mapping_or_empty(item.get("result"))
    passed = result.get("passed")
    if isinstance(passed, bool):
        return not passed
    if isinstance(passed, str):
        return passed.strip().lower() in {"false", "0", "no"}
    return False


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


def _mapping_or_empty(value: object) -> dict[str, Any]:
    return cast(dict[str, Any], dict(value)) if isinstance(value, Mapping) else {}


def _int_or_none(value: object) -> int | None:
    if value is None:
        return None
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    if isinstance(value, str):
        try:
            return int(value)
        except ValueError:
            return None
    return None


def _int_value(value: object, *, fallback: int) -> int:
    converted = _int_or_none(value)
    return fallback if converted is None else converted


def _bounded_int(value: object, *, default: int, maximum: int) -> int:
    converted = _int_or_none(value)
    if converted is None or converted <= 0:
        return default
    return min(converted, maximum)


def _text(value: object) -> str:
    return str(value or "")


def _clean_base_url(value: str) -> str:
    return value.strip().rstrip("/")
