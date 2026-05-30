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

        outbox_events_raw, error = self._read_list(
            "/v1/outbox-events",
            {"limit": safe_event_limit},
        )
        if error:
            errors.append({"endpoint": "outbox-events", "error": error})
            outbox_events_raw = []
        else:
            successful_reads += 1

        delivery_adapters_raw, error = self._read_list("/v1/delivery-adapters")
        if error:
            errors.append({"endpoint": "delivery-adapters", "error": error})
            delivery_adapters_raw = []
        else:
            successful_reads += 1

        queue_backend_raw, error = self._read_mapping("/v1/queue-backend")
        if error:
            errors.append({"endpoint": "queue-backend", "error": error})
            queue_backend_raw = {}
        else:
            successful_reads += 1

        send_ledger_metrics_raw, error = self._read_mapping(
            "/v1/send-ledger/metrics",
            {"limit": safe_limit},
        )
        if error:
            errors.append({"endpoint": "send-ledger-metrics", "error": error})
            send_ledger_metrics_raw = {}
        else:
            successful_reads += 1

        inbox_metrics_raw, error = self._read_mapping(
            "/v1/inbox-metrics",
            {"limit": safe_limit},
        )
        if error:
            errors.append({"endpoint": "inbox-metrics", "error": error})
            inbox_metrics_raw = {}
        else:
            successful_reads += 1

        agent_job_metrics_raw, error = self._read_mapping(
            "/v1/job-metrics",
            {"job_limit": safe_limit, "event_limit": safe_event_limit},
        )
        if error:
            errors.append({"endpoint": "job-metrics", "error": error})
            agent_job_metrics_raw = {}
        else:
            successful_reads += 1

        outbox_metrics_raw, error = self._read_mapping(
            "/v1/outbox-metrics",
            {"delivery_limit": safe_limit, "event_limit": safe_event_limit},
        )
        if error:
            errors.append({"endpoint": "outbox-metrics", "error": error})
            outbox_metrics_raw = {}
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
        outbox_events = [
            _normalize_outbox_event(item)
            for item in outbox_events_raw
            if isinstance(item, Mapping)
        ]
        delivery_adapters = [
            _normalize_delivery_adapter(item)
            for item in delivery_adapters_raw
            if isinstance(item, Mapping)
        ]
        queue_backend = _normalize_queue_backend(queue_backend_raw)
        send_ledger_metrics = _normalize_send_ledger_metrics(send_ledger_metrics_raw)
        inbox_metrics = _normalize_inbox_metrics(inbox_metrics_raw)
        agent_job_metrics = _normalize_agent_job_metrics(agent_job_metrics_raw)
        outbox_metrics = _normalize_outbox_metrics(outbox_metrics_raw)

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
        enabled_adapters = [item for item in delivery_adapters if item["enabled"]]
        disabled_adapters = [item for item in delivery_adapters if not item["enabled"]]

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
            "outbox_events": len(outbox_events),
            "rag_eval_failures": len(rag_eval_failures),
            "delivery_adapters": len(delivery_adapters),
            "delivery_adapters_enabled": len(enabled_adapters),
            "delivery_adapters_disabled": len(disabled_adapters),
            "queue_backend_provider": queue_backend["provider"],
            "queue_backend_mode": queue_backend["mode"],
            "queue_consumer_concurrency": queue_backend["consumer_concurrency"],
            "queue_max_in_flight": queue_backend["max_in_flight"],
            "queue_external_lease_ready": queue_backend["external_lease_ready"],
            "send_ledger_records": send_ledger_metrics["sampled_records"],
            "send_ledger_repeated_hashes": send_ledger_metrics["repeated_content_hashes"],
            "inbox_metric_events": inbox_metrics["sampled_events"],
            "inbox_metric_observe_only": inbox_metrics["observe_only_total"],
            "inbox_metric_with_attachments": inbox_metrics["with_attachments"],
            "agent_job_metric_events": agent_job_metrics["sampled_events"],
            "agent_job_metric_dead_letters": agent_job_metrics["dead_letters"]["current_total"],
            "outbox_metric_events": outbox_metrics["sampled_events"],
            "outbox_metric_dead_letters": outbox_metrics["dead_letters"]["current_total"],
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
            outbox_events=outbox_events,
            diagnostics=diagnostics,
            rag_eval_failures=rag_eval_failures,
            delivery_adapters=delivery_adapters,
            disabled_adapters=disabled_adapters,
            queue_backend=queue_backend,
            send_ledger_metrics=send_ledger_metrics,
            inbox_metrics=inbox_metrics,
            agent_job_metrics=agent_job_metrics,
            outbox_metrics=outbox_metrics,
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
            "recent_outbox_events": outbox_events,
            "diagnostics": diagnostics,
            "delivery_adapters": delivery_adapters,
            "queue_backend": queue_backend,
            "send_ledger_metrics": send_ledger_metrics,
            "inbox_metrics": inbox_metrics,
            "agent_job_metrics": agent_job_metrics,
            "outbox_metrics": outbox_metrics,
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


