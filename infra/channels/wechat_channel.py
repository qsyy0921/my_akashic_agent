from __future__ import annotations

from typing import Any

from core.net.http import HttpRequester


class WechatWebhookChannel:
    """Outbound-only WeCom/WeChat group robot webhook channel."""

    def __init__(
        self,
        *,
        webhook_url: str,
        requester: HttpRequester,
        mentioned_list: list[str] | None = None,
        mentioned_mobile_list: list[str] | None = None,
    ) -> None:
        self._webhook_url = webhook_url
        self._requester = requester
        self._mentioned_list = list(mentioned_list or [])
        self._mentioned_mobile_list = list(mentioned_mobile_list or [])

    async def start(self) -> None:
        return None

    async def stop(self) -> None:
        return None

    async def send(self, chat_id: str, message: str) -> None:
        text = str(message or "").strip()
        if not text:
            return
        content: dict[str, Any] = {"content": text}
        if self._mentioned_list:
            content["mentioned_list"] = self._mentioned_list
        if self._mentioned_mobile_list:
            content["mentioned_mobile_list"] = self._mentioned_mobile_list
        payload = {
            "msgtype": "text",
            "text": content,
        }
        response = await self._requester.post(
            self._webhook_url,
            json=payload,
            timeout_s=15.0,
        )
        if response.status_code >= 400:
            raise RuntimeError(f"微信 webhook HTTP {response.status_code}: {response.text}")
        try:
            body = response.json()
        except Exception as exc:
            raise RuntimeError("微信 webhook 响应不是 JSON") from exc

        errcode = body.get("errcode", body.get("code", 0))
        if errcode not in (0, "0"):
            errmsg = body.get("errmsg", body.get("msg", body))
            raise RuntimeError(f"微信 webhook 发送失败: {errmsg}")
