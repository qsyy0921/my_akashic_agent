from __future__ import annotations

import base64
import hashlib
import hmac
import time
from typing import Any

from core.net.http import HttpRequester


class FeishuWebhookChannel:
    """Outbound-only Feishu custom bot webhook channel."""

    def __init__(
        self,
        *,
        webhook_url: str,
        secret: str = "",
        requester: HttpRequester,
    ) -> None:
        self._webhook_url = webhook_url
        self._secret = secret
        self._requester = requester

    async def start(self) -> None:
        return None

    async def stop(self) -> None:
        return None

    async def send(self, chat_id: str, message: str) -> None:
        text = str(message or "").strip()
        if not text:
            return
        payload: dict[str, Any] = {
            "msg_type": "text",
            "content": {"text": text},
        }
        if self._secret:
            timestamp = str(int(time.time()))
            payload["timestamp"] = timestamp
            payload["sign"] = _feishu_sign(self._secret, timestamp)
        response = await self._requester.post(
            self._webhook_url,
            json=payload,
            timeout_s=15.0,
        )
        if response.status_code >= 400:
            raise RuntimeError(f"飞书 webhook HTTP {response.status_code}: {response.text}")
        try:
            body = response.json()
        except Exception as exc:
            raise RuntimeError("飞书 webhook 响应不是 JSON") from exc

        code = body.get("code", body.get("StatusCode", 0))
        if code not in (0, "0"):
            msg = body.get("msg", body.get("StatusMessage", body))
            raise RuntimeError(f"飞书 webhook 发送失败: {msg}")


def _feishu_sign(secret: str, timestamp: str) -> str:
    string_to_sign = f"{timestamp}\n{secret}".encode("utf-8")
    digest = hmac.new(string_to_sign, b"", hashlib.sha256).digest()
    return base64.b64encode(digest).decode("utf-8")
