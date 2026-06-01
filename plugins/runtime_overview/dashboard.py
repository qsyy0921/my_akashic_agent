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

        go_overview, go_overview_error = self._read_mapping(
            "/v1/runtime-overview",
            {
                "limit": safe_limit,
                "event_limit": safe_event_limit,
                "stale_after_seconds": safe_stale_after,
            },
        )
        if not go_overview_error and go_overview:
            return _normalize_go_runtime_overview(
                go_overview,
                runtime_base_url=self.runtime_base_url,
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

        observe_targets_raw, error = self._read_mapping("/v1/observe-targets")
        if error:
            errors.append({"endpoint": "observe-targets", "error": error})
            observe_targets_raw = {}
        else:
            successful_reads += 1

        observe_capture_raw, error = self._read_mapping(
            "/v1/observe-capture-diagnostics",
            {"limit": safe_limit},
        )
        if error:
            errors.append({"endpoint": "observe-capture-diagnostics", "error": error})
            observe_capture_raw = {}
        else:
            successful_reads += 1

        media_asset_retention_raw, error = self._read_mapping(
            "/v1/media-assets/retention-diagnostics",
            {"limit": safe_limit},
        )
        if error:
            errors.append({"endpoint": "media-asset-retention-diagnostics", "error": error})
            media_asset_retention_raw = {}
        else:
            successful_reads += 1

        media_asset_retention_plan_raw, error = self._read_mapping(
            "/v1/media-assets/retention-plan",
            {"limit": safe_limit},
        )
        if error:
            errors.append({"endpoint": "media-asset-retention-plan", "error": error})
            media_asset_retention_plan_raw = {}
        else:
            successful_reads += 1

        receiver_statuses_raw, error = self._read_mapping("/v1/receiver-statuses")
        if error:
            errors.append({"endpoint": "receiver-statuses", "error": error})
            receiver_statuses_raw = {}
        else:
            successful_reads += 1

        receiver_leases_raw, error = self._read_mapping("/v1/receiver-leases")
        if error:
            errors.append({"endpoint": "receiver-leases", "error": error})
            receiver_leases_raw = {}
        else:
            successful_reads += 1

        scheduler_jobs_raw, error = self._read_mapping(
            "/v1/scheduler/diagnostics",
            {"limit": safe_limit},
        )
        if error:
            errors.append({"endpoint": "scheduler-diagnostics", "error": error})
            scheduler_jobs_raw = {}
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
        observe_targets = _normalize_observe_targets(observe_targets_raw)
        observe_capture = _normalize_observe_capture(observe_capture_raw)
        media_asset_retention = _normalize_media_asset_retention_diagnostics(
            media_asset_retention_raw
        )
        media_asset_retention_plan = _normalize_media_asset_retention_plan(
            media_asset_retention_plan_raw
        )
        media_asset_retention_cleanup = _normalize_media_asset_retention_cleanup(
            {},
            plan=media_asset_retention_plan,
        )
        receiver_statuses = _normalize_receiver_statuses(receiver_statuses_raw)
        receiver_leases = _normalize_receiver_leases(receiver_leases_raw)
        scheduler_jobs = _normalize_scheduler_job_diagnostics(scheduler_jobs_raw)

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
            "observe_targets": observe_targets["totals"]["targets"],
            "observe_targets_enabled": observe_targets["totals"]["enabled"],
            "observe_targets_observe_only": observe_targets["totals"]["observe_only"],
            "observe_targets_reply_allowed": observe_targets["totals"]["reply_allowed"],
            "observe_target_groups": observe_targets["totals"]["groups"],
            "observe_capture_targets": observe_capture["totals"]["targets"],
            "observe_capture_ready": observe_capture["totals"]["ready"],
            "observe_capture_warning": observe_capture["totals"]["warning"],
            "observe_capture_blocked": observe_capture["totals"]["blocked"],
            "observe_capture_text": observe_capture["totals"]["text_covered"],
            "observe_capture_image": observe_capture["totals"]["image_covered"],
            "observe_capture_file": observe_capture["totals"]["file_covered"],
            "observe_capture_content_ready": observe_capture["totals"]["content_ready_assets"],
            "observe_capture_receiver_connected": observe_capture["totals"]["receiver_connected"],
            "observe_capture_receiver_status_connected": observe_capture["totals"].get(
                "receiver_status_connected",
                0,
            ),
            "observe_capture_receiver_activity_recent": observe_capture["totals"].get(
                "receiver_activity_recent",
                0,
            ),
            "media_asset_retention_assets": media_asset_retention["totals"]["assets"],
            "media_asset_retention_cleanup_due": media_asset_retention["totals"]["cleanup_due"],
            "media_asset_retention_permanent": media_asset_retention["totals"]["permanent"],
            "media_asset_retention_default": media_asset_retention["totals"]["default"],
            "media_asset_retention_ephemeral": media_asset_retention["totals"]["ephemeral"],
            "media_asset_retention_unknown": media_asset_retention["totals"]["unknown"],
            "media_asset_retention_plan_ready": media_asset_retention_plan["ready"],
            "media_asset_retention_plan_reason": media_asset_retention_plan["reason"],
            "media_asset_retention_plan_blockers": len(
                media_asset_retention_plan["blockers"]
            ),
            "media_asset_retention_plan_assets": media_asset_retention_plan["asset_count"],
            "media_asset_retention_plan_candidates": media_asset_retention_plan[
                "candidate_count"
            ],
            "media_asset_retention_plan_required_steps": len(
                media_asset_retention_plan["required_steps"]
            ),
            "media_asset_retention_cleanup_ready": media_asset_retention_cleanup[
                "ready"
            ],
            "media_asset_retention_cleanup_reason": media_asset_retention_cleanup[
                "reason"
            ],
            "media_asset_retention_cleanup_blockers": len(
                media_asset_retention_cleanup["blockers"]
            ),
            "media_asset_retention_cleanup_candidates": media_asset_retention_cleanup[
                "candidate_count"
            ],
            "media_asset_retention_cleanup_applied": media_asset_retention_cleanup[
                "totals"
            ]["applied"],
            "media_asset_retention_cleanup_failed": media_asset_retention_cleanup[
                "totals"
            ]["failed"],
            "media_asset_retention_cleanup_recent_audits": len(
                media_asset_retention_cleanup["recent_audits"]
            ),
            "receiver_statuses": receiver_statuses["totals"]["receivers"],
            "receiver_status_connected": receiver_statuses["totals"]["connected"],
            "receiver_status_suspended": receiver_statuses["totals"]["suspended"],
            "receiver_status_failed": receiver_statuses["totals"]["failed"],
            "receiver_status_qq": receiver_statuses["totals"]["qq"],
            "receiver_status_telegram": receiver_statuses["totals"]["telegram"],
            "receiver_leases": receiver_leases["totals"]["leases"],
            "receiver_leases_active": receiver_leases["totals"]["active"],
            "receiver_leases_expired": receiver_leases["totals"]["expired"],
            "receiver_lease_cleanup_required": receiver_leases["totals"]["expired"] > 0,
            "receiver_lease_cleanup_endpoint": "/v1/receiver-leases/cleanup-expired",
            "scheduler_jobs": scheduler_jobs["sampled_jobs"],
            "scheduler_jobs_enabled": scheduler_jobs["enabled_jobs"],
            "scheduler_jobs_disabled": scheduler_jobs["disabled_jobs"],
            "scheduler_jobs_overdue": scheduler_jobs["overdue_jobs"],
            "scheduler_jobs_due_soon": scheduler_jobs["due_soon_jobs"],
            "scheduler_jobs_soft": scheduler_jobs["soft_jobs"],
            "scheduler_jobs_instant": scheduler_jobs["instant_jobs"],
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
            observe_targets=observe_targets,
            observe_capture=observe_capture,
            media_asset_retention=media_asset_retention,
            media_asset_retention_plan=media_asset_retention_plan,
            media_asset_retention_cleanup=media_asset_retention_cleanup,
            receiver_statuses=receiver_statuses,
            receiver_leases=receiver_leases,
            scheduler_jobs=scheduler_jobs,
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
            "observe_targets": observe_targets,
            "observe_capture": observe_capture,
            "media_asset_retention_diagnostics": media_asset_retention,
            "media_asset_retention_plan": media_asset_retention_plan,
            "media_asset_retention_cleanup": media_asset_retention_cleanup,
            "receiver_statuses": receiver_statuses,
            "receiver_leases": receiver_leases,
            "scheduler_jobs": scheduler_jobs,
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

    def get_delivery_adapter_health(
        self,
        *,
        timeout_seconds: int = 3,
    ) -> dict[str, Any]:
        safe_timeout = _bounded_int(timeout_seconds, default=3, maximum=30)
        health, error = self._read_mapping(
            "/v1/delivery-adapters/health",
            {"timeout_seconds": safe_timeout},
            timeout_seconds=float(safe_timeout) + 0.5,
        )
        if error:
            return {
                "items": [],
                "totals": {
                    "adapters": 0,
                    "healthy": 0,
                    "unhealthy": 0,
                    "authenticated": 0,
                },
                "notes": [],
                "status": {
                    "runtime_url": self.runtime_base_url,
                    "available": False,
                    "error": error,
                    "side_effect": "none",
                },
            }
        return _normalize_delivery_adapter_health(
            health,
            runtime_base_url=self.runtime_base_url,
        )

    def get_delivery_smoke_readiness(
        self,
        *,
        group_ids: list[str] | None = None,
        include_synthetic_media: bool = True,
    ) -> dict[str, Any]:
        body = {
            "group_ids": group_ids or [],
            "include_synthetic_media": bool(include_synthetic_media),
        }
        smoke, error = self._post_mapping(
            "/v1/delivery-smoke/readiness",
            body,
            timeout_seconds=max(self.request_timeout_seconds, 5.0),
        )
        if error:
            return {
                "ready": False,
                "reason": "runtime_unavailable",
                "cases": [],
                "totals": {"cases": 0, "ready": 0, "not_ready": 0},
                "blockers": ["runtime_unavailable"],
                "notes": [],
                "status": {
                    "runtime_url": self.runtime_base_url,
                    "available": False,
                    "error": error,
                    "side_effect": "none",
                },
                "side_effect": "none",
            }
        return _normalize_delivery_smoke_readiness(
            smoke,
            runtime_base_url=self.runtime_base_url,
        )

    def _read_list(
        self,
        path: str,
        params: Mapping[str, object] | None = None,
        *,
        timeout_seconds: float | None = None,
    ) -> tuple[list[Any], str | None]:
        data, error = self._request_json(path, params, timeout_seconds=timeout_seconds)
        if error:
            return [], error
        if not isinstance(data, list):
            return [], f"runtime {path} response is not a list"
        return data, None

    def _read_mapping(
        self,
        path: str,
        params: Mapping[str, object] | None = None,
        *,
        timeout_seconds: float | None = None,
    ) -> tuple[dict[str, Any], str | None]:
        data, error = self._request_json(path, params, timeout_seconds=timeout_seconds)
        if error:
            return {}, error
        if not isinstance(data, Mapping):
            return {}, f"runtime {path} response is not an object"
        return dict(data), None

    def _post_mapping(
        self,
        path: str,
        json_body: Mapping[str, object] | None = None,
        *,
        timeout_seconds: float | None = None,
    ) -> tuple[dict[str, Any], str | None]:
        data, error = self._request_json(
            path,
            method="POST",
            json_body=json_body or {},
            timeout_seconds=timeout_seconds,
        )
        if error:
            return {}, error
        if not isinstance(data, Mapping):
            return {}, f"runtime {path} response is not an object"
        return dict(data), None

    def _request_json(
        self,
        path: str,
        params: Mapping[str, object] | None = None,
        *,
        method: str = "GET",
        json_body: Mapping[str, object] | None = None,
        timeout_seconds: float | None = None,
    ) -> tuple[dict[str, Any] | list[Any] | None, str | None]:
        if not self.runtime_base_url:
            return None, "runtime base url is empty"
        query = urllib.parse.urlencode(
            {key: value for key, value in (params or {}).items() if value is not None}
        )
        url = f"{self.runtime_base_url}{path}"
        if query:
            url = f"{url}?{query}"
        data: bytes | None = None
        headers: dict[str, str] = {}
        if json_body is not None:
            data = json.dumps(dict(json_body)).encode("utf-8")
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(
            url,
            data=data,
            headers=headers,
            method=method.upper(),
        )
        try:
            with urllib.request.urlopen(
                request,
                timeout=timeout_seconds or self.request_timeout_seconds,
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

    @app.get("/api/dashboard/runtime-overview/delivery-adapter-health")
    def get_delivery_adapter_health(
        timeout_seconds: int = Query(3, ge=1, le=30),
    ) -> dict[str, Any]:
        return reader.get_delivery_adapter_health(timeout_seconds=timeout_seconds)

    @app.get("/api/dashboard/runtime-overview/delivery-smoke-readiness")
    def get_delivery_smoke_readiness(
        group_ids: str = Query("", max_length=200),
        include_synthetic_media: bool = Query(True),
    ) -> dict[str, Any]:
        return reader.get_delivery_smoke_readiness(
            group_ids=_csv_values(group_ids),
            include_synthetic_media=include_synthetic_media,
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


def _normalize_delivery_adapter_health(
    item: Mapping[str, Any],
    *,
    runtime_base_url: str,
) -> dict[str, Any]:
    items_raw = item.get("items")
    if not isinstance(items_raw, list):
        items_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    items = [
        _normalize_delivery_adapter_health_item(value)
        for value in items_raw
        if isinstance(value, Mapping)
    ]
    totals_raw = _mapping_or_empty(item.get("totals"))
    return {
        "items": items,
        "totals": {
            "adapters": _int_value(totals_raw.get("adapters"), fallback=len(items)),
            "healthy": _int_value(
                totals_raw.get("healthy"),
                fallback=sum(1 for value in items if value["healthy"]),
            ),
            "unhealthy": _int_value(
                totals_raw.get("unhealthy"),
                fallback=sum(1 for value in items if not value["healthy"]),
            ),
            "authenticated": _int_value(
                totals_raw.get("authenticated"),
                fallback=sum(1 for value in items if value["authenticated"]),
            ),
        },
        "notes": [str(value) for value in notes_raw],
        "status": {
            "runtime_url": runtime_base_url,
            "available": True,
            "side_effect": "none",
        },
    }


def _normalize_delivery_adapter_health_item(item: Mapping[str, Any]) -> dict[str, Any]:
    attributes = _mapping_or_empty(item.get("attributes"))
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    return {
        "provider": _text(item.get("provider")),
        "channel": _text(item.get("channel")),
        "transport": _text(item.get("transport")),
        "endpoint": _text(item.get("endpoint")),
        "healthy": bool(item.get("healthy")),
        "reachable": bool(item.get("reachable")),
        "authenticated": bool(item.get("authenticated")),
        "account_id": _text(item.get("account_id")),
        "account_name": _text(item.get("account_name")),
        "error_kind": _text(item.get("error_kind")),
        "error_message": _text(item.get("error_message")),
        "checked_at": _text(item.get("checked_at")),
        "latency_ms": _int_value(item.get("latency_ms"), fallback=0),
        "side_effect": _text(item.get("side_effect") or "none"),
        "attributes": attributes,
        "access_token_present": bool(item.get("access_token_present")),
        "notes": [str(value) for value in notes_raw],
    }


def _normalize_delivery_smoke_readiness(
    item: Mapping[str, Any],
    *,
    runtime_base_url: str,
) -> dict[str, Any]:
    cases_raw = item.get("cases")
    if not isinstance(cases_raw, list):
        cases_raw = []
    cases = [
        _normalize_delivery_smoke_case(value)
        for value in cases_raw
        if isinstance(value, Mapping)
    ]
    totals_raw = _mapping_or_empty(item.get("totals"))
    blockers_raw = item.get("blockers")
    if not isinstance(blockers_raw, list):
        blockers_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    ready_count = sum(1 for value in cases if value["ready"])
    return {
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason")),
        "cases": cases,
        "totals": {
            "cases": _int_value(totals_raw.get("cases"), fallback=len(cases)),
            "ready": _int_value(totals_raw.get("ready"), fallback=ready_count),
            "not_ready": _int_value(
                totals_raw.get("not_ready"),
                fallback=max(0, len(cases) - ready_count),
            ),
        },
        "blockers": [str(value) for value in blockers_raw],
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": [str(value) for value in notes_raw],
        "status": {
            "runtime_url": runtime_base_url,
            "available": True,
            "side_effect": "none",
        },
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_delivery_smoke_case(item: Mapping[str, Any]) -> dict[str, Any]:
    missing_channels = item.get("missing_channels")
    if not isinstance(missing_channels, list):
        missing_channels = []
    return {
        "name": _text(item.get("name")),
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason")),
        "missing_channels": [str(value) for value in missing_channels],
        "plan": _mapping_or_empty(item.get("plan")),
        "attributes": _mapping_or_empty(item.get("attributes")),
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


def _normalize_queue_topology(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "provider": _text(item.get("provider") or "unknown"),
        "mode": _text(item.get("mode") or "unknown"),
        "migration_phase": _text(item.get("migration_phase")),
        "selected_provider": _text(item.get("selected_provider")),
        "recommended_provider": _text(item.get("recommended_provider")),
        "state_store_authoritative": bool(item.get("state_store_authoritative")),
        "external_queue_active": bool(item.get("external_queue_active")),
        "external_lease_ready": bool(item.get("external_lease_ready")),
        "execution_scope": _text(item.get("execution_scope")),
        "nodes": _mapping_list(item.get("nodes")),
        "edges": _mapping_list(item.get("edges")),
        "work_kinds": _mapping_list(item.get("work_kinds")),
        "blockers": _string_list(item.get("blockers")),
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
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


def _normalize_inbound_dedupe_metrics(item: Mapping[str, Any]) -> dict[str, Any]:
    scopes = item.get("scopes")
    if not isinstance(scopes, list):
        scopes = []
    notes = item.get("notes")
    if not isinstance(notes, list):
        notes = []
    totals = _mapping_or_empty(item.get("totals"))
    return {
        "sampled_records": _int_value(item.get("sampled_records"), fallback=0),
        "active_records": _int_value(item.get("active_records"), fallback=0),
        "expired_records": _int_value(item.get("expired_records"), fallback=0),
        "duplicate_records": _int_value(item.get("duplicate_records"), fallback=0),
        "seen_total": _int_value(item.get("seen_total"), fallback=0),
        "duplicate_seen_total": _int_value(
            item.get("duplicate_seen_total"),
            fallback=0,
        ),
        "scopes": [dict(value) for value in scopes if isinstance(value, Mapping)],
        "totals": dict(totals),
        "notes": [str(value) for value in notes],
        "side_effect": _text(item.get("side_effect")),
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


def _normalize_go_runtime_overview(
    item: Mapping[str, Any],
    *,
    runtime_base_url: str,
) -> dict[str, Any]:
    summary = _summary_with_defaults(_mapping_or_empty(item.get("summary")))
    delivery_adapters_raw = item.get("delivery_adapters")
    if not isinstance(delivery_adapters_raw, list):
        delivery_adapters_raw = []
    cards_raw = item.get("cards")
    if not isinstance(cards_raw, list):
        cards_raw = []

    delivery_adapters = [
        _normalize_delivery_adapter(value)
        for value in delivery_adapters_raw
        if isinstance(value, Mapping)
    ]
    queue_backend = _normalize_queue_backend(_mapping_or_empty(item.get("queue_backend")))
    queue_topology = _normalize_queue_topology(_mapping_or_empty(item.get("queue_topology")))
    send_ledger_metrics = _normalize_send_ledger_metrics(
        _mapping_or_empty(item.get("send_ledger_metrics"))
    )
    inbox_metrics = _normalize_inbox_metrics(_mapping_or_empty(item.get("inbox_metrics")))
    inbound_dedupe_metrics = _normalize_inbound_dedupe_metrics(
        _mapping_or_empty(item.get("inbound_dedupe_metrics"))
    )
    agent_job_metrics = _normalize_agent_job_metrics(
        _mapping_or_empty(item.get("agent_job_metrics"))
    )
    outbox_metrics = _normalize_outbox_metrics(_mapping_or_empty(item.get("outbox_metrics")))
    runtime_workers = _normalize_runtime_workers(
        _mapping_or_empty(item.get("runtime_workers"))
    )
    observe_targets = _normalize_observe_targets(
        _mapping_or_empty(item.get("observe_targets"))
    )
    observe_capture = _normalize_observe_capture(
        _mapping_or_empty(item.get("observe_capture"))
    )
    receiver_statuses = _normalize_receiver_statuses(
        _mapping_or_empty(item.get("receiver_statuses"))
    )
    receiver_leases = _normalize_receiver_leases(
        _mapping_or_empty(item.get("receiver_leases"))
    )
    scheduler_jobs = _normalize_scheduler_job_diagnostics(
        _mapping_or_empty(item.get("scheduler_jobs"))
    )
    delivery_smoke_readiness = _normalize_delivery_smoke_readiness(
        _mapping_or_empty(item.get("delivery_smoke_readiness")),
        runtime_base_url=runtime_base_url,
    )
    media_asset_content = _normalize_media_asset_content_diagnostics(
        _mapping_or_empty(item.get("media_asset_content_diagnostics"))
    )
    media_asset_retention = _normalize_media_asset_retention_diagnostics(
        _mapping_or_empty(item.get("media_asset_retention_diagnostics"))
    )
    media_asset_retention_plan = _normalize_media_asset_retention_plan(
        _mapping_or_empty(item.get("media_asset_retention_plan"))
    )
    media_asset_retention_cleanup = _normalize_media_asset_retention_cleanup(
        _mapping_or_empty(item.get("media_asset_retention_cleanup")),
        plan=media_asset_retention_plan,
    )
    agent_job_capacity_plan = _normalize_agent_job_capacity_plan(
        _mapping_or_empty(item.get("agent_job_capacity_plan"))
    )
    agent_job_priority_plan = _normalize_agent_job_priority_plan(
        _mapping_or_empty(item.get("agent_job_priority_plan"))
    )
    agent_job_external_lease_readiness = (
        _normalize_agent_job_external_lease_readiness(
            _mapping_or_empty(item.get("agent_job_external_lease_readiness"))
        )
    )
    agent_job_external_lease_plan = _normalize_agent_job_external_lease_plan(
        _mapping_or_empty(item.get("agent_job_external_lease_plan"))
    )
    outbound_cutover_plan = _normalize_outbound_cutover_plan(
        _mapping_or_empty(item.get("outbound_cutover_plan"))
    )
    control_mutation_policy = _normalize_control_mutation_policy(
        _mapping_or_empty(item.get("control_mutation_policy"))
    )
    operator_approvals = _normalize_operator_approvals(
        _mapping_or_empty(item.get("operator_approvals"))
    )
    control_mutations = _normalize_control_mutations(
        _mapping_or_empty(item.get("control_mutations"))
    )
    knowledge_job_planner_cutover_plan = (
        _normalize_knowledge_job_planner_cutover_plan(
            _mapping_or_empty(item.get("knowledge_job_planner_cutover_plan"))
        )
    )
    runtime_config = _mapping_or_empty(item.get("runtime_config"))
    diagnostics = _mapping_or_empty(item.get("diagnostics"))
    status = _mapping_or_empty(item.get("status"))
    errors_raw = status.get("errors")
    if not isinstance(errors_raw, list):
        errors_raw = []
    errors = [
        {
            "endpoint": _text(_mapping_or_empty(value).get("endpoint")),
            "error": _text(_mapping_or_empty(value).get("error")),
        }
        for value in errors_raw
        if isinstance(value, Mapping)
    ]

    cards = [
        _normalize_go_runtime_card(value)
        for value in cards_raw
        if isinstance(value, Mapping)
    ]

    return {
        "summary": summary,
        "cards": cards,
        "jobs_by_status": agent_job_metrics["jobs_by_status"],
        "jobs_by_type": agent_job_metrics["jobs_by_type"],
        "outbox_by_status": outbox_metrics["deliveries_by_status"],
        "worker_leases": {
            "agent_jobs": [],
            "outbox": [],
        },
        "stale_items": [],
        "dead_letters": {
            "agent_jobs": agent_job_metrics["dead_letters"]["recent"],
            "outbox": outbox_metrics["dead_letters"]["recent"],
        },
        "checkpoint_lag": _checkpoint_lag_from_diagnostics(diagnostics),
        "recent_events": [],
        "recent_outbox_events": [],
        "diagnostics": diagnostics,
        "delivery_adapters": delivery_adapters,
        "delivery_smoke_readiness": delivery_smoke_readiness,
        "queue_backend": queue_backend,
        "queue_topology": queue_topology,
        "runtime_config": dict(runtime_config),
        "runtime_workers": runtime_workers,
        "observe_targets": observe_targets,
        "observe_capture": observe_capture,
        "media_asset_content_diagnostics": media_asset_content,
        "media_asset_retention_diagnostics": media_asset_retention,
        "media_asset_retention_plan": media_asset_retention_plan,
        "media_asset_retention_cleanup": media_asset_retention_cleanup,
        "agent_job_capacity_plan": agent_job_capacity_plan,
        "agent_job_priority_plan": agent_job_priority_plan,
        "agent_job_external_lease_readiness": agent_job_external_lease_readiness,
        "agent_job_external_lease_plan": agent_job_external_lease_plan,
        "outbound_cutover_plan": outbound_cutover_plan,
        "control_mutation_policy": control_mutation_policy,
        "operator_approvals": operator_approvals,
        "control_mutations": control_mutations,
        "knowledge_job_planner_cutover_plan": knowledge_job_planner_cutover_plan,
        "receiver_statuses": receiver_statuses,
        "receiver_leases": receiver_leases,
        "scheduler_jobs": scheduler_jobs,
        "send_ledger_metrics": send_ledger_metrics,
        "inbox_metrics": inbox_metrics,
        "inbound_dedupe_metrics": inbound_dedupe_metrics,
        "agent_job_metrics": agent_job_metrics,
        "outbox_metrics": outbox_metrics,
        "status": {
            "runtime_url": runtime_base_url,
            "runtime_available": bool(status.get("runtime_available", True)),
            "health_available": bool(status.get("health_available", True)),
            "partial": bool(status.get("partial")) or bool(errors),
            "errors": errors,
        },
    }


def _summary_with_defaults(item: Mapping[str, Any]) -> dict[str, Any]:
    summary = dict(item)
    defaults: dict[str, Any] = {
        "jobs_total": 0,
        "outbox_total": 0,
        "checkpoints_total": 0,
        "worker_leases": 0,
        "agent_job_leases": 0,
        "outbox_leases": 0,
        "stale_jobs": 0,
        "dead_letters": 0,
        "checkpoint_lag_max": 0,
        "job_events": 0,
        "outbox_events": 0,
        "rag_eval_failures": 0,
        "delivery_adapters": 0,
        "delivery_adapters_enabled": 0,
        "delivery_adapters_disabled": 0,
        "delivery_smoke_ready": False,
        "delivery_smoke_reason": "",
        "delivery_smoke_cases": 0,
        "delivery_smoke_ready_cases": 0,
        "delivery_smoke_not_ready_cases": 0,
        "delivery_smoke_blockers": 0,
        "queue_backend_provider": "unknown",
        "queue_backend_mode": "unknown",
        "queue_consumer_concurrency": 0,
        "queue_max_in_flight": 0,
        "queue_external_lease_ready": False,
        "queue_topology_nodes": 0,
        "queue_topology_edges": 0,
        "queue_topology_work_kinds": 0,
        "queue_topology_blockers": 0,
        "queue_topology_external_lease_ready": False,
        "queue_topology_outbox_execution_owner": "unknown",
        "queue_topology_agent_job_execution_owner": "unknown",
        "queue_topology_agent_job_ack_owner": "unknown",
        "runtime_config_blockers": 0,
        "runtime_config_onebot_missing": 0,
        "runtime_workers": 0,
        "runtime_workers_enabled": 0,
        "runtime_workers_running": 0,
        "observe_targets": 0,
        "observe_targets_enabled": 0,
        "observe_targets_observe_only": 0,
        "observe_targets_reply_allowed": 0,
        "observe_target_groups": 0,
        "observe_capture_targets": 0,
        "observe_capture_ready": 0,
        "observe_capture_warning": 0,
        "observe_capture_blocked": 0,
        "observe_capture_text": 0,
        "observe_capture_image": 0,
        "observe_capture_file": 0,
        "observe_capture_content_ready": 0,
        "observe_capture_receiver_connected": 0,
        "observe_capture_receiver_status_connected": 0,
        "observe_capture_receiver_activity_recent": 0,
        "media_asset_content_assets": 0,
        "media_asset_content_ready": 0,
        "media_asset_content_forbidden": 0,
        "media_asset_content_unavailable": 0,
        "media_asset_content_disabled": 0,
        "media_asset_content_error": 0,
        "media_asset_retention_assets": 0,
        "media_asset_retention_cleanup_due": 0,
        "media_asset_retention_permanent": 0,
        "media_asset_retention_default": 0,
        "media_asset_retention_ephemeral": 0,
        "media_asset_retention_unknown": 0,
        "media_asset_retention_plan_ready": False,
        "media_asset_retention_plan_reason": "unknown",
        "media_asset_retention_plan_blockers": 0,
        "media_asset_retention_plan_assets": 0,
        "media_asset_retention_plan_candidates": 0,
        "media_asset_retention_plan_required_steps": 0,
        "media_asset_retention_cleanup_ready": False,
        "media_asset_retention_cleanup_reason": "unknown",
        "media_asset_retention_cleanup_blockers": 0,
        "media_asset_retention_cleanup_candidates": 0,
        "media_asset_retention_cleanup_applied": 0,
        "media_asset_retention_cleanup_failed": 0,
        "media_asset_retention_cleanup_recent_audits": 0,
        "agent_job_capacity_ready": False,
        "agent_job_capacity_reason": "unknown",
        "agent_job_capacity_blockers": 0,
        "agent_job_capacity_job_types": 0,
        "agent_job_capacity_mapped_job_types": 0,
        "agent_job_capacity_unmapped_job_types": 0,
        "agent_job_capacity_high_pressure_job_types": 0,
        "agent_job_capacity_blocked_job_types": 0,
        "agent_job_capacity_worker_warning_job_types": 0,
        "agent_job_capacity_active_worker_job_types": 0,
        "agent_job_capacity_stale_worker_job_types": 0,
        "agent_job_capacity_failed_worker_job_types": 0,
        "agent_job_capacity_max_pending": 0,
        "agent_job_capacity_max_active": 0,
        "agent_job_capacity_oldest_pending_age_seconds": 0,
        "agent_job_priority_ready": False,
        "agent_job_priority_reason": "unknown",
        "agent_job_priority_blockers": 0,
        "agent_job_priority_job_types": 0,
        "agent_job_priority_high_priority_job_types": 0,
        "agent_job_priority_blocked_job_types": 0,
        "agent_job_priority_warning_job_types": 0,
        "agent_job_priority_max_priority_score": 0,
        "agent_job_external_lease_ready": False,
        "agent_job_external_lease_reason": "unknown",
        "agent_job_external_lease_blockers": 0,
        "agent_job_external_lease_result_ack_ready": False,
        "agent_job_external_lease_worker_ready": False,
        "agent_job_external_lease_strict_token": False,
        "agent_job_external_lease_execution_owner": "unknown",
        "agent_job_external_lease_execution_scope": "unknown",
        "agent_job_external_lease_plan_ready": False,
        "agent_job_external_lease_plan_decision": "unknown",
        "agent_job_external_lease_plan_blockers": 0,
        "agent_job_external_lease_plan_current_owner": "unknown",
        "agent_job_external_lease_plan_desired_owner": "unknown",
        "agent_job_external_lease_plan_recommended_owner": "unknown",
        "outbound_cutover_plan_ready": False,
        "outbound_cutover_plan_decision": "unknown",
        "outbound_cutover_plan_blockers": 0,
        "outbound_cutover_plan_current_owner": "unknown",
        "outbound_cutover_plan_desired_owner": "unknown",
        "outbound_cutover_plan_recommended_owner": "unknown",
        "control_mutation_policy_allowed": False,
        "control_mutation_policy_reason": "unknown",
        "control_mutation_policy_targets": 0,
        "control_mutation_policy_actions": 0,
        "operator_approvals_total": 0,
        "operator_approvals_active": 0,
        "operator_approvals_approved": 0,
        "operator_approvals_rejected": 0,
        "operator_approvals_revoked": 0,
        "control_mutations_total": 0,
        "control_mutations_planned": 0,
        "control_mutations_applied": 0,
        "control_mutations_failed": 0,
        "control_mutations_rolled_back": 0,
        "knowledge_job_planner_cutover_plan_ready": False,
        "knowledge_job_planner_cutover_plan_decision": "unknown",
        "knowledge_job_planner_cutover_plan_blockers": 0,
        "knowledge_job_planner_cutover_plan_current_owner": "unknown",
        "knowledge_job_planner_cutover_plan_desired_owner": "unknown",
        "knowledge_job_planner_cutover_plan_recommended_owner": "unknown",
        "receiver_statuses": 0,
        "receiver_status_connected": 0,
        "receiver_status_suspended": 0,
        "receiver_status_failed": 0,
        "receiver_status_qq": 0,
        "receiver_status_telegram": 0,
        "receiver_leases": 0,
        "receiver_leases_active": 0,
        "receiver_leases_expired": 0,
        "receiver_lease_cleanup_required": False,
        "receiver_lease_cleanup_endpoint": "/v1/receiver-leases/cleanup-expired",
        "scheduler_jobs": 0,
        "scheduler_jobs_enabled": 0,
        "scheduler_jobs_disabled": 0,
        "scheduler_jobs_overdue": 0,
        "scheduler_jobs_due_soon": 0,
        "scheduler_jobs_soft": 0,
        "scheduler_jobs_instant": 0,
        "send_ledger_records": 0,
        "send_ledger_repeated_hashes": 0,
        "inbox_metric_events": 0,
        "inbox_metric_observe_only": 0,
        "inbox_metric_with_attachments": 0,
        "inbound_dedupe_records": 0,
        "inbound_dedupe_active_records": 0,
        "inbound_dedupe_duplicate_records": 0,
        "inbound_dedupe_seen_total": 0,
        "inbound_dedupe_duplicate_seen_total": 0,
        "inbound_dedupe_scopes": 0,
        "agent_job_metric_events": 0,
        "agent_job_metric_dead_letters": 0,
        "outbox_metric_events": 0,
        "outbox_metric_dead_letters": 0,
    }
    for key, value in defaults.items():
        summary.setdefault(key, value)
    return summary


def _normalize_runtime_workers(item: Mapping[str, Any]) -> dict[str, Any]:
    workers_raw = item.get("workers")
    if not isinstance(workers_raw, list):
        workers_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    return {
        "workers": [
            dict(value)
            for value in workers_raw
            if isinstance(value, Mapping)
        ],
        "totals": {
            "workers": _int_value(_mapping_or_empty(item.get("totals")).get("workers"), fallback=0),
            "enabled": _int_value(_mapping_or_empty(item.get("totals")).get("enabled"), fallback=0),
            "running": _int_value(_mapping_or_empty(item.get("totals")).get("running"), fallback=0),
            "disabled": _int_value(_mapping_or_empty(item.get("totals")).get("disabled"), fallback=0),
        },
        "notes": [str(value) for value in notes_raw],
    }


def _normalize_observe_targets(item: Mapping[str, Any]) -> dict[str, Any]:
    targets_raw = item.get("targets")
    if not isinstance(targets_raw, list):
        targets_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    totals = _mapping_or_empty(item.get("totals"))
    targets = [
        _normalize_observe_target(value)
        for value in targets_raw
        if isinstance(value, Mapping)
    ]
    return {
        "targets": targets,
        "totals": {
            "targets": _int_value(totals.get("targets"), fallback=len(targets)),
            "enabled": _int_value(
                totals.get("enabled"),
                fallback=sum(1 for value in targets if value["enabled"]),
            ),
            "disabled": _int_value(totals.get("disabled"), fallback=0),
            "observe_only": _int_value(
                totals.get("observe_only"),
                fallback=sum(1 for value in targets if value["observe_only"]),
            ),
            "reply_allowed": _int_value(
                totals.get("reply_allowed"),
                fallback=sum(1 for value in targets if value["reply_allowed"]),
            ),
            "groups": _int_value(
                totals.get("groups"),
                fallback=sum(
                    1
                    for value in targets
                    if value["channel"]["conversation_type"] == "group"
                ),
            ),
        },
        "notes": [str(value) for value in notes_raw],
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_observe_target(item: Mapping[str, Any]) -> dict[str, Any]:
    channel = _mapping_or_empty(item.get("channel"))
    return {
        "target_id": _text(item.get("target_id")),
        "channel": {
            "kind": _text(channel.get("kind")),
            "account_id": _text(channel.get("account_id")),
            "conversation_id": _text(channel.get("conversation_id")),
            "conversation_type": _text(channel.get("conversation_type")),
        },
        "observe_only": bool(item.get("observe_only")),
        "reply_allowed": bool(item.get("reply_allowed")),
        "require_at": bool(item.get("require_at")),
        "allow_from": [str(value) for value in item.get("allow_from", [])]
        if isinstance(item.get("allow_from"), list)
        else [],
        "enabled": bool(item.get("enabled")),
        "source": _text(item.get("source")),
        "metadata": _mapping_or_empty(item.get("metadata")),
        "updated_at": _text(item.get("updated_at")),
    }


def _normalize_observe_capture(item: Mapping[str, Any]) -> dict[str, Any]:
    targets_raw = item.get("targets")
    if not isinstance(targets_raw, list):
        targets_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    totals = _mapping_or_empty(item.get("totals"))
    targets = [
        _normalize_observe_capture_target(value)
        for value in targets_raw
        if isinstance(value, Mapping)
    ]
    return {
        "targets": targets,
        "totals": {
            "targets": _int_value(totals.get("targets"), fallback=len(targets)),
            "enabled": _int_value(
                totals.get("enabled"),
                fallback=sum(1 for value in targets if value["enabled"]),
            ),
            "ready": _int_value(
                totals.get("ready"),
                fallback=sum(1 for value in targets if value["status"] == "ok"),
            ),
            "warning": _int_value(
                totals.get("warning"),
                fallback=sum(1 for value in targets if value["status"] == "warn"),
            ),
            "blocked": _int_value(
                totals.get("blocked"),
                fallback=sum(1 for value in targets if value["status"] == "danger"),
            ),
            "receiver_connected": _int_value(totals.get("receiver_connected"), fallback=0),
            "receiver_status_connected": _int_value(
                totals.get("receiver_status_connected"),
                fallback=0,
            ),
            "receiver_activity_recent": _int_value(
                totals.get("receiver_activity_recent"),
                fallback=0,
            ),
            "text_covered": _int_value(totals.get("text_covered"), fallback=0),
            "attachment_covered": _int_value(totals.get("attachment_covered"), fallback=0),
            "image_covered": _int_value(totals.get("image_covered"), fallback=0),
            "file_covered": _int_value(totals.get("file_covered"), fallback=0),
            "media_assets": _int_value(totals.get("media_assets"), fallback=0),
            "image_assets": _int_value(totals.get("image_assets"), fallback=0),
            "file_assets": _int_value(totals.get("file_assets"), fallback=0),
            "content_ready_assets": _int_value(totals.get("content_ready_assets"), fallback=0),
            "content_unavailable_assets": _int_value(totals.get("content_unavailable_assets"), fallback=0),
            "content_forbidden_assets": _int_value(totals.get("content_forbidden_assets"), fallback=0),
            "content_disabled_assets": _int_value(totals.get("content_disabled_assets"), fallback=0),
        },
        "notes": [str(value) for value in notes_raw],
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_observe_capture_target(item: Mapping[str, Any]) -> dict[str, Any]:
    channel = _mapping_or_empty(item.get("channel"))
    coverage = _mapping_or_empty(item.get("coverage"))
    blockers = item.get("blockers")
    if not isinstance(blockers, list):
        blockers = []
    return {
        "target_id": _text(item.get("target_id")),
        "channel": {
            "kind": _text(channel.get("kind")),
            "account_id": _text(channel.get("account_id")),
            "conversation_id": _text(channel.get("conversation_id")),
            "conversation_type": _text(channel.get("conversation_type")),
        },
        "enabled": bool(item.get("enabled")),
        "observe_only": bool(item.get("observe_only")),
        "receiver_connected": bool(item.get("receiver_connected")),
        "receiver_status_connected": bool(item.get("receiver_status_connected")),
        "receiver_activity_recent": bool(item.get("receiver_activity_recent")),
        "receiver_connection_source": _text(item.get("receiver_connection_source")),
        "receiver_id": _text(item.get("receiver_id")),
        "receiver_status": _text(item.get("receiver_status")),
        "status": _text(item.get("status")),
        "blockers": [str(value) for value in blockers],
        "inbox_events": _int_value(item.get("inbox_events"), fallback=0),
        "text_events": _int_value(item.get("text_events"), fallback=0),
        "attachment_events": _int_value(item.get("attachment_events"), fallback=0),
        "attachment_count": _int_value(item.get("attachment_count"), fallback=0),
        "media_assets": _int_value(item.get("media_assets"), fallback=0),
        "image_assets": _int_value(item.get("image_assets"), fallback=0),
        "file_assets": _int_value(item.get("file_assets"), fallback=0),
        "content_ready_assets": _int_value(item.get("content_ready_assets"), fallback=0),
        "content_unavailable_assets": _int_value(item.get("content_unavailable_assets"), fallback=0),
        "content_forbidden_assets": _int_value(item.get("content_forbidden_assets"), fallback=0),
        "content_disabled_assets": _int_value(item.get("content_disabled_assets"), fallback=0),
        "latest_received_at": _text(item.get("latest_received_at")),
        "latest_asset_at": _text(item.get("latest_asset_at")),
        "coverage": {
            "text_seen": bool(coverage.get("text_seen")),
            "attachment_seen": bool(coverage.get("attachment_seen")),
            "image_seen": bool(coverage.get("image_seen")),
            "file_seen": bool(coverage.get("file_seen")),
            "media_content_ready": bool(coverage.get("media_content_ready")),
        },
    }


def _normalize_media_asset_content_diagnostics(item: Mapping[str, Any]) -> dict[str, Any]:
    items_raw = item.get("items")
    if not isinstance(items_raw, list):
        items_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    assets = [
        _normalize_media_asset_content_item(value)
        for value in items_raw
        if isinstance(value, Mapping)
    ]
    totals = _mapping_or_empty(item.get("totals"))
    return {
        "items": assets,
        "totals": {
            "assets": _int_value(totals.get("assets"), fallback=len(assets)),
            "ready": _int_value(
                totals.get("ready"),
                fallback=sum(1 for value in assets if value["content_status"] == "ready"),
            ),
            "forbidden": _int_value(
                totals.get("forbidden"),
                fallback=sum(
                    1 for value in assets if value["content_status"] == "forbidden"
                ),
            ),
            "unavailable": _int_value(
                totals.get("unavailable"),
                fallback=sum(
                    1 for value in assets if value["content_status"] == "unavailable"
                ),
            ),
            "disabled": _int_value(
                totals.get("disabled"),
                fallback=sum(1 for value in assets if value["content_status"] == "disabled"),
            ),
            "error": _int_value(
                totals.get("error"),
                fallback=sum(1 for value in assets if value["content_status"] == "error"),
            ),
        },
        "notes": [str(value) for value in notes_raw],
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_media_asset_content_item(item: Mapping[str, Any]) -> dict[str, Any]:
    channel = _mapping_or_empty(item.get("channel"))
    return {
        "asset_id": _text(item.get("asset_id")),
        "channel": {
            "kind": _text(channel.get("kind")),
            "account_id": _text(channel.get("account_id")),
            "conversation_id": _text(channel.get("conversation_id")),
            "conversation_type": _text(channel.get("conversation_type")),
        },
        "source_message_id": _text(item.get("source_message_id")),
        "sender_id": _text(item.get("sender_id")),
        "kind": _text(item.get("kind")),
        "mime_type": _text(item.get("mime_type")),
        "name": _text(item.get("name")),
        "size_bytes": _int_value(item.get("size_bytes"), fallback=0),
        "content_status": _text(item.get("content_status")),
        "content_reason": _text(item.get("content_reason")),
        "content_endpoint": _text(item.get("content_endpoint")),
        "content_mime_type": _text(item.get("content_mime_type")),
        "content_size_bytes": _int_value(item.get("content_size_bytes"), fallback=0),
        "updated_at": _text(item.get("updated_at")),
    }


def _normalize_media_asset_retention_diagnostics(item: Mapping[str, Any]) -> dict[str, Any]:
    items_raw = item.get("items")
    if not isinstance(items_raw, list):
        items_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    assets = [
        _normalize_media_asset_retention_item(value)
        for value in items_raw
        if isinstance(value, Mapping)
    ]
    totals = _mapping_or_empty(item.get("totals"))
    return {
        "items": assets,
        "totals": {
            "assets": _int_value(totals.get("assets"), fallback=len(assets)),
            "cleanup_due": _int_value(
                totals.get("cleanup_due"),
                fallback=sum(1 for value in assets if value["cleanup_due"]),
            ),
            "permanent": _int_value(
                totals.get("permanent"),
                fallback=sum(
                    1 for value in assets if value["retention_class"] == "permanent"
                ),
            ),
            "default": _int_value(
                totals.get("default"),
                fallback=sum(1 for value in assets if value["retention_class"] == "default"),
            ),
            "ephemeral": _int_value(
                totals.get("ephemeral"),
                fallback=sum(
                    1 for value in assets if value["retention_class"] == "ephemeral"
                ),
            ),
            "unknown": _int_value(
                totals.get("unknown"),
                fallback=sum(1 for value in assets if value["retention_class"] == "unknown"),
            ),
        },
        "notes": [str(value) for value in notes_raw],
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_media_asset_retention_item(item: Mapping[str, Any]) -> dict[str, Any]:
    channel = _mapping_or_empty(item.get("channel"))
    return {
        "asset_id": _text(item.get("asset_id")),
        "channel": {
            "kind": _text(channel.get("kind")),
            "account_id": _text(channel.get("account_id")),
            "conversation_id": _text(channel.get("conversation_id")),
            "conversation_type": _text(channel.get("conversation_type")),
        },
        "source_message_id": _text(item.get("source_message_id")),
        "sender_id": _text(item.get("sender_id")),
        "kind": _text(item.get("kind")),
        "mime_type": _text(item.get("mime_type")),
        "name": _text(item.get("name")),
        "size_bytes": _int_value(item.get("size_bytes"), fallback=0),
        "retention": _text(item.get("retention")),
        "retention_class": _text(item.get("retention_class") or "unknown"),
        "cleanup_due": bool(item.get("cleanup_due")),
        "age_seconds": _int_value(item.get("age_seconds"), fallback=0),
        "ttl_seconds": _int_value(item.get("ttl_seconds"), fallback=0),
        "cleanup_after": _text(item.get("cleanup_after")),
        "cleanup_reason": _text(item.get("cleanup_reason")),
        "created_at": _text(item.get("created_at")),
        "updated_at": _text(item.get("updated_at")),
    }


def _normalize_media_asset_retention_plan(item: Mapping[str, Any]) -> dict[str, Any]:
    candidates_raw = item.get("candidates")
    if not isinstance(candidates_raw, list):
        candidates_raw = []
    required_raw = item.get("required_steps")
    if not isinstance(required_raw, list):
        required_raw = []
    verify_raw = item.get("verify_steps")
    if not isinstance(verify_raw, list):
        verify_raw = []
    rollback_raw = item.get("rollback_steps")
    if not isinstance(rollback_raw, list):
        rollback_raw = []
    candidates = [
        _normalize_media_asset_retention_item(value)
        for value in candidates_raw
        if isinstance(value, Mapping)
    ]
    diagnostics = _normalize_media_asset_retention_diagnostics(
        _mapping_or_empty(item.get("diagnostics"))
    )
    return {
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason") or "unknown"),
        "blockers": _string_list(item.get("blockers")),
        "asset_count": _int_value(
            item.get("asset_count"),
            fallback=diagnostics["totals"]["assets"],
        ),
        "candidate_count": _int_value(
            item.get("candidate_count"),
            fallback=len(candidates),
        ),
        "candidates": candidates,
        "required_steps": [
            _normalize_media_asset_retention_plan_step(value)
            for value in required_raw
            if isinstance(value, Mapping)
        ],
        "verify_steps": [
            _normalize_media_asset_retention_plan_step(value)
            for value in verify_raw
            if isinstance(value, Mapping)
        ],
        "rollback_steps": [
            _normalize_media_asset_retention_plan_step(value)
            for value in rollback_raw
            if isinstance(value, Mapping)
        ],
        "diagnostics": diagnostics,
        "side_effect": _text(item.get("side_effect") or "none"),
        "notes": _string_list(item.get("notes")),
    }


def _normalize_media_asset_retention_plan_step(
    item: Mapping[str, Any],
) -> dict[str, Any]:
    return {
        "name": _text(item.get("name")),
        "description": _text(item.get("description")),
        "endpoint": _text(item.get("endpoint")),
        "method": _text(item.get("method")),
        "metadata": dict(_mapping_or_empty(item.get("metadata"))),
    }


def _normalize_media_asset_retention_cleanup(
    item: Mapping[str, Any],
    *,
    plan: Mapping[str, Any],
) -> dict[str, Any]:
    audits_raw = item.get("recent_audits")
    if not isinstance(audits_raw, list):
        audits_raw = []
    recent_audits = [
        _normalize_control_mutation(value)
        for value in audits_raw
        if isinstance(value, Mapping)
    ]
    totals = _mapping_or_empty(item.get("totals"))
    endpoints = _mapping_or_empty(item.get("endpoints"))
    return {
        "ready": bool(item.get("ready", plan.get("ready", False))),
        "reason": _text(item.get("reason") or plan.get("reason") or "unknown"),
        "blockers": _string_list(item.get("blockers") or plan.get("blockers")),
        "asset_count": _int_value(
            item.get("asset_count"),
            fallback=_int_value(plan.get("asset_count"), fallback=0),
        ),
        "candidate_count": _int_value(
            item.get("candidate_count"),
            fallback=_int_value(plan.get("candidate_count"), fallback=0),
        ),
        "recent_audits": recent_audits,
        "totals": {
            "audits": _int_value(totals.get("audits"), fallback=len(recent_audits)),
            "planned": _int_value(
                totals.get("planned"),
                fallback=sum(1 for value in recent_audits if value["status"] == "planned"),
            ),
            "applied": _int_value(
                totals.get("applied"),
                fallback=sum(1 for value in recent_audits if value["status"] == "applied"),
            ),
            "failed": _int_value(
                totals.get("failed"),
                fallback=sum(1 for value in recent_audits if value["status"] == "failed"),
            ),
            "rolled_back": _int_value(
                totals.get("rolled_back"),
                fallback=sum(
                    1 for value in recent_audits if value["status"] == "rolled_back"
                ),
            ),
        },
        "endpoints": {
            "plan": _text(endpoints.get("plan") or "/v1/media-assets/retention-plan"),
            "preflight": _text(
                endpoints.get("preflight")
                or "/v1/media-assets/retention-cleanup/preflight"
            ),
            "cleanup": _text(
                endpoints.get("cleanup") or "/v1/media-assets/retention-cleanup"
            ),
        },
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_agent_job_capacity_plan(item: Mapping[str, Any]) -> dict[str, Any]:
    summary = _mapping_or_empty(item.get("summary"))
    items_raw = item.get("items")
    if not isinstance(items_raw, list):
        items_raw = []
    return {
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason") or "unknown"),
        "summary": {
            "job_types": _int_value(summary.get("job_types"), fallback=0),
            "mapped_job_types": _int_value(summary.get("mapped_job_types"), fallback=0),
            "unmapped_job_types": _int_value(
                summary.get("unmapped_job_types"),
                fallback=0,
            ),
            "high_pressure_job_types": _int_value(
                summary.get("high_pressure_job_types"),
                fallback=0,
            ),
            "capacity_blocked_job_types": _int_value(
                summary.get("capacity_blocked_job_types"),
                fallback=0,
            ),
            "worker_warning_job_types": _int_value(
                summary.get("worker_warning_job_types"),
                fallback=0,
            ),
            "active_worker_job_types": _int_value(
                summary.get("active_worker_job_types"),
                fallback=0,
            ),
            "stale_worker_job_types": _int_value(
                summary.get("stale_worker_job_types"),
                fallback=0,
            ),
            "failed_worker_job_types": _int_value(
                summary.get("failed_worker_job_types"),
                fallback=0,
            ),
            "max_pending": _int_value(summary.get("max_pending"), fallback=0),
            "max_active": _int_value(summary.get("max_active"), fallback=0),
            "oldest_pending_age_seconds": _int_value(
                summary.get("oldest_pending_age_seconds"),
                fallback=0,
            ),
        },
        "items": [
            _normalize_agent_job_capacity_item(value)
            for value in items_raw
            if isinstance(value, Mapping)
        ],
        "verification_steps": _normalize_operator_steps(
            item.get("verification_steps")
        ),
        "blockers": _string_list(item.get("blockers")),
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_agent_job_capacity_item(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "job_type": _text(item.get("job_type")),
        "severity": _text(item.get("severity")),
        "action": _text(item.get("action")),
        "recommendation": _text(item.get("recommendation")),
        "pending": _int_value(item.get("pending"), fallback=0),
        "leased": _int_value(item.get("leased"), fallback=0),
        "running": _int_value(item.get("running"), fallback=0),
        "active": _int_value(item.get("active"), fallback=0),
        "oldest_pending_age_seconds": _int_value(
            item.get("oldest_pending_age_seconds"),
            fallback=0,
        ),
        "high_pressure": bool(item.get("high_pressure")),
        "pressure_reason": _text(item.get("pressure_reason")),
        "coverage": _mapping_or_empty(item.get("coverage")),
    }


def _normalize_agent_job_priority_plan(item: Mapping[str, Any]) -> dict[str, Any]:
    summary = _mapping_or_empty(item.get("summary"))
    items_raw = item.get("items")
    if not isinstance(items_raw, list):
        items_raw = []
    return {
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason") or "unknown"),
        "summary": {
            "job_types": _int_value(summary.get("job_types"), fallback=0),
            "high_priority_job_types": _int_value(
                summary.get("high_priority_job_types"),
                fallback=0,
            ),
            "blocked_job_types": _int_value(
                summary.get("blocked_job_types"),
                fallback=0,
            ),
            "warning_job_types": _int_value(
                summary.get("warning_job_types"),
                fallback=0,
            ),
            "max_priority_score": _int_value(
                summary.get("max_priority_score"),
                fallback=0,
            ),
        },
        "items": [
            _normalize_agent_job_priority_item(value)
            for value in items_raw
            if isinstance(value, Mapping)
        ],
        "verification_steps": _normalize_operator_steps(
            item.get("verification_steps")
        ),
        "blockers": _string_list(item.get("blockers")),
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_agent_job_priority_item(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "rank": _int_value(item.get("rank"), fallback=0),
        "job_type": _text(item.get("job_type")),
        "priority_class": _text(item.get("priority_class")),
        "priority_score": _int_value(item.get("priority_score"), fallback=0),
        "action": _text(item.get("action")),
        "recommendation": _text(item.get("recommendation")),
        "pending": _int_value(item.get("pending"), fallback=0),
        "leased": _int_value(item.get("leased"), fallback=0),
        "running": _int_value(item.get("running"), fallback=0),
        "active": _int_value(item.get("active"), fallback=0),
        "oldest_pending_age_seconds": _int_value(
            item.get("oldest_pending_age_seconds"),
            fallback=0,
        ),
        "high_pressure": bool(item.get("high_pressure")),
        "pressure_reason": _text(item.get("pressure_reason")),
        "coverage": _mapping_or_empty(item.get("coverage")),
    }


def _normalize_agent_job_external_lease_readiness(
    item: Mapping[str, Any],
) -> dict[str, Any]:
    return {
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason") or "unknown"),
        "external_lease_ready": bool(item.get("external_lease_ready")),
        "agent_job_result_ack_ready": bool(item.get("agent_job_result_ack_ready")),
        "strict_lease_token_enabled": bool(item.get("strict_lease_token_enabled")),
        "agent_job_worker_ready": bool(item.get("agent_job_worker_ready")),
        "execution_owner": _text(item.get("execution_owner") or "unknown"),
        "queue_provider": _text(item.get("queue_provider")),
        "queue_mode": _text(item.get("queue_mode")),
        "execution_scope": _text(item.get("execution_scope") or "unknown"),
        "allowed_work_kinds": _string_list(item.get("allowed_work_kinds")),
        "blocked_work_kinds": _mapping_list(item.get("blocked_work_kinds")),
        "required_checks": _mapping_list(item.get("required_checks")),
        "worker_coverage": _mapping_list(item.get("worker_coverage")),
        "blockers": _string_list(item.get("blockers")),
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_agent_job_external_lease_plan(
    item: Mapping[str, Any],
) -> dict[str, Any]:
    return {
        "ready": bool(item.get("ready")),
        "decision": _text(item.get("decision") or "unknown"),
        "desired_execution_owner": _text(
            item.get("desired_execution_owner") or "unknown"
        ),
        "recommended_execution_owner": _text(
            item.get("recommended_execution_owner") or "unknown"
        ),
        "current_execution_owner": _text(
            item.get("current_execution_owner") or "unknown"
        ),
        "readiness": _normalize_agent_job_external_lease_readiness(
            _mapping_or_empty(item.get("readiness"))
        ),
        "required_checks": _normalize_operator_steps(item.get("required_checks")),
        "enable_steps": _normalize_operator_steps(item.get("enable_steps")),
        "verification_steps": _normalize_operator_steps(
            item.get("verification_steps")
        ),
        "rollback_steps": _normalize_operator_steps(item.get("rollback_steps")),
        "blockers": _string_list(item.get("blockers")),
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_outbound_cutover_plan(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "ready": bool(item.get("ready")),
        "decision": _text(item.get("decision") or "unknown"),
        "desired_execution_owner": _text(
            item.get("desired_execution_owner") or "unknown"
        ),
        "recommended_execution_owner": _text(
            item.get("recommended_execution_owner") or "unknown"
        ),
        "current_execution_owner": _text(
            item.get("current_execution_owner") or "unknown"
        ),
        "readiness": _normalize_outbound_cutover_readiness(
            _mapping_or_empty(item.get("readiness"))
        ),
        "required_checks": _normalize_operator_steps(item.get("required_checks")),
        "enable_steps": _normalize_operator_steps(item.get("enable_steps")),
        "verification_steps": _normalize_operator_steps(
            item.get("verification_steps")
        ),
        "rollback_steps": _normalize_operator_steps(item.get("rollback_steps")),
        "blockers": _string_list(item.get("blockers")),
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_outbound_cutover_readiness(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason") or "unknown"),
        "onebot_ready": bool(item.get("onebot_ready")),
        "smoke_ready": bool(item.get("smoke_ready")),
        "execution_ready": bool(item.get("execution_ready")),
        "execution_owner": _text(item.get("execution_owner") or "unknown"),
        "local_outbox_worker_ready": bool(item.get("local_outbox_worker_ready")),
        "external_lease_outbox_ready": bool(item.get("external_lease_outbox_ready")),
        "queue_provider": _text(item.get("queue_provider")),
        "queue_mode": _text(item.get("queue_mode")),
        "external_lease_scope": _text(item.get("external_lease_scope")),
        "expected_onebot_channels": _string_list(
            item.get("expected_onebot_channels")
        ),
        "missing_onebot_channels": _string_list(item.get("missing_onebot_channels")),
        "runtime_config_readiness": _mapping_or_empty(
            item.get("runtime_config_readiness")
        ),
        "delivery_smoke_readiness": _normalize_delivery_smoke_readiness(
            _mapping_or_empty(item.get("delivery_smoke_readiness")),
            runtime_base_url="",
        ),
        "blockers": _string_list(item.get("blockers")),
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_control_mutation_policy(item: Mapping[str, Any]) -> dict[str, Any]:
    intents_raw = item.get("intents")
    if not isinstance(intents_raw, list):
        intents_raw = []
    return {
        "allowed": bool(item.get("allowed")),
        "reason": _text(item.get("reason")),
        "blockers": _string_list(item.get("blockers")),
        "target_kind": _text(item.get("target_kind")),
        "intents": [
            _normalize_control_mutation_policy_intent(value)
            for value in intents_raw
            if isinstance(value, Mapping)
        ],
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_control_mutation_policy_intent(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "target_kind": _text(item.get("target_kind")),
        "actions": _string_list(item.get("actions")),
    }


def _normalize_operator_approvals(item: Mapping[str, Any]) -> dict[str, Any]:
    approvals_raw = item.get("approvals")
    if not isinstance(approvals_raw, list):
        approvals_raw = []
    totals = _mapping_or_empty(item.get("totals"))
    return {
        "approvals": [
            _normalize_operator_approval(value)
            for value in approvals_raw
            if isinstance(value, Mapping)
        ],
        "totals": {
            "approvals": _int_value(totals.get("approvals"), fallback=0),
            "active": _int_value(totals.get("active"), fallback=0),
            "approved": _int_value(totals.get("approved"), fallback=0),
            "rejected": _int_value(totals.get("rejected"), fallback=0),
            "revoked": _int_value(totals.get("revoked"), fallback=0),
        },
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "runtime_state_only"),
    }


def _normalize_operator_approval(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "approval_id": _text(item.get("approval_id")),
        "target_kind": _text(item.get("target_kind")),
        "target_id": _text(item.get("target_id")),
        "decision": _text(item.get("decision")),
        "operator_id": _text(item.get("operator_id")),
        "reason": _text(item.get("reason")),
        "active": bool(item.get("active")),
        "expires_at": _text(item.get("expires_at")),
        "created_at": _text(item.get("created_at")),
        "metadata": _mapping_or_empty(item.get("metadata")),
    }


def _normalize_control_mutations(item: Mapping[str, Any]) -> dict[str, Any]:
    mutations_raw = item.get("mutations")
    if not isinstance(mutations_raw, list):
        mutations_raw = []
    totals = _mapping_or_empty(item.get("totals"))
    return {
        "mutations": [
            _normalize_control_mutation(value)
            for value in mutations_raw
            if isinstance(value, Mapping)
        ],
        "totals": {
            "mutations": _int_value(totals.get("mutations"), fallback=0),
            "planned": _int_value(totals.get("planned"), fallback=0),
            "applied": _int_value(totals.get("applied"), fallback=0),
            "failed": _int_value(totals.get("failed"), fallback=0),
            "rolled_back": _int_value(totals.get("rolled_back"), fallback=0),
        },
        "notes": _string_list(item.get("notes")),
        "side_effect": _text(item.get("side_effect") or "runtime_state_only"),
    }


def _normalize_control_mutation(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "mutation_id": _text(item.get("mutation_id")),
        "target_kind": _text(item.get("target_kind")),
        "target_id": _text(item.get("target_id")),
        "action": _text(item.get("action")),
        "status": _text(item.get("status")),
        "operator_id": _text(item.get("operator_id")),
        "approval_id": _text(item.get("approval_id")),
        "reason": _text(item.get("reason")),
        "rollback_of": _text(item.get("rollback_of")),
        "rollback_ref": _text(item.get("rollback_ref")),
        "created_at": _text(item.get("created_at")),
        "metadata": _mapping_or_empty(item.get("metadata")),
    }


def _normalize_knowledge_job_planner_cutover_plan(
    item: Mapping[str, Any],
) -> dict[str, Any]:
    blockers = item.get("blockers")
    if not isinstance(blockers, list):
        blockers = []
    notes = item.get("notes")
    if not isinstance(notes, list):
        notes = []
    return {
        "ready": bool(item.get("ready")),
        "decision": _text(item.get("decision") or "unknown"),
        "desired_admission_owner": _text(
            item.get("desired_admission_owner") or "unknown"
        ),
        "recommended_admission_owner": _text(
            item.get("recommended_admission_owner") or "unknown"
        ),
        "current_admission_owner": _text(
            item.get("current_admission_owner") or "unknown"
        ),
        "readiness": _normalize_knowledge_job_planner_readiness_summary(
            _mapping_or_empty(item.get("readiness"))
        ),
        "required_checks": _normalize_knowledge_job_planner_cutover_steps(
            item.get("required_checks")
        ),
        "enable_steps": _normalize_knowledge_job_planner_cutover_steps(
            item.get("enable_steps")
        ),
        "verification_steps": _normalize_knowledge_job_planner_cutover_steps(
            item.get("verification_steps")
        ),
        "rollback_steps": _normalize_knowledge_job_planner_cutover_steps(
            item.get("rollback_steps")
        ),
        "blockers": [str(value) for value in blockers],
        "attributes": _mapping_or_empty(item.get("attributes")),
        "notes": [str(value) for value in notes],
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_knowledge_job_planner_readiness_summary(
    item: Mapping[str, Any],
) -> dict[str, Any]:
    blockers = item.get("blockers")
    if not isinstance(blockers, list):
        blockers = []
    return {
        "ready": bool(item.get("ready")),
        "reason": _text(item.get("reason")),
        "planner_enabled": bool(item.get("planner_enabled")),
        "planner_running": bool(item.get("planner_running")),
        "knowledge_worker_ready": bool(item.get("knowledge_worker_ready")),
        "knowledge_worker_active": _int_value(
            item.get("knowledge_worker_active"),
            fallback=0,
        ),
        "knowledge_worker_stale": _int_value(
            item.get("knowledge_worker_stale"),
            fallback=0,
        ),
        "knowledge_worker_failed": _int_value(
            item.get("knowledge_worker_failed"),
            fallback=0,
        ),
        "knowledge_worker_stopped": _int_value(
            item.get("knowledge_worker_stopped"),
            fallback=0,
        ),
        "blockers": [str(value) for value in blockers],
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_knowledge_job_planner_cutover_steps(
    value: object,
) -> list[dict[str, Any]]:
    return _normalize_operator_steps(value)


def _normalize_knowledge_job_planner_cutover_step(
    item: Mapping[str, Any],
) -> dict[str, Any]:
    return _normalize_operator_step(item)


def _normalize_operator_steps(value: object) -> list[dict[str, Any]]:
    if not isinstance(value, list):
        return []
    return [
        _normalize_operator_step(item)
        for item in value
        if isinstance(item, Mapping)
    ]


def _normalize_operator_step(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "step_index": _int_value(item.get("step_index"), fallback=0),
        "phase": _text(item.get("phase")),
        "action": _text(item.get("action")),
        "detail": _text(item.get("detail")),
        "method": _text(item.get("method")),
        "endpoint": _text(item.get("endpoint")),
        "env": _mapping_or_empty(item.get("env")),
    }


def _normalize_receiver_statuses(item: Mapping[str, Any]) -> dict[str, Any]:
    receivers_raw = item.get("receivers")
    if not isinstance(receivers_raw, list):
        receivers_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    totals = _mapping_or_empty(item.get("totals"))
    receivers = [
        _normalize_receiver_status(value)
        for value in receivers_raw
        if isinstance(value, Mapping)
    ]
    return {
        "receivers": receivers,
        "totals": {
            "receivers": _int_value(
                totals.get("receivers"), fallback=len(receivers)
            ),
            "starting": _int_value(totals.get("starting"), fallback=0),
            "connected": _int_value(
                totals.get("connected"),
                fallback=sum(1 for value in receivers if value["status"] == "connected"),
            ),
            "suspended": _int_value(
                totals.get("suspended"),
                fallback=sum(1 for value in receivers if value["status"] == "suspended"),
            ),
            "failed": _int_value(
                totals.get("failed"),
                fallback=sum(1 for value in receivers if value["status"] == "failed"),
            ),
            "stopped": _int_value(totals.get("stopped"), fallback=0),
            "qq": _int_value(
                totals.get("qq"),
                fallback=sum(1 for value in receivers if value["kind"] == "qq"),
            ),
            "telegram": _int_value(
                totals.get("telegram"),
                fallback=sum(1 for value in receivers if value["kind"] == "telegram"),
            ),
        },
        "notes": [str(value) for value in notes_raw],
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_receiver_status(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "receiver_id": _text(item.get("receiver_id")),
        "kind": _text(item.get("kind")),
        "channel_name": _text(item.get("channel_name")),
        "account_id": _text(item.get("account_id")),
        "endpoint": _text(item.get("endpoint")),
        "status": _text(item.get("status")),
        "reason": _text(item.get("reason")),
        "last_error": _text(item.get("last_error")),
        "source": _text(item.get("source")),
        "metadata": dict(_mapping_or_empty(item.get("metadata"))),
        "updated_at": _text(item.get("updated_at")),
    }


def _normalize_receiver_leases(item: Mapping[str, Any]) -> dict[str, Any]:
    leases_raw = item.get("leases")
    if not isinstance(leases_raw, list):
        leases_raw = []
    notes_raw = item.get("notes")
    if not isinstance(notes_raw, list):
        notes_raw = []
    totals = _mapping_or_empty(item.get("totals"))
    leases = [
        _normalize_receiver_lease(value)
        for value in leases_raw
        if isinstance(value, Mapping)
    ]
    return {
        "leases": leases,
        "totals": {
            "leases": _int_value(totals.get("leases"), fallback=len(leases)),
            "active": _int_value(
                totals.get("active"),
                fallback=sum(1 for value in leases if value["active"]),
            ),
            "expired": _int_value(totals.get("expired"), fallback=0),
            "qq": _int_value(
                totals.get("qq"),
                fallback=sum(1 for value in leases if value["kind"] == "qq"),
            ),
            "telegram": _int_value(
                totals.get("telegram"),
                fallback=sum(1 for value in leases if value["kind"] == "telegram"),
            ),
        },
        "notes": [str(value) for value in notes_raw],
        "side_effect": _text(item.get("side_effect") or "runtime_state_only"),
    }


def _normalize_receiver_lease(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "receiver_id": _text(item.get("receiver_id")),
        "kind": _text(item.get("kind")),
        "channel_name": _text(item.get("channel_name")),
        "account_id": _text(item.get("account_id")),
        "holder_id": _text(item.get("holder_id")),
        "lease_token_present": bool(item.get("lease_token_present")),
        "active": bool(item.get("active")),
        "expires_at": _text(item.get("expires_at")),
        "acquired_at": _text(item.get("acquired_at")),
        "updated_at": _text(item.get("updated_at")),
        "metadata": dict(_mapping_or_empty(item.get("metadata"))),
    }


def _normalize_scheduler_job_diagnostics(item: Mapping[str, Any]) -> dict[str, Any]:
    recent_raw = item.get("recent")
    if not isinstance(recent_raw, list):
        recent_raw = []
    return {
        "sampled_jobs": _int_value(item.get("sampled_jobs"), fallback=0),
        "enabled_jobs": _int_value(item.get("enabled_jobs"), fallback=0),
        "disabled_jobs": _int_value(item.get("disabled_jobs"), fallback=0),
        "overdue_jobs": _int_value(item.get("overdue_jobs"), fallback=0),
        "due_soon_jobs": _int_value(item.get("due_soon_jobs"), fallback=0),
        "instant_jobs": _int_value(item.get("instant_jobs"), fallback=0),
        "soft_jobs": _int_value(item.get("soft_jobs"), fallback=0),
        "next_fire_at": _text(item.get("next_fire_at")),
        "jobs_by_trigger": _mapping_or_empty(item.get("jobs_by_trigger")),
        "jobs_by_tier": _mapping_or_empty(item.get("jobs_by_tier")),
        "jobs_by_channel": _mapping_or_empty(item.get("jobs_by_channel")),
        "jobs_by_status": _mapping_or_empty(item.get("jobs_by_status")),
        "recent": [
            _normalize_scheduler_job_sample(value)
            for value in recent_raw
            if isinstance(value, Mapping)
        ],
        "due_soon_seconds": _int_value(item.get("due_soon_seconds"), fallback=300),
        "side_effect": _text(item.get("side_effect") or "none"),
    }


def _normalize_scheduler_job_sample(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "id": _text(item.get("id")),
        "trigger": _text(item.get("trigger")),
        "tier": _text(item.get("tier")),
        "channel": _text(item.get("channel")),
        "chat_id": _text(item.get("chat_id")),
        "fire_at": _text(item.get("fire_at")),
        "status": _text(item.get("status")),
        "run_count": _int_value(item.get("run_count"), fallback=0),
        "enabled": bool(item.get("enabled")),
        "overdue_by_seconds": _int_value(item.get("overdue_by_seconds"), fallback=0),
        "due_in_seconds": _int_value(item.get("due_in_seconds"), fallback=0),
    }


def _normalize_go_runtime_card(item: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "id": _text(item.get("id")),
        "label": _text(item.get("label")),
        "value": item.get("value"),
        "status": _text(item.get("status")),
        "detail": _mapping_or_empty(item.get("detail")),
    }


def _checkpoint_lag_from_diagnostics(diagnostics: Mapping[str, Any]) -> list[dict[str, Any]]:
    workers = diagnostics.get("workers")
    if not isinstance(workers, list):
        return []
    checkpoints: list[dict[str, Any]] = []
    for worker in workers:
        if not isinstance(worker, Mapping):
            continue
        raw_items = worker.get("checkpoints")
        if not isinstance(raw_items, list):
            continue
        for item in raw_items:
            if isinstance(item, Mapping):
                checkpoint = _normalize_checkpoint(item)
                if checkpoint["checkpoint_lag_messages"] is not None:
                    checkpoints.append(checkpoint)
    return checkpoints


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
    observe_targets: dict[str, Any],
    observe_capture: dict[str, Any],
    media_asset_retention: dict[str, Any],
    media_asset_retention_plan: dict[str, Any],
    media_asset_retention_cleanup: dict[str, Any],
    receiver_statuses: dict[str, Any],
    receiver_leases: dict[str, Any],
    scheduler_jobs: dict[str, Any],
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
            "observe_targets",
            "Observe Targets",
            summary.get("observe_targets_enabled", 0),
            "ok" if summary.get("observe_targets_enabled") else "muted",
            {"observe_targets": observe_targets},
        ),
        _card(
            "observe_capture",
            "Observe Capture",
            f"{summary.get('observe_capture_ready', 0)}/{summary.get('observe_capture_targets', 0)}",
            _observe_capture_status(summary),
            {"observe_capture": observe_capture},
        ),
        _card(
            "media_asset_retention",
            "Media Asset Retention",
            _media_asset_retention_value(summary),
            _media_asset_retention_status(summary),
            {"media_asset_retention_diagnostics": media_asset_retention},
        ),
        _card(
            "media_asset_retention_plan",
            "Media Asset Retention Plan",
            _media_asset_retention_plan_value(summary),
            _media_asset_retention_plan_status(summary),
            {"media_asset_retention_plan": media_asset_retention_plan},
        ),
        _card(
            "media_asset_retention_cleanup",
            "Media Asset Retention Cleanup",
            _media_asset_retention_cleanup_value(summary),
            _media_asset_retention_cleanup_status(summary),
            {"media_asset_retention_cleanup": media_asset_retention_cleanup},
        ),
        _card(
            "receiver_statuses",
            "Receiver Statuses",
            summary.get("receiver_status_connected", 0),
            _receiver_status_status(summary),
            {"receiver_statuses": receiver_statuses},
        ),
        _card(
            "receiver_leases",
            "Receiver Leases",
            _receiver_lease_value(summary),
            _receiver_lease_status(summary),
            {
                "receiver_leases": receiver_leases,
                "cleanup_endpoint": summary.get(
                    "receiver_lease_cleanup_endpoint",
                    "/v1/receiver-leases/cleanup-expired",
                ),
            },
        ),
        _card(
            "scheduler_jobs",
            "Scheduler Jobs",
            _scheduler_job_value(summary),
            _scheduler_job_status(summary),
            {"scheduler_jobs": scheduler_jobs},
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


def _observe_capture_status(summary: Mapping[str, Any]) -> str:
    targets = _int_value(summary.get("observe_capture_targets"), fallback=0)
    if targets <= 0:
        return "muted"
    if _int_value(summary.get("observe_capture_blocked"), fallback=0) > 0:
        return "danger"
    if _int_value(summary.get("observe_capture_ready"), fallback=0) < _int_value(
        summary.get("observe_capture_targets"),
        fallback=0,
    ):
        return "warn"
    return "ok"


def _media_asset_retention_status(summary: Mapping[str, Any]) -> str:
    assets = _int_value(summary.get("media_asset_retention_assets"), fallback=0)
    if assets <= 0:
        return "muted"
    if _int_value(summary.get("media_asset_retention_cleanup_due"), fallback=0) > 0:
        return "warn"
    return "ok"


def _media_asset_retention_value(summary: Mapping[str, Any]) -> str:
    return (
        f"{_int_value(summary.get('media_asset_retention_cleanup_due'), fallback=0)}/"
        f"{_int_value(summary.get('media_asset_retention_assets'), fallback=0)}"
    )


def _media_asset_retention_plan_status(summary: Mapping[str, Any]) -> str:
    reason = _text(summary.get("media_asset_retention_plan_reason"))
    if reason == "unknown":
        return "muted"
    if bool(summary.get("media_asset_retention_plan_ready")):
        return "warn"
    if _int_value(summary.get("media_asset_retention_plan_blockers"), fallback=0) > 0:
        return "ok"
    return "muted"


def _media_asset_retention_plan_value(summary: Mapping[str, Any]) -> str:
    reason = _text(summary.get("media_asset_retention_plan_reason"))
    if reason == "unknown":
        return "unknown"
    if bool(summary.get("media_asset_retention_plan_ready")):
        return (
            f"{_int_value(summary.get('media_asset_retention_plan_candidates'), fallback=0)}/"
            f"{_int_value(summary.get('media_asset_retention_plan_assets'), fallback=0)}"
        )
    return (
        f"{reason}:"
        f"{_int_value(summary.get('media_asset_retention_plan_blockers'), fallback=0)}"
    )


def _media_asset_retention_cleanup_status(summary: Mapping[str, Any]) -> str:
    reason = _text(summary.get("media_asset_retention_cleanup_reason"))
    if reason == "unknown":
        return "muted"
    if _int_value(summary.get("media_asset_retention_cleanup_failed"), fallback=0) > 0:
        return "danger"
    if _int_value(summary.get("media_asset_retention_cleanup_candidates"), fallback=0) > 0:
        return "warn"
    if _int_value(summary.get("media_asset_retention_cleanup_applied"), fallback=0) > 0:
        return "ok"
    return "muted"


def _media_asset_retention_cleanup_value(summary: Mapping[str, Any]) -> str:
    reason = _text(summary.get("media_asset_retention_cleanup_reason"))
    if reason == "unknown":
        return "unknown"
    return (
        f"{_int_value(summary.get('media_asset_retention_cleanup_candidates'), fallback=0)}/"
        f"{_int_value(summary.get('media_asset_retention_cleanup_applied'), fallback=0)}"
    )


def _receiver_status_status(summary: Mapping[str, Any]) -> str:
    receivers = _int_value(summary.get("receiver_statuses"), fallback=0)
    if receivers <= 0:
        return "muted"
    if _int_value(summary.get("receiver_status_failed"), fallback=0) > 0:
        return "danger"
    if _int_value(summary.get("receiver_status_suspended"), fallback=0) > 0:
        return "warn"
    if _int_value(summary.get("receiver_status_connected"), fallback=0) <= 0:
        return "warn"
    return "ok"


def _receiver_lease_status(summary: Mapping[str, Any]) -> str:
    leases = _int_value(summary.get("receiver_leases"), fallback=0)
    if leases <= 0:
        return "muted"
    if _int_value(summary.get("receiver_leases_expired"), fallback=0) > 0:
        return "warn"
    if _int_value(summary.get("receiver_leases_active"), fallback=0) <= 0:
        return "warn"
    return "ok"


def _receiver_lease_value(summary: Mapping[str, Any]) -> str:
    leases = _int_value(summary.get("receiver_leases"), fallback=0)
    if leases <= 0:
        return "0"
    return (
        f"{_int_value(summary.get('receiver_leases_active'), fallback=0)}/"
        f"{_int_value(summary.get('receiver_leases_expired'), fallback=0)}"
    )


def _scheduler_job_status(summary: Mapping[str, Any]) -> str:
    jobs = _int_value(summary.get("scheduler_jobs"), fallback=0)
    if jobs <= 0:
        return "muted"
    if _int_value(summary.get("scheduler_jobs_overdue"), fallback=0) > 0:
        return "warn"
    return "ok"


def _scheduler_job_value(summary: Mapping[str, Any]) -> object:
    jobs = _int_value(summary.get("scheduler_jobs"), fallback=0)
    if jobs <= 0:
        return 0
    enabled = _int_value(summary.get("scheduler_jobs_enabled"), fallback=0)
    return f"{enabled}/{jobs}"


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


def _mapping_list(value: object) -> list[dict[str, Any]]:
    if not isinstance(value, list):
        return []
    return [dict(item) for item in value if isinstance(item, Mapping)]


def _string_list(value: object) -> list[str]:
    if not isinstance(value, list):
        return []
    return [str(item) for item in value]


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


def _csv_values(value: str) -> list[str]:
    return [
        item.strip()
        for item in str(value or "").split(",")
        if item.strip()
    ]


def _clean_base_url(value: str) -> str:
    return value.strip().rstrip("/")
