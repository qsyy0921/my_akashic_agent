from __future__ import annotations

import json
from datetime import datetime
from pathlib import Path
from typing import Any

from agent.tools.base import Tool
from group_memory.sources import GroupMessageSource, SessionGroupMessageSource
from integrations.ragflow import RAGFlowClient, RAGFlowError
from session.store import SessionStore


def _json(data: Any) -> str:
    return json.dumps(data, ensure_ascii=False)


def _json_error(message: str) -> str:
    return _json({"ok": False, "error": message})


class RAGFlowListDatasetsTool(Tool):
    name = "ragflow_list_datasets"
    description = "列出 RAGFlow datasets，用于选择不同来源数据的知识库。"
    parameters = {
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "按名称过滤，可选"},
            "page": {"type": "integer", "minimum": 1, "default": 1},
            "page_size": {"type": "integer", "minimum": 1, "maximum": 100, "default": 30},
        },
    }

    def __init__(self, client: RAGFlowClient) -> None:
        self._client = client

    async def execute(self, name: str = "", page: int = 1, page_size: int = 30, **_: Any) -> str:
        try:
            data = await self._client.list_datasets(name=name, page=page, page_size=page_size)
            return _json({"ok": True, "data": data})
        except Exception as exc:
            return _json_error(str(exc))


class RAGFlowRetrieveTool(Tool):
    name = "ragflow_retrieve"
    description = (
        "通过 RAGFlow 对多来源知识库执行高级 RAG 检索。支持 keyword、GraphRAG(use_kg)、metadata filter。"
    )
    parameters = {
        "type": "object",
        "properties": {
            "question": {"type": "string", "description": "检索问题"},
            "dataset_ids": {
                "type": "array",
                "items": {"type": "string"},
                "description": "RAGFlow dataset id 列表。为空则使用默认配置。",
            },
            "page_size": {"type": "integer", "minimum": 1, "maximum": 50, "default": 8},
            "keyword": {"type": "boolean", "description": "是否启用关键词检索"},
            "use_kg": {"type": "boolean", "description": "是否启用 RAGFlow KG/GraphRAG 检索"},
            "metadata_condition": {
                "type": "object",
                "description": "RAGFlow metadata_condition，可选",
            },
        },
        "required": ["question"],
    }

    def __init__(self, client: RAGFlowClient) -> None:
        self._client = client

    async def execute(
        self,
        question: str,
        dataset_ids: list[str] | None = None,
        page_size: int | None = None,
        keyword: bool | None = None,
        use_kg: bool | None = None,
        metadata_condition: dict[str, Any] | None = None,
        **_: Any,
    ) -> str:
        try:
            data = await self._client.retrieve(
                question=question,
                dataset_ids=dataset_ids or [],
                page_size=page_size,
                keyword=keyword,
                use_kg=use_kg,
                metadata_condition=metadata_condition,
            )
            return _json({"ok": True, "data": data})
        except Exception as exc:
            return _json_error(str(exc))


class RAGFlowUploadTextTool(Tool):
    name = "ragflow_upload_text"
    description = "把一段文本作为 txt 文档上传到 RAGFlow dataset，并可自动触发解析索引。"
    parameters = {
        "type": "object",
        "properties": {
            "dataset_id": {"type": "string", "description": "RAGFlow dataset id"},
            "text": {"type": "string", "description": "要上传的文本内容"},
            "display_name": {"type": "string", "description": "文档显示名称，默认自动生成"},
            "parse": {"type": "boolean", "default": True, "description": "上传后是否触发解析"},
        },
        "required": ["dataset_id", "text"],
    }

    def __init__(self, client: RAGFlowClient) -> None:
        self._client = client

    async def execute(
        self,
        dataset_id: str,
        text: str,
        display_name: str = "",
        parse: bool = True,
        **_: Any,
    ) -> str:
        try:
            name = display_name.strip() or f"akashic_text_{datetime.now().strftime('%Y%m%d_%H%M%S')}.txt"
            data = await self._client.upload_document_bytes(
                dataset_id=dataset_id,
                display_name=name,
                content=str(text or "").encode("utf-8"),
                content_type="text/plain",
                parse=parse,
            )
            return _json({"ok": True, "data": data})
        except Exception as exc:
            return _json_error(str(exc))


