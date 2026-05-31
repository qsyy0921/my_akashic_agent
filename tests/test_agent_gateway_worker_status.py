from __future__ import annotations

import logging
from typing import Any

import pytest

from integrations.agent_gateway_worker_status import AgentWorkerStatusReporter


class _FakeClient:
    def __init__(self) -> None:
        self.calls: list[dict[str, Any]] = []

    async def report_agent_worker_status(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(kwargs)
        return kwargs


@pytest.mark.asyncio
async def test_agent_worker_status_heartbeat_reports_running_keepalive() -> None:
    client = _FakeClient()
    reporter = AgentWorkerStatusReporter(
        client=client,
        worker_id="worker-a",
        worker_type="knowledge",
        logger=logging.getLogger("test"),
        label="test_worker",
        lease_ttl_seconds=60,
    )

    await reporter.running(current_job_id="job-1")
    heartbeat = reporter.running_heartbeat(
        current_job_id="job-1",
        interval_seconds=1,
    )
    await heartbeat.beat_once()

    assert len(client.calls) == 2
    assert client.calls[0]["status"] == "running"
    assert client.calls[1]["status"] == "running"
    assert client.calls[1]["current_job_id"] == "job-1"
    assert client.calls[1]["worker_id"] == "worker-a"
    assert client.calls[1]["worker_type"] == "knowledge"
    assert client.calls[1]["instance_id"].startswith("worker-a:")
    assert client.calls[1]["lease_ttl_seconds"] == 60
