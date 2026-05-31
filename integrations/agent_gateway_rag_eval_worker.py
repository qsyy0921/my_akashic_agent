from __future__ import annotations

import asyncio
import json
import logging
import re
import time
from pathlib import Path
from typing import Any, Protocol

from integrations.agent_gateway import (
    AgentGatewayClient,
    AgentGatewayError,
    AgentGatewayNoJob,
)
from integrations.agent_gateway_heartbeat import AgentJobLeaseHeartbeat
from integrations.agent_gateway_worker_status import AgentWorkerStatusReporter

logger = logging.getLogger(__name__)


class RagEvalExecutor(Protocol):
    async def execute(self, job: dict[str, Any]) -> dict[str, Any]:
        ...


class GroupMemoryFixtureEvaluator:
    def __init__(self, *, workspace: Path) -> None:
        self._workspace = Path(workspace)

    async def execute(self, job: dict[str, Any]) -> dict[str, Any]:
        from eval.group_memory.runner import run_group_memory_fixture_eval

        payload = _payload(job)
        fixture = str(
            payload.get("fixture")
            or payload.get("fixture_path")
            or "tests/fixtures/group_memory_open_strategy_dataset.json"
        )
        job_id = str(job.get("job_id") or "rag_eval")
        default_workspace = self._workspace / "eval" / "rag" / _safe_name(job_id)
        workspace = Path(str(payload.get("workspace") or default_workspace))
        return await asyncio.to_thread(
            run_group_memory_fixture_eval,
            fixture_path=fixture,
            workspace=workspace,
            min_top1_accuracy=_float_value(payload.get("min_top1_accuracy"), 1.0),
            min_evidence_coverage=_float_value(
                payload.get("min_evidence_coverage"),
                1.0,
            ),
        )


class AgentGatewayRagEvalWorker:
    def __init__(
        self,
        *,
        client: AgentGatewayClient,
        evaluator: RagEvalExecutor,
        worker_id: str,
        lease_ttl_seconds: int = 300,
        poll_interval_seconds: float = 2.0,
    ) -> None:
        self._client = client
        self._evaluator = evaluator
        self._worker_id = str(worker_id or "akashic-python-worker")
        self._lease_ttl = max(10, int(lease_ttl_seconds or 300))
        self._poll_interval = max(0.5, float(poll_interval_seconds or 2.0))
        self._heartbeat_interval = max(5.0, min(float(self._lease_ttl) / 3.0, 60.0))
        self._stopped = asyncio.Event()
        self._status = AgentWorkerStatusReporter(
            client=client,
            worker_id=self._worker_id,
            worker_type="rag_eval",
            logger=logger,
            label="agent_runtime_rag_eval_worker",
            lease_ttl_seconds=self._lease_ttl,
        )

    async def process_once(self) -> dict[str, Any]:
        try:
            job = await self._client.lease_next(
                worker_id=self._worker_id,
                job_type="rag_eval",
                ttl_seconds=self._lease_ttl,
            )
        except AgentGatewayNoJob:
            await self._status.idle(reason="no_job")
            return {"processed": False, "reason": "no_job"}

        job_id = str(job.get("job_id") or "")
        lease_token = _lease_token(job)
        heartbeat = AgentJobLeaseHeartbeat(
            client=self._client,
            job_id=job_id,
            lease_token=lease_token,
            ttl_seconds=self._lease_ttl,
            interval_seconds=self._heartbeat_interval,
            logger=logger,
            label="agent_runtime_rag_eval_worker",
        )
        status_heartbeat = self._status.running_heartbeat(
            current_job_id=job_id,
            interval_seconds=self._heartbeat_interval,
        )
        try:
            await self._status.running(current_job_id=job_id)
            await self._client.mark_running(job_id, lease_token=lease_token)
            heartbeat.start()
            status_heartbeat.start()
            result = await self._evaluator.execute(job)
            await self._client.complete_job(
                job_id,
                lease_token=lease_token,
                result=_string_result(result),
            )
            await self._status.succeeded(last_job_id=job_id)
            return {
                "processed": True,
                "job_id": job_id,
                "job_type": "rag_eval",
                "result": result,
            }
        except Exception as exc:
            await heartbeat.stop()
            message = str(exc)
            logger.exception(
                "[agent_runtime_rag_eval_worker] job failed job_id=%s",
                job_id,
            )
            if job_id:
                await self._client.fail_job(
                    job_id,
                    lease_token=lease_token,
                    error_message=message,
                )
            await self._status.failed(last_job_id=job_id, error=message)
            return {
                "processed": True,
                "job_id": job_id,
                "job_type": "rag_eval",
                "failed": True,
                "error": message,
            }
        finally:
            await status_heartbeat.stop()
            await heartbeat.stop()

    async def run(self) -> None:
        logger.info(
            "[agent_runtime_rag_eval_worker] loop started worker_id=%s",
            self._worker_id,
        )
        await self._status.starting()
        try:
            while not self._stopped.is_set():
                try:
                    result = await self.process_once()
                    if result.get("processed") and not result.get("failed"):
                        logger.info(
                            "[agent_runtime_rag_eval_worker] processed %s",
                            result,
                        )
                except AgentGatewayError as exc:
                    logger.warning(
                        "[agent_runtime_rag_eval_worker] runtime error: %s",
                        exc,
                    )
                except Exception:
                    logger.exception("[agent_runtime_rag_eval_worker] unexpected loop error")
                try:
                    await asyncio.wait_for(
                        self._stopped.wait(),
                        timeout=self._poll_interval,
                    )
                except asyncio.TimeoutError:
                    continue
        finally:
            await self._status.stopped()
            logger.info("[agent_runtime_rag_eval_worker] loop stopped")

    def stop(self) -> None:
        self._stopped.set()


def _payload(job: dict[str, Any]) -> dict[str, Any]:
    payload = job.get("payload")
    return payload if isinstance(payload, dict) else {}


def _lease_token(job: dict[str, Any]) -> str:
    return str(job.get("lease_token") or "")


def _string_result(result: dict[str, Any]) -> dict[str, str]:
    stringified: dict[str, str] = {}
    for key, value in result.items():
        if isinstance(value, str):
            stringified[key] = value
        else:
            stringified[key] = json.dumps(value, ensure_ascii=False)
    return stringified


def _float_value(value: Any, fallback: float) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return fallback


def _safe_name(value: str) -> str:
    cleaned = re.sub(r"[^a-zA-Z0-9_.-]+", "_", value.strip())
    return cleaned[:120] or "rag_eval"
