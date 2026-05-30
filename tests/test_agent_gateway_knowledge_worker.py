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

    async def mark_running(self, job_id: str) -> dict[str, Any]:
        self.calls.append(("mark_running", job_id))
        return {"job_id": job_id}

    async def complete_job(
        self,
        job_id: str,
        *,
        result: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("complete_job", job_id, result))
        return {"job_id": job_id}

    async def fail_job(self, job_id: str, *, error_message: str) -> dict[str, Any]:
        self.calls.append(("fail_job", job_id, error_message))
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
    assert client.created[1]["payload"]["dataset_id"] == "ds-1"


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
