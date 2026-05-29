from __future__ import annotations

"""兼容层：agent runtime 命名的别名导出。"""

from .agent_gateway import (
    AgentGatewayClient as AgentRuntimeClient,
    AgentGatewayError as AgentRuntimeError,
    AgentGatewayNoJob as AgentRuntimeNoJob,
)

__all__ = ["AgentRuntimeClient", "AgentRuntimeError", "AgentRuntimeNoJob"]

