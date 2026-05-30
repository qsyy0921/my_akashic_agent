from __future__ import annotations

from typing import Any

import pytest

from integrations.agent_gateway import AgentGatewayDeliveryDispatchError
from integrations.agent_gateway import AgentGatewayDeliveryDispatchUnavailable
from integrations.agent_gateway import AgentGatewayNoJob
from integrations.agent_gateway import AgentGatewayDeliveryPlanError
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


class _FakePushToolWithDirect(_FakePushTool):
    def __init__(self, result: str = "文本已发送") -> None:
        super().__init__(result)
        self.direct_calls: list[dict[str, Any]] = []

    async def execute_direct(self, **kwargs: Any) -> str:
        self.direct_calls.append(kwargs)
        return self.result


class _FakePlanClient(_FakeClient):
    def __init__(
        self,
        delivery: dict[str, Any],
        plan: dict[str, Any] | None = None,
        plan_error: Exception | None = None,
    ) -> None:
        super().__init__(delivery)
        self.plan = plan
        self.plan_error = plan_error

    async def plan_outbox_dispatch(
        self,
        event_id: str,
        *,
        channel_by_account: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("plan_outbox_dispatch", event_id, channel_by_account or {}))
        if self.plan_error is not None:
            raise self.plan_error
        return self.plan or {"event_id": event_id, "steps": []}


