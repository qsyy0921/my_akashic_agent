from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from agent.config_models import AgentGatewayIntegrationConfig
from integrations.agent_gateway import (
    AgentGatewayClient,
    AgentGatewayDeliveryDispatchError,
    AgentGatewayDeliveryDispatchUnavailable,
    AgentGatewayDeliveryPlanError,
    AgentGatewayError,
    AgentGatewayNoJob,
)


def _client(handler) -> AgentGatewayClient:
    return AgentGatewayClient(
        AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="http://agent-gateway.local",
            worker_id="worker-a",
            lease_ttl_seconds=120,
        ),
        transport=httpx.MockTransport(handler),
    )


def _ok(data: Any) -> httpx.Response:
    return httpx.Response(200, json={"code": "OK", "data": data})


@pytest.mark.asyncio
async def test_agent_gateway_client_creates_leases_and_completes_job():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        if request.url.path == "/v1/jobs":
            assert body["job_type"] == "image_generation"
            return httpx.Response(
                202,
                json={
                    "code": "OK",
                    "data": {"job_id": body["job_id"], "status": "pending"},
                },
            )
        if request.url.path == "/v1/jobs/lease-next":
            assert body == {
                "worker_id": "worker-a",
                "job_type": "image_generation",
                "ttl_seconds": 120,
            }
            return _ok({"job_id": "job-1", "status": "leased", "lease_token": "tok-1"})
        if request.url.path == "/v1/jobs/job-1/running":
            assert body == {"lease_token": "tok-1"}
            return _ok({"job_id": "job-1", "status": "running"})
        if request.url.path == "/v1/jobs/job-1/renew":
            assert body == {"lease_token": "tok-1", "ttl_seconds": 120}
            return _ok({"job_id": "job-1", "status": "running"})
        if request.url.path == "/v1/jobs/job-1/succeeded":
            assert body == {"result": {"asset_id": "asset-1"}, "lease_token": "tok-1"}
            return _ok({"job_id": "job-1", "status": "succeeded"})
        return httpx.Response(404, text="not found")

    client = _client(handler)
    created = await client.create_job(
        job_id="job-1",
        job_type="image_generation",
        agent_id="akashic-python-worker",
        route={
            "kind": "qq",
            "account_id": "2365524513",
            "conversation_id": "1049511700",
            "conversation_type": "private",
        },
        source_event_ids=["qq:private:1"],
        payload={"prompt": "古装美女"},
        max_attempts=2,
    )
    leased = await client.lease_next(job_type="image_generation")
    running = await client.mark_running("job-1", lease_token=leased["lease_token"])
    renewed = await client.renew_job("job-1", lease_token=leased["lease_token"])
    succeeded = await client.complete_job(
        "job-1",
        lease_token=leased["lease_token"],
        result={"asset_id": "asset-1"},
    )

    assert created["status"] == "pending"
    assert leased["status"] == "leased"
    assert running["status"] == "running"
    assert renewed["status"] == "running"
    assert succeeded["status"] == "succeeded"
    assert [path for _, path, _ in calls] == [
        "/v1/jobs",
        "/v1/jobs/lease-next",
        "/v1/jobs/job-1/running",
        "/v1/jobs/job-1/renew",
        "/v1/jobs/job-1/succeeded",
    ]


@pytest.mark.asyncio
async def test_agent_gateway_client_leases_exact_queue_work_id():
    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        assert request.method == "POST"
        assert request.url.path == "/v1/jobs/lease-work"
        assert body == {
            "work_kind": "agent_job",
            "work_id": "job-queue-1",
            "aggregate_id": "job-queue-1",
            "subject": "akashic.work.agent_job.rag_ingest",
            "worker_id": "worker-a",
            "ttl_seconds": 120,
        }
        return _ok(
            {
                "job_id": "job-queue-1",
                "status": "leased",
                "lease_token": "tok-work-1",
            }
        )

    leased = await _client(handler).lease_work(
        work_kind="agent_job",
        work_id="job-queue-1",
        aggregate_id="job-queue-1",
        subject="akashic.work.agent_job.rag_ingest",
    )

    assert leased["status"] == "leased"
    assert leased["lease_token"] == "tok-work-1"


