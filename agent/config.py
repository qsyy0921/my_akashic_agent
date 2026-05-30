"""
配置加载模块
从 config.toml 读取配置，支持 ${ENV_VAR} 格式的环境变量插值。
"""

from __future__ import annotations

import os
import re
import sys
import tomllib
import zlib
from pathlib import Path
from zoneinfo import ZoneInfo

from agent.config_models import (
    AgentGatewayIntegrationConfig,
    ChannelsConfig,
    ChatGPTProxyIntegrationConfig,
    Config,
    FeishuWebhookChannelConfig,
    FitbitIntegrationConfig,
    MemoryConfig,
    MemoryEmbeddingConfig,
    PeerAgentConfig,
    QQBotChannelConfig,
    QQBotGroupConfig,
    QQChannelConfig,
    QQGroupConfig,
    RAGFlowIntegrationConfig,
    ShadowGatewayIntegrationConfig,
    ShadowRuntimeIntegrationConfig,
    TelegramChannelConfig,
    WechatWebhookChannelConfig,
    WiringConfig,
)
from proactive_v2.config import ProactiveConfig
from proactive_v2.config_loader import ProactiveConfigError, load_proactive_config

_PRESETS: dict[str, str] = {
    "qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "deepseek": "https://api.deepseek.com/v1",
    "openai": "https://api.openai.com/v1",
}

# CLI channel 默认 Unix socket 路径
DEFAULT_SOCKET = "127.0.0.1:8765" if os.name == "nt" else "/tmp/akashic.sock"


def _normalize_cli_socket_endpoint(value: str | None) -> str:
    text = str(value or "").strip()
    if not text:
        return DEFAULT_SOCKET
    if os.name != "nt":
        return text
    host, sep, port = text.rpartition(":")
    if sep and host:
        try:
            int(port)
            return text
        except ValueError:
            pass
    port_seed = zlib.crc32(text.encode("utf-8")) % 20000
    return f"127.0.0.1:{20000 + port_seed}"


def _validated_timezone(tz_name: str, *, enabled: bool) -> str:
    """仅当 anyaction_enabled=True 时校验时区合法性，无效则启动时 fail-fast。"""
    if not enabled:
        return tz_name
    try:
        ZoneInfo(tz_name)
        return tz_name
    except Exception:
        raise ValueError(
            f"proactive.anyaction_timezone 无效: {tz_name!r}，"
            "请使用 IANA 格式，如 'Asia/Shanghai'"
        )