class _FakeDispatchClient(_FakePlanClient):
    def __init__(
        self,
        delivery: dict[str, Any],
        dispatch: dict[str, Any] | None = None,
        dispatch_error: Exception | None = None,
        plan: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(delivery, plan=plan)
        self.dispatch = dispatch
        self.dispatch_error = dispatch_error

    async def dispatch_outbox_delivery(
        self,
        event_id: str,
        *,
        channel_by_account: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        self.calls.append(
            ("dispatch_outbox_delivery", event_id, channel_by_account or {})
        )
        if self.dispatch_error is not None:
            raise self.dispatch_error
        return self.dispatch or {
            "event_id": event_id,
            "results": [
                {
                    "step_index": 1,
                    "kind": "text",
                    "channel": "telegram",
                    "chat_id": "8655199155",
                    "status": "sent",
                    "provider": "telegram",
                    "provider_message_id": "123",
                }
            ],
        }


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


def _telegram_delivery(**overrides: Any) -> dict[str, Any]:
    return _delivery(
        event_id="telegram:private:1",
        channel={
            "kind": "telegram",
            "account_id": "",
            "conversation_id": "8655199155",
            "conversation_type": "private",
        },
        **overrides,
    )


def _worker(
    client: _FakeClient,
    push_tool: _FakePushTool,
    *,
    runtime_dispatch_channels: list[str] | None = None,
) -> AgentGatewayOutboxWorker:
    return AgentGatewayOutboxWorker(
        client=client,  # type: ignore[arg-type]
        push_tool=push_tool,
        worker_id="worker-a",
        channel_by_account={"2365524513": "qq_2365524513"},
        runtime_dispatch_channels=runtime_dispatch_channels,
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
async def test_outbox_worker_uses_direct_push_path_to_avoid_requeue():
    client = _FakeClient(_delivery())
    push_tool = _FakePushToolWithDirect()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["processed"] is True
    assert push_tool.calls == []
    assert push_tool.direct_calls == [
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
async def test_outbox_worker_prefers_runtime_dispatch_plan():
    client = _FakePlanClient(
        _delivery(),
        {
            "event_id": "qq:private:1",
            "steps": [
                {
                    "step_index": 1,
                    "kind": "image",
                    "channel": "qq_2365524513",
                    "chat_id": "1049511700",
                    "message": "hello",
                    "image": "E:/agent/akashic/.tmp/a.png",
                },
                {
                    "step_index": 2,
                    "kind": "text",
                    "channel": "qq_2365524513",
                    "chat_id": "1049511700",
                    "message": "done",
                },
            ],
        },
    )
    push_tool = _FakePushTool("图片已发送")
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["dispatch_count"] == 2
    assert client.calls[1] == (
        "plan_outbox_dispatch",
        "qq:private:1",
        {"2365524513": "qq_2365524513"},
    )
    assert push_tool.calls == [
        {
            "channel": "qq_2365524513",
            "chat_id": "1049511700",
            "message": "hello",
            "image": "E:/agent/akashic/.tmp/a.png",
        },
        {
            "channel": "qq_2365524513",
            "chat_id": "1049511700",
            "message": "done",
        },
    ]


@pytest.mark.asyncio
async def test_outbox_worker_prefers_go_runtime_dispatch_for_telegram():
    client = _FakeDispatchClient(_telegram_delivery())
    push_tool = _FakePushTool()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["processed"] is True
    assert result["dispatch_count"] == 1
    assert client.calls[1] == (
        "dispatch_outbox_delivery",
        "telegram:private:1",
        {"2365524513": "qq_2365524513"},
    )
    assert push_tool.calls == []
    assert client.calls[-1] == ("mark_outbox_succeeded", "telegram:private:1")


@pytest.mark.asyncio
async def test_outbox_worker_prefers_go_runtime_dispatch_for_configured_qq_channel():
    client = _FakeDispatchClient(
        _delivery(),
        dispatch={
            "event_id": "qq:private:1",
            "results": [
                {
                    "step_index": 1,
                    "kind": "text",
                    "channel": "qq_2365524513",
                    "chat_id": "1049511700",
                    "status": "sent",
                    "provider": "onebot",
                    "provider_message_id": "42",
                }
            ],
        },
    )
    push_tool = _FakePushTool()
    worker = _worker(
        client,
        push_tool,
        runtime_dispatch_channels=["telegram", "qq_2365524513"],
    )

    result = await worker.process_once()

    assert result["processed"] is True
    assert result["dispatch_count"] == 1
    assert client.calls[1] == (
        "dispatch_outbox_delivery",
        "qq:private:1",
        {"2365524513": "qq_2365524513"},
    )
    assert push_tool.calls == []
    assert client.calls[-1] == ("mark_outbox_succeeded", "qq:private:1")


@pytest.mark.asyncio
async def test_outbox_worker_falls_back_when_go_runtime_dispatch_unavailable():
    client = _FakeDispatchClient(
        _telegram_delivery(),
        dispatch_error=AgentGatewayDeliveryDispatchUnavailable(
            "delivery adapter unavailable"
        ),
        plan={
            "event_id": "telegram:private:1",
            "steps": [
                {
                    "step_index": 1,
                    "kind": "text",
                    "channel": "telegram",
                    "chat_id": "8655199155",
                    "message": "hello",
                }
            ],
        },
    )
    push_tool = _FakePushTool()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["dispatch_count"] == 1
    assert client.calls[1][0] == "dispatch_outbox_delivery"
    assert client.calls[2][0] == "plan_outbox_dispatch"
    assert push_tool.calls == [
        {
            "channel": "telegram",
            "chat_id": "8655199155",
            "message": "hello",
        }
    ]


@pytest.mark.asyncio
async def test_outbox_worker_uses_go_runtime_dispatch_error_kind():
    client = _FakeDispatchClient(
        _telegram_delivery(),
        dispatch_error=AgentGatewayDeliveryDispatchError(
            "platform_error",
            "telegram platform error",
        ),
    )
    push_tool = _FakePushTool()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["failed"] is True
    assert result["error_kind"] == "platform_error"
    assert push_tool.calls == []
    assert client.calls[-1] == (
        "mark_outbox_failed",
        "telegram:private:1",
        "platform_error",
        "telegram platform error",
    )


@pytest.mark.asyncio
async def test_outbox_worker_uses_runtime_plan_error_kind():
    client = _FakePlanClient(
        _delivery(),
        plan_error=AgentGatewayDeliveryPlanError(
            "route_error",
            "outbox delivery missing channel kind",
        ),
    )
    push_tool = _FakePushTool()
    worker = _worker(client, push_tool)

    result = await worker.process_once()

    assert result["failed"] is True
    assert result["error_kind"] == "route_error"
    assert push_tool.calls == []
    assert client.calls[-1] == (
        "mark_outbox_failed",
        "qq:private:1",
        "route_error",
        "outbox delivery missing channel kind",
    )


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
