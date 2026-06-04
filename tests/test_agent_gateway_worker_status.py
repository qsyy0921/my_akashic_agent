from __future__ import annotations

import logging
from typing import Any

import pytest

from integrations.agent_gateway import AgentGatewayHTTPError
from integrations.agent_gateway_worker_status import (
    AgentWorkerStatusReporter,
    _extract_existing_instance_id_from_conflict,
    _extract_pid_from_instance_id,
)


class _FakeClient:
    def __init__(self) -> None:
        self.calls: list[dict[str, Any]] = []

    async def report_agent_worker_status(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(kwargs)
        return kwargs


class _ConflictClient:
    def __init__(self, *, existing_instance_id: str) -> None:
        self.existing_instance_id = existing_instance_id
        self.calls: list[dict[str, Any]] = []

    async def report_agent_worker_status(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(kwargs)
        if len(self.calls) == 1:
            raise AgentGatewayHTTPError(
                409,
                (
                    "agent worker status lease conflict: "
                    f"worker_id=worker-a existing_instance_id={self.existing_instance_id} "
                    "lease_until=2026-06-03T00:00:00Z"
                ),
            )
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


@pytest.mark.asyncio
async def test_agent_worker_status_reporter_retries_once_for_stale_conflict(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    client = _ConflictClient(existing_instance_id="worker-a:34567:stale")
    reporter = AgentWorkerStatusReporter(
        client=client,
        worker_id="worker-a",
        worker_type="knowledge",
        logger=logging.getLogger("test"),
        label="test_worker",
        lease_ttl_seconds=60,
    )
    monkeypatch.setattr(
        "integrations.agent_gateway_worker_status._process_exists",
        lambda pid: False,
    )

    await reporter.running(current_job_id="job-1")

    assert len(client.calls) == 2
    assert client.calls[0]["replace_existing_instance_id"] == ""
    assert client.calls[1]["replace_existing_instance_id"] == "worker-a:34567:stale"
    assert client.calls[1]["current_job_id"] == "job-1"


@pytest.mark.asyncio
async def test_agent_worker_status_reporter_does_not_takeover_live_conflict(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    client = _ConflictClient(existing_instance_id="worker-a:34567:live")
    reporter = AgentWorkerStatusReporter(
        client=client,
        worker_id="worker-a",
        worker_type="knowledge",
        logger=logging.getLogger("test"),
        label="test_worker",
        lease_ttl_seconds=60,
    )
    monkeypatch.setattr(
        "integrations.agent_gateway_worker_status._process_exists",
        lambda pid: True,
    )

    await reporter.running(current_job_id="job-1")

    assert len(client.calls) == 1


def test_extract_existing_instance_id_from_conflict_text() -> None:
    value = _extract_existing_instance_id_from_conflict(
        "agent worker status lease conflict: worker_id=worker-a "
        "existing_instance_id=akashic-python-worker:outbox:12345:abc lease_until=2026-06-03T00:00:00Z"
    )
    assert value == "akashic-python-worker:outbox:12345:abc"


def test_extract_pid_from_instance_id_uses_rightmost_segments() -> None:
    assert _extract_pid_from_instance_id("akashic-python-worker:outbox:12345:abc") == 12345
    assert _extract_pid_from_instance_id("missing-pid") is None
