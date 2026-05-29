from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from agent.config_models import RAGFlowIntegrationConfig
from agent.tools.ragflow import (
    RAGFlowIndexQQGroupTool,
    RAGFlowListDatasetsTool,
    RAGFlowRetrieveTool,
    RAGFlowUploadTextTool,
)
from integrations.ragflow import RAGFlowClient
from session.store import SessionStore


def _client(handler) -> RAGFlowClient:
    return RAGFlowClient(
        RAGFlowIntegrationConfig(
            enabled=True,
            base_url="http://ragflow.local",
            api_key="ragflow-key",
            proxy_url="http://127.0.0.1:7897",
            default_dataset_ids=["default-ds"],
        ),
        transport=httpx.MockTransport(handler),
    )


def _json_response(data: Any) -> httpx.Response:
    return httpx.Response(200, json={"code": 0, "data": data})


@pytest.mark.asyncio
async def test_ragflow_retrieve_uses_default_dataset_and_auth_header():
    seen: dict[str, Any] = {}

    async def handler(request: httpx.Request) -> httpx.Response:
        seen["path"] = request.url.path
        seen["auth"] = request.headers.get("Authorization")
        seen["body"] = json.loads(request.content.decode())
        return _json_response(
            {
                "chunks": [
                    {
                        "id": "chunk-1",
                        "content": "Boss A P2 先清小怪",
                        "similarity": 0.88,
                    }
                ]
            }
        )

    tool = RAGFlowRetrieveTool(_client(handler))
    payload = json.loads(await tool.execute(question="Boss A 怎么打"))

    assert payload["ok"] is True
    assert seen["path"] == "/api/v1/retrieval"
    assert seen["auth"] == "Bearer ragflow-key"
    assert seen["body"]["dataset_ids"] == ["default-ds"]
    assert seen["body"]["keyword"] is True
    assert payload["data"]["chunks"][0]["id"] == "chunk-1"


@pytest.mark.asyncio
async def test_ragflow_list_datasets_calls_api():
    async def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        assert request.url.path == "/api/v1/datasets"
        assert request.url.params["include_parsing_status"] == "true"
        return _json_response([{"id": "ds1", "name": "qq"}])

    payload = json.loads(await RAGFlowListDatasetsTool(_client(handler)).execute())

    assert payload["ok"] is True
    assert payload["data"][0]["id"] == "ds1"


@pytest.mark.asyncio
async def test_ragflow_upload_text_uploads_and_triggers_parse():
    paths: list[str] = []

    async def handler(request: httpx.Request) -> httpx.Response:
        paths.append(request.url.path)
        if request.url.path.endswith("/documents"):
            assert request.method == "POST"
            return _json_response([{"id": "doc1", "name": "note.txt"}])
        if request.url.path.endswith("/chunks"):
            body = json.loads(request.content.decode())
            assert body == {"document_ids": ["doc1"]}
            return _json_response({"started": True})
        return httpx.Response(404)

    payload = json.loads(
        await RAGFlowUploadTextTool(_client(handler)).execute(
            dataset_id="ds1",
            text="hello ragflow",
            display_name="note.txt",
        )
    )

    assert payload["ok"] is True
    assert paths == [
        "/api/v1/datasets/ds1/documents",
        "/api/v1/datasets/ds1/chunks",
    ]
    assert payload["data"]["document_ids"] == ["doc1"]


@pytest.mark.asyncio
async def test_ragflow_index_qq_group_exports_observed_messages(tmp_path):
    store = SessionStore(tmp_path / "sessions.db")
    store.upsert_session(
        "qq:gqq:284331268",
        created_at="2026-01-01T00:00:00+00:00",
        updated_at="2026-01-01T00:00:00+00:00",
        last_consolidated=0,
        metadata={"group_id": "284331268", "observe_only": True},
    )
    store.insert_message(
        "qq:gqq:284331268",
        role="user",
        content="[QQ群 284331268 | user_a] Boss A P2 先清小怪",
        ts="2026-01-01T00:00:00+00:00",
        seq=0,
        extra={"sender_id": "user_a"},
    )

    async def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path.endswith("/documents"):
            return _json_response([{"id": "doc1"}])
        if request.url.path.endswith("/chunks"):
            return _json_response({"started": True})
        return httpx.Response(404)

    payload = json.loads(
        await RAGFlowIndexQQGroupTool(_client(handler), store).execute(
            dataset_id="ds1",
            group_id="284331268",
        )
    )

    assert payload["ok"] is True
    assert payload["message_count"] == 1
    assert payload["data"]["document_ids"] == ["doc1"]
