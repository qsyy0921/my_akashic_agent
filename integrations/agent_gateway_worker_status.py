from __future__ import annotations

import asyncio
import logging
import os
import re
import secrets
from typing import Any

from integrations.agent_gateway import AgentGatewayHTTPError

_LEASE_CONFLICT_INSTANCE_ID_PATTERN = re.compile(r"existing_instance_id=([^\s]+)")


class AgentWorkerStatusReporter:
    def __init__(
        self,
        *,
        client: Any,
        worker_id: str,
        worker_type: str,
        logger: logging.Logger,
        label: str,
        lease_ttl_seconds: int | None = None,
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
            int(lease_ttl_seconds or getattr(config, "lease_ttl_seconds", 120) or 120),
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

    def running_heartbeat(
        self,
        *,
        current_job_id: str,
        interval_seconds: float,
    ) -> "AgentWorkerStatusHeartbeat":
        return AgentWorkerStatusHeartbeat(
            reporter=self,
            current_job_id=current_job_id,
            interval_seconds=interval_seconds,
        )

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
            await self._report_once(
                method,
                status=status,
                current_job_id=current_job_id,
                last_job_id=last_job_id,
                last_error=last_error,
                metadata=metadata,
            )
        except AgentGatewayHTTPError as exc:
            recovered = await self._retry_conflict_takeover(
                method,
                exc,
                status=status,
                current_job_id=current_job_id,
                last_job_id=last_job_id,
                last_error=last_error,
                metadata=metadata,
            )
            if recovered:
                return
            self._logger.warning(
                "[%s] agent worker status report failed",
                self._label,
                exc_info=True,
            )
        except Exception:
            self._logger.warning(
                "[%s] agent worker status report failed",
                self._label,
                exc_info=True,
            )

    async def _report_once(
        self,
        method: Any,
        *,
        status: str,
        current_job_id: str = "",
        last_job_id: str = "",
        last_error: str = "",
        metadata: dict[str, str] | None = None,
        replace_existing_instance_id: str = "",
    ) -> dict[str, Any]:
        return await method(
            worker_id=self._worker_id,
            instance_id=self._instance_id,
            replace_existing_instance_id=replace_existing_instance_id,
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

    async def _retry_conflict_takeover(
        self,
        method: Any,
        exc: AgentGatewayHTTPError,
        *,
        status: str,
        current_job_id: str = "",
        last_job_id: str = "",
        last_error: str = "",
        metadata: dict[str, str] | None = None,
    ) -> bool:
        if int(getattr(exc, "status_code", 0) or 0) != 409:
            return False
        existing_instance_id = _extract_existing_instance_id_from_conflict(exc.text)
        if not existing_instance_id or existing_instance_id == self._instance_id:
            return False
        existing_pid = _extract_pid_from_instance_id(existing_instance_id)
        if existing_pid is None:
            return False
        pid_alive = await asyncio.to_thread(_process_exists, existing_pid)
        if pid_alive:
            return False
        await self._report_once(
            method,
            status=status,
            current_job_id=current_job_id,
            last_job_id=last_job_id,
            last_error=last_error,
            metadata=metadata,
            replace_existing_instance_id=existing_instance_id,
        )
        self._logger.info(
            "[%s] took over stale agent worker status lease from instance %s",
            self._label,
            existing_instance_id,
        )
        return True


class AgentWorkerStatusHeartbeat:
    def __init__(
        self,
        *,
        reporter: AgentWorkerStatusReporter,
        current_job_id: str,
        interval_seconds: float,
    ) -> None:
        self._reporter = reporter
        self._current_job_id = str(current_job_id or "")
        self._interval_seconds = max(1.0, float(interval_seconds or 30.0))
        self._task: asyncio.Task[None] | None = None

    def start(self) -> None:
        if not self._current_job_id or self._task is not None:
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

    async def beat_once(self) -> None:
        await self._reporter._report("running", current_job_id=self._current_job_id)

    async def _run(self) -> None:
        while True:
            await asyncio.sleep(self._interval_seconds)
            await self.beat_once()


def _extract_existing_instance_id_from_conflict(text: str) -> str:
    match = _LEASE_CONFLICT_INSTANCE_ID_PATTERN.search(str(text or ""))
    if not match:
        return ""
    return str(match.group(1) or "").strip()


def _extract_pid_from_instance_id(instance_id: str) -> int | None:
    value = str(instance_id or "").strip()
    if not value:
        return None
    parts = value.rsplit(":", 2)
    if len(parts) != 3:
        return None
    try:
        pid = int(parts[1])
    except (TypeError, ValueError):
        return None
    return pid if pid > 0 else None


def _process_exists(pid: int) -> bool:
    try:
        import ctypes

        kernel32 = ctypes.windll.kernel32
        process = kernel32.OpenProcess(0x1000, 0, int(pid))
        if process:
            kernel32.CloseHandle(process)
            return True
        if kernel32.GetLastError() == 5:
            return True
        return False
    except Exception:
        pass
    try:
        os.kill(int(pid), 0)
    except OSError:
        return False
    except Exception:
        return True
    return True
