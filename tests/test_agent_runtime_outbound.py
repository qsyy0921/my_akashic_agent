from __future__ import annotations

from typing import Any

import pytest

from integrations.agent_runtime_outbound import AgentRuntimeOutboundEnqueuer


class _FakeRuntimeClient:
    def __init__(self) -> None:
        self.calls: list[dict[str, Any]] = []

    async def send_outbound(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(kwargs)
        return {"event_id": kwargs["event_id"], "status": "pending"}


@pytest.mark.asyncio
async def test_agent_runtime_outbound_enqueuer_maps_message_push_to_outbound():
    client = _FakeRuntimeClient()
    enqueuer = AgentRuntimeOutboundEnqueuer(
        client,  # type: ignore[arg-type]
        account_id_by_channel={"telegram": "telegram-bot"},
    )

    result = await enqueuer.enqueue(
        channel="telegram",
        chat_id="-100",
        message="hello",
        image="E:/agent/akashic/.tmp/a.png",
        file="E:/agent/akashic/.tmp/a.pdf",
    )

    assert "Go outbox 已接收" in result
    assert len(client.calls) == 1
    call = client.calls[0]
    assert str(call["event_id"]).startswith("pyout:telegram:-100:")
    assert call["channel"] == {
        "kind": "telegram",
        "platform": "telegram",
        "account_id": "telegram-bot",
        "conversation_id": "-100",
        "conversation_type": "group",
    }
    assert call["content"] == "hello"
    assert call["attachments"] == [
        {
            "kind": "image",
            "url": "E:/agent/akashic/.tmp/a.png",
            "name": "a.png",
            "mime_type": "image/png",
        },
        {
            "kind": "file",
            "url": "E:/agent/akashic/.tmp/a.pdf",
            "name": "a.pdf",
            "mime_type": "application/pdf",
        },
    ]
    assert call["metadata"] == {
        "source": "message_push",
        "runtime_outbound": "true",
    }


@pytest.mark.asyncio
async def test_agent_runtime_outbound_enqueuer_rejects_empty_payload():
    enqueuer = AgentRuntimeOutboundEnqueuer(_FakeRuntimeClient())  # type: ignore[arg-type]

    with pytest.raises(ValueError, match="requires message"):
        await enqueuer.enqueue(channel="telegram", chat_id="100")
