from __future__ import annotations

import asyncio
import json
import logging
import time
from typing import Any, Protocol

from integrations.agent_gateway import (
    AgentGatewayClient,
    AgentGatewayError,
    AgentGatewayNoJob,
)

logger = logging.getLogger(__name__)


class GroupMemoryIngestor(Protocol):
    def ingest_group(self, group_id: str) -> Any:
        ...


class AsyncGroupIndexer(Protocol):
    async def execute(self, **kwargs: Any) -> str:
        ...


class AgentGatewayKnowledgeWorker:
    def __init__(
        self,
        *,
        client: AgentGatewayClient,
        group_memory: GroupMemoryIngestor,
        worker_id: str,
        group_accounts: dict[str, str],
        ragflow_indexer: AsyncGroupIndexer | None = None,
        ragflow_dataset_ids: list[str] | None = None,
        lease_ttl_seconds: int = 300,
        poll_interval_seconds: float = 2.0,
        enqueue_interval_seconds: float = 60.0,
        now_fn: Any = time.time,
    ) -> None:
        self._client = client
        self._group_memory = group_memory
        self._worker_id = str(worker_id or "akashic-python-worker")
        self._group_accounts = {
            str(group_id).strip(): str(account_id).strip()
            for group_id, account_id in group_accounts.items()
            if str(group_id).strip() and str(account_id).strip()
        }
        self._ragflow_indexer = ragflow_indexer
        self._ragflow_dataset_ids = [
            str(value).strip() for value in (ragflow_dataset_ids or []) if str(value).strip()
        ]
        self._lease_ttl = max(10, int(lease_ttl_seconds or 300))
        self._poll_interval = max(0.5, float(poll_interval_seconds or 2.0))
        self._enqueue_interval = max(5.0, float(enqueue_interval_seconds or 60.0))
        self._now_fn = now_fn
        self._stopped = asyncio.Event()
        self._last_enqueue_bucket = -1

    async def enqueue_once(self) -> dict[str, Any]:
        bucket = int(float(self._now_fn()) // self._enqueue_interval)
        created = 0
        for group_id, account_id in self._group_accounts.items():
            await self._create_group_memory_job(group_id, account_id, bucket)
            created += 1
            if self._ragflow_indexer is not None:
                for dataset_id in self._ragflow_dataset_ids:
                    await self._create_rag_ingest_job(
                        group_id,
                        account_id,
                        dataset_id,
                        bucket,
                    )
                    created += 1
        self._last_enqueue_bucket = bucket
        return {
            "bucket": bucket,
            "created_or_existing": created,
            "groups": sorted(self._group_accounts),
        }

    async def process_once(self) -> dict[str, Any]:
        try:
            job = await self._lease_next_knowledge_job()
        except AgentGatewayNoJob:
            return {"processed": False, "reason": "no_job"}

        job_id = str(job.get("job_id") or "")
        job_type = str(job.get("job_type") or "")
        try:
            await self._client.mark_running(job_id)
            if job_type == "group_memory_extract":
                result = await self._process_group_memory_job(job)
            elif job_type == "rag_ingest":
                result = await self._process_rag_ingest_job(job)
            else:
                raise RuntimeError(f"unsupported knowledge job type: {job_type}")
            await self._client.complete_job(job_id, result=_string_result(result))
            return {
                "processed": True,
                "job_id": job_id,
                "job_type": job_type,
                "result": result,
            }
        except Exception as exc:
            message = str(exc)
            logger.exception("[agent_runtime_knowledge_worker] job failed job_id=%s", job_id)
            if job_id:
                await self._client.fail_job(job_id, error_message=message)
            return {
                "processed": True,
                "job_id": job_id,
                "job_type": job_type,
                "failed": True,
                "error": message,
            }

    async def run(self) -> None:
        logger.info(
            "[agent_runtime_knowledge_worker] loop started worker_id=%s groups=%s",
            self._worker_id,
            sorted(self._group_accounts),
        )
        try:
            while not self._stopped.is_set():
                try:
                    bucket = int(float(self._now_fn()) // self._enqueue_interval)
                    if bucket != self._last_enqueue_bucket:
                        summary = await self.enqueue_once()
                        logger.info("[agent_runtime_knowledge_worker] enqueued %s", summary)
                    result = await self.process_once()
                    if result.get("processed") and not result.get("failed"):
                        logger.info("[agent_runtime_knowledge_worker] processed %s", result)
                except AgentGatewayError as exc:
                    logger.warning("[agent_runtime_knowledge_worker] runtime error: %s", exc)
                except Exception:
                    logger.exception("[agent_runtime_knowledge_worker] unexpected loop error")
                try:
                    await asyncio.wait_for(
                        self._stopped.wait(),
                        timeout=self._poll_interval,
                    )
                except asyncio.TimeoutError:
                    continue
        finally:
            logger.info("[agent_runtime_knowledge_worker] loop stopped")

    def stop(self) -> None:
        self._stopped.set()

    async def _lease_next_knowledge_job(self) -> dict[str, Any]:
        last_no_job: AgentGatewayNoJob | None = None
        for job_type in ("group_memory_extract", "rag_ingest"):
            try:
                return await self._client.lease_next(
                    worker_id=self._worker_id,
                    job_type=job_type,
                    ttl_seconds=self._lease_ttl,
                )
            except AgentGatewayNoJob as exc:
                last_no_job = exc
        raise last_no_job or AgentGatewayNoJob("no knowledge jobs")

    async def _create_group_memory_job(
        self,
        group_id: str,
        account_id: str,
        bucket: int,
    ) -> None:
        await self._client.create_job(
            job_id=f"group_memory_extract:qq:{group_id}:{bucket}",
            job_type="group_memory_extract",
            agent_id=self._worker_id,
            route=_qq_group_route(account_id, group_id),
            payload={
                "group_id": group_id,
                "session_key": f"qq:gqq:{group_id}",
                "observe_only": "true",
            },
            max_attempts=2,
            metadata={"scheduler": "agent_runtime_knowledge_worker"},
        )

    async def _create_rag_ingest_job(
        self,
        group_id: str,
        account_id: str,
        dataset_id: str,
        bucket: int,
    ) -> None:
        await self._client.create_job(
            job_id=f"rag_ingest:qq:{group_id}:{dataset_id}:{bucket}",
            job_type="rag_ingest",
            agent_id=self._worker_id,
            route=_qq_group_route(account_id, group_id),
            payload={
                "group_id": group_id,
                "session_key": f"qq:gqq:{group_id}",
                "dataset_id": dataset_id,
                "max_messages": "1000",
                "parse": "true",
                "observe_only": "true",
            },
            max_attempts=2,
            metadata={"scheduler": "agent_runtime_knowledge_worker"},
        )

    async def _process_group_memory_job(self, job: dict[str, Any]) -> dict[str, Any]:
        payload = _payload(job)
        group_id = str(payload.get("group_id") or job.get("route", {}).get("conversation_id") or "")
        if not group_id.strip():
            raise RuntimeError("group_memory_extract job requires group_id")
        stats = self._group_memory.ingest_group(group_id.strip())
        data = dict(getattr(stats, "__dict__", {}) or {})
        data.setdefault("group_id", group_id.strip())
        data.setdefault("session_key", f"qq:gqq:{group_id.strip()}")
        return data

    async def _process_rag_ingest_job(self, job: dict[str, Any]) -> dict[str, Any]:
        if self._ragflow_indexer is None:
            raise RuntimeError("ragflow indexer is not configured")
        payload = _payload(job)
        group_id = str(payload.get("group_id") or "").strip()
        dataset_id = str(payload.get("dataset_id") or "").strip()
        if not group_id or not dataset_id:
            raise RuntimeError("rag_ingest job requires group_id and dataset_id")
        raw = await self._ragflow_indexer.execute(
            dataset_id=dataset_id,
            group_id=group_id,
            max_messages=_positive_int(payload.get("max_messages"), 1000),
            since_seq=_non_negative_int(payload.get("since_seq"), 0),
            parse=_bool_text(payload.get("parse"), True),
        )
        try:
            result = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise RuntimeError(f"ragflow indexer returned non-json result: {raw[:500]}") from exc
        if not result.get("ok"):
            raise RuntimeError(str(result.get("error") or result))
        return {
            "group_id": group_id,
            "dataset_id": dataset_id,
            "message_count": result.get("message_count"),
            "display_name": result.get("display_name"),
            "data": result.get("data"),
        }


def _qq_group_route(account_id: str, group_id: str) -> dict[str, str]:
    return {
        "kind": "qq",
        "account_id": account_id,
        "conversation_id": group_id,
        "conversation_type": "group",
    }


def _payload(job: dict[str, Any]) -> dict[str, Any]:
    payload = job.get("payload")
    return payload if isinstance(payload, dict) else {}


def _string_result(result: dict[str, Any]) -> dict[str, str]:
    stringified: dict[str, str] = {}
    for key, value in result.items():
        if isinstance(value, str):
            stringified[key] = value
        else:
            stringified[key] = json.dumps(value, ensure_ascii=False)
    return stringified


def _positive_int(value: Any, fallback: int) -> int:
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        return fallback
    return max(1, parsed)


def _non_negative_int(value: Any, fallback: int) -> int:
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        return fallback
    return max(0, parsed)


def _bool_text(value: Any, fallback: bool) -> bool:
    if value is None or value == "":
        return fallback
    return str(value).strip().lower() in {"1", "true", "yes", "y", "on"}