def load_config(path: str | Path = "config.toml") -> Config:
    data = _load_config_data(path)

    llm = _as_dict(data.get("llm"))
    llm_main = _as_dict(llm.get("main"))
    llm_fast = _as_dict(llm.get("fast"))
    llm_agent = _as_dict(llm.get("agent"))
    llm_vl = _as_dict(llm.get("vl"))
    agent_cfg = _as_dict(data.get("agent"))
    agent_context = _as_dict(agent_cfg.get("context"))
    agent_tools = _as_dict(agent_cfg.get("tools"))
    agent_maintenance = _as_dict(agent_cfg.get("maintenance"))
    provider = str(llm.get("provider") or data["provider"])
    channels = _load_channels_config(data)
    proactive = _load_proactive_config(data)
    memory = _load_memory_config(data)
    peer_agents = _load_peer_agents_config(data)
    fitbit = _load_fitbit_config(data)
    chatgpt_proxy = _load_chatgpt_proxy_config(data)
    ragflow = _load_ragflow_config(data)
    shadow_gateway = _load_shadow_gateway_config(data)
    agent_gateway = _load_agent_gateway_config(data)
    wiring = _load_wiring_config(data)

    return Config(
        provider=provider,
        model=str(llm_main.get("model") or data["model"]),
        api_key=_resolve(str(llm_main.get("api_key") or data.get("api_key", ""))),
        system_prompt=str(
            agent_cfg.get("system_prompt")
            or data.get("system_prompt", "You are a helpful assistant.")
        ),
        max_tokens=int(agent_cfg.get("max_tokens", data.get("max_tokens", 8192))),
        max_iterations=int(
            agent_cfg.get("max_iterations", data.get("max_iterations", 10))
        ),
        memory_window=int(
            agent_context.get("memory_window", data.get("memory_window", 40))
        ),
        base_url=str(
            llm_main.get("base_url")
            or data.get("base_url")
            or _PRESETS.get(provider)
            or ""
        ),
        extra_body=_load_extra_body(data),
        channels=channels,
        proactive=proactive,
        memory_optimizer_enabled=bool(
            agent_maintenance.get(
                "memory_optimizer_enabled",
                data.get("memory_optimizer_enabled", True),
            )
        ),
        memory_optimizer_interval_seconds=int(
            agent_maintenance.get(
                "memory_optimizer_interval_seconds",
                data.get("memory_optimizer_interval_seconds", 64800),
            )
        ),
        light_model=str(llm_fast.get("model") or data.get("light_model", "")),
        light_api_key=_resolve(
            str(llm_fast.get("api_key") or data.get("light_api_key", ""))
        ),
        light_base_url=str(llm_fast.get("base_url") or data.get("light_base_url", "")),
        agent_model=str(llm_agent.get("model") or data.get("agent_model", "")),
        agent_api_key=_resolve(
            str(llm_agent.get("api_key") or data.get("agent_api_key", ""))
        ),
        agent_base_url=str(llm_agent.get("base_url") or data.get("agent_base_url", "")),
        memory=memory,
        fitbit=fitbit,
        chatgpt_proxy=chatgpt_proxy,
        ragflow=ragflow,
        shadow_gateway=shadow_gateway,
        agent_gateway=agent_gateway,
        tool_search_enabled=bool(
            agent_tools.get("search_enabled", data.get("tool_search_enabled", False))
        ),
        spawn_enabled=bool(
            agent_tools.get("spawn_enabled", data.get("spawn_enabled", True))
        ),
        dev_mode=bool(
            agent_cfg.get(
                "dev_mode",
                agent_cfg.get(
                    "dev_model",
                    data.get("dev_mode", data.get("dev_model", False)),
                ),
            )
        ),
        multimodal=bool(llm_main.get("multimodal", True)),
        vl_model=str(llm_vl.get("model") or data.get("vl_model", "")),
        vl_api_key=_resolve(str(llm_vl.get("api_key") or data.get("vl_api_key", ""))),
        vl_base_url=str(llm_vl.get("base_url") or data.get("vl_base_url", "")),
        peer_agents=peer_agents,
        wiring=wiring,
    )


def _load_channels_config(data: dict) -> ChannelsConfig:
    channels_data = data.get("channels", {})

    telegram = None
    if tg := channels_data.get("telegram"):
        token = _resolve_optional_string(tg.get("token", ""))
        if bool(tg.get("enabled", True)) and token:
            telegram = TelegramChannelConfig(
                token=token,
                allow_from=_resolve_string_list(
                    tg.get("allow_from", tg.get("allowFrom", []))
                ),
                channel_name=str(tg.get("channel_name", "telegram")),
            )

    feishu = None
    if fs := channels_data.get("feishu"):
        webhook_url = _resolve_optional_string(
            fs.get("webhook_url", fs.get("webhookUrl", ""))
        )
        if bool(fs.get("enabled", True)) and webhook_url:
            feishu = FeishuWebhookChannelConfig(
                webhook_url=webhook_url,
                secret=_resolve_optional_string(fs.get("secret", "")),
                channel_name=str(fs.get("channel_name", "feishu")),
            )

    wechat = None
    if wc := channels_data.get("wechat"):
        webhook_url = _resolve_optional_string(
            wc.get("webhook_url", wc.get("webhookUrl", ""))
        )
        if bool(wc.get("enabled", True)) and webhook_url:
            wechat = WechatWebhookChannelConfig(
                webhook_url=webhook_url,
                mentioned_list=_resolve_string_list(
                    wc.get("mentioned_list", wc.get("mentionedList", []))
                ),
                mentioned_mobile_list=_resolve_string_list(
                    wc.get("mentioned_mobile_list", wc.get("mentionedMobileList", []))
                ),
                channel_name=str(wc.get("channel_name", "wechat")),
            )

    qq = None
    qq_accounts: list[QQChannelConfig] = []
    if qq_data := channels_data.get("qq"):
        if bool(qq_data.get("enabled", True)):
            qq = _load_qq_channel_config(qq_data, default_channel_name="qq")
            for index, account_data in enumerate(
                qq_data.get("accounts", []) or [], start=2
            ):
                account = _load_qq_channel_config(
                    _as_dict(account_data),
                    default_channel_name=f"qq_{index}",
                )
                if account is not None:
                    qq_accounts.append(account)

    qqbot = None
    if qqbot_data := channels_data.get("qqbot"):
        app_id = _resolve_optional_string(
            qqbot_data.get("app_id", qqbot_data.get("appId", ""))
        )
        client_secret = _resolve_optional_string(
            qqbot_data.get("client_secret", qqbot_data.get("clientSecret", ""))
        )
        if bool(qqbot_data.get("enabled", True)) and app_id and client_secret:
            groups = []
            for g in qqbot_data.get("groups", []):
                group_openid = _resolve_optional_string(
                    g["group_openid"] if "group_openid" in g else g["groupOpenid"]
                )
                if not group_openid:
                    continue
                groups.append(
                    QQBotGroupConfig(
                        group_openid=group_openid,
                        allow_from=_resolve_string_list(
                            g.get("allow_from", g.get("allowFrom", []))
                        ),
                        require_at=g.get("require_at", g.get("requireAt", True)),
                        allow_proactive=bool(
                            g.get("allow_proactive", g.get("allowProactive", False))
                        ),
                    )
                )
            qqbot = QQBotChannelConfig(
                app_id=app_id,
                client_secret=client_secret,
                allow_from=_resolve_string_list(
                    qqbot_data.get("allow_from", qqbot_data.get("allowFrom", []))
                ),
                groups=groups,
            )

    socket_value = channels_data.get("socket") or channels_data.get("cli", {}).get(
        "socket", DEFAULT_SOCKET
    )
    channels = ChannelsConfig(
        telegram=telegram,
        feishu=feishu,
        wechat=wechat,
        qq=qq,
        qq_accounts=qq_accounts,
        qqbot=qqbot,
        socket=_normalize_cli_socket_endpoint(socket_value),
    )
    channels.socket = _normalize_cli_socket_endpoint(channels.socket)
    return channels


