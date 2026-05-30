from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from agent.config_models import AgentGatewayIntegrationConfig
from integrations.agent_gateway import (
    AgentGatewayClient,
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
            return _ok({"job_id": "job-1", "status": "leased"})
        if request.url.path == "/v1/jobs/job-1/running":
            return _ok({"job_id": "job-1", "status": "running"})
        if request.url.path == "/v1/jobs/job-1/succeeded":
            assert body == {"result": {"asset_id": "asset-1"}}
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
    running = await client.mark_running("job-1")
    succeeded = await client.complete_job("job-1", result={"asset_id": "asset-1"})

    assert created["status"] == "pending"
    assert leased["status"] == "leased"
    assert running["status"] == "running"
    assert succeeded["status"] == "succeeded"
    assert [path for _, path, _ in calls] == [
        "/v1/jobs",
        "/v1/jobs/lease-next",
        "/v1/jobs/job-1/running",
        "/v1/jobs/job-1/succeeded",
    ]


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
