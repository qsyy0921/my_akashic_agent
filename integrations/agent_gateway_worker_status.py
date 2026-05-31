from __future__ import annotations

import logging
import os
import secrets
from typing import Any


class AgentWorkerStatusReporter:
    def __init__(
        self,
        *,
        client: Any,
        worker_id: str,
        worker_type: str,
        logger: logging.Logger,
        label: str,
    ) -> None:
        self._client = client
        self._worker_id = str(worker_id or "akashic-python-worker")
        self._instance_id = f"{self._worker_id}:{os.getpid()}:{secrets.token_hex(6)}"
        self._worker_type = str(worker_type or "unknown")
        self._logger = logger
        self._label = str(label or self._worker_type)
        self._processed_total = 0
        self._failed_total = 0
        config = getattr(client, "_config", None)
        self._lease_ttl_seconds = max(
            30,
            int(getattr(config, "lease_ttl_seconds", 120) or 120),
        )

    async def starting(self) -> None:
        await self._report("starting")

    async def idle(self, *, reason: str = "") -> None:
        metadata = {"reason": reason} if reason else None
        await self._report("idle", metadata=metadata)

    async def running(self, *, current_job_id: str) -> None:
        await self._report("running", current_job_id=current_job_id)

    async def succeeded(self, *, last_job_id: str) -> None:
        self._processed_total += 1
        await self._report("idle", last_job_id=last_job_id)

    async def failed(self, *, last_job_id: str, error: str) -> None:
        self._failed_total += 1
        await self._report("failed", last_job_id=last_job_id, last_error=error)

    async def stopped(self) -> None:
        await self._report("stopped")

    async def _report(
        self,
        status: str,
        *,
        current_job_id: str = "",
        last_job_id: str = "",
        last_error: str = "",
        metadata: dict[str, str] | None = None,
    ) -> None:
        method = getattr(self._client, "report_agent_worker_status", None)
        if not callable(method):
            return
        try:
            await method(
                worker_id=self._worker_id,
                instance_id=self._instance_id,
                worker_type=self._worker_type,
                status=status,
                current_job_id=current_job_id,
                last_job_id=last_job_id,
                last_error=last_error,
                processed_total=self._processed_total,
                failed_total=self._failed_total,
                lease_ttl_seconds=self._lease_ttl_seconds,
                metadata=metadata or {},
            )
        except Exception:
            self._logger.warning(
                "[%s] agent worker status report failed",
                self._label,
                exc_info=True,
            )
