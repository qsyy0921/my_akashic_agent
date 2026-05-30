from __future__ import annotations

import asyncio
import json
import logging
from pathlib import Path
from typing import Any

from agent.tools.base import Tool
from integrations.agent_gateway import (
    AgentGatewayClient,
    AgentGatewayError,
    AgentGatewayNoJob,
)

logger = logging.getLogger(__name__)


class AgentGatewayImageWorker:
    def __init__(
        self,
        *,
        client: AgentGatewayClient,
        image_tool: Tool,
        worker_id: str,
        lease_ttl_seconds: int = 300,
        poll_interval_seconds: float = 2.0,
    ) -> None:
        self._client = client
        self._image_tool = image_tool
        self._worker_id = str(worker_id or "akashic-python-worker")
        self._lease_ttl = max(10, int(lease_ttl_seconds or 300))
        self._poll_interval = max(0.5, float(poll_interval_seconds or 2.0))
        self._stopped = asyncio.Event()

    async def process_once(self) -> dict[str, Any]:
        try:
            job = await self._client.lease_next(
                worker_id=self._worker_id,
                job_type="image_generation",
                ttl_seconds=self._lease_ttl,
            )
        except AgentGatewayNoJob:
            return {"processed": False, "reason": "no_job"}

        job_id = str(job.get("job_id") or "")
        lease_token = _lease_token(job)
        legacy_job_id = _legacy_image_job_id(job)
        try:
            await self._client.mark_running(job_id, lease_token=lease_token)
            if legacy_job_id:
                await self._client.mark_image_job_running(legacy_job_id)
            tool_result = await self._execute_image_tool(job)
            attachments = _attachments_from_result(tool_result)
            if not attachments:
                raise RuntimeError("image tool did not return saved image paths")
            metadata = {
                "agent_job_id": job_id,
                "worker_id": self._worker_id,
                "result_count": str(len(attachments)),
            }
            if legacy_job_id:
                await self._client.complete_image_job(
                    legacy_job_id,
                    results=attachments,
                    metadata=metadata,
                )
            await self._client.complete_job(
                job_id,
                lease_token=lease_token,
                result={
                    "legacy_image_job_id": legacy_job_id,
                    "paths": json.dumps(
                        [item["url"] for item in attachments],
                        ensure_ascii=False,
                    ),
                    "count": str(len(attachments)),
                },
            )
            return {
                "processed": True,
                "job_id": job_id,
                "legacy_image_job_id": legacy_job_id,
                "count": len(attachments),
            }
        except Exception as exc:
            message = str(exc)
            logger.exception("[agent_runtime_image_worker] job failed job_id=%s", job_id)
            await self._safe_fail(job_id, legacy_job_id, lease_token, message)
            return {
                "processed": True,
                "job_id": job_id,
                "legacy_image_job_id": legacy_job_id,
                "failed": True,
                "error": message,
            }

    async def run(self) -> None:
        logger.info(
            "[agent_runtime_image_worker] loop started worker_id=%s",
            self._worker_id,
        )
        try:
            while not self._stopped.is_set():
                try:
                    result = await self.process_once()
                    if result.get("processed") and not result.get("failed"):
                        logger.info("[agent_runtime_image_worker] processed %s", result)
                except AgentGatewayError as exc:
                    logger.warning("[agent_runtime_image_worker] runtime error: %s", exc)
                except Exception:
                    logger.exception("[agent_runtime_image_worker] unexpected loop error")
                try:
                    await asyncio.wait_for(
                        self._stopped.wait(),
                        timeout=self._poll_interval,
                    )
                except asyncio.TimeoutError:
                    continue
        finally:
            logger.info("[agent_runtime_image_worker] loop stopped")

    def stop(self) -> None:
        self._stopped.set()

    async def _execute_image_tool(self, job: dict[str, Any]) -> dict[str, Any]:
        payload = job.get("payload") if isinstance(job.get("payload"), dict) else {}
        raw = await self._image_tool.execute(
            prompt=str(payload.get("prompt") or ""),
            size=str(payload.get("size") or "1024x1024"),
            n=_positive_int(payload.get("count"), 1),
            model=str(payload.get("model") or ""),
            response_format=None,
        )
        if not isinstance(raw, str):
            raw = raw.preview()
        try:
            result = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise RuntimeError(f"image tool returned non-json result: {raw[:500]}") from exc
        if not result.get("ok"):
            raise RuntimeError(str(result.get("error") or result))
        return result

    async def _safe_fail(
        self,
        job_id: str,
        legacy_job_id: str,
        lease_token: str,
        message: str,
    ) -> None:
        if legacy_job_id:
            try:
                await self._client.fail_image_job(legacy_job_id, error_message=message)
            except Exception:
                logger.warning(
                    "[agent_runtime_image_worker] legacy fail update failed job_id=%s",
                    legacy_job_id,
                    exc_info=True,
                )
        if job_id:
            try:
                await self._client.fail_job(
                    job_id,
                    lease_token=lease_token,
                    error_message=message,
                )
            except Exception:
                logger.warning(
                    "[agent_runtime_image_worker] generic fail update failed job_id=%s",
                    job_id,
                    exc_info=True,
                )


def _legacy_image_job_id(job: dict[str, Any]) -> str:
    payload = job.get("payload") if isinstance(job.get("payload"), dict) else {}
    metadata = job.get("metadata") if isinstance(job.get("metadata"), dict) else {}
    return str(
        payload.get("legacy_image_job_id")
        or metadata.get("legacy_image_job_id")
        or job.get("job_id")
        or ""
    )


def _lease_token(job: dict[str, Any]) -> str:
    return str(job.get("lease_token") or "")


def _attachments_from_result(result: dict[str, Any]) -> list[dict[str, Any]]:
    paths = result.get("paths")
    if not isinstance(paths, list):
        return []
    attachments: list[dict[str, Any]] = []
    for index, value in enumerate(paths, start=1):
        path = Path(str(value))
        attachments.append(
            {
                "id": f"generated-image-{index}",
                "kind": "image",
                "url": _path_to_file_url(path),
                "mime_type": _mime_for_path(path),
                "name": path.name or f"generated-image-{index}",
                "size_bytes": path.stat().st_size if path.exists() else 0,
            }
        )
    return attachments


def _path_to_file_url(path: Path) -> str:
    return path.resolve().as_uri()


def _mime_for_path(path: Path) -> str:
    suffix = path.suffix.lower()
    if suffix in {".jpg", ".jpeg"}:
        return "image/jpeg"
    if suffix == ".webp":
        return "image/webp"
    if suffix == ".gif":
        return "image/gif"
    return "image/png"


def _positive_int(value: Any, fallback: int) -> int:
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        return fallback
    return max(1, parsed)