def _load_qq_groups(groups_data: list[dict]) -> list[QQGroupConfig]:
    groups: list[QQGroupConfig] = []
    for g in groups_data:
        group_id = _resolve_optional_string(
            g["group_id"] if "group_id" in g else g.get("groupId", "")
        )
        if not group_id:
            continue
        groups.append(
            QQGroupConfig(
                group_id=group_id,
                allow_from=_resolve_string_list(
                    g.get("allow_from", g.get("allowFrom", []))
                ),
                require_at=g.get("require_at", g.get("requireAt", True)),
                observe_only=bool(g.get("observe_only", g.get("observeOnly", False))),
            )
        )
    return groups


def _load_qq_channel_config(
    qq_data: dict,
    *,
    default_channel_name: str,
) -> QQChannelConfig | None:
    bot_uin = _resolve_optional_string(
        qq_data.get("bot_uin", qq_data.get("botUin", ""))
    )
    if not bot_uin:
        return None
    return QQChannelConfig(
        bot_uin=bot_uin,
        allow_from=_resolve_string_list(
            qq_data.get("allow_from", qq_data.get("allowFrom", []))
        ),
        bot_peer_ids=_resolve_string_list(
            qq_data.get("bot_peer_ids", qq_data.get("botPeerIds", []))
        ),
        peer_trigger_prefixes=_resolve_string_list(
            qq_data.get(
                "peer_trigger_prefixes",
                qq_data.get("peerTriggerPrefixes", []),
            )
        ),
        groups=_load_qq_groups(qq_data.get("groups", []) or []),
        websocket_open_timeout_seconds=float(
            qq_data.get("websocket_open_timeout_seconds", 5.0)
        ),
        channel_name=str(qq_data.get("channel_name", default_channel_name)),
        websocket_uri=str(
            qq_data.get("websocket_uri", qq_data.get("websocketUri", ""))
        ),
        websocket_token=str(
            qq_data.get("websocket_token", qq_data.get("websocketToken", "NcatBot"))
            or "NcatBot"
        ),
    )


def _load_proactive_config(data: dict) -> ProactiveConfig:
    proactive = ProactiveConfig()
    if p := data.get("proactive"):
        try:
            proactive = load_proactive_config(p)
        except ProactiveConfigError as e:
            print(f"❌ Proactive 配置错误: {e}", file=sys.stderr)
            sys.exit(1)
    return proactive


def _load_memory_config(data: dict) -> MemoryConfig:
    memory = _as_dict(data.get("memory"))
    embedding = _as_dict(memory.get("embedding"))
    return MemoryConfig(
        enabled=bool(memory.get("enabled", False)),
        engine=str(memory.get("engine", "") or ""),
        embedding=MemoryEmbeddingConfig(
            model=str(embedding.get("model", "text-embedding-v3")),
            api_key=_resolve(str(embedding.get("api_key", ""))),
            base_url=str(embedding.get("base_url", "")),
        ),
    )


