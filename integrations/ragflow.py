from __future__ import annotations

import json
import mimetypes
from dataclasses import asdict
from typing import Any

import httpx

from agent.config_models import RAGFlowIntegrationConfig


class RAGFlowError(RuntimeError):
    pass


class RAGFlowClient:
    def __init__(
        self,
        config: RAGFlowIntegrationConfig,
        *,
        transport: httpx.AsyncBaseTransport | None = None,
    ) -> None:
        self._config = config
        self._base_url = str(config.base_url or "").rstrip("/")
        self._transport = transport

    @property
    def config_snapshot(self) -> dict[str, Any]:
        data = asdict(self._config)
        if data.get("api_key"):
            data["api_key"] = "***"
        return data

    async def health(self) -> dict[str, Any]:
        return await self._request("GET", "/api/v1/system/healthz", unwrap=False)

    async def list_datasets(
        self,
        *,
        page: int = 1,
        page_size: int = 30,
        name: str = "",
        include_parsing_status: bool = True,
    ) -> dict[str, Any]:
        params = {
            "page": max(1, int(page)),
            "page_size": max(1, min(int(page_size), 100)),
            "include_parsing_status": str(bool(include_parsing_status)).lower(),
        }
        if name:
            params["name"] = name
        return await self._request("GET", "/api/v1/datasets", params=params)

    async def create_dataset(
        self,
        *,
        name: str,
        chunk_method: str = "naive",
        description: str = "",
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "name": name,
            "chunk_method": chunk_method or "naive",
        }
        if description:
            payload["description"] = description
        return await self._request("POST", "/api/v1/datasets", json_body=payload)

    async def retrieve(
        self,
        *,
        question: str,
        dataset_ids: list[str],
        document_ids: list[str] | None = None,
        page: int = 1,
        page_size: int | None = None,
        similarity_threshold: float | None = None,
        vector_similarity_weight: float | None = None,
        top_k: int | None = None,
        keyword: bool | None = None,
        use_kg: bool | None = None,
        metadata_condition: dict[str, Any] | None = None,
        toc_enhance: bool = False,
    ) -> dict[str, Any]:
        clean_datasets = [str(v).strip() for v in dataset_ids if str(v).strip()]
        if not clean_datasets:
            clean_datasets = list(self._config.default_dataset_ids)
        if not clean_datasets:
            raise RAGFlowError("RAGFlow dataset_ids 为空，请传入 dataset_ids 或配置 default_dataset_ids")
        payload = {
            "dataset_ids": clean_datasets,
            "document_ids": document_ids or [],
            "question": question,
            "page": max(1, int(page)),
            "page_size": int(page_size or self._config.default_page_size),
            "similarity_threshold": float(
                similarity_threshold
                if similarity_threshold is not None
                else self._config.default_similarity_threshold
            ),
            "vector_similarity_weight": float(
                vector_similarity_weight
                if vector_similarity_weight is not None
                else self._config.default_vector_similarity_weight
            ),
            "top_k": int(top_k or self._config.default_top_k),
            "keyword": self._config.default_keyword if keyword is None else bool(keyword),
            "use_kg": self._config.default_use_kg if use_kg is None else bool(use_kg),
            "toc_enhance": bool(toc_enhance),
        }
        if metadata_condition:
            payload["metadata_condition"] = metadata_condition
        return await self._request("POST", "/api/v1/retrieval", json_body=payload)

    async def upload_document_bytes(
        self,
        *,
        dataset_id: str,
        display_name: str,
        content: bytes,
        content_type: str | None = None,
        parse: bool = True,
    ) -> dict[str, Any]:
        if not dataset_id.strip():
            raise RAGFlowError("dataset_id 不能为空")
        mime = content_type or mimetypes.guess_type(display_name)[0] or "text/plain"
        data = await self._request(
            "POST",
            f"/api/v1/datasets/{dataset_id}/documents",
            files={"file": (display_name, content, mime)},
        )
        docs = data if isinstance(data, list) else data.get("documents", data)
        document_ids = _extract_document_ids(docs)
        parse_result: dict[str, Any] | None = None
        if parse and document_ids:
            parse_result = await self.parse_documents(
                dataset_id=dataset_id,
                document_ids=document_ids,
            )
        return {
            "documents": docs,
            "document_ids": document_ids,
            "parse_result": parse_result,
        }

    async def parse_documents(
        self,
        *,
        dataset_id: str,
        document_ids: list[str],
    ) -> dict[str, Any]:
        if not document_ids:
            return {"document_ids": [], "skipped": True}
        return await self._request(
            "POST",
            f"/api/v1/datasets/{dataset_id}/chunks",
            json_body={"document_ids": document_ids},
        )

    async def _request(
        self,
        method: str,
        path: str,
        *,
        params: dict[str, Any] | None = None,
        json_body: Any = None,
        files: dict[str, tuple[str, bytes, str]] | None = None,
        unwrap: bool = True,
    ) -> Any:
        if not self._config.enabled:
            raise RAGFlowError("RAGFlow 未启用")
        if not self._base_url:
            raise RAGFlowError("RAGFlow base_url 未配置")
        if not self._config.api_key:
            raise RAGFlowError("RAGFlow api_key 未配置")
        url = self._base_url + path
        headers = {"Authorization": f"Bearer {self._config.api_key}"}
        if files is None:
            headers["Content-Type"] = "application/json"
        # Tests inject a MockTransport; do not route those requests through a
        # configured proxy because httpx treats proxy transport separately.
        proxy = None if self._transport is not None else self._config.proxy_url.strip() or None
        async with httpx.AsyncClient(
            proxy=proxy,
            timeout=self._config.request_timeout_seconds,
            transport=self._transport,
            trust_env=True,
        ) as client:
            response = await client.request(
                method.upper(),
                url,
                headers=headers,
                params=params,
                json=json_body if files is None else None,
                files=files,
            )
        if response.status_code >= 400:
            raise RAGFlowError(
                f"RAGFlow HTTP {response.status_code}: {response.text[:500]}"
            )
        try:
            payload = response.json()
        except json.JSONDecodeError:
            return {"status_code": response.status_code, "text": response.text}
        if not unwrap:
            return payload
        code = payload.get("code") if isinstance(payload, dict) else None
        if code not in (None, 0):
            raise RAGFlowError(str(payload.get("message") or payload))
        if isinstance(payload, dict) and "data" in payload:
            return payload["data"]
        return payload


def _extract_document_ids(value: Any) -> list[str]:
    docs = value if isinstance(value, list) else [value]
    ids: list[str] = []
    for item in docs:
        if not isinstance(item, dict):
            continue
        doc_id = str(item.get("id") or item.get("document_id") or "").strip()
        if doc_id:
            ids.append(doc_id)
    return ids