def _normalize_outbox_event(item: Mapping[str, Any]) -> dict[str, Any]:
    channel = _mapping_or_empty(item.get("channel"))
    return {
        "event_id": _text(item.get("event_id")),
        "delivery_id": _text(item.get("delivery_id")),
        "channel": channel,
        "channel_kind": _text(channel.get("kind")),
        "account_id": _text(channel.get("account_id")),
        "conversation_id": _text(channel.get("conversation_id")),
        "conversation_type": _text(channel.get("conversation_type")),
        "event_type": _text(item.get("event_type")),
        "status": _text(item.get("status")),
        "attempt": _int_value(item.get("attempt"), fallback=0),
        "max_attempts": _int_value(item.get("max_attempts"), fallback=0),
        "lease_owner": _text(item.get("lease_owner")),
        "lease_expires_at": _text(item.get("lease_expires_at")),
        "error_kind": _text(item.get("error_kind")),
        "error_message": _text(item.get("error_message")),
        "occurred_at": _text(item.get("occurred_at")),
        "metadata": _mapping_or_empty(item.get("metadata")),
    }


def _normalize_delivery_adapter(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "provider": _text(item.get("provider")),
        "channel": _text(item.get("channel")),
        "transport": _text(item.get("transport")),
        "enabled": bool(item.get("enabled")),
        "endpoint_configured": bool(item.get("endpoint_configured")),
        "access_token_configured": bool(item.get("access_token_configured")),
        "endpoint": _text(item.get("endpoint")),
        "notes": [str(value) for value in item.get("notes", [])]
        if isinstance(item.get("notes"), list)
        else [],
    }


def _normalize_queue_backend(item: Mapping[str, Any]) -> dict[str, Any]:
    external_lease = _mapping_or_empty(item.get("external_lease"))
    blockers = item.get("blockers")
    if not isinstance(blockers, list):
        blockers = external_lease.get("blockers")
    if not isinstance(blockers, list):
        blockers = []
    return {
        "provider": _text(item.get("provider") or "unknown"),
        "mode": _text(item.get("mode") or "unknown"),
        "migration_phase": _text(item.get("migration_phase")),
        "stream": _text(item.get("stream")),
        "subject_prefix": _text(item.get("subject_prefix")),
        "external_queue_configured": bool(item.get("external_queue_configured")),
        "external_queue_active": bool(item.get("external_queue_active")),
        "state_store_authoritative": bool(item.get("state_store_authoritative")),
        "lease_owner": _text(item.get("lease_owner")),
        "consumer_model": _text(item.get("consumer_model")),
        "consumer_concurrency": _int_value(item.get("consumer_concurrency"), fallback=0),
        "max_in_flight": _int_value(item.get("max_in_flight"), fallback=0),
        "outbox_queue_source": _text(item.get("outbox_queue_source")),
        "agent_job_queue_source": _text(item.get("agent_job_queue_source")),
        "dsn_configured": bool(item.get("dsn_configured")),
        "recommended_first_backend": _text(item.get("recommended_first_backend")),
        "external_lease_ready": bool(external_lease.get("allow_execution")),
        "external_lease_gate_state": _text(external_lease.get("gate_state")),
        "external_lease_execution_scope": _text(external_lease.get("execution_scope")),
        "external_lease_blockers": [str(value) for value in blockers],
        "shadow_publish": _mapping_or_empty(item.get("shadow_publish")),
        "dual_read_compare": _mapping_or_empty(item.get("dual_read_compare")),
        "external_lease": external_lease,
        "notes": [str(value) for value in item.get("notes", [])]
        if isinstance(item.get("notes"), list)
        else [],
    }


