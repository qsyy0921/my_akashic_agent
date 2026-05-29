from __future__ import annotations

"""兼容层：将 knowledge worker 的新命名映射到现有 agent_gateway 实现。"""

from .agent_gateway_knowledge_worker import (
    AgentGatewayKnowledgeWorker as AgentRuntimeKnowledgeWorker,
)

__all__ = ["AgentRuntimeKnowledgeWorker"]