class RAGFlowUploadFileTool(Tool):
    name = "ragflow_upload_file"
    description = "把本地文件上传到 RAGFlow dataset，并可自动触发解析索引。"
    parameters = {
        "type": "object",
        "properties": {
            "dataset_id": {"type": "string", "description": "RAGFlow dataset id"},
            "path": {"type": "string", "description": "本地文件路径，支持相对 workspace"},
            "display_name": {"type": "string", "description": "文档显示名称，可选"},
            "parse": {"type": "boolean", "default": True, "description": "上传后是否触发解析"},
        },
        "required": ["dataset_id", "path"],
    }

    def __init__(self, client: RAGFlowClient, workspace: Path) -> None:
        self._client = client
        self._workspace = workspace

    async def execute(
        self,
        dataset_id: str,
        path: str,
        display_name: str = "",
        parse: bool = True,
        **_: Any,
    ) -> str:
        try:
            p = Path(path)
            if not p.is_absolute():
                p = self._workspace / p
            if not p.is_file():
                return _json_error(f"文件不存在: {p}")
            data = await self._client.upload_document_bytes(
                dataset_id=dataset_id,
                display_name=display_name.strip() or p.name,
                content=p.read_bytes(),
                parse=parse,
            )
            return _json({"ok": True, "data": data})
        except Exception as exc:
            return _json_error(str(exc))


class RAGFlowIndexQQGroupTool(Tool):
    name = "ragflow_index_qq_group"
    description = (
        "把 Akashic 已观察到的 QQ 群消息导出成文档上传到 RAGFlow，用 RAGFlow 统一处理群聊来源数据。"
    )
    parameters = {
        "type": "object",
        "properties": {
            "dataset_id": {"type": "string", "description": "RAGFlow dataset id"},
            "group_id": {"type": "string", "description": "QQ群号"},
            "max_messages": {"type": "integer", "minimum": 1, "maximum": 5000, "default": 1000},
            "since_seq": {"type": "integer", "minimum": 0, "default": 0},
            "parse": {"type": "boolean", "default": True, "description": "上传后是否触发解析"},
        },
        "required": ["dataset_id", "group_id"],
    }

    def __init__(
        self,
        client: RAGFlowClient,
        session_store: SessionStore,
        *,
        message_source: GroupMessageSource | None = None,
    ) -> None:
        self._client = client
        self._sessions = session_store
        self._source = message_source or SessionGroupMessageSource(session_store)

    async def execute(
        self,
        dataset_id: str,
        group_id: str,
        max_messages: int = 1000,
        since_seq: int = 0,
        parse: bool = True,
        **_: Any,
    ) -> str:
        try:
            session_key = f"qq:gqq:{str(group_id).strip()}"
            limit = max(1, min(int(max_messages), 5000))
            rows = self._source.fetch_new_messages(
                session_key=session_key,
                group_id=str(group_id).strip(),
                after_seq=max(-1, int(since_seq) - 1),
                limit=limit,
            )
            selected = [
                row
                for row in rows
                if int(row.get("seq", -1)) >= int(since_seq) and str(row.get("role") or "") == "user"
            ][:limit]
            if not selected:
                return _json_error(f"没有可索引的群消息: {session_key}")
            lines = [
                f"# QQ Group {group_id}",
                "",
                "每行格式: seq | timestamp | sender | content",
                "",
            ]
            for row in selected:
                lines.append(
                    " | ".join(
                        [
                            str(row.get("seq", "")),
                            str(row.get("timestamp", row.get("ts", ""))),
                            str(row.get("sender_id", "")),
                            str(row.get("content", "")).replace("\n", " "),
                        ]
                    )
                )
            name = (
                f"qq_group_{group_id}_seq{selected[0].get('seq')}"
                f"_{selected[-1].get('seq')}.txt"
            )
            data = await self._client.upload_document_bytes(
                dataset_id=dataset_id,
                display_name=name,
                content="\n".join(lines).encode("utf-8"),
                content_type="text/plain",
                parse=parse,
            )
            return _json(
                {
                    "ok": True,
                    "session_key": session_key,
                    "message_count": len(selected),
                    "display_name": name,
                    "data": data,
                }
            )
        except RAGFlowError as exc:
            return _json_error(str(exc))
        except Exception as exc:
            return _json_error(str(exc))
