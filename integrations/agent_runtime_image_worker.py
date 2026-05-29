from __future__ import annotations

"""兼容层：将 image worker 的新命名映射到现有 agent_gateway 实现。"""

from .agent_gateway_image_worker import AgentGatewayImageWorker as AgentRuntimeImageWorker

__all__ = ["AgentRuntimeImageWorker"]

