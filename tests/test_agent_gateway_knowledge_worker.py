from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

from integrations.agent_gateway import AgentGatewayNoJob
from integrations.agent_gateway_knowledge_worker import AgentGatewayKnowledgeWorker

CONTRACT_DIR = Path(__file__).resolve().parents[1] / "tests" / "fixtures" / "contracts"


class _Stats:
    def __init__(self, **values: Any) -> None:
        self.__dict__.update(values)


class _FakeGroupMemory:
    def __init__(self) -> None:
        self.groups: list[str] = []

    def ingest_group(self, group_id: str) -> _Stats:
        self.groups.append(group_id)
        return _Stats(
            session_key=f"qq:gqq:{group_id}",
            group_id=group_id,
            scanned=3,
            candidates=2,
            added=1,
            reinforced=1,
            cursor=42,
        )


class _FakeRagflowIndexer:
    def __init__(self, *, error: str = "") -> None:
        self.calls: list[dict[str, Any]] = []
        self._error = error

    async def execute(self, **kwargs: Any) -> str:
        self.calls.append(kwargs)
        if self._error:
            return json.dumps({"ok": False, "error": self._error})
        since_seq = int(kwargs.get("since_seq", 1))
        end_seq = since_seq + 2
        return json.dumps(
            {
                "ok": True,
                "message_count": 3,
                "display_name": f"qq_group_284331268_seq{since_seq}_{end_seq}.txt",
                "start_seq": since_seq,
                "end_seq": end_seq,
                "data": {"document_ids": ["doc-1"]},
            }
        )