def _load_peer_agents_config(data: dict) -> list[PeerAgentConfig]:
    integrations = _as_dict(data.get("integrations"))
    peer_agents = integrations.get("peer_agents", data.get("peer_agents", []))
    return [
        PeerAgentConfig(
            name=pa["name"],
            base_url=pa["base_url"],
            launcher=pa["launcher"],
            cwd=pa.get("cwd"),
            description=pa.get("description", ""),
            health_path=pa.get("health_path", "/health"),
            startup_timeout_s=int(pa.get("startup_timeout_s", 30)),
            shutdown_timeout_s=int(pa.get("shutdown_timeout_s", 10)),
        )
        for pa in peer_agents
    ]


def _load_fitbit_config(data: dict) -> FitbitIntegrationConfig:
    integrations = _as_dict(data.get("integrations"))
    fitbit = _as_dict(integrations.get("fitbit"))
    return FitbitIntegrationConfig(
        enabled=bool(fitbit.get("enabled", False)),
    )


def _load_chatgpt_proxy_config(data: dict) -> ChatGPTProxyIntegrationConfig:
    integrations = _as_dict(data.get("integrations"))
    proxy = _as_dict(integrations.get("chatgpt_proxy"))
    return ChatGPTProxyIntegrationConfig(
        enabled=bool(proxy.get("enabled", False)),
        base_url=_resolve_optional_string(proxy.get("base_url", "")),
        api_key=_resolve_optional_string(proxy.get("api_key", "")),
        image_model=str(proxy.get("image_model", "gpt-image-1") or "gpt-image-1"),
        image_path=str(
            proxy.get("image_path", "/images/generations") or "/images/generations"
        ),
        response_format=str(proxy.get("response_format", "b64_json") or ""),
        output_dir=str(
            proxy.get("output_dir", "generated_images") or "generated_images"
        ),
    )


def _load_ragflow_config(data: dict) -> RAGFlowIntegrationConfig:
    integrations = _as_dict(data.get("integrations"))
    ragflow = _as_dict(integrations.get("ragflow"))
    return RAGFlowIntegrationConfig(
        enabled=bool(ragflow.get("enabled", False)),
        base_url=_resolve_optional_string(
            ragflow.get("base_url", "http://127.0.0.1:9380")
        )
        or "http://127.0.0.1:9380",
        api_key=_resolve_optional_string(ragflow.get("api_key", "")),
        proxy_url=_resolve_optional_string(
            ragflow.get("proxy_url", ragflow.get("proxyUrl", ""))
        ),
        default_dataset_ids=_resolve_string_list(
            ragflow.get(
                "default_dataset_ids",
                ragflow.get("defaultDatasetIds", []),
            )
        ),
        default_keyword=bool(ragflow.get("default_keyword", True)),
        default_use_kg=bool(ragflow.get("default_use_kg", False)),
        default_top_k=int(ragflow.get("default_top_k", 1024)),
        default_page_size=int(ragflow.get("default_page_size", 8)),
        default_similarity_threshold=float(
            ragflow.get("default_similarity_threshold", 0.2)
        ),
        default_vector_similarity_weight=float(
            ragflow.get("default_vector_similarity_weight", 0.3)
        ),
        request_timeout_seconds=float(ragflow.get("request_timeout_seconds", 60.0)),
    )


def _load_shadow_gateway_config(data: dict) -> ShadowRuntimeIntegrationConfig:
    integrations = _as_dict(data.get("integrations"))
    # 新命名优先：shadow_runtime；兼容历史配置 shadow_gateway。
    raw = _as_dict(
        integrations.get("shadow_runtime")
        if "shadow_runtime" in integrations
        else integrations.get("shadow_gateway")
    )
    return ShadowGatewayIntegrationConfig(
        enabled=bool(raw.get("enabled", False)),
        endpoint=str(raw.get("endpoint", "http://127.0.0.1:8780/v1/shadow/inbound")),
        log_path=str(raw.get("log_path", "shadow/inbound.jsonl")),
        request_timeout_seconds=float(raw.get("request_timeout_seconds", 2.0)),
        agent_id=str(raw.get("agent_id", "shadow") or "shadow"),
    )


