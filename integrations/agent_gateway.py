from __future__ import annotations

import json
from dataclasses import asdict
from typing import Any
from urllib.parse import quote

import httpx

from agent.config_models import AgentGatewayIntegrationConfig


class AgentGatewayError(RuntimeError):
    pass


class AgentGatewayNoJob(AgentGatewayError):
    pass


class AgentGatewayClient:
    def __init__(
        self,
        config: AgentGatewayIntegrationConfig,
        *,
        transport: httpx.AsyncBaseTransport | None = None,
    ) -> None:
        self._config = config
        self._base_url = str(config.base_url or "").rstrip("/")
        self._transport = transport

    @property
    def config_snapshot(self) -> dict[str, Any]:
        return asdict(self._config)

    async def health(self) -> dict[str, Any]:
        return await self._request("GET", "/healthz")

    async def create_job(
        self,
        *,
        job_id: str,
        job_type: str,
        agent_id: str,
        route: dict[str, Any],
        source_event_ids: list[str] | None = None,
        source_asset_ids: list[str] | None = None,
        payload: dict[str, str] | None = None,
        max_attempts: int | None = None,
        metadata: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        body: dict[str, Any] = {
            "job_id": job_id,
            "job_type": job_type,
            "agent_id": agent_id,
            "route": route,
            "source_event_ids": source_event_ids or [],
            "source_asset_ids": source_asset_ids or [],
            "payload": payload or {},
            "metadata": metadata or {},
        }
        if max_attempts is not None:
            body["max_attempts"] = int(max_attempts)
        return await self._request("POST", "/v1/jobs", json_body=body)

    async def list_jobs(
        self,
        *,
        job_type: str = "",
        status: str = "",
        limit: int = 50,
    ) -> list[dict[str, Any]]:
        params: dict[str, Any] = {"limit": max(1, min(int(limit), 200))}
        if job_type:
            params["type"] = job_type
        if status:
            params["status"] = status
        data = await self._request("GET", "/v1/jobs", params=params)
        if not isinstance(data, list):
            raise AgentGatewayError("agent runtime jobs response is not a list")
        return data

    async def get_job(self, job_id: str) -> dict[str, Any]:
        return await self._request("GET", f"/v1/jobs/{job_id}")

    async def list_inbox_events(
        self,
        *,
        channel_kind: str = "",
        account_id: str = "",
        conversation_id: str = "",
        conversation_type: str = "",
        sender_id: str = "",
        decision_action: str = "",
        observe_only: bool | None = None,
        after_seq: int | None = None,
        order: str = "",
        limit: int = 50,
    ) -> list[dict[str, Any]]:
        params: dict[str, Any] = {"limit": max(1, min(int(limit), 200))}
        if channel_kind:
            params["channel_kind"] = str(channel_kind)
        if account_id:
            params["account_id"] = str(account_id)
        if conversation_id:
            params["conversation_id"] = str(conversation_id)
        if conversation_type:
            params["conversation_type"] = str(conversation_type)
        if sender_id:
            params["sender_id"] = str(sender_id)
        if decision_action:
            params["decision_action"] = str(decision_action)
        if observe_only is not None:
            params["observe_only"] = "true" if observe_only else "false"
        if after_seq is not None:
            params["after_seq"] = max(0, int(after_seq))
        if order:
            params["order"] = str(order)
        data = await self._request("GET", "/v1/inbox", params=params)
        if not isinstance(data, list):
            raise AgentGatewayError("agent runtime inbox response is not a list")
        return data

    async def lease_next(
        self,
        *,
        worker_id: str | None = None,
        job_type: str = "",
        ttl_seconds: int | None = None,
    ) -> dict[str, Any]:
        body = {
            "worker_id": worker_id or self._config.worker_id,
            "job_type": job_type,
            "ttl_seconds": int(ttl_seconds or self._config.lease_ttl_seconds),
        }
        return await self._request(
            "POST",
            "/v1/jobs/lease-next",
            json_body=body,
            no_job_on_404=True,
        )

    async def lease_job(
        self,
        job_id: str,
        *,
        worker_id: str | None = None,
        ttl_seconds: int | None = None,
    ) -> dict[str, Any]:
        body = {
            "worker_id": worker_id or self._config.worker_id,
            "ttl_seconds": int(ttl_seconds or self._config.lease_ttl_seconds),
        }
        return await self._request("POST", f"/v1/jobs/{job_id}/lease", json_body=body)

    async def mark_running(self, job_id: str) -> dict[str, Any]:
        return await self._request("POST", f"/v1/jobs/{job_id}/running", json_body={})

    async def complete_job(
        self,
        job_id: str,
        *,
        result: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        return await self._request(
            "POST",
            f"/v1/jobs/{job_id}/succeeded",
            json_body={"result": result or {}},
        )

    async def fail_job(self, job_id: str, *, error_message: str) -> dict[str, Any]:
        return await self._request(
            "POST",
            f"/v1/jobs/{job_id}/failed",
            json_body={"error_message": error_message},
        )

    async def mark_image_job_running(self, job_id: str) -> dict[str, Any]:
        return await self._request(
            "POST",
            f"/v1/image-jobs/{job_id}/running",
            json_body={},
        )

    async def complete_image_job(
        self,
        job_id: str,
        *,
        results: list[dict[str, Any]],
        metadata: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        return await self._request(
            "POST",
            f"/v1/image-jobs/{job_id}/succeeded",
            json_body={"results": results, "metadata": metadata or {}},
        )

    async def fail_image_job(
        self, job_id: str, *, error_message: str
    ) -> dict[str, Any]:
        return await self._request(
            "POST",
            f"/v1/image-jobs/{job_id}/failed",
            json_body={"error_message": error_message},
        )

    async def record_send(
        self,
        *,
        from_bot_id: str,
        conversation_id: str,
        content: str = "",
        content_hash: str = "",
    ) -> dict[str, Any]:
        body: dict[str, Any] = {
            "from_bot_id": str(from_bot_id),
            "conversation_id": str(conversation_id),
        }
        if content_hash:
            body["content_hash"] = str(content_hash)
        else:
            body["content"] = str(content)
        return await self._request(
            "POST",
            "/v1/send-ledger/records",
            json_body=body,
        )

    async def recently_sent(
        self,
        *,
        from_bot_id: str,
        conversation_id: str,
        content: str = "",
        content_hash: str = "",
        window_seconds: int = 15,
    ) -> bool:
        params: dict[str, Any] = {
            "from_bot_id": str(from_bot_id),
            "conversation_id": str(conversation_id),
            "window_seconds": max(1, int(window_seconds)),
        }
        if content_hash:
            params["content_hash"] = str(content_hash)
        else:
            params["content"] = str(content)
        data = await self._request(
            "GET",
            "/v1/send-ledger/recent",
            params=params,
        )
        if not isinstance(data, dict):
            raise AgentGatewayError(
                "agent runtime recent-send response is not an object"
            )
        return bool(data.get("recent"))

    async def retry_job(self, job_id: str) -> dict[str, Any]:
        return await self._request("POST", f"/v1/jobs/{job_id}/retry", json_body={})

    async def cancel_job(self, job_id: str) -> dict[str, Any]:
        return await self._request("POST", f"/v1/jobs/{job_id}/cancel", json_body={})

    async def lease_next_outbox(
        self,
        *,
        worker_id: str | None = None,
        ttl_seconds: int | None = None,
    ) -> dict[str, Any]:
        body = {
            "worker_id": worker_id or self._config.worker_id,
            "ttl_seconds": int(ttl_seconds or self._config.lease_ttl_seconds),
        }
        return await self._request(
            "POST",
            "/v1/outbox/lease-next",
            json_body=body,
            no_job_on_404=True,
        )

    async def mark_outbox_succeeded(self, event_id: str) -> dict[str, Any]:
        encoded_id = quote(str(event_id), safe="")
        return await self._request(
            "POST",
            f"/v1/outbox/{encoded_id}/succeeded",
            json_body={},
        )

    async def mark_outbox_failed(
        self,
        event_id: str,
        *,
        error_message: str,
        error_kind: str = "",
    ) -> dict[str, Any]:
        encoded_id = quote(str(event_id), safe="")
        body: dict[str, Any] = {"error_message": error_message}
        if error_kind:
            body["error_kind"] = str(error_kind)
        return await self._request(
            "POST",
            f"/v1/outbox/{encoded_id}/failed",
            json_body=body,
        )

    async def _request(
        self,
        method: str,
        path: str,
        *,
        params: dict[str, Any] | None = None,
        json_body: Any = None,
        no_job_on_404: bool = False,
    ) -> Any:
        if not self._config.enabled:
            raise AgentGatewayError("agent runtime client 未启用")
        if not self._base_url:
            raise AgentGatewayError("agent runtime base_url 未配置")
        url = self._base_url + path
        async with httpx.AsyncClient(
            timeout=self._config.request_timeout_seconds,
            transport=self._transport,
            trust_env=True,
        ) as client:
            response = await client.request(
                method.upper(),
                url,
                params=params,
                json=json_body,
                headers={"Content-Type": "application/json"},
            )
        if response.status_code == 404 and no_job_on_404:
            raise AgentGatewayNoJob(response.text.strip() or "no leaseable job")
        if response.status_code >= 400:
            raise AgentGatewayError(
                f"agent runtime HTTP {response.status_code}: {response.text[:500]}"
            )
        try:
            payload = response.json()
        except json.JSONDecodeError:
            return {"status_code": response.status_code, "text": response.text}
        if not isinstance(payload, dict):
            return payload
        code = payload.get("code")
        if code not in (None, "OK", 0):
            raise AgentGatewayError(str(payload.get("message") or payload))
        return payload.get("data", payload)
