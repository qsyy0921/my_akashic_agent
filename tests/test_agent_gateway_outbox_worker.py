from __future__ import annotations

from typing import Any

import pytest

from integrations.agent_gateway import AgentGatewayNoJob
from integrations.agent_gateway_outbox_worker import AgentGatewayOutboxWorker


class _FakeClient:
    def __init__(self, delivery: dict[str, Any] | None = None) -> None:
        self.delivery = delivery
        self.calls: list[tuple[str, Any]] = []

    async def lease_next_outbox(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(("lease_next_outbox", kwargs))
        if self.delivery is None:
            raise AgentGatewayNoJob("none")
        return self.delivery

    async def mark_outbox_succeeded(self, event_id: str) -> dict[str, Any]:
        self.calls.append(("mark_outbox_succeeded", event_id))
        return {"event_id": event_id, "status": "succeeded"}

    async def mark_outbox_failed(
        self,
        event_id: str,
        *,
        error_message: str,
        error_kind: str = "",
    ) -> dict[str, Any]:
        self.calls.append(("mark_outbox_failed", event_id, error_kind, error_message))
        return {"event_id": event_id, "status": "failed"}


class _FakePushTool:
    def __init__(self, result: str = "文本已发送") -> None:
        self.result = result
        self.calls: list[dict[str, Any]] = []

    async def execute(self, **kwargs: Any) -> str:
        self.calls.append(kwargs)
        return self.result


def _delivery(**overrides: Any) -> dict[str, Any]:
    value = {
        "event_id": "qq:private:1",
        "channel": {
            "kind": "qq",
            "account_id": "2365524513",
            "conversation_id": "1049511700",
            "conversation_type": "private",
        },
        "content": "hello",
        "attachments": [],
    }
    value.update(overrides)
    return value


def _worker(
    client: _FakeClient,
    push_tool: _FakePushTool,
) -> AgentGatewayOutboxWorker:
    return AgentGatewayOutboxWorker(
        client=client,  # type: ignore[arg-type]
        push_tool=push_tool,
        worker_id="worker-a",
        channel_by_account={"2365524513": "qq_2365524513"},
        lease_ttl_seconds=60,
        poll_interval_seconds=0.5,
    )


@pytest.mark.asyncio
async def test_outbox_worker_dispatches_text_and_marks_succeeded():
    client = _FakeClient(_delivery())
    push_tool = _FakePushTool()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["processed"] is True
    assert result["event_id"] == "qq:private:1"
    assert push_tool.calls == [
        {
            "channel": "qq_2365524513",
            "chat_id": "1049511700",
            "message": "hello",
        }
    ]
    assert client.calls[-1] == ("mark_outbox_succeeded", "qq:private:1")


@pytest.mark.asyncio
async def test_outbox_worker_dispatches_media_with_message_once():
    client = _FakeClient(
        _delivery(
            attachments=[
                {"kind": "image", "url": "file:///E:/agent/akashic/.tmp/a.png"},
                {"kind": "file", "url": "file:///E:/agent/akashic/.tmp/a.pdf"},
            ]
        )
    )
    push_tool = _FakePushTool("图片已发送")
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["dispatch_count"] == 2
    assert push_tool.calls[0] == {
        "channel": "qq_2365524513",
        "chat_id": "1049511700",
        "message": "hello",
        "image": "E:/agent/akashic/.tmp/a.png",
    }
    assert push_tool.calls[1] == {
        "channel": "qq_2365524513",
        "chat_id": "1049511700",
        "message": "",
        "file": "E:/agent/akashic/.tmp/a.pdf",
    }
    assert client.calls[-1] == ("mark_outbox_succeeded", "qq:private:1")


@pytest.mark.asyncio
async def test_outbox_worker_marks_failed_when_push_tool_returns_error():
    client = _FakeClient(_delivery())
    push_tool = _FakePushTool("发送失败：platform timeout")
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["failed"] is True
    assert result["error_kind"] == "platform_timeout"
    assert client.calls[-1] == (
        "mark_outbox_failed",
        "qq:private:1",
        "platform_timeout",
        "发送失败：platform timeout",
    )


@pytest.mark.asyncio
async def test_outbox_worker_classifies_route_errors():
    client = _FakeClient(
        _delivery(channel={"kind": "", "conversation_id": "1049511700"})
    )
    push_tool = _FakePushTool()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["failed"] is True
    assert result["error_kind"] == "route_error"
    assert client.calls[-1] == (
        "mark_outbox_failed",
        "qq:private:1",
        "route_error",
        "outbox delivery missing channel kind",
    )


@pytest.mark.asyncio
async def test_outbox_worker_returns_idle_when_no_delivery():
    client = _FakeClient()
    push_tool = _FakePushTool()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result == {"processed": False, "reason": "no_delivery"}