def _load_agent_gateway_config(data: dict) -> AgentGatewayIntegrationConfig:
    integrations = _as_dict(data.get("integrations"))
    # 新命名优先：agent_runtime；兼容历史配置 agent_gateway。
    raw = _as_dict(
        integrations.get("agent_runtime")
        if "agent_runtime" in integrations
        else integrations.get("agent_gateway")
    )
    return AgentGatewayIntegrationConfig(
        enabled=bool(raw.get("enabled", False)),
        base_url=_resolve_optional_string(raw.get("base_url", "http://127.0.0.1:8780"))
        or "http://127.0.0.1:8780",
        request_timeout_seconds=float(raw.get("request_timeout_seconds", 5.0)),
        worker_id=str(
            raw.get("worker_id", "akashic-python-worker") or "akashic-python-worker"
        ),
        lease_ttl_seconds=int(raw.get("lease_ttl_seconds", 300)),
        poll_interval_seconds=float(raw.get("poll_interval_seconds", 2.0)),
        knowledge_job_interval_seconds=float(
            raw.get("knowledge_job_interval_seconds", 60.0)
        ),
        outbox_worker_enabled=bool(raw.get("outbox_worker_enabled", False)),
    )


def _load_wiring_config(data: dict) -> WiringConfig:
    agent_cfg = _as_dict(data.get("agent"))
    raw = _as_dict(agent_cfg.get("wiring")) or data.get("wiring", {}) or {}
    toolsets = raw.get(
        "toolsets",
        ["meta_common", "spawn", "schedule", "mcp"],
    )
    if not isinstance(toolsets, list):
        toolsets = ["meta_common", "spawn", "schedule", "mcp"]
    return WiringConfig(
        context=str(raw.get("context", "default") or "default"),
        memory=str(raw.get("memory", "default") or "default"),
        toolsets=[str(name) for name in toolsets if str(name).strip()],
    )


def _load_extra_body(data: dict) -> dict:
    llm = _as_dict(data.get("llm"))
    llm_main = _as_dict(llm.get("main"))
    extra_body = dict(data.get("extra_body", {}))
    thinking = llm_main.get("thinking")
    if isinstance(thinking, dict):
        extra_body["thinking"] = thinking
    if "enable_thinking" in llm_main:
        extra_body["enable_thinking"] = bool(llm_main.get("enable_thinking"))
    if "reasoning_effort" in llm_main:
        effort = str(llm_main.get("reasoning_effort") or "").strip()
        if effort:
            extra_body["reasoning_effort"] = effort
    return extra_body


def _as_dict(value: object) -> dict:
    return value if isinstance(value, dict) else {}


def _resolve(value: str) -> str:
    resolved = re.sub(
        r"\$\{(\w+)\}", lambda m: os.environ.get(m.group(1), m.group(0)), value
    )
    # 若仍是未展开的占位符，尝试从 workspace/memory/<VAR_NAME> 文件读取
    m = re.fullmatch(r"\$\{(\w+)\}", resolved)
    if m:
        key_file = Path.home() / ".akashic" / "workspace" / "memory" / m.group(1)
        if key_file.exists():
            resolved = key_file.read_text(encoding="utf-8").strip()
    return resolved


def _normalize_optional_config_text(value: str) -> str:
    text = str(value or "").strip()
    if not text:
        return ""
    if re.fullmatch(r"\$\{(\w+)\}", text):
        return ""
    return text


def _resolve_optional_string(value: object) -> str:
    return _normalize_optional_config_text(_resolve(str(value or "")))


def _resolve_string_list(values: object) -> list[str]:
    raw_values = values if isinstance(values, list) else [values]
    resolved_values: list[str] = []
    for value in raw_values:
        text = _normalize_optional_config_text(_resolve(str(value or "")))
        if not text:
            continue
        resolved_values.extend(part.strip() for part in text.split(",") if part.strip())
    return resolved_values


def _load_config_data(path: str | Path) -> dict:
    path = Path(path)
    if path.suffix.lower() != ".toml":
        raise ValueError(f"主配置仅支持 TOML: {path.suffix}")
    return tomllib.loads(path.read_text(encoding="utf-8"))


__all__ = [
    "ChannelsConfig",
    "ChatGPTProxyIntegrationConfig",
    "Config",
    "DEFAULT_SOCKET",
    "FeishuWebhookChannelConfig",
    "MemoryConfig",
    "MemoryEmbeddingConfig",
    "QQChannelConfig",
    "QQGroupConfig",
    "RAGFlowIntegrationConfig",
    "ShadowGatewayIntegrationConfig",
    "TelegramChannelConfig",
    "WechatWebhookChannelConfig",
    "_validated_timezone",
    "load_config",
]
