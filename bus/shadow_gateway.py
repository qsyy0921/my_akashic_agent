from __future__ import annotations

import asyncio
import json
import logging
import mimetypes
import urllib.error
import urllib.request
from collections.abc import Mapping
from dataclasses import dataclass
from datetime import datetime
from hashlib import sha256
from pathlib import Path
from typing import Any

from core.contracts import ContractFixture
from bus.events import InboundItem, InboundMessage

logger = logging.getLogger(__name__)

SCHEMA_VERSION = "2026-05-30.v1"
GROUP_PREFIX = "gqq:"


@dataclass(frozen=True)
class ShadowGatewaySettings:
    enabled: bool = False
    endpoint: str = ""
    log_path: str = ""
    request_timeout_seconds: float = 2.0
    agent_id: str = "shadow"


class ShadowGatewayObserver:
    def __init__(
        self,
        *,
        settings: ShadowGatewaySettings,
        workspace: Path,
        channel_account_ids: Mapping[str, str] | None = None,
    ) -> None:
        self._settings = settings
        self._workspace = workspace
        self._channel_account_ids = dict(channel_account_ids or {})
        self._lock = asyncio.Lock()

    async def __call__(self, item: InboundItem) -> None:
        if not isinstance(item, InboundMessage):
            return
        payload = inbound_message_to_shadow_contract(
            item,
            channel_account_ids=self._channel_account_ids,
            default_agent_id=self._settings.agent_id,
        )
        await self._append_log(payload)
        if self._settings.endpoint:
            await self._post(payload)

    async def _append_log(self, payload: dict[str, Any]) -> None:
        path = self._resolve_log_path()
        if path is None:
            return
        line = json.dumps(payload, ensure_ascii=False, sort_keys=True) + "\n"
        async with self._lock:
            path.parent.mkdir(parents=True, exist_ok=True)
            await asyncio.to_thread(_append_text, path, line)

    def _resolve_log_path(self) -> Path | None:
        raw = str(self._settings.log_path or "").strip()
        if not raw:
            raw = "shadow/inbound.jsonl"
        path = Path(raw)
        if not path.is_absolute():
            path = self._workspace / path
        return path

    async def _post(self, payload: dict[str, Any]) -> None:
        try:
            await asyncio.to_thread(
                _post_json,
                self._settings.endpoint,
                payload,
                max(0.1, float(self._settings.request_timeout_seconds)),
            )
        except (OSError, urllib.error.URLError, TimeoutError) as exc:
            logger.warning("shadow gateway post failed: %s", exc)


def build_shadow_gateway_observer(
    *,
    settings: ShadowGatewaySettings,
    workspace: Path,
    channel_account_ids: Mapping[str, str] | None = None,
) -> ShadowGatewayObserver | None:
    if not settings.enabled:
        return None
    return ShadowGatewayObserver(
        settings=settings,
        workspace=workspace,
        channel_account_ids=channel_account_ids,
    )


def inbound_message_to_shadow_contract(
    msg: InboundMessage,
    *,
    channel_account_ids: Mapping[str, str] | None = None,
    default_agent_id: str = "shadow",
) -> dict[str, Any]:
    metadata = _string_metadata(msg.metadata)
    platform = _platform_for_channel(msg.channel)
    account_id = (
        metadata.get("account_id")
        or metadata.get("bot_uin")
        or (channel_account_ids or {}).get(msg.channel)
        or msg.channel
    )
    conversation_type = _conversation_type(msg.chat_id, metadata)
    conversation_id = _conversation_id(msg.chat_id, conversation_type)
    timestamp = _timestamp(msg.timestamp)
    event_id = _event_id(
        platform=platform,
        account_id=account_id,
        conversation_type=conversation_type,
        conversation_id=conversation_id,
        sender=msg.sender,
        content=msg.content,
        timestamp=timestamp,
        metadata=metadata,
    )
    attachments = _attachments(
        media=msg.media,
        platform=platform,
        conversation_id=conversation_id,
        event_id=event_id,
    )
    source_asset_ids = [str(item["id"]) for item in attachments if item.get("id")]
    payload: dict[str, Any] = {
        "schema_version": SCHEMA_VERSION,
        "kind": "MessageEnvelope",
        "event_id": event_id,
        "platform": platform,
        "account_id": account_id,
        "conversation_id": conversation_id,
        "conversation_type": conversation_type,
        "agent_id": metadata.get("agent_id") or default_agent_id or "shadow",
        "channel": {
            "kind": platform,
            "platform": platform,
            "account_id": account_id,
            "conversation_id": conversation_id,
            "conversation_type": conversation_type,
        },
        "sender": {
            "id": str(msg.sender),
            "display_name": metadata.get("username", ""),
            "kind": metadata.get("sender_kind", "human"),
        },
        "content": str(msg.content or ""),
        "attachments": attachments,
        "timestamp": timestamp,
        "source_message_ids": [],
        "source_asset_ids": source_asset_ids,
        "metadata": {
            **metadata,
            "shadow_mode": "true",
            "original_channel": msg.channel,
            "original_chat_id": msg.chat_id,
            "session_key": msg.session_key,
        },
    }
    ContractFixture.from_dict(payload)
    return payload


