from __future__ import annotations

import asyncio
import json
from datetime import datetime
from pathlib import Path
from uuid import uuid4

import pytest

from bus.events import InboundMessage
from bus.queue import MessageBus
from bus.shadow_gateway import (
    ObservedSessionShadowMirror,
    ShadowGatewayObserver,
    ShadowGatewaySettings,
    inbound_message_to_shadow_contract,
    session_message_to_shadow_inbound,
)
from core.contracts import ContractFixture
from session.manager import SessionManager


ROOT = Path(__file__).resolve().parents[1]


def test_inbound_message_to_shadow_contract_preserves_route_and_metadata():
    workdir = _test_workdir()
    image = workdir / "screen.png"
    image.write_bytes(b"png")
    msg = InboundMessage(
        channel="qq_2365524513",
        sender="1049511700",
        chat_id="gqq:27234224",
        content="看看这张图",
        timestamp=datetime.fromisoformat("2026-05-30T10:00:00+08:00"),
        media=[str(image)],
        metadata={
            "chat_type": "group",
            "group_id": "27234224",
            "platform_message_id": "msg-1",
            "observe_only": True,
        },
    )

    payload = inbound_message_to_shadow_contract(
        msg,
        channel_account_ids={"qq_2365524513": "2365524513"},
        default_agent_id="hardware-observer",
    )
    fixture = ContractFixture.from_dict(payload)

    assert fixture.event_id == "qq:2365524513:group:27234224:msg-1"
    assert fixture.platform == "qq"
    assert fixture.account_id == "2365524513"
    assert fixture.conversation_id == "27234224"
    assert fixture.conversation_type == "group"
    assert fixture.source_asset_ids
    assert fixture.metadata["observe_only"] == "true"
    assert fixture.metadata["original_channel"] == "qq_2365524513"


@pytest.mark.asyncio
async def test_shadow_observer_writes_jsonl_after_bus_enqueue():
    bus = MessageBus()
    workdir = _test_workdir()
    observer = ShadowGatewayObserver(
        settings=ShadowGatewaySettings(enabled=True, log_path="shadow/inbound.jsonl"),
        workspace=workdir,
        channel_account_ids={"qq": "1049511700"},
    )
    bus.add_inbound_observer(observer)
    msg = InboundMessage(
        channel="qq",
        sender="2365524513",
        chat_id="1049511700",
        content="/ask hello",
    )

    await bus.publish_inbound(msg)
    consumed = await asyncio.wait_for(bus.consume_inbound(), timeout=1)

    assert consumed is msg
    await asyncio.sleep(0.1)
    log_path = workdir / "shadow" / "inbound.jsonl"
    lines = log_path.read_text(encoding="utf-8").splitlines()
    payload = json.loads(lines[-1])
    assert payload["account_id"] == "1049511700"
    assert payload["metadata"]["session_key"] == "qq:1049511700"


@pytest.mark.asyncio
async def test_observe_only_session_messages_are_mirrored_to_shadow_log():
    workdir = _test_workdir()
    observer = ShadowGatewayObserver(
        settings=ShadowGatewaySettings(enabled=True, log_path="shadow/session.jsonl"),
        workspace=workdir,
        channel_account_ids={"qq": "2365524513"},
    )
    manager = SessionManager(workdir)
    manager.add_message_observer(ObservedSessionShadowMirror(observer))
    session = manager.get_or_create("qq:gqq:27234224")
    session.add_message(
        "user",
        "[QQ群 27234224 | 2948770636] [图片 x 1]",
        observe_only=True,
        chat_type="group",
        group_id="27234224",
        sender_id="2948770636",
        platform_message_id="msg-498",
    )

    await manager.save_async(session)

    log_path = workdir / "shadow" / "session.jsonl"
    for _ in range(20):
        if log_path.exists():
            break
        await asyncio.sleep(0.05)
    payload = json.loads(log_path.read_text(encoding="utf-8").splitlines()[-1])
    assert payload["event_id"] == "qq:2365524513:group:27234224:msg-498"
    assert payload["conversation_type"] == "group"
    assert payload["metadata"]["observe_only"] == "true"


def test_non_observe_only_session_messages_are_not_shadowed():
    inbound = session_message_to_shadow_inbound(
        "qq:gqq:27234224",
        {
            "role": "user",
            "content": "normal saved message",
            "timestamp": "2026-05-30T10:00:00+08:00",
            "extra": {"chat_type": "group"},
        },
    )

    assert inbound is None


@pytest.mark.asyncio
async def test_inbound_observer_failure_does_not_block_message_bus():
    bus = MessageBus()

    async def failing_observer(_msg):
        raise RuntimeError("boom")

    bus.add_inbound_observer(failing_observer)
    msg = InboundMessage(channel="telegram", sender="u", chat_id="1", content="hi")

    await bus.publish_inbound(msg)

    assert await asyncio.wait_for(bus.consume_inbound(), timeout=1) is msg


def _test_workdir() -> Path:
    path = ROOT / ".tmp" / "pytest-shadow" / uuid4().hex
    path.mkdir(parents=True, exist_ok=True)
    return path
