from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

from integrations.agent_gateway import AgentGatewayNoJob
from integrations.agent_gateway_rag_eval_worker import (
    AgentGatewayRagEvalWorker,
    GroupMemoryFixtureEvaluator,
)


class _FakeEvaluator:
    def __init__(self, *, error: str = "") -> None:
        self.calls: list[dict[str, Any]] = []
        self._error = error

    async def execute(self, job: dict[str, Any]) -> dict[str, Any]:
        self.calls.append(job)
        if self._error:
            raise RuntimeError(self._error)
        return {
            "suite": "group_memory_open_fixture",
            "questions": 2,
            "top1_accuracy": 1.0,
            "evidence_coverage": 1.0,
            "passed": True,
        }


class _FakeGatewayClient:
    def __init__(self, *, jobs: list[dict[str, Any]] | None = None) -> None:
        self.jobs = jobs or []
        self.calls: list[tuple[str, Any]] = []

    async def lease_next(self, **kwargs: Any) -> dict[str, Any]:
        self.calls.append(("lease_next", kwargs))
        if not self.jobs:
            raise AgentGatewayNoJob("none")
        job = self.jobs.pop(0)
        assert kwargs["job_type"] == "rag_eval"
        return job

    async def mark_running(
        self,
        job_id: str,
        *,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("mark_running", job_id, lease_token))
        return {"job_id": job_id, "status": "running"}

    async def complete_job(
        self,
        job_id: str,
        *,
        result: dict[str, str] | None = None,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("complete_job", job_id, result, lease_token))
        return {"job_id": job_id, "status": "succeeded"}

    async def fail_job(
        self,
        job_id: str,
        *,
        error_message: str,
        lease_token: str | None = None,
    ) -> dict[str, Any]:
        self.calls.append(("fail_job", job_id, error_message, lease_token))
        return {"job_id": job_id, "status": "failed"}


def _worker(
    client: _FakeGatewayClient,
    evaluator: _FakeEvaluator,
) -> AgentGatewayRagEvalWorker:
    return AgentGatewayRagEvalWorker(
        client=client,  # type: ignore[arg-type]
        evaluator=evaluator,
        worker_id="eval-worker-a",
        lease_ttl_seconds=60,
        poll_interval_seconds=0.5,
    )


@pytest.mark.asyncio
async def test_rag_eval_worker_processes_generic_job():
    evaluator = _FakeEvaluator()
    client = _FakeGatewayClient(
        jobs=[
            {
                "job_id": "rag_eval:qq:284331268:fixture:1",
                "job_type": "rag_eval",
                "payload": {
                    "fixture": "tests/fixtures/group_memory_open_strategy_dataset.json"
                },
            }
        ]
    )
    worker = _worker(client, evaluator)

    result = await worker.process_once()

    assert result["processed"] is True
    assert result["job_type"] == "rag_eval"
    assert evaluator.calls[0]["job_id"] == "rag_eval:qq:284331268:fixture:1"
    complete = next(call for call in client.calls if call[0] == "complete_job")
    assert complete[2]["passed"] == "true"
    assert complete[2]["top1_accuracy"] == "1.0"


@pytest.mark.asyncio
async def test_rag_eval_worker_fails_job_when_executor_errors():
    client = _FakeGatewayClient(
        jobs=[{"job_id": "rag_eval:bad", "job_type": "rag_eval", "payload": {}}]
    )
    worker = _worker(client, _FakeEvaluator(error="eval exploded"))

    result = await worker.process_once()

    assert result["failed"] is True
    assert any(call[0] == "fail_job" for call in client.calls)


@pytest.mark.asyncio
async def test_rag_eval_worker_returns_idle_when_no_job():
    result = await _worker(_FakeGatewayClient(), _FakeEvaluator()).process_once()

    assert result == {"processed": False, "reason": "no_job"}


@pytest.mark.asyncio
async def test_group_memory_fixture_evaluator_runs_checked_in_fixture(tmp_path: Path):
    evaluator = GroupMemoryFixtureEvaluator(workspace=tmp_path)

    result = await evaluator.execute(
        {
            "job_id": "rag_eval:fixture-smoke",
            "payload": {
                "fixture": "tests/fixtures/group_memory_open_strategy_dataset.json",
                "min_top1_accuracy": "1.0",
                "min_evidence_coverage": "1.0",
            },
        }
    )

    assert result["questions"] == 2
    assert result["top1_accuracy"] == 1.0
    assert result["evidence_coverage"] == 1.0
    assert result["passed"] is True
    json.dumps(result, ensure_ascii=False)
