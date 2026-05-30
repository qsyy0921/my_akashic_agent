from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

from proactive_v2.config import ProactiveConfig


@dataclass
class TelegramChannelConfig:
    token: str
    allow_from: list[str] = field(default_factory=list)
    channel_name: str = "telegram"


@dataclass
class FeishuWebhookChannelConfig:
    webhook_url: str
    secret: str = ""
    channel_name: str = "feishu"


@dataclass
class WechatWebhookChannelConfig:
    webhook_url: str
    mentioned_list: list[str] = field(default_factory=list)
    mentioned_mobile_list: list[str] = field(default_factory=list)
    channel_name: str = "wechat"


@dataclass
class QQGroupConfig:
    group_id: str
    allow_from: list[str] = field(default_factory=list)
    require_at: bool = True
    observe_only: bool = False


@dataclass
class QQChannelConfig:
    bot_uin: str
    allow_from: list[str] = field(default_factory=list)
    bot_peer_ids: list[str] = field(default_factory=list)
    peer_trigger_prefixes: list[str] = field(default_factory=list)
    groups: list[QQGroupConfig] = field(default_factory=list)
    websocket_open_timeout_seconds: float = 5.0
    channel_name: str = "qq"
    websocket_uri: str = ""
    websocket_token: str = "NcatBot"


@dataclass
class QQBotGroupConfig:
    group_openid: str
    allow_from: list[str] = field(default_factory=list)
    require_at: bool = True
    allow_proactive: bool = False


@dataclass
class QQBotChannelConfig:
    app_id: str
    client_secret: str
    allow_from: list[str] = field(default_factory=list)
    groups: list[QQBotGroupConfig] = field(default_factory=list)


@dataclass
class ChannelsConfig:
    telegram: TelegramChannelConfig | None = None
    feishu: FeishuWebhookChannelConfig | None = None
    wechat: WechatWebhookChannelConfig | None = None
    qq: QQChannelConfig | None = None
    qq_accounts: list[QQChannelConfig] = field(default_factory=list)
    qqbot: QQBotChannelConfig | None = None
    socket: str = "/tmp/akashic.sock"


@dataclass
class MemoryEmbeddingConfig:
    model: str = "text-embedding-v3"
    api_key: str = ""
    base_url: str = ""


@dataclass
class MemoryConfig:
    enabled: bool = False
    engine: str = ""
    embedding: MemoryEmbeddingConfig = field(default_factory=MemoryEmbeddingConfig)


@dataclass
class FitbitIntegrationConfig:
    enabled: bool = False


@dataclass
class ChatGPTProxyIntegrationConfig:
    enabled: bool = False
    base_url: str = ""
    api_key: str = ""
    image_model: str = "gpt-image-1"
    image_path: str = "/images/generations"
    response_format: str = "b64_json"
    output_dir: str = "generated_images"


@dataclass
class RAGFlowIntegrationConfig:
    enabled: bool = False
    base_url: str = "http://127.0.0.1:9380"
    api_key: str = ""
    proxy_url: str = ""
    default_dataset_ids: list[str] = field(default_factory=list)
    default_keyword: bool = True
    default_use_kg: bool = False
    default_top_k: int = 1024
    default_page_size: int = 8
    default_similarity_threshold: float = 0.2
    default_vector_similarity_weight: float = 0.3
    request_timeout_seconds: float = 60.0


@dataclass
class ShadowGatewayIntegrationConfig:
    enabled: bool = False
    endpoint: str = "http://127.0.0.1:8780/v1/shadow/inbound"
    log_path: str = "shadow/inbound.jsonl"
    request_timeout_seconds: float = 2.0
    agent_id: str = "shadow"


# 兼容别名：shadow 旁路现在由 agent-runtime 承接，历史配置名仍可读取。
ShadowRuntimeIntegrationConfig = ShadowGatewayIntegrationConfig


@dataclass
class AgentGatewayIntegrationConfig:
    enabled: bool = False
    base_url: str = "http://127.0.0.1:8780"
    request_timeout_seconds: float = 5.0
    worker_id: str = "akashic-python-worker"
    lease_ttl_seconds: int = 300
    poll_interval_seconds: float = 2.0
    knowledge_job_interval_seconds: float = 60.0
    rag_eval_worker_enabled: bool = False
    outbox_worker_enabled: bool = False
    outbound_channels: list[str] = field(default_factory=lambda: ["telegram"])