def _append_text(path: Path, line: str) -> None:
    with path.open("a", encoding="utf-8") as handle:
        handle.write(line)


def _post_json(endpoint: str, payload: dict[str, Any], timeout: float) -> None:
    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    request = urllib.request.Request(
        endpoint,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=timeout) as response:
        response.read()


def _string_metadata(metadata: Mapping[str, Any] | None) -> dict[str, str]:
    result: dict[str, str] = {}
    for key, value in (metadata or {}).items():
        if value is None:
            continue
        if isinstance(value, str):
            result[str(key)] = value
        elif isinstance(value, bool):
            result[str(key)] = "true" if value else "false"
        elif isinstance(value, int | float):
            result[str(key)] = str(value)
        else:
            result[str(key)] = json.dumps(value, ensure_ascii=False, sort_keys=True)
    return result


def _platform_for_channel(channel: str) -> str:
    if channel.startswith("qq"):
        return "qq"
    if channel.startswith("telegram"):
        return "telegram"
    if channel.startswith("feishu"):
        return "feishu"
    if channel.startswith("wechat"):
        return "wechat"
    return channel


def _conversation_type(chat_id: str, metadata: Mapping[str, str]) -> str:
    chat_type = str(metadata.get("chat_type", "")).strip()
    if chat_type in {"private", "group"}:
        return chat_type
    if chat_id.startswith(GROUP_PREFIX):
        return "group"
    if chat_id.startswith("-"):
        return "group"
    return "private"


def _conversation_id(chat_id: str, conversation_type: str) -> str:
    if conversation_type == "group" and chat_id.startswith(GROUP_PREFIX):
        return chat_id[len(GROUP_PREFIX) :]
    return chat_id


def _timestamp(value: datetime) -> str:
    if value.tzinfo is None:
        return value.astimezone().isoformat()
    return value.isoformat()


def _event_id(
    *,
    platform: str,
    account_id: str,
    conversation_type: str,
    conversation_id: str,
    sender: str,
    content: str,
    timestamp: str,
    metadata: Mapping[str, str],
) -> str:
    platform_message_id = (
        metadata.get("platform_message_id")
        or metadata.get("message_id")
        or metadata.get("raw_message_id")
    )
    if platform_message_id:
        suffix = platform_message_id
    else:
        seed = "\n".join(
            [platform, account_id, conversation_id, sender, timestamp, content]
        )
        suffix = sha256(seed.encode("utf-8")).hexdigest()[:16]
    return f"{platform}:{account_id}:{conversation_type}:{conversation_id}:{suffix}"


def _attachments(
    *,
    media: list[str],
    platform: str,
    conversation_id: str,
    event_id: str,
) -> list[dict[str, Any]]:
    attachments: list[dict[str, Any]] = []
    for index, raw in enumerate(media or [], start=1):
        path = Path(raw)
        mime_type, _ = mimetypes.guess_type(str(path))
        mime_type = mime_type or "application/octet-stream"
        kind = "image" if mime_type.startswith("image/") else "file"
        stat_size = path.stat().st_size if path.is_file() else 0
        asset_seed = f"{event_id}:{index}:{raw}"
        asset_hash = sha256(asset_seed.encode("utf-8")).hexdigest()[:16]
        asset_id = f"asset:{platform}:{kind}:{conversation_id}:{asset_hash}:{index}"
        url = path.resolve().as_uri() if path.exists() else str(raw)
        attachments.append(
            {
                "id": asset_id,
                "kind": kind,
                "url": url,
                "mime_type": mime_type,
                "name": path.name or f"attachment-{index}",
                "size_bytes": stat_size,
            }
        )
    return attachments