@pytest.mark.asyncio
async def test_agent_gateway_client_recovers_expired_jobs():
    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        assert request.method == "POST"
        assert request.url.path == "/v1/jobs/recover-expired"
        assert body == {"limit": 7}
        return _ok(
            {
                "scanned": 1,
                "recovered": 1,
                "dead_lettered": 0,
                "items": [
                    {
                        "action": "recovered",
                        "job": {"job_id": "job-expired-1", "status": "pending"},
                    }
                ],
            }
        )

    result = await _client(handler).recover_expired_jobs(limit=7)

    assert result["recovered"] == 1
    assert result["items"][0]["action"] == "recovered"


@pytest.mark.asyncio
async def test_agent_gateway_client_lists_jobs_with_filters():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        assert request.url.path == "/v1/jobs"
        assert request.url.params["type"] == "rag_ingest"
        assert request.url.params["status"] == "pending"
        assert request.url.params["limit"] == "5"
        return _ok([{"job_id": "job-1", "status": "pending"}])

    items = await _client(handler).list_jobs(
        job_type="rag_ingest",
        status="pending",
        limit=5,
    )

    assert items == [{"job_id": "job-1", "status": "pending"}]


@pytest.mark.asyncio
async def test_agent_gateway_client_lists_inbox_events_with_cursor_filters():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        assert request.url.path == "/v1/inbox"
        assert dict(request.url.params) == {
            "limit": "20",
            "channel_kind": "qq",
            "conversation_id": "27234224",
            "conversation_type": "group",
            "observe_only": "true",
            "after_seq": "3",
            "order": "asc",
        }
        return _ok([{"event_id": "event-4", "metadata": {"seq": "4"}}])

    items = await _client(handler).list_inbox_events(
        channel_kind="qq",
        conversation_id="27234224",
        conversation_type="group",
        observe_only=True,
        after_seq=3,
        order="asc",
        limit=20,
    )

    assert items == [{"event_id": "event-4", "metadata": {"seq": "4"}}]


@pytest.mark.asyncio
async def test_agent_gateway_client_gets_and_updates_knowledge_checkpoint():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        raw_path = request.url.raw_path.decode()
        if request.method == "GET":
            assert raw_path == "/v1/knowledge-checkpoints/ragflow%3Aqq%3A27234224%3Ads1"
            return _ok({"checkpoint_id": "ragflow:qq:27234224:ds1", "cursor": 41})
        if request.method == "PUT":
            assert raw_path == "/v1/knowledge-checkpoints/ragflow%3Aqq%3A27234224%3Ads1"
            assert body == {"cursor": 42, "metadata": {"group_id": "27234224"}}
            return _ok({"checkpoint_id": "ragflow:qq:27234224:ds1", "cursor": 42})
        return httpx.Response(404, text="not found")

    client = _client(handler)
    found = await client.get_knowledge_checkpoint("ragflow:qq:27234224:ds1")
    updated = await client.update_knowledge_checkpoint(
        "ragflow:qq:27234224:ds1",
        cursor=42,
        metadata={"group_id": "27234224"},
    )

    assert found["cursor"] == 41
    assert updated["cursor"] == 42
    assert [call[0] for call in calls] == ["GET", "PUT"]


@pytest.mark.asyncio
async def test_agent_gateway_client_updates_legacy_image_job_state():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        return _ok({"job_id": "img-1", "status": "ok"})

    client = _client(handler)

    await client.mark_image_job_running("img-1")
    await client.complete_image_job(
        "img-1",
        results=[{"kind": "image", "url": "file:///x.png"}],
        metadata={"agent_job_id": "img-1"},
    )
    await client.fail_image_job("img-1", error_message="failed")

    assert calls == [
        ("POST", "/v1/image-jobs/img-1/running", {}),
        (
            "POST",
            "/v1/image-jobs/img-1/succeeded",
            {
                "results": [{"kind": "image", "url": "file:///x.png"}],
                "metadata": {"agent_job_id": "img-1"},
            },
        ),
        ("POST", "/v1/image-jobs/img-1/failed", {"error_message": "failed"}),
    ]