def _normalize_send_ledger_metrics(item: Mapping[str, Any]) -> dict[str, Any]:
    repeated = item.get("repeated_hashes")
    if not isinstance(repeated, list):
        repeated = []
    recent = item.get("recent")
    if not isinstance(recent, list):
        recent = []
    return {
        "sampled_records": _int_value(item.get("sampled_records"), fallback=0),
        "unique_bots": _int_value(item.get("unique_bots"), fallback=0),
        "unique_conversations": _int_value(item.get("unique_conversations"), fallback=0),
        "unique_content_hashes": _int_value(item.get("unique_content_hashes"), fallback=0),
        "repeated_content_hashes": _int_value(
            item.get("repeated_content_hashes"),
            fallback=0,
        ),
        "records_by_bot": _mapping_or_empty(item.get("records_by_bot")),
        "records_by_conversation": _mapping_or_empty(item.get("records_by_conversation")),
        "repeated_hashes": [
            dict(value) for value in repeated if isinstance(value, Mapping)
        ],
        "recent": [dict(value) for value in recent if isinstance(value, Mapping)],
    }


def _normalize_inbox_metrics(item: Mapping[str, Any]) -> dict[str, Any]:
    recent = item.get("recent")
    if not isinstance(recent, list):
        recent = []
    return {
        "sampled_events": _int_value(item.get("sampled_events"), fallback=0),
        "observe_only_total": _int_value(item.get("observe_only_total"), fallback=0),
        "reply_eligible_total": _int_value(item.get("reply_eligible_total"), fallback=0),
        "with_attachments": _int_value(item.get("with_attachments"), fallback=0),
        "attachment_count": _int_value(item.get("attachment_count"), fallback=0),
        "unique_senders": _int_value(item.get("unique_senders"), fallback=0),
        "events_by_channel_kind": _mapping_or_empty(item.get("events_by_channel_kind")),
        "events_by_conversation": _mapping_or_empty(item.get("events_by_conversation")),
        "events_by_decision_action": _mapping_or_empty(
            item.get("events_by_decision_action")
        ),
        "events_by_sender_kind": _mapping_or_empty(item.get("events_by_sender_kind")),
        "recent": [dict(value) for value in recent if isinstance(value, Mapping)],
    }


def _normalize_agent_job_metrics(item: Mapping[str, Any]) -> dict[str, Any]:
    throughput = _mapping_or_empty(item.get("throughput"))
    dead_letters = _mapping_or_empty(item.get("dead_letters"))
    recent = dead_letters.get("recent")
    if not isinstance(recent, list):
        recent = []
    return {
        "sampled_jobs": _int_value(item.get("sampled_jobs"), fallback=0),
        "sampled_events": _int_value(item.get("sampled_events"), fallback=0),
        "jobs_by_status": _mapping_or_empty(item.get("jobs_by_status")),
        "jobs_by_type": _mapping_or_empty(item.get("jobs_by_type")),
        "throughput": {
            "events_by_type": _mapping_or_empty(throughput.get("events_by_type")),
            "created": _int_value(throughput.get("created"), fallback=0),
            "leased": _int_value(throughput.get("leased"), fallback=0),
            "renewed": _int_value(throughput.get("renewed"), fallback=0),
            "running": _int_value(throughput.get("running"), fallback=0),
            "succeeded": _int_value(throughput.get("succeeded"), fallback=0),
            "failed": _int_value(throughput.get("failed"), fallback=0),
            "retry": _int_value(throughput.get("retry"), fallback=0),
            "lease_expired": _int_value(throughput.get("lease_expired"), fallback=0),
            "cancelled": _int_value(throughput.get("cancelled"), fallback=0),
            "terminal_events": _int_value(throughput.get("terminal_events"), fallback=0),
        },
        "dead_letters": {
            "current_total": _int_value(dead_letters.get("current_total"), fallback=0),
            "by_type": _mapping_or_empty(dead_letters.get("by_type")),
            "recent": [dict(item) for item in recent if isinstance(item, Mapping)],
        },
        "notes": [str(value) for value in item.get("notes", [])]
        if isinstance(item.get("notes"), list)
        else [],
    }


