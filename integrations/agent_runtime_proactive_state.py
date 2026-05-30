from __future__ import annotations

import logging
from datetime import datetime
from typing import Any

import httpx

from agent.config_models import AgentRuntimeIntegrationConfig
from core.common.timekit import parse_iso as _parse_iso, utcnow as _utcnow
from proactive_v2.state import ProactiveStateStore

logger = logging.getLogger(__name__)


class AgentRuntimeProactiveStateError(RuntimeError):
    pass


class AgentRuntimeProactiveStateStore:
    """Runtime-backed proactive scheduling state with SQLite compatibility fallback."""

    def __init__(
        self,
        config: AgentRuntimeIntegrationConfig,
        fallback: ProactiveStateStore,
        *,
        transport: httpx.BaseTransport | None = None,
    ) -> None:
        self._config = config
        self._fallback = fallback
        self._base_url = str(getattr(config, "base_url", "") or "").strip().rstrip("/")
        self._transport = transport
        self.workspace_dir = fallback.workspace_dir

    def close(self) -> None:
        self._fallback.close()

    def record_tick_log_start(self, **kwargs: Any) -> None:
        self._fallback.record_tick_log_start(**kwargs)

    def record_tick_log_finish(self, **kwargs: Any) -> None:
        self._fallback.record_tick_log_finish(**kwargs)

    def record_tick_step_log(self, **kwargs: Any) -> None:
        self._fallback.record_tick_step_log(**kwargs)

    def is_item_seen(
        self,
        source_key: str,
        item_id: str,
        ttl_hours: int,
        now: datetime | None = None,
    ) -> bool:
        return self._fallback.is_item_seen(source_key, item_id, ttl_hours, now)

    def mark_items_seen(
        self,
        entries: list[tuple[str, str]],
        now: datetime | None = None,
    ) -> None:
        self._fallback.mark_items_seen(entries, now)

    def is_delivery_duplicate(
        self,
        session_key: str,
        delivery_key: str,
        window_hours: int,
        now: datetime | None = None,
    ) -> bool:
        timestamp = now or _utcnow()
        try:
            data = self._request(
                "GET",
                "/v1/proactive/deliveries/duplicate",
                params={
                    "session_key": session_key,
                    "delivery_key": delivery_key,
                    "window_hours": max(1, int(window_hours)),
                    "timestamp": timestamp.isoformat(),
                },
            )
            if isinstance(data, dict):
                duplicate = bool(data.get("duplicate"))
                if duplicate:
                    return True
                return self._fallback.is_delivery_duplicate(
                    session_key,
                    delivery_key,
                    window_hours,
                    timestamp,
                )
            raise AgentRuntimeProactiveStateError(
                "duplicate response is not an object"
            )
        except Exception as exc:
            self._log_fallback("is_delivery_duplicate", exc)
            return self._fallback.is_delivery_duplicate(
                session_key,
                delivery_key,
                window_hours,
                timestamp,
            )

    def mark_delivery(
        self,
        session_key: str,
        delivery_key: str,
        now: datetime | None = None,
    ) -> None:
        timestamp = now or _utcnow()
        try:
            self._request(
                "POST",
                "/v1/proactive/deliveries",
                json_body={
                    "session_key": session_key,
                    "delivery_key": delivery_key,
                    "timestamp": timestamp.isoformat(),
                },
            )
        except Exception as exc:
            self._log_fallback("mark_delivery", exc)
        self._fallback.mark_delivery(session_key, delivery_key, timestamp)

    def count_deliveries_in_window(
        self,
        session_key: str,
        window_hours: int,
        now: datetime | None = None,
    ) -> int:
        timestamp = now or _utcnow()
        try:
            data = self._request(
                "GET",
                "/v1/proactive/deliveries/count",
                params={
                    "session_key": session_key,
                    "window_hours": max(1, int(window_hours)),
                    "timestamp": timestamp.isoformat(),
                },
            )
            if isinstance(data, dict):
                runtime_count = int(data.get("count") or 0)
                fallback_count = self._fallback.count_deliveries_in_window(
                    session_key,
                    window_hours,
                    timestamp,
                )
                return max(runtime_count, fallback_count)
            raise AgentRuntimeProactiveStateError("count response is not an object")
        except Exception as exc:
            self._log_fallback("count_deliveries_in_window", exc)
            return self._fallback.count_deliveries_in_window(
                session_key,
                window_hours,
                timestamp,
            )

    def get_semantic_items(
        self,
        window_hours: int,
        max_candidates: int,
        now: datetime | None = None,
    ) -> list[dict[str, str]]:
        return self._fallback.get_semantic_items(window_hours, max_candidates, now)

    def mark_semantic_items(
        self,
        entries: list[dict[str, str]],
        now: datetime | None = None,
    ) -> None:
        self._fallback.mark_semantic_items(entries, now)

    def is_rejection_cooled(
        self,
        source_key: str,
        item_id: str,
        ttl_hours: int,
        now: datetime | None = None,
    ) -> bool:
        return self._fallback.is_rejection_cooled(
            source_key,
            item_id,
            ttl_hours,
            now,
        )

    def mark_rejection_cooldown(
        self,
        entries: list[tuple[str, str]],
        hours: int,
        now: datetime | None = None,
    ) -> None:
        self._fallback.mark_rejection_cooldown(entries, hours, now)

    def cleanup(
        self,
        seen_ttl_hours: int,
        delivery_ttl_hours: int,
        semantic_ttl_hours: int,
        rejection_cooldown_ttl_hours: int = 0,
    ) -> None:
        self._fallback.cleanup(
            seen_ttl_hours,
            delivery_ttl_hours,
            semantic_ttl_hours,
            rejection_cooldown_ttl_hours,
        )

    def get_bg_context_last_main_at(self) -> datetime | None:
        return self._fallback.get_bg_context_last_main_at()

    def mark_bg_context_main_send(self, now: datetime | None = None) -> None:
        self._fallback.mark_bg_context_main_send(now)

    def get_last_drift_at(self, session_key: str) -> datetime | None:
        try:
            runtime_timestamp = self._get_timestamp(
                "/v1/proactive/drift-runs/last",
                session_key=session_key,
            )
            fallback_timestamp = self._fallback.get_last_drift_at(session_key)
            return _max_datetime(runtime_timestamp, fallback_timestamp)
        except Exception as exc:
            self._log_fallback("get_last_drift_at", exc)
            return self._fallback.get_last_drift_at(session_key)

    def mark_drift_run(
        self,
        session_key: str,
        now: datetime | None = None,
    ) -> None:
        timestamp = now or _utcnow()
        try:
            self._request(
                "POST",
                "/v1/proactive/drift-runs",
                json_body={
                    "session_key": session_key,
                    "timestamp": timestamp.isoformat(),
                },
            )
        except Exception as exc:
            self._log_fallback("mark_drift_run", exc)
        self._fallback.mark_drift_run(session_key, timestamp)

    def get_last_context_only_at(self, session_key: str) -> datetime | None:
        try:
            runtime_timestamp = self._get_timestamp(
                "/v1/proactive/context-only/last",
                session_key=session_key,
            )
            fallback_timestamp = self._fallback.get_last_context_only_at(session_key)
            return _max_datetime(runtime_timestamp, fallback_timestamp)
        except Exception as exc:
            self._log_fallback("get_last_context_only_at", exc)
            return self._fallback.get_last_context_only_at(session_key)

    def mark_context_only_send(
        self,
        session_key: str,
        now: datetime | None = None,
    ) -> None:
        timestamp = now or _utcnow()
        try:
            self._request(
                "POST",
                "/v1/proactive/context-only",
                json_body={
                    "session_key": session_key,
                    "timestamp": timestamp.isoformat(),
                },
            )
        except Exception as exc:
            self._log_fallback("mark_context_only_send", exc)
        self._fallback.mark_context_only_send(session_key, timestamp)

    def count_context_only_in_window(
        self,
        session_key: str,
        window_hours: int,
        now: datetime | None = None,
    ) -> int:
        timestamp = now or _utcnow()
        try:
            data = self._request(
                "GET",
                "/v1/proactive/context-only/count",
                params={
                    "session_key": session_key,
                    "window_hours": max(1, int(window_hours)),
                    "timestamp": timestamp.isoformat(),
                },
            )
            if isinstance(data, dict):
                runtime_count = int(data.get("count") or 0)
                fallback_count = self._fallback.count_context_only_in_window(
                    session_key,
                    window_hours,
                    timestamp,
                )
                return max(runtime_count, fallback_count)
            raise AgentRuntimeProactiveStateError(
                "context-only count response is not an object"
            )
        except Exception as exc:
            self._log_fallback("count_context_only_in_window", exc)
            return self._fallback.count_context_only_in_window(
                session_key,
                window_hours,
                timestamp,
            )

    def _get_timestamp(self, path: str, *, session_key: str) -> datetime | None:
        data = self._request("GET", path, params={"session_key": session_key})
        if not isinstance(data, dict):
            raise AgentRuntimeProactiveStateError("timestamp response is not an object")
        if not bool(data.get("found")):
            return None
        timestamp = _parse_iso(str(data.get("timestamp") or ""))
        if timestamp is None:
            raise AgentRuntimeProactiveStateError("timestamp response is invalid")
        return timestamp

    def _request(
        self,
        method: str,
        path: str,
        *,
        params: dict[str, Any] | None = None,
        json_body: dict[str, Any] | None = None,
    ) -> Any:
        if not bool(getattr(self._config, "enabled", False)):
            raise AgentRuntimeProactiveStateError("agent runtime client disabled")
        if not self._base_url:
            raise AgentRuntimeProactiveStateError("agent runtime base_url missing")
        timeout = float(getattr(self._config, "request_timeout_seconds", 5.0) or 5.0)
        with httpx.Client(
            timeout=timeout,
            transport=self._transport,
            trust_env=True,
        ) as client:
            response = client.request(
                method.upper(),
                self._base_url + path,
                params=params,
                json=json_body,
                headers={"Content-Type": "application/json"},
            )
        if response.status_code >= 400:
            raise AgentRuntimeProactiveStateError(
                f"agent runtime HTTP {response.status_code}: {response.text[:500]}"
            )
        payload = response.json()
        if not isinstance(payload, dict):
            return payload
        code = payload.get("code")
        if code not in (None, "OK", 0):
            raise AgentRuntimeProactiveStateError(str(payload.get("message") or payload))
        return payload.get("data", payload)

    def _log_fallback(self, method: str, exc: Exception) -> None:
        logger.warning(
            "[agent_runtime_proactive_state] %s failed, using SQLite fallback: %s",
            method,
            exc,
        )


def _max_datetime(left: datetime | None, right: datetime | None) -> datetime | None:
    if left is None:
        return right
    if right is None:
        return left
    return left if left >= right else right
