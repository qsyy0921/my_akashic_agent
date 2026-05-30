from __future__ import annotations

import mimetypes
import uuid
from pathlib import Path
from typing import Any

from integrations.agent_gateway import AgentGatewayClient


class AgentRuntimeOutboundEnqueuer:
    def __init__(
        self,
        client: AgentGatewayClient,
        *,
        account_id_by_channel: dict[str, str] | None = None,
    ) -> None:
        self._client = client
        self._account_id_by_channel = {
            str(key).strip(): str(value).strip()
            for key, value in (account_id_by_channel or {}).items()
            if str(key).strip() and str(value).strip()
        }

    async def enqueue(
        self,
        *,
        channel: str,
        chat_id: str,
        message: str = "",
        file: str | None = None,
        image: str | None = None,
    ) -> str:
        channel = str(channel or "").strip()
        chat_id = str(chat_id or "").strip()
        message = str(message or "")
        attachments = _attachments(file=file, image=image)
        if not channel or not chat_id:
            raise ValueError("runtime outbound requires channel and chat_id")
        if not message and not attachments:
            raise ValueError("runtime outbound requires message, file, or image")
        event_id = f"pyout:{channel}:{chat_id}:{uuid.uuid4().hex}"
        await self._client.send_outbound(
            event_id=event_id,
            channel={
                "kind": channel,
                "platform": channel,
                "account_id": self._account_id_by_channel.get(channel, channel),
                "conversation_id": chat_id,
                "conversation_type": _conversation_type(channel, chat_id),
            },
            content=message,
            attachments=attachments,
            metadata={
                "source": "message_push",
                "runtime_outbound": "true",
            },
        )
        return _queued_result(message=message, file=file, image=image)


def _attachments(
    *,
    file: str | None,
    image: str | None,
) -> list[dict[str, Any]]:
    result: list[dict[str, Any]] = []
    if image:
        result.append(_attachment("image", image))
    if file:
        result.append(_attachment("file", file))
    return result


def _attachment(kind: str, value: str) -> dict[str, Any]:
    path = str(value or "").strip()
    item: dict[str, Any] = {
        "kind": kind,
        "url": path,
    }
    name = Path(path).name
    if name:
        item["name"] = name
    mime_type, _ = mimetypes.guess_type(name or path)
    if mime_type:
        item["mime_type"] = mime_type
    return item


def _conversation_type(channel: str, chat_id: str) -> str:
    if channel.startswith("telegram") and str(chat_id).startswith("-"):
        return "group"
    return "private"


def _queued_result(
    *,
    message: str,
    file: str | None,
    image: str | None,
) -> str:
    parts: list[str] = []
    if message:
        parts.append("文本已发送（Go outbox 已接收）")
    if image:
        parts.append("图片已发送（Go outbox 已接收）")
    if file:
        parts.append(f"文件 {Path(str(file)).name!r} 已发送（Go outbox 已接收）")
    return "；".join(parts) if parts else "消息已发送（Go outbox 已接收）"
