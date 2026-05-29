from __future__ import annotations

import copy
import json
from typing import Any

import pytest

from agent.config_models import AgentGatewayIntegrationConfig
from group_memory import GroupMemoryService
from integrations.agent_gateway import (
    AgentGatewayError,
    AgentGatewayNoJob,
)
from integrations.agent_gateway_knowledge_worker import AgentGatewayKnowledgeWorker
from integrations.agent_gateway import AgentGatewayClient
from session.manager import SessionManager


class _FakeGatewayService:
    def __init__(self) -> None:
        self.jobs: dict[str, dict[str, Any]] = {}
        self._ordered: list[str] = []

    async def create_job(
        self,
        *,
        job_id: str,
        job_type: str,
        agent_id: str,
        route: dict[str, str],
        payload: dict[str, Any],
        source_event_ids: list[str] | None = None,
        source_asset_ids: list[str] | None = None,
        max_attempts: int = 3,
        metadata: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        if job_id not in self.jobs:
            now = _now_iso()
            self.jobs[job_id] = {
                "job_id": job_id,
                "job_type": job_type,
                "agent_id": agent_id,
                "route": route,
                "payload": payload,
                "source_event_ids": list(source_event_ids or []),
                "source_asset_ids": list(source_asset_ids or []),
                "attempts": 0,
                "max_attempts": int(max_attempts),
                "status": "pending",
                "result": {},
                "error_message": "",
                "metadata": metadata or {},
                "lease_owner": "",
                "lease_expires_at": "",
                "created_at": now,
                "updated_at": now,
            }
            self._ordered.append(job_id)
        return self.jobs[job_id]

    async def lease_next(
        self,
        *,
        worker_id: str,
        job_type: str,
        ttl_seconds: int = 300,
    ) -> dict[str, Any]:
        for job_id in self._ordered:
            job = self.jobs[job_id]
            if job["job_type"] == job_type and job["status"] == "pending":
                return self._set_status(
                    job_id,
                    "leased",
                    lease_owner=worker_id,
                    lease_expires_at=_now_iso(ttl_seconds),
                )
        raise AgentGatewayNoJob("no leaseable agent job")

    async def mark_running(self, job_id: str) -> dict[str, Any]:
        if job_id not in self.jobs:
            raise AgentGatewayError("missing job")
        return self._set_status(job_id, "running")

    async def complete_job(
        self,
        job_id: str,
        *,
        result: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        if job_id not in self.jobs:
            raise AgentGatewayError("missing job")
        updates = {"result": dict(result or {}), "lease_owner": "", "lease_expires_at": ""}
        return self._set_status(job_id, "succeeded", **updates)

    async def fail_job(self, job_id: str, *, error_message: str) -> dict[str, Any]:
        if job_id not in self.jobs:
            raise AgentGatewayError("missing job")
        job = self.jobs[job_id]
        attempts = int(job["attempts"]) + 1
        status = "failed" if attempts >= int(job["max_attempts"]) else "pending"
        return self._set_status(job_id, status, attempts=attempts, error_message=error_message)

    async def list_jobs(self, **_query: Any) -> list[dict[str, Any]]:
        return [copy.deepcopy(job) for job in self.jobs.values()]

    def snapshot(self, job_id: str) -> dict[str, Any]:
        return copy.deepcopy(self.jobs[job_id])

    def _set_status(self, job_id: str, status: str, **updates: Any) -> dict[str, Any]:
        job = self.jobs[job_id]
        job["status"] = status
        job["updated_at"] = _now_iso()
        job.update(updates)
        return copy.deepcopy(job)


class _FakeRagflowIndexer:
    def __init__(self) -> None:
        self.calls: list[dict[str, Any]] = []

    async def execute(self, **kwargs: Any) -> str:
        self.calls.append(kwargs)
        return json.dumps(
            {
                "ok": True,
                "message_count": 2,
                "display_name": "qq_group_284331268_seq1_3.txt",
                "data": {
                    "dataset_id": kwargs.get("dataset_id"),
                    "document_ids": ["mock-doc-1"],
                },
            }
        )


class _AdapterClient(AgentGatewayClient):
    def __init__(self, fake: _FakeGatewayService) -> None:
        # Keep compatibility with AgentGatewayClient internals while routing all calls
        # through the in-process fake job service.
        super().__init__(AgentGatewayIntegrationConfig(enabled=True, base_url="http://localhost"))
        self._fake = fake

    async def create_job(self, **kwargs: Any) -> dict[str, Any]:
        return await self._fake.create_job(**kwargs)

    async def lease_next(self, **kwargs: Any) -> dict[str, Any]:
        return await self._fake.lease_next(**kwargs)

    async def mark_running(self, job_id: str) -> dict[str, Any]:
        return await self._fake.mark_running(job_id)

    async def complete_job(self, job_id: str, *, result: dict[str, str] | None = None) -> dict[str, Any]:
        return await self._fake.complete_job(job_id, result=result)

    async def fail_job(self, job_id: str, *, error_message: str) -> dict[str, Any]:
        return await self._fake.fail_job(job_id, error_message=error_message)


@pytest.mark.asyncio
async def test_smoke_knowledge_worker_end_to_end_with_real_group_memory_and_fake_ragflow(
    tmp_path,
) -> None:
    workdir = tmp_path / "smoke"
    session_manager = SessionManager(workdir)
    session = session_manager.get_or_create("qq:gqq:284331268")
    session.add_message(
        "user",
        "我想在主机群里看一下游戏主机 5600X+3060 的功耗和兼容性。",
        sender_id="1049511700",
        chat_type="group",
        group_id="284331268",
        platform_message_id="msg-1001",
    )
    session.add_message(
        "user",
        "另外，散热要考虑静音方案，有没有推荐风冷的？",
        sender_id="2948770636",
        chat_type="group",
        group_id="284331268",
        platform_message_id="msg-1002",
    )
    await session_manager.save_async(session)

    fake_gateway = _FakeGatewayService()
    ragflow_indexer = _FakeRagflowIndexer()
    worker = AgentGatewayKnowledgeWorker(
        client=_AdapterClient(fake_gateway),
        group_memory=GroupMemoryService.from_workspace(
            workdir,
            session_store=session_manager._store,
            batch_max_messages=100,
        ),
        worker_id="smoke-worker",
        group_accounts={"284331268": "2365524513"},
        ragflow_indexer=ragflow_indexer,
        ragflow_dataset_ids=["ds-smoke-1", "ds-smoke-2"],
        lease_ttl_seconds=60,
        poll_interval_seconds=0.2,
        enqueue_interval_seconds=5,
        now_fn=lambda: 120.0,
    )

    enqueue_summary = await worker.enqueue_once()
    assert enqueue_summary["created_or_existing"] == 3  # 1 group + 2 rag datasets
    expected_bucket = enqueue_summary["bucket"]
    expected_group_job_id = f"group_memory_extract:qq:284331268:{expected_bucket}"
    assert {item["job_type"] for item in fake_gateway.jobs.values()} == {
        "group_memory_extract",
        "rag_ingest",
    }
    assert fake_gateway.jobs[expected_group_job_id]["payload"]["group_id"] == "284331268"

    first = await worker.process_once()
    assert first["processed"] is True
    assert first["job_type"] == "group_memory_extract"
    assert first["job_id"] == expected_group_job_id

    group_result = fake_gateway.snapshot(expected_group_job_id)
    assert group_result["status"] == "succeeded"
    assert group_result["result"]["group_id"] == "284331268"
    assert group_result["result"]["session_key"] == "qq:gqq:284331268"

    second = await worker.process_once()
    assert second["processed"] is True
    assert second["job_type"] == "rag_ingest"
    assert ragflow_indexer.calls, "ragflow fake indexer should be invoked"
    assert second["result"]["dataset_id"] in {"ds-smoke-1", "ds-smoke-2"}

    third = await worker.process_once()
    assert third["processed"] is True
    assert third["job_type"] == "rag_ingest"

    assert len(ragflow_indexer.calls) == 2
    assert {entry["result"]["dataset_id"] for entry in [second, third]} <= {"ds-smoke-1", "ds-smoke-2"}

    failed = await worker.process_once()
    assert failed == {"processed": False, "reason": "no_job"}

    rag_jobs = [
        job
        for job in fake_gateway.jobs.values()
        if job["job_type"] == "rag_ingest"
    ]
    assert len(rag_jobs) == 2
    assert all(job["status"] == "succeeded" for job in rag_jobs)


def _now_iso(plus_seconds: int = 0) -> str:
    from datetime import datetime, timezone, timedelta

    now = datetime.now(timezone.utc) + timedelta(seconds=plus_seconds)
    return now.isoformat().replace("+00:00", "Z")