class _FakeGatewayClient:
    def __init__(self, *, jobs: list[dict[str, Any]] | None = None) -> None:
        self.jobs = jobs or []
        self.created: list[dict[str, Any]] = []
        self.checkpoints: dict[str, dict[str, Any]] = {}
        self.calls: list[tuple[str, Any]] = []

    async def create_job(self, **kwargs: Any) -> dict[str, Any]:
        self.created.append(kwargs)
        return {"job_id": kwargs["job_id"], "status": "pending"}

    async def lease_next(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(("lease_next", kwargs))
        job_type = kwargs.get("job_type")
        for index, job in enumerate(self.jobs):
            if job.get("job_type") == job_type:
                return self.jobs.pop(index)
        raise AgentGatewayNoJob("none")

    async def mark_running(
        self,
        job_id: str,
        *,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("mark_running", job_id, lease_token))
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

    async def fail_job(
        self,
        job_id: str,
        *,
        error_message: str,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("fail_job", job_id, error_message, lease_token))
        return {"job_id": job_id}

    async def get_knowledge_checkpoint(self, checkpoint_id: str) -> dict[str, Any] | None:
        self.calls.append(("get_knowledge_checkpoint", checkpoint_id))
        return self.checkpoints.get(checkpoint_id)

    async def update_knowledge_checkpoint(
        self,
        checkpoint_id: str,
        *,
        cursor: int,
        metadata: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("update_knowledge_checkpoint", checkpoint_id, cursor, metadata))
        value = {
            "checkpoint_id": checkpoint_id,
            "cursor": cursor,
            "metadata": metadata or {},
        }
        self.checkpoints[checkpoint_id] = value
        return value


def _worker(
    client: _FakeGatewayClient,
    *,
    ragflow_indexer: _FakeRagflowIndexer | None = None,
) -> AgentGatewayKnowledgeWorker:
    return AgentGatewayKnowledgeWorker(
        client=client,  # type: ignore[arg-type]
        group_memory=_FakeGroupMemory(),
        worker_id="worker-a",
        group_accounts={"284331268": "2365524513"},
        ragflow_indexer=ragflow_indexer,
        ragflow_dataset_ids=["ds-1"] if ragflow_indexer is not None else [],
        lease_ttl_seconds=60,
        poll_interval_seconds=0.5,
        enqueue_interval_seconds=60,
        now_fn=lambda: 120.0,
    )


@pytest.mark.asyncio
async def test_knowledge_worker_enqueues_group_memory_and_rag_jobs():
    client = _FakeGatewayClient()
    worker = _worker(client, ragflow_indexer=_FakeRagflowIndexer())

    summary = await worker.enqueue_once()

    assert summary["created_or_existing"] == 2
    assert [item["job_type"] for item in client.created] == [
        "group_memory_extract",
        "rag_ingest",
    ]
    assert client.created[0]["route"]["conversation_id"] == "284331268"
    assert client.created[0]["payload"]["observe_only"] == "true"
    assert client.created[0]["dedupe_key"] == (
        "knowledge:group_memory_extract:qq:2365524513:284331268"
    )
    assert client.created[1]["payload"]["dataset_id"] == "ds-1"
    assert client.created[1]["dedupe_key"] == (
        "knowledge:rag_ingest:qq:2365524513:284331268:ds-1"
    )


@pytest.mark.asyncio
async def test_knowledge_worker_uses_group_specific_dataset_bindings():
    client = _FakeGatewayClient()
    worker = AgentGatewayKnowledgeWorker(
        client=client,  # type: ignore[arg-type]
        group_memory=_FakeGroupMemory(),
        worker_id="worker-a",
        group_accounts={
            "284331268": "2365524513",
            "3219982": "1049511700",
        },
        ragflow_indexer=_FakeRagflowIndexer(),
        ragflow_dataset_ids=["ds-default"],
        ragflow_dataset_ids_by_group={
            "284331268": ["ds-hardware", "ds-guides"],
            "3219982": [],
        },
        lease_ttl_seconds=60,
        poll_interval_seconds=0.5,
        enqueue_interval_seconds=60,
        now_fn=lambda: 120.0,
    )

    summary = await worker.enqueue_once()

    assert summary["created_or_existing"] == 4
    rag_jobs = [
        job for job in client.created if job["job_type"] == "rag_ingest"
    ]
    assert [job["payload"]["dataset_id"] for job in rag_jobs] == [
        "ds-hardware",
        "ds-guides",
    ]
    assert all(job["route"]["conversation_id"] == "284331268" for job in rag_jobs)


@pytest.mark.asyncio
async def test_knowledge_worker_reports_runtime_dedupe_suppression():
    class _DedupeGateway(_FakeGatewayClient):
        async def create_job(self, **kwargs: Any) -> dict[str, Any]:
            self.created.append(kwargs)
            return {"job_id": "group_memory_extract:qq:284331268:previous", "status": "pending"}

    client = _DedupeGateway()
    worker = _worker(client)

    summary = await worker.enqueue_once()

    assert summary["created_or_existing"] == 1
    assert summary["suppressed_by_runtime_dedupe"] == 1


@pytest.mark.asyncio
async def test_knowledge_worker_skips_legacy_enqueue_when_go_planner_enabled():
    class _RuntimePlannerGateway(_FakeGatewayClient):
        async def get_runtime_config(self) -> dict[str, Any]:
            return {"workers": {"knowledge_job_planner_enabled": True}}

    client = _RuntimePlannerGateway()
    worker = _worker(client, ragflow_indexer=_FakeRagflowIndexer())

    summary = await worker.enqueue_if_due_once()

    assert summary["enqueued"] is False
    assert summary["reason"] == "go_runtime_knowledge_job_planner_enabled"
    assert client.created == []


@pytest.mark.asyncio
async def test_knowledge_worker_keeps_enqueue_fallback_without_go_planner_flag():
    class _RuntimePlannerGateway(_FakeGatewayClient):
        async def get_runtime_config(self) -> dict[str, Any]:
            return {"workers": {"knowledge_job_planner_enabled": False}}

    client = _RuntimePlannerGateway()
    worker = _worker(client, ragflow_indexer=_FakeRagflowIndexer())

    summary = await worker.enqueue_if_due_once()

    assert summary["enqueued"] is True
    assert summary["created_or_existing"] == 2
    assert [item["job_type"] for item in client.created] == [
        "group_memory_extract",
        "rag_ingest",
    ]


@pytest.mark.asyncio
async def test_knowledge_worker_processes_group_memory_job():
    client = _FakeGatewayClient(
        jobs=[
            {
                "job_id": "gm-1",
                "job_type": "group_memory_extract",
                "payload": {"group_id": "284331268"},
            }
        ]
    )
    worker = _worker(client)

    result = await worker.process_once()

    assert result["processed"] is True
    assert result["job_type"] == "group_memory_extract"
    complete = next(call for call in client.calls if call[0] == "complete_job")
    assert complete[2]["group_id"] == "284331268"
    assert complete[2]["scanned"] == "3"


@pytest.mark.asyncio
async def test_knowledge_worker_processes_rag_ingest_job():
    indexer = _FakeRagflowIndexer()
    client = _FakeGatewayClient(
        jobs=[
            {
                "job_id": "rag-1",
                "job_type": "rag_ingest",
                "payload": {"group_id": "284331268", "dataset_id": "ds-1"},
            }
        ]
    )
    worker = _worker(client, ragflow_indexer=indexer)

    result = await worker.process_once()

    assert result["processed"] is True
    assert indexer.calls[0]["group_id"] == "284331268"
    assert indexer.calls[0]["dataset_id"] == "ds-1"
    assert indexer.calls[0]["since_seq"] == 0
    complete = next(call for call in client.calls if call[0] == "complete_job")
    assert complete[2]["dataset_id"] == "ds-1"
    assert complete[2]["message_count"] == "3"
    assert client.checkpoints["ragflow:qq:284331268:ds-1"]["cursor"] == 2
    assert client.checkpoints["ragflow:qq:284331268:ds-1"]["metadata"] == {
        "job_type": "rag_ingest",
        "source": "qq",
        "group_id": "284331268",
        "dataset_id": "ds-1",
        "display_name": "qq_group_284331268_seq0_2.txt",
        "last_message_count": "3",
        "last_document_count": "1",
        "last_start_seq": "0",
        "last_end_seq": "2",
        "last_parse_requested": "true",
    }


@pytest.mark.asyncio
async def test_knowledge_worker_uses_rag_ingest_checkpoint_cursor():
    indexer = _FakeRagflowIndexer()
    client = _FakeGatewayClient(
        jobs=[
            {
                "job_id": "rag-1",
                "job_type": "rag_ingest",
                "payload": {"group_id": "284331268", "dataset_id": "ds-1"},
            }
        ]
    )
    client.checkpoints["ragflow:qq:284331268:ds-1"] = {
        "checkpoint_id": "ragflow:qq:284331268:ds-1",
        "cursor": 10,
    }
    worker = _worker(client, ragflow_indexer=indexer)

    result = await worker.process_once()

    assert result["processed"] is True
    assert indexer.calls[0]["since_seq"] == 11
    assert any(call[0] == "update_knowledge_checkpoint" for call in client.calls)


@pytest.mark.asyncio
async def test_knowledge_worker_uses_checkpoint_contract_fixture_cursor():
    fixture = _load_contract("knowledge_checkpoint.ragflow.qq.json")
    indexer = _FakeRagflowIndexer()
    client = _FakeGatewayClient(
        jobs=[
            {
                "job_id": "rag-contract-1",
                "job_type": "rag_ingest",
                "payload": {
                    "group_id": fixture["conversation_id"],
                    "dataset_id": fixture["metadata"]["dataset_id"],
                },
            }
        ]
    )
    client.checkpoints[fixture["checkpoint_id"]] = {
        "checkpoint_id": fixture["checkpoint_id"],
        "cursor": fixture["cursor"],
        "metadata": fixture["metadata"],
    }
    worker = _worker(client, ragflow_indexer=indexer)

    result = await worker.process_once()

    assert result["processed"] is True
    assert indexer.calls[0]["group_id"] == fixture["conversation_id"]
    assert indexer.calls[0]["dataset_id"] == fixture["metadata"]["dataset_id"]
    assert indexer.calls[0]["since_seq"] == fixture["cursor"] + 1
    update = next(call for call in client.calls if call[0] == "update_knowledge_checkpoint")
    assert update[1] == fixture["checkpoint_id"]


@pytest.mark.asyncio
async def test_knowledge_worker_fails_job_when_ragflow_errors():
    client = _FakeGatewayClient(
        jobs=[
            {
                "job_id": "rag-1",
                "job_type": "rag_ingest",
                "payload": {"group_id": "284331268", "dataset_id": "ds-1"},
            }
        ]
    )
    worker = _worker(client, ragflow_indexer=_FakeRagflowIndexer(error="bad dataset"))

    result = await worker.process_once()

    assert result["failed"] is True
    assert any(call[0] == "fail_job" for call in client.calls)


@pytest.mark.asyncio
async def test_knowledge_worker_returns_idle_when_no_jobs():
    client = _FakeGatewayClient()
    worker = _worker(client)

    result = await worker.process_once()

    assert result == {"processed": False, "reason": "no_job"}


def _load_contract(name: str) -> dict[str, Any]:
    return json.loads((CONTRACT_DIR / name).read_text(encoding="utf-8"))
