from __future__ import annotations

import asyncio
import logging
from typing import Any

import pytest

from integrations.agent_gateway_heartbeat import AgentJobLeaseHeartbeat


class _FakeRenewClient:
    def __init__(self) -> None:
        self.calls: list[tuple[str, dict[str, Any]]] = []
        self.renewed = asyncio.Event()

    async def renew_job(
        self,
        job_id: str,
        *,
        lease_token: str,
        ttl_seconds: int,
    ) -> dict[str, Any]:
        self.calls.append(
            (
                job_id,
                {
                    "lease_token": lease_token,
                    "ttl_seconds": ttl_seconds,
                },
            )
        )
        self.renewed.set()
        return {"job_id": job_id, "status": "running"}


@pytest.mark.asyncio
async def test_agent_job_lease_heartbeat_renews_until_stopped():
    client = _FakeRenewClient()
    heartbeat = AgentJobLeaseHeartbeat(
        client=client,
        job_id="job-1",
        lease_token="tok-1",
        ttl_seconds=120,
        interval_seconds=0.01,
        logger=logging.getLogger(__name__),
        label="test",
    )

    heartbeat.start()
    await asyncio.wait_for(client.renewed.wait(), timeout=2)
    await heartbeat.stop()

    assert client.calls[0] == (
        "job-1",
        {
            "lease_token": "tok-1",
            "ttl_seconds": 120,
        },
    )


@pytest.mark.asyncio
async def test_agent_job_lease_heartbeat_noops_without_token():
    client = _FakeRenewClient()
    heartbeat = AgentJobLeaseHeartbeat(
        client=client,
        job_id="job-1",
        lease_token="",
        ttl_seconds=120,
        interval_seconds=0.01,
        logger=logging.getLogger(__name__),
        label="test",
    )

    heartbeat.start()
    await asyncio.sleep(0.03)
    await heartbeat.stop()

    assert client.calls == []
