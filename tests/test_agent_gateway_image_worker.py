from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

from agent.tools.base import Tool
from integrations.agent_gateway import AgentGatewayNoJob
from integrations.agent_gateway_image_worker import AgentGatewayImageWorker


class _FakeImageTool(Tool):
    name = "fake_image"
    description = "fake image tool"
    parameters = {"type": "object", "properties": {}}

    def __init__(
        self,
        *,
        result: dict[str, Any] | None = None,
        error: str = "",
    ) -> None:
        self.calls: list[dict[str, Any]] = []
        self._result = result
        self._error = error

    async def execute(self, **kwargs: Any) -> str:
        self.calls.append(kwargs)
        if self._error:
            raise RuntimeError(self._error)
        return json.dumps(self._result or {"ok": False, "error": "missing"})


class _FakeGatewayClient:
    def __init__(self, *, job: dict[str, Any] | None) -> None:
        self.job = job
        self.calls: list[tuple[str, Any]] = []

    async def lease_next(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(("lease_next", kwargs))
        if self.job is None:
            raise AgentGatewayNoJob("none")
        return self.job

    async def mark_running(
        self,
        job_id: str,
        *,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("mark_running", job_id, lease_token))
        return {"job_id": job_id}

    async def mark_image_job_running(self, job_id: str) -> dict[str, Any]:
        self.calls.append(("mark_image_job_running", job_id))
        return {"job_id": job_id}

    async def complete_image_job(
        self,
        job_id: str,
        *,
        results: list[dict[str, Any]],
        metadata: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("complete_image_job", job_id, results, metadata))
        return {"job_id": job_id}

    async def complete_job(
        self,
        job_id: str,
        *,
        result: dict[str, str] | None = None,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("complete_job", job_id, result, lease_token))
        return {"job_id": job_id}

    async def fail_image_job(self, job_id: str, *, error_message: str) -> dict[str, Any]:
        self.calls.append(("fail_image_job", job_id, error_message))
        return {"job_id": job_id}

    async def fail_job(
        self,
        job_id: str,
        *,
        error_message: str,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("fail_job", job_id, error_message, lease_token))
        return {"job_id": job_id}


class _FakeStatusGatewayClient(_FakeGatewayClient):
    async def report_agent_worker_status(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(("report_agent_worker_status", kwargs))
        return kwargs


def _job() -> dict[str, Any]:
    return {
        "job_id": "img-1",
        "lease_token": "lease-img-1",
        "payload": {
            "legacy_image_job_id": "img-1",
            "prompt": "古装美女",
            "size": "1024x1024",
            "count": "1",
            "model": "gpt-image",
        },
    }


@pytest.mark.asyncio
async def test_image_worker_processes_generic_image_job(tmp_path: Path):
    image_path = tmp_path / "generated.png"
    image_path.write_bytes(b"png")
    client = _FakeGatewayClient(job=_job())
    tool = _FakeImageTool(result={"ok": True, "paths": [str(image_path)]})
    worker = AgentGatewayImageWorker(
        client=client,  # type: ignore[arg-type]
        image_tool=tool,
        worker_id="worker-a",
        lease_ttl_seconds=60,
        poll_interval_seconds=0.5,
    )

    result = await worker.process_once()

    assert result["processed"] is True
    assert result["count"] == 1
    assert tool.calls[0]["prompt"] == "古装美女"
    assert tool.calls[0]["n"] == 1
    complete_legacy = next(
        call for call in client.calls if call[0] == "complete_image_job"
    )
    attachment = complete_legacy[2][0]
    assert attachment["kind"] == "image"
    assert attachment["url"].startswith("file:///")
    assert attachment["size_bytes"] == 3
    assert client.calls[-1][0] == "complete_job"
    assert client.calls[-1][1] == "img-1"
    assert json.loads(client.calls[-1][2]["paths"])[0].startswith("file:///")
    assert client.calls[-1][3] == "lease-img-1"


@pytest.mark.asyncio
async def test_image_worker_fails_both_generic_and_legacy_jobs():
    client = _FakeGatewayClient(job=_job())
    tool = _FakeImageTool(error="boom")
    worker = AgentGatewayImageWorker(
        client=client,  # type: ignore[arg-type]
        image_tool=tool,
        worker_id="worker-a",
    )

    result = await worker.process_once()

    assert result["failed"] is True
    assert any(call[0] == "fail_image_job" for call in client.calls)
    assert any(call[0] == "fail_job" for call in client.calls)


@pytest.mark.asyncio
async def test_image_worker_returns_idle_when_no_job():
    client = _FakeGatewayClient(job=None)
    tool = _FakeImageTool(result={"ok": True, "paths": []})
    worker = AgentGatewayImageWorker(
        client=client,  # type: ignore[arg-type]
        image_tool=tool,
        worker_id="worker-a",
    )

    result = await worker.process_once()

    assert result == {"processed": False, "reason": "no_job"}


@pytest.mark.asyncio
async def test_image_worker_reports_worker_status_on_process():
    client = _FakeStatusGatewayClient(job=None)
    tool = _FakeImageTool(result={"ok": True, "paths": []})
    worker = AgentGatewayImageWorker(
        client=client,  # type: ignore[arg-type]
        image_tool=tool,
        worker_id="worker-a",
    )

    result = await worker.process_once()

    assert result == {"processed": False, "reason": "no_job"}
    status_call = next(call for call in client.calls if call[0] == "report_agent_worker_status")
    assert status_call[1]["worker_type"] == "image_generation"
    assert status_call[1]["status"] == "idle"
    assert status_call[1]["instance_id"].startswith("worker-a:")
    assert status_call[1]["lease_ttl_seconds"] == 300
    assert status_call[1]["metadata"] == {"reason": "no_job"}