def _normalize_outbox_metrics(item: Mapping[str, Any]) -> dict[str, Any]:
    throughput = _mapping_or_empty(item.get("throughput"))
    dead_letters = _mapping_or_empty(item.get("dead_letters"))
    recent = dead_letters.get("recent")
    if not isinstance(recent, list):
        recent = []
    return {
        "sampled_deliveries": _int_value(item.get("sampled_deliveries"), fallback=0),
        "sampled_events": _int_value(item.get("sampled_events"), fallback=0),
        "deliveries_by_status": _mapping_or_empty(item.get("deliveries_by_status")),
        "deliveries_by_channel_kind": _mapping_or_empty(
            item.get("deliveries_by_channel_kind")
        ),
        "throughput": {
            "events_by_type": _mapping_or_empty(throughput.get("events_by_type")),
            "queued": _int_value(throughput.get("queued"), fallback=0),
            "leased": _int_value(throughput.get("leased"), fallback=0),
            "dispatching": _int_value(throughput.get("dispatching"), fallback=0),
            "succeeded": _int_value(throughput.get("succeeded"), fallback=0),
            "failed": _int_value(throughput.get("failed"), fallback=0),
            "retry": _int_value(throughput.get("retry"), fallback=0),
            "dead_lettered": _int_value(throughput.get("dead_lettered"), fallback=0),
            "terminal_events": _int_value(throughput.get("terminal_events"), fallback=0),
        },
        "dead_letters": {
            "current_total": _int_value(dead_letters.get("current_total"), fallback=0),
            "by_channel_kind": _mapping_or_empty(dead_letters.get("by_channel_kind")),
            "recent": [dict(item) for item in recent if isinstance(item, Mapping)],
        },
        "notes": [str(value) for value in item.get("notes", [])]
        if isinstance(item.get("notes"), list)
        else [],
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
    outbox_events: list[dict[str, Any]],
    diagnostics: Mapping[str, Any],
    rag_eval_failures: list[dict[str, Any]],
    delivery_adapters: list[dict[str, Any]],
    disabled_adapters: list[dict[str, Any]],
    queue_backend: dict[str, Any],
    send_ledger_metrics: dict[str, Any],
    inbox_metrics: dict[str, Any],
    agent_job_metrics: dict[str, Any],
    outbox_metrics: dict[str, Any],
) -> list[dict[str, Any]]:
    health_status = _text(health.get("status")) or ("degraded" if errors else "unknown")
    queue_mode = _text(queue_backend.get("mode"))
    queue_provider = _text(queue_backend.get("provider"))
    queue_status = _queue_backend_status(queue_backend)
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
            "agent_job_metrics",
            "Agent Job Metrics",
            summary.get("agent_job_metric_events", 0),
            "danger" if summary.get("agent_job_metric_dead_letters") else "ok",
            {"agent_job_metrics": agent_job_metrics},
        ),
        _card(
            "outbox_metrics",
            "Outbox Metrics",
            summary.get("outbox_metric_events", 0),
            "danger" if summary.get("outbox_metric_dead_letters") else "ok",
            {"outbox_metrics": outbox_metrics},
        ),
        _card(
            "outbox_events",
            "Outbox Events",
            summary.get("outbox_events", 0),
            "ok" if summary.get("outbox_events") else "muted",
            {"recent_events": outbox_events},
        ),
        _card(
            "rag_eval_failures",
            "RAG Eval Failures",
            summary.get("rag_eval_failures", 0),
            "danger" if summary.get("rag_eval_failures") else "ok",
            {"items": rag_eval_failures},
        ),
        _card(
            "delivery_adapters",
            "Delivery Adapters",
            summary.get("delivery_adapters_enabled", 0),
            "warn" if disabled_adapters else ("ok" if delivery_adapters else "muted"),
            {
                "enabled_count": summary.get("delivery_adapters_enabled", 0),
                "disabled_count": summary.get("delivery_adapters_disabled", 0),
                "items": delivery_adapters,
            },
        ),
        _card(
            "queue_backend",
            "Queue Backend",
            f"{queue_provider}/{queue_mode}",
            queue_status,
            {"queue_backend": queue_backend},
        ),
        _card(
            "send_ledger_metrics",
            "Send Ledger Metrics",
            summary.get("send_ledger_records", 0),
            "warn" if summary.get("send_ledger_repeated_hashes") else (
                "ok" if summary.get("send_ledger_records") else "muted"
            ),
            {"send_ledger_metrics": send_ledger_metrics},
        ),
        _card(
            "inbox_metrics",
            "Inbox Metrics",
            summary.get("inbox_metric_events", 0),
            "ok" if summary.get("inbox_metric_events") else "muted",
            {"inbox_metrics": inbox_metrics},
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


def _queue_backend_status(queue_backend: Mapping[str, Any]) -> str:
    provider = _text(queue_backend.get("provider"))
    mode = _text(queue_backend.get("mode"))
    if not provider or provider == "unknown":
        return "muted"
    if provider == "local" or mode == "local_state_store":
        return "ok"
    if bool(queue_backend.get("external_queue_active")):
        return "ok"
    if mode == "external_lease" and not bool(queue_backend.get("external_lease_ready")):
        return "warn"
    if bool(queue_backend.get("external_queue_configured")):
        return "ok"
    return "warn"


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
