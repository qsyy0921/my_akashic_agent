from __future__ import annotations

import asyncio
import logging
from typing import Any
from urllib.parse import unquote, urlparse

from integrations.agent_gateway import (
    AgentGatewayClient,
    AgentGatewayError,
    AgentGatewayNoJob,
)

logger = logging.getLogger(__name__)


class DeliveryDispatchError(RuntimeError):
    def __init__(self, kind: str, message: str) -> None:
        super().__init__(message)
        self.kind = kind or "unknown"


class AgentGatewayOutboxWorker:
    def __init__(
        self,
        *,
        client: AgentGatewayClient,
        push_tool: Any,
        worker_id: str,
        channel_by_account: dict[str, str] | None = None,
        lease_ttl_seconds: int = 300,
        poll_interval_seconds: float = 2.0,
    ) -> None:
        self._client = client
        self._push_tool = push_tool
        self._worker_id = str(worker_id or "akashic-python-worker")
        self._channel_by_account = {
            str(key): str(value)
            for key, value in (channel_by_account or {}).items()
            if str(key).strip() and str(value).strip()
        }
        self._lease_ttl = max(10, int(lease_ttl_seconds or 300))
        self._poll_interval = max(0.5, float(poll_interval_seconds or 2.0))
        self._stopped = asyncio.Event()

    async def process_once(self) -> dict[str, Any]:
        try:
            delivery = await self._client.lease_next_outbox(
                worker_id=self._worker_id,
                ttl_seconds=self._lease_ttl,
            )
        except AgentGatewayNoJob:
            return {"processed": False, "reason": "no_delivery"}

        event_id = str(delivery.get("event_id") or "")
        try:
            results = await self._dispatch_delivery(delivery)
            await self._client.mark_outbox_succeeded(event_id)
            return {
                "processed": True,
                "event_id": event_id,
                "dispatch_count": len(results),
            }
        except Exception as exc:
            message = str(exc)
            error_kind = getattr(exc, "kind", "unknown")
            logger.exception(
                "[agent_runtime_outbox_worker] delivery failed event_id=%s", event_id
            )
            await self._safe_fail(event_id, message, error_kind=str(error_kind))
            return {
                "processed": True,
                "event_id": event_id,
                "failed": True,
                "error_kind": str(error_kind),
                "error": message,
            }

    async def run(self) -> None:
        logger.info(
            "[agent_runtime_outbox_worker] loop started worker_id=%s",
            self._worker_id,
        )
        try:
            while not self._stopped.is_set():
                try:
                    result = await self.process_once()
                    if result.get("processed") and not result.get("failed"):
                        logger.info(
                            "[agent_runtime_outbox_worker] processed %s", result
                        )
                except AgentGatewayError as exc:
                    logger.warning(
                        "[agent_runtime_outbox_worker] runtime error: %s", exc
                    )
                except Exception:
                    logger.exception(
                        "[agent_runtime_outbox_worker] unexpected loop error"
                    )
                try:
                    await asyncio.wait_for(
                        self._stopped.wait(),
                        timeout=self._poll_interval,
                    )
                except asyncio.TimeoutError:
                    continue
        finally:
            logger.info("[agent_runtime_outbox_worker] loop stopped")

    def stop(self) -> None:
        self._stopped.set()

    async def _dispatch_delivery(self, delivery: dict[str, Any]) -> list[str]:
        route = _delivery_route(delivery)
        channel_name = self._resolve_channel_name(route)
        chat_id = str(route.get("conversation_id") or "").strip()
        if not channel_name:
            raise DeliveryDispatchError(
                "route_error",
                "outbox delivery missing channel kind",
            )
        if not chat_id:
            raise DeliveryDispatchError(
                "route_error",
                "outbox delivery missing conversation_id",
            )

        content = str(delivery.get("content") or "")
        attachments = _delivery_attachments(delivery)
        images = [
            _normalise_attachment_uri(item)
            for item in attachments
            if str(item.get("kind") or "").lower() == "image"
        ]
        files = [
            _normalise_attachment_uri(item)
            for item in attachments
            if str(item.get("kind") or "").lower() != "image"
        ]

        results: list[str] = []
        message = content
        for image in images:
            result = await self._send(
                channel=channel_name,
                chat_id=chat_id,
                message=message,
                image=image,
            )
            results.append(result)
            message = ""
        for file_path in files:
            result = await self._send(
                channel=channel_name,
                chat_id=chat_id,
                message=message,
                file=file_path,
            )
            results.append(result)
            message = ""
        if message or not results:
            results.append(
                await self._send(
                    channel=channel_name,
                    chat_id=chat_id,
                    message=message,
                )
            )
        return results

    def _resolve_channel_name(self, route: dict[str, Any]) -> str:
        account_id = str(route.get("account_id") or "").strip()
        if account_id and account_id in self._channel_by_account:
            return self._channel_by_account[account_id]
        return str(route.get("kind") or "").strip()

    async def _send(self, **kwargs: Any) -> str:
        result = await self._push_tool.execute(**kwargs)
        text = str(result or "")
        if _is_send_failure(text):
            raise DeliveryDispatchError(_failure_kind(text), text)
        return text

    async def _safe_fail(
        self,
        event_id: str,
        message: str,
        *,
        error_kind: str,
    ) -> None:
        if not event_id:
            return
        try:
            await self._client.mark_outbox_failed(
                event_id,
                error_kind=error_kind,
                error_message=message,
            )
        except Exception:
            logger.warning(
                "[agent_runtime_outbox_worker] fail update failed event_id=%s",
                event_id,
                exc_info=True,
            )


def _delivery_route(delivery: dict[str, Any]) -> dict[str, Any]:
    route = delivery.get("channel") if isinstance(delivery.get("channel"), dict) else {}
    if route:
        return route
    return {
        "kind": delivery.get("channel_kind"),
        "account_id": delivery.get("account_id"),
        "conversation_id": delivery.get("conversation_id"),
        "conversation_type": delivery.get("conversation_type"),
    }


def _delivery_attachments(delivery: dict[str, Any]) -> list[dict[str, Any]]:
    raw = delivery.get("attachments")
    if not isinstance(raw, list):
        return []
    return [item for item in raw if isinstance(item, dict) and item.get("url")]


def _normalise_attachment_uri(attachment: dict[str, Any]) -> str:
    value = str(attachment.get("url") or "").strip()
    parsed = urlparse(value)
    if parsed.scheme != "file":
        return value
    path = unquote(parsed.path)
    if len(path) >= 3 and path[0] == "/" and path[2] == ":":
        path = path[1:]
    return path


def _is_send_failure(result: str) -> bool:
    return any(
        marker in result
        for marker in (
            "发送失败",
            "未注册",
            "不支持",
            "没有可用",
            "至少提供一个",
            "错误：",
        )
    )


def _failure_kind(result: str) -> str:
    text = str(result or "").lower()
    if "未注册" in result:
        return "route_error"
    if "不支持" in result:
        return "unsupported_media"
    if "没有可用" in result:
        return "sender_unavailable"
    if "timeout" in text or "timed out" in text or "超时" in result:
        return "platform_timeout"
    if "至少提供一个" in result or "错误：" in result:
        return "validation_error"
    if "发送失败" in result:
        return "platform_error"
    return "unknown"
