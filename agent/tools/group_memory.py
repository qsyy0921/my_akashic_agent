from __future__ import annotations

import json
from typing import Any

from agent.tools.base import Tool
from group_memory.service import GroupMemoryService


class IngestGroupMemoryTool(Tool):
    name = "ingest_group_memory"
    description = (
        "处理已观察到的QQ群消息，抽取游戏攻略、争议和版本线索，写入群 memory。"
        "适合在查询前手动刷新某个 observe-only QQ 群的攻略库。"
    )
    parameters = {
        "type": "object",
        "properties": {
            "group_id": {
                "type": "string",
                "description": "QQ群号，例如 284331268",
            },
        },
        "required": ["group_id"],
    }

    def __init__(self, service: GroupMemoryService) -> None:
        self._service = service

    async def execute(self, group_id: str, **_: Any) -> str:
        stats = self._service.ingest_group(str(group_id).strip())
        return json.dumps(stats.__dict__, ensure_ascii=False)


class RecallGroupMemoryTool(Tool):
    name = "recall_group_memory"
    description = (
        "检索群 memory 中沉淀的游戏攻略。返回结构化攻略、状态、置信度和证据引用。"
        "回答群攻略问题时优先使用本工具，再用 show_group_strategy_evidence 取证据原文。"
    )
    parameters = {
        "type": "object",
        "properties": {
            "query": {"type": "string", "description": "攻略问题或关键词"},
            "group_id": {
                "type": "string",
                "description": "QQ群号。为空时跨群检索，但推荐指定。",
            },
            "status": {
                "type": "string",
                "enum": ["", "pending", "validated", "disputed", "needs_review", "outdated"],
                "description": "可选状态过滤",
            },
            "limit": {
                "type": "integer",
                "minimum": 1,
                "maximum": 20,
                "default": 5,
                "description": "最多返回条数",
            },
        },
        "required": ["query"],
    }

    def __init__(self, service: GroupMemoryService) -> None:
        self._service = service

    async def execute(
        self,
        query: str,
        group_id: str = "",
        status: str = "",
        limit: int = 5,
        **_: Any,
    ) -> str:
        group = str(group_id or "").strip() or None
        if group:
            self._service.ingest_group(group)
        result = self._service.retrieve(
            str(query or ""),
            group_id=group,
            status=str(status or "").strip() or None,
            limit=limit,
            evidence_limit=3,
        )
        hits = result.hits
        return json.dumps(
            {
                "count": len(hits),
                "trace": {
                    "query": result.trace.query,
                    "group_id": result.trace.group_id,
                    "status": result.trace.status,
                    "channels": result.trace.channels,
                    "fused_count": result.trace.fused_count,
                },
                "items": [
                    {
                        "id": h.id,
                        "group_id": h.group_id,
                        "game": h.game,
                        "topic": h.topic,
                        "category": h.category,
                        "summary": h.summary,
                        "status": h.status,
                        "confidence": round(h.confidence, 3),
                        "reinforcement": h.reinforcement,
                        "version_hint": h.version_hint,
                        "score": round(h.score, 3),
                        "evidence_count": h.evidence_count,
                        "retrieval_channels": h.channels,
                        "evidence": h.evidence,
                    }
                    for h in hits
                ],
            },
            ensure_ascii=False,
        )


class ListGroupStrategiesTool(Tool):
    name = "list_group_strategies"
    description = "列出某个QQ群当前沉淀的攻略 memory，可按状态过滤。"
    parameters = {
        "type": "object",
        "properties": {
            "group_id": {"type": "string", "description": "QQ群号，可选"},
            "status": {
                "type": "string",
                "enum": ["", "pending", "validated", "disputed", "needs_review", "outdated"],
                "description": "状态过滤，可选",
            },
            "limit": {
                "type": "integer",
                "minimum": 1,
                "maximum": 100,
                "default": 20,
            },
        },
    }

    def __init__(self, service: GroupMemoryService) -> None:
        self._service = service

    async def execute(
        self,
        group_id: str = "",
        status: str = "",
        limit: int = 20,
        **_: Any,
    ) -> str:
        rows = self._service.list_strategies(
            group_id=str(group_id or "").strip() or None,
            status=str(status or "").strip() or None,
            limit=limit,
        )
        return json.dumps({"count": len(rows), "items": rows}, ensure_ascii=False)


class ShowGroupStrategyEvidenceTool(Tool):
    name = "show_group_strategy_evidence"
    description = (
        "读取某条群攻略 memory 的证据消息。最终回答中引用攻略结论前，应使用本工具确认来源。"
    )
    parameters = {
        "type": "object",
        "properties": {
            "strategy_id": {"type": "string", "description": "攻略 memory id"},
            "limit": {
                "type": "integer",
                "minimum": 1,
                "maximum": 50,
                "default": 10,
            },
        },
        "required": ["strategy_id"],
    }

    def __init__(self, service: GroupMemoryService) -> None:
        self._service = service

    async def execute(self, strategy_id: str, limit: int = 10, **_: Any) -> str:
        evidence = self._service.list_evidence(str(strategy_id).strip(), limit=limit)
        return json.dumps(
            {"count": len(evidence), "strategy_id": strategy_id, "evidence": evidence},
            ensure_ascii=False,
        )