# 兼容别名：历史上该配置/集成在代码中叫 AgentGateway，运行时对外文档与目录使用
# agent-runtime，使用别名避免一次性大规模迁移。
AgentRuntimeIntegrationConfig = AgentGatewayIntegrationConfig


@dataclass
class PeerAgentConfig:
    name: str
    base_url: str
    launcher: list[str]  # 拉起命令，如 ["uv", "run", "python", "-m", "app.a2a_server"]
    cwd: str | None = None  # 子进程工作目录，None 表示继承父进程
    description: str = ""  # 工具描述，用于 LLM 路由；服务器在线时会被 AgentCard 覆盖
    health_path: str = "/health"
    startup_timeout_s: int = 30
    shutdown_timeout_s: int = 10


@dataclass
class WiringConfig:
    context: str = "default"
    memory: str = "default"
    toolsets: list[str] = field(
        default_factory=lambda: [
            "meta_common",
            "spawn",
            "schedule",
            "mcp",
        ]
    )


@dataclass
class Config:
    provider: str
    model: str
    api_key: str
    system_prompt: str
    max_tokens: int = 8192
    max_iterations: int = 10
    memory_window: int = 40
    base_url: str | None = None
    extra_body: dict = field(default_factory=dict)
    channels: ChannelsConfig = field(default_factory=ChannelsConfig)
    proactive: ProactiveConfig = field(default_factory=ProactiveConfig)
    memory_optimizer_enabled: bool = True
    memory_optimizer_interval_seconds: int = 64800
    light_model: str = ""
    light_api_key: str = ""
    light_base_url: str = ""
    agent_model: str = ""
    agent_api_key: str = ""
    agent_base_url: str = ""
    memory: MemoryConfig = field(default_factory=MemoryConfig)
    fitbit: FitbitIntegrationConfig = field(default_factory=FitbitIntegrationConfig)
    chatgpt_proxy: ChatGPTProxyIntegrationConfig = field(
        default_factory=ChatGPTProxyIntegrationConfig
    )
    ragflow: RAGFlowIntegrationConfig = field(default_factory=RAGFlowIntegrationConfig)
    shadow_gateway: ShadowGatewayIntegrationConfig = field(
        default_factory=ShadowGatewayIntegrationConfig
    )
    agent_gateway: AgentGatewayIntegrationConfig = field(
        default_factory=AgentGatewayIntegrationConfig
    )
    multimodal: bool = True
    vl_model: str = ""
    vl_api_key: str = ""
    vl_base_url: str = ""
    tool_search_enabled: bool = False
    spawn_enabled: bool = True
    dev_mode: bool = False
    peer_agents: list[PeerAgentConfig] = field(default_factory=list)
    wiring: WiringConfig = field(default_factory=WiringConfig)

    @property
    def shadow_runtime(self) -> ShadowRuntimeIntegrationConfig:
        return self.shadow_gateway

    @property
    def agent_runtime(self) -> AgentRuntimeIntegrationConfig:
        return self.agent_gateway

    @classmethod
    def load(cls, path: str | Path = "config.toml") -> Config:
        from importlib import import_module

        return import_module("agent.config").load_config(path)


__all__ = [
    "ChannelsConfig",
    "Config",
    "FeishuWebhookChannelConfig",
    "FitbitIntegrationConfig",
    "ChatGPTProxyIntegrationConfig",
    "RAGFlowIntegrationConfig",
    "ShadowGatewayIntegrationConfig",
    "ShadowRuntimeIntegrationConfig",
    "AgentGatewayIntegrationConfig",
    "AgentRuntimeIntegrationConfig",
    "MemoryConfig",
    "MemoryEmbeddingConfig",
    "PeerAgentConfig",
    "QQChannelConfig",
    "QQBotChannelConfig",
    "QQBotGroupConfig",
    "QQGroupConfig",
    "TelegramChannelConfig",
    "WechatWebhookChannelConfig",
    "WiringConfig",
]
