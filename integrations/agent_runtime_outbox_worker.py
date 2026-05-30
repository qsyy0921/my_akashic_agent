from __future__ import annotations

"""兼容层：将 outbox worker 的新命名映射到现有 agent_gateway 实现。"""

from .agent_gateway_outbox_worker import (
    AgentGatewayOutboxWorker as AgentRuntimeOutboxWorker,
)

__all__ = ["AgentRuntimeOutboxWorker"]
