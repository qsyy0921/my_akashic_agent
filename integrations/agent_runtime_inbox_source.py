from __future__ import annotations

import json
import re
from typing import Any

import httpx

from agent.config_models import AgentGatewayIntegrationConfig
from integrations.agent_gateway import AgentGatewayError


class AgentRuntimeInboxGroupMessageSource:
    """Group-memory source backed by agent-runtime `/v1/inbox`."""

    def __init__(
        self,
        config: AgentGatewayIntegrationConfig,
        *,
        transport: httpx.BaseTransport | None = None,
    ) -> None:
        self._config = config
        self._base_url = str(config.base_url or "").rstrip("/")
        self._transport = transport

    def fetch_new_messages(
        self,
        *,
        session_key: str,
        group_id: str,
        after_seq: int,
        limit: int,
    ) -> list[dict[str, Any]]:
        data = self._request(
            "/v1/inbox",
            params={
                "channel_kind": "qq",
                "conversation_id": str(group_id),
                "conversation_type": "group",
                "observe_only": "true",
                "after_seq": max(-1, int(after_seq)),
                "order": "asc",
                "limit": max(1, min(int(limit), 5000)),
            },
        )
        if not isinstance(data, list):
            raise AgentGatewayError("agent runtime inbox response is not a list")
        rows = [
            _inbox_event_to_session_row(event, session_key=session_key, group_id=group_id)
            for event in data
            if isinstance(event, dict)
        ]
        return [row for row in rows if row is not None]

    def _request(self, path: str, *, params: dict[str, Any]) -> Any:
        if not self._config.enabled:
            raise AgentGatewayError("agent runtime inbox source 未启用")
        if not self._base_url:
            raise AgentGatewayError("agent runtime base_url 未配置")
        with httpx.Client(
            timeout=self._config.request_timeout_seconds,
            transport=self._transport,
            trust_env=True,
        ) as client:
            response = client.get(self._base_url + path, params=params)
        if response.status_code >= 400:
            raise AgentGatewayError(
                f"agent runtime HTTP {response.status_code}: {response.text[:500]}"
            )
        try:
            payload = response.json()
        except json.JSONDecodeError:
            raise AgentGatewayError("agent runtime inbox returned non-json response")
        if not isinstance(payload, dict):
            return payload
        code = payload.get("code")
        if code not in (None, "OK", 0):
            raise AgentGatewayError(str(payload.get("message") or payload))
        return payload.get("data", payload)


def _inbox_event_to_session_row(
    event: dict[str, Any],
    *,
    session_key: str,
    group_id: str,
) -> dict[str, Any] | None:
    metadata = event.get("metadata") if isinstance(event.get("metadata"), dict) else {}
    seq = _event_seq(event, metadata)
    if seq is None:
        return None
    sender = event.get("sender") if isinstance(event.get("sender"), dict) else {}
    sender_id = str(sender.get("id") or metadata.get("sender_id") or "").strip()
    if not sender_id:
        sender_id = "unknown"
    content = str(event.get("content") or "").strip()
    if not content:
        return None
    if not _looks_like_group_observation(content):
        content = f"[QQ群 {group_id} | {sender_id}] {content}"
    message_id = str(
        metadata.get("session_message_id")
        or metadata.get("message_id")
        or event.get("event_id")
        or f"{session_key}:{seq}"
    )
    return {
        "id": message_id,
        "session_key": str(metadata.get("session_key") or session_key),
        "seq": seq,
        "role": "user",
        "content": content,
        "timestamp": str(event.get("timestamp") or event.get("received_at") or ""),
        "sender_id": sender_id,
    }


def _event_seq(event: dict[str, Any], metadata: dict[str, Any]) -> int | None:
    for raw in (
        metadata.get("seq"),
        metadata.get("session_seq"),
        _seq_from_id(str(metadata.get("session_message_id") or "")),
        _seq_from_id(str(event.get("event_id") or "")),
    ):
        if raw in (None, ""):
            continue
        try:
            parsed = int(raw)
        except (TypeError, ValueError):
            continue
        if parsed >= 0:
            return parsed
    return None


def _seq_from_id(value: str) -> str:
    match = re.search(r":(\d+)$", value.strip())
    return match.group(1) if match else ""


def _looks_like_group_observation(content: str) -> bool:
    return bool(re.match(r"^\[QQ群\s+[^|]+\|\s*[^\]]+\]", content.strip()))