@pytest.mark.asyncio
async def test_agent_gateway_client_records_and_checks_send_ledger():
    calls: list[tuple[str, str, dict[str, Any], dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        params = dict(request.url.params)
        calls.append((request.method, request.url.path, body, params))
        if request.url.path == "/v1/send-ledger/records":
            assert body == {
                "from_bot_id": "1049511700",
                "conversation_id": "2365524513",
                "content": "hello",
            }
            return httpx.Response(
                202,
                json={
                    "code": "OK",
                    "data": {
                        "from_bot_id": "1049511700",
                        "conversation_id": "2365524513",
                        "content_hash": "hash-1",
                    },
                },
            )
        if request.url.path == "/v1/send-ledger/recent":
            assert params == {
                "from_bot_id": "1049511700",
                "conversation_id": "2365524513",
                "window_seconds": "60",
                "content": "hello",
            }
            return _ok({"recent": True, "content_hash": "hash-1"})
        return httpx.Response(404, text="not found")

    client = _client(handler)

    recorded = await client.record_send(
        from_bot_id="1049511700",
        conversation_id="2365524513",
        content="hello",
    )
    recent = await client.recently_sent(
        from_bot_id="1049511700",
        conversation_id="2365524513",
        content="hello",
        window_seconds=60,
    )

    assert recorded["content_hash"] == "hash-1"
    assert recent is True
    assert [call[1] for call in calls] == [
        "/v1/send-ledger/records",
        "/v1/send-ledger/recent",
    ]


@pytest.mark.asyncio
async def test_agent_gateway_client_checks_private_echo():
    calls: list[tuple[str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        params = dict(request.url.params)
        calls.append((request.url.path, params))
        assert request.url.path == "/v1/send-ledger/private-echo"
        assert params == {
            "from_user_id": "1049511700",
            "to_bot_id": "2365524513",
            "window_seconds": "180",
            "has_image": "true",
        }
        return _ok({"echo": True, "reason": "recent_image_echo"})

    client = _client(handler)

    echo = await client.private_echo(
        from_user_id="1049511700",
        to_bot_id="2365524513",
        has_image=True,
    )

    assert echo["echo"] is True
    assert echo["reason"] == "recent_image_echo"
    assert calls[0][0] == "/v1/send-ledger/private-echo"


@pytest.mark.asyncio
async def test_agent_gateway_client_lists_delivery_adapters():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        assert request.url.path == "/v1/delivery-adapters"
        return _ok(
            [
                {
                    "provider": "onebot",
                    "channel": "qq_2365524513",
                    "transport": "websocket",
                    "enabled": True,
                    "endpoint_configured": True,
                    "access_token_configured": True,
                }
            ]
        )

    adapters = await _client(handler).list_delivery_adapters()

    assert adapters[0]["channel"] == "qq_2365524513"
    assert adapters[0]["transport"] == "websocket"


@pytest.mark.asyncio
async def test_agent_gateway_client_checks_delivery_adapter_health():
    calls: list[tuple[str, str, dict[str, str]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        calls.append((request.method, request.url.path, dict(request.url.params)))
        assert request.method == "GET"
        assert request.url.path == "/v1/delivery-adapters/health"
        return _ok(
            {
                "items": [
                    {
                        "provider": "onebot",
                        "channel": "qq_2365524513",
                        "healthy": True,
                        "side_effect": "none",
                    }
                ],
                "totals": {"adapters": 1, "healthy": 1},
            }
        )

    health = await _client(handler).check_delivery_adapter_health(timeout_seconds=2)

    assert health["items"][0]["channel"] == "qq_2365524513"
    assert health["items"][0]["side_effect"] == "none"
    assert calls[0][2]["timeout_seconds"] == "2"


@pytest.mark.asyncio
async def test_agent_gateway_client_gets_queue_backend():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        assert request.url.path == "/v1/queue-backend"
        return _ok(
            {
                "provider": "nats_jetstream",
                "mode": "dual_read_compare",
                "consumer_concurrency": 8,
                "max_in_flight": 64,
            }
        )

    backend = await _client(handler).get_queue_backend()

    assert backend["provider"] == "nats_jetstream"
    assert backend["consumer_concurrency"] == 8


@pytest.mark.asyncio
async def test_agent_gateway_client_sends_outbound_to_runtime():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        assert request.method == "POST"
        assert request.url.path == "/v1/outbound"
        assert body == {
            "event_id": "pyout:telegram:100:1",
            "channel": {
                "kind": "telegram",
                "account_id": "telegram",
                "conversation_id": "100",
                "conversation_type": "private",
            },
            "content": "hello",
            "attachments": [{"kind": "image", "url": "file:///tmp/a.png"}],
            "metadata": {"source": "message_push"},
        }
        return _ok({"event_id": body["event_id"], "status": "pending"})

    result = await _client(handler).send_outbound(
        event_id="pyout:telegram:100:1",
        channel={
            "kind": "telegram",
            "account_id": "telegram",
            "conversation_id": "100",
            "conversation_type": "private",
        },
        content="hello",
        attachments=[{"kind": "image", "url": "file:///tmp/a.png"}],
        metadata={"source": "message_push"},
    )

    assert result == {"event_id": "pyout:telegram:100:1", "status": "pending"}
    assert len(calls) == 1


@pytest.mark.asyncio
async def test_agent_gateway_client_leases_and_updates_outbox_delivery():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        if request.url.path == "/v1/outbox/lease-next":
            assert body == {"worker_id": "worker-a", "ttl_seconds": 120}
            return _ok({"event_id": "qq:private:1", "status": "dispatching"})
        if request.url.path == "/v1/outbox/qq:private:1/succeeded":
            assert body == {}
            return _ok({"event_id": "qq:private:1", "status": "succeeded"})
        if request.url.path == "/v1/outbox/qq:private:2/failed":
            assert body == {
                "error_kind": "platform_timeout",
                "error_message": "platform timeout",
            }
            return _ok({"event_id": "qq:private:2", "status": "failed"})
        return httpx.Response(404, text="not found")

    client = _client(handler)

    leased = await client.lease_next_outbox()
    succeeded = await client.mark_outbox_succeeded("qq:private:1")
    failed = await client.mark_outbox_failed(
        "qq:private:2",
        error_kind="platform_timeout",
        error_message="platform timeout",
    )

    assert leased["status"] == "dispatching"
    assert succeeded["status"] == "succeeded"
    assert failed["status"] == "failed"
    assert [path for _, path, _ in calls] == [
        "/v1/outbox/lease-next",
        "/v1/outbox/qq:private:1/succeeded",
        "/v1/outbox/qq:private:2/failed",
    ]


@pytest.mark.asyncio
async def test_agent_gateway_client_requests_outbox_dispatch_plan():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        assert request.url.path == "/v1/delivery-dispatch/plan"
        assert body == {
            "event_id": "qq:private:1",
            "channel_by_account": {"2365524513": "qq_2365524513"},
        }
        return _ok(
            {
                "event_id": "qq:private:1",
                "steps": [
                    {
                        "step_index": 1,
                        "kind": "text",
                        "channel": "qq_2365524513",
                        "chat_id": "1049511700",
                        "message": "hello",
                    }
                ],
            }
        )

    plan = await _client(handler).plan_outbox_dispatch(
        "qq:private:1",
        channel_by_account={"2365524513": "qq_2365524513"},
    )

    assert plan["steps"][0]["channel"] == "qq_2365524513"
    assert calls[0][1] == "/v1/delivery-dispatch/plan"


@pytest.mark.asyncio
async def test_agent_gateway_client_checks_outbox_dispatch_readiness():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        assert request.url.path == "/v1/delivery-dispatch/readiness"
        assert body == {
            "event_id": "qq:private:1",
            "channel_by_account": {"2365524513": "qq_2365524513"},
        }
        return _ok(
            {
                "event_id": "qq:private:1",
                "channel": "qq_2365524513",
                "ready": False,
                "reason": "delivery_adapter_unavailable",
                "missing_channels": ["qq_2365524513"],
                "plan": {
                    "event_id": "qq:private:1",
                    "channel": "qq_2365524513",
                    "chat_id": "1049511700",
                    "step_count": 1,
                    "steps": [
                        {
                            "step_index": 1,
                            "kind": "text",
                            "channel": "qq_2365524513",
                            "chat_id": "1049511700",
                            "message": "hello",
                        }
                    ],
                },
            }
        )

    readiness = await _client(handler).check_outbox_dispatch_readiness(
        "qq:private:1",
        channel_by_account={"2365524513": "qq_2365524513"},
    )

    assert readiness["ready"] is False
    assert readiness["missing_channels"] == ["qq_2365524513"]
    assert calls[0][1] == "/v1/delivery-dispatch/readiness"


@pytest.mark.asyncio
async def test_agent_gateway_client_maps_dispatch_plan_error_kind():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/delivery-dispatch/plan"
        return httpx.Response(
            400,
            json={
                "code": "INVALID_ARGUMENT",
                "message": "outbox delivery missing channel kind",
                "data": {"error_kind": "route_error"},
            },
        )

    with pytest.raises(AgentGatewayDeliveryPlanError) as raised:
        await _client(handler).plan_outbox_dispatch("qq:private:bad")

    assert raised.value.kind == "route_error"


@pytest.mark.asyncio
async def test_agent_gateway_client_dispatches_outbox_delivery():
    calls: list[tuple[str, str, dict[str, Any]]] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content.decode() or "{}")
        calls.append((request.method, request.url.path, body))
        assert request.url.path == "/v1/delivery-dispatch/send"
        return _ok(
            {
                "event_id": "telegram:private:1",
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
        )

    result = await _client(handler).dispatch_outbox_delivery(
        "telegram:private:1",
        channel_by_account={"bot": "telegram"},
    )

    assert result["results"][0]["provider_message_id"] == "123"
    assert calls[0] == (
        "POST",
        "/v1/delivery-dispatch/send",
        {
            "event_id": "telegram:private:1",
            "channel_by_account": {"bot": "telegram"},
        },
    )


@pytest.mark.asyncio
async def test_agent_gateway_client_maps_dispatch_unavailable_to_fallback_signal():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/delivery-dispatch/send"
        return httpx.Response(
            501,
            json={
                "code": "INVALID_ARGUMENT",
                "message": "delivery adapter unavailable for channel \"telegram\"",
                "data": {"error_kind": "sender_unavailable"},
            },
        )

    with pytest.raises(AgentGatewayDeliveryDispatchUnavailable):
        await _client(handler).dispatch_outbox_delivery("telegram:private:1")


@pytest.mark.asyncio
async def test_agent_gateway_client_maps_dispatch_error_kind():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/delivery-dispatch/send"
        return httpx.Response(
            502,
            json={
                "code": "INVALID_ARGUMENT",
                "message": "telegram platform error",
                "data": {"error_kind": "platform_error"},
            },
        )

    with pytest.raises(AgentGatewayDeliveryDispatchError) as raised:
        await _client(handler).dispatch_outbox_delivery("telegram:private:1")

    assert raised.value.kind == "platform_error"


@pytest.mark.asyncio
async def test_agent_gateway_client_maps_empty_lease_to_no_job():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/jobs/lease-next"
        return httpx.Response(404, text="no leaseable agent job found")

    with pytest.raises(AgentGatewayNoJob):
        await _client(handler).lease_next(job_type="rag_eval")


@pytest.mark.asyncio
async def test_agent_gateway_client_maps_empty_outbox_lease_to_no_job():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/outbox/lease-next"
        return httpx.Response(404, text="no leaseable outbox delivery found")

    with pytest.raises(AgentGatewayNoJob):
        await _client(handler).lease_next_outbox()


@pytest.mark.asyncio
async def test_agent_gateway_client_requires_enabled_config():
    client = AgentGatewayClient(
        AgentGatewayIntegrationConfig(
            enabled=False, base_url="http://agent-gateway.local"
        )
    )

    with pytest.raises(AgentGatewayError, match="未启用"):
        await client.health()
