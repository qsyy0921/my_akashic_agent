from __future__ import annotations

import asyncio
import logging
from typing import Any


class AgentJobLeaseHeartbeat:
    def __init__(
        self,
        *,
        client: Any,
        job_id: str,
        lease_token: str,
        ttl_seconds: int,
        interval_seconds: float,
        logger: logging.Logger,
        label: str,
    ) -> None:
        self._client = client
        self._job_id = str(job_id or "")
        self._lease_token = str(lease_token or "")
        self._ttl_seconds = max(1, int(ttl_seconds or 300))
        self._interval_seconds = max(1.0, float(interval_seconds or 30.0))
        self._logger = logger
        self._label = str(label or "agent_job")
        self._task: asyncio.Task[None] | None = None

    def start(self) -> None:
        if not self._job_id or not self._lease_token or self._task is not None:
            return
        self._task = asyncio.create_task(self._run())

    async def stop(self) -> None:
        task = self._task
        self._task = None
        if task is None:
            return
        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            return

    async def _run(self) -> None:
        while True:
            await asyncio.sleep(self._interval_seconds)
            try:
                await self._client.renew_job(
                    self._job_id,
                    lease_token=self._lease_token,
                    ttl_seconds=self._ttl_seconds,
                )
            except asyncio.CancelledError:
                raise
            except Exception:
                self._logger.warning(
                    "[%s] agent job lease renew failed job_id=%s",
                    self._label,
                    self._job_id,
                    exc_info=True,
                )
