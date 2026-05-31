from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import TYPE_CHECKING, Any, Callable, cast

if TYPE_CHECKING:
    from agent.plugins.manager import PluginManager

logger = logging.getLogger(__name__)

from agent.config_models import Config, WiringConfig
from agent.context import ContextBuilder
from agent.peer_agent.process_manager import PeerProcessManager
from agent.peer_agent.poller import PeerAgentPoller
from agent.peer_agent.registry import PeerAgentRegistry
from agent.looping.core import AgentLoop
from agent.looping.ports import (
    AgentLoopConfig,
    AgentLoopDeps,
    LLMConfig,
    LLMServices,
    MemoryConfig,
    MemoryServices,
    SessionServices,
)
from agent.mcp.registry import McpServerRegistry
from agent.provider import LLMProvider
from agent.retrieval.default_pipeline import DefaultMemoryRetrievalPipeline
from agent.scheduler import SchedulerService
from agent.tools.message_push import MessagePushTool
from agent.tools.registry import ToolRegistry
from agent.turns.outbound import BusOutboundPort
from bootstrap.toolsets.mcp import McpToolsetProvider
from bootstrap.toolsets.memory import MemoryToolsetProvider
from bootstrap.toolsets.meta import (
    CommonMetaToolsetProvider,
    SpawnToolsetProvider,
    build_readonly_tools,
)
from bootstrap.toolsets.peer import build_peer_agent_resources
from bootstrap.toolsets.protocol import ToolsetDeps
from bootstrap.toolsets.schedule import (
    SchedulerToolsetProvider,
    build_scheduler,
)
from bootstrap.wiring import (
    wire_turn_lifecycle,
    resolve_context_factory,
    resolve_memory_toolset_provider,
    resolve_toolset_provider,
)
from agent.lifecycle.facade import TurnLifecycle
from bootstrap.providers import build_providers, build_vl_provider
from bus.event_bus import EventBus
from bus.processing import ProcessingState
from bus.queue import MessageBus
from bus.shadow_gateway import (
    ObservedSessionShadowMirror,
    ShadowGatewaySettings,
    build_shadow_gateway_observer,
)
from core.memory.markdown import MemoryLifecycleBindRequest, MarkdownMemoryMaintenance
from core.memory.runtime import MemoryRuntime
from core.net.http import SharedHttpResources
from proactive_v2.presence import PresenceStore
from session.manager import Session, SessionManager


@dataclass
class CoreRuntime:
    config: Config
    http_resources: SharedHttpResources
    loop: AgentLoop
    bus: MessageBus
    event_bus: EventBus
    tools: ToolRegistry
    push_tool: MessagePushTool
    session_manager: SessionManager
    scheduler: SchedulerService
    provider: LLMProvider
    light_provider: LLMProvider | None
    vl_provider: LLMProvider | None
    mcp_registry: McpServerRegistry
    memory_runtime: MemoryRuntime
    presence: PresenceStore
    peer_process_manager: PeerProcessManager | None
    peer_poller: PeerAgentPoller | None
    agent_provider: LLMProvider | None = None
    plugin_manager: "PluginManager | None" = None

    async def start(self) -> None:
        self.mcp_registry.start_connect_all_background()

        if (
            self.peer_poller is not None
            and self.peer_process_manager is not None
            and self.config.peer_agents
        ):
            peer_registry = PeerAgentRegistry(
                process_manager=self.peer_process_manager,
                poller=self.peer_poller,
                requester=self.http_resources.local_service,
            )
            peer_tools = await peer_registry.discover_all(self.config.peer_agents)
            for t in peer_tools:
                self.tools.register(
                    t,
                    always_on=False,
                    risk="external-side-effect",
                )
            self.peer_poller.start()
        if self.plugin_manager is not None:
            await self.plugin_manager.load_all()
            logger.info("插件加载完成: %d 个", self.plugin_manager.loaded_count)
            self.loop.add_before_turn_plugin_modules(
                self.plugin_manager.before_turn_modules,
            )
            self.loop.add_before_reasoning_plugin_modules(
                self.plugin_manager.before_reasoning_modules,
            )
            self.loop.add_prompt_render_plugin_modules(
                self.plugin_manager.prompt_render_modules,
            )
            self.loop.add_before_step_plugin_modules(
                self.plugin_manager.before_step_modules,
            )
            self.loop.add_after_step_plugin_modules(
                self.plugin_manager.after_step_modules,
            )
            self.loop.add_after_reasoning_plugin_modules(
                self.plugin_manager.after_reasoning_modules,
            )
            self.loop.add_after_turn_plugin_modules(
                self.plugin_manager.after_turn_modules,
            )
            if self.plugin_manager.tool_hooks:
                self.loop.add_tool_hooks(self.plugin_manager.tool_hooks)
                spawn_tool = self.tools.get_tool("spawn")
                if spawn_tool is not None and hasattr(spawn_tool, "add_tool_hooks"):
                    spawn_tool.add_tool_hooks(self.plugin_manager.tool_hooks)

    async def inspect_modules(self) -> str:
        if self.plugin_manager is not None:
            await self.plugin_manager.load_all()

        from agent.lifecycle.phase import inspect_phase
        from agent.lifecycle.phases.after_reasoning import (
            default_after_reasoning_modules,
        )
        from agent.lifecycle.phases.after_step import default_after_step_modules
        from agent.lifecycle.phases.after_turn import default_after_turn_modules
        from agent.lifecycle.phases.before_reasoning import (
            default_before_reasoning_modules,
        )
        from agent.lifecycle.phases.before_step import default_before_step_modules
        from agent.lifecycle.phases.before_turn import default_before_turn_modules
        from agent.lifecycle.phases.prompt_render import default_prompt_render_modules

        manager = self.plugin_manager
        before_turn_modules = manager.before_turn_modules if manager is not None else []
        before_reasoning_modules = (
            manager.before_reasoning_modules if manager is not None else []
        )
        prompt_render_modules = manager.prompt_render_modules if manager is not None else []
        before_step_modules = manager.before_step_modules if manager is not None else []
        after_step_modules = manager.after_step_modules if manager is not None else []
        after_reasoning_modules = (
            manager.after_reasoning_modules if manager is not None else []
        )
        after_turn_modules = manager.after_turn_modules if manager is not None else []

        agent_core = cast(Any, getattr(self.loop, "_agent_core"))
        pipeline = agent_core.pipeline
        reasoner = getattr(self.loop, "_reasoner", None)
        context = getattr(reasoner, "_context", None)

        phases = [
            (
                "before_turn",
                default_before_turn_modules(
                    self.event_bus,
                    self.session_manager,
                    cast(Any, getattr(pipeline, "_context_store", None)),
                    plugin_modules=cast(Any, before_turn_modules),
                ),
            ),
            (
                "before_reasoning",
                default_before_reasoning_modules(
                    self.event_bus,
                    self.tools,
                    self.session_manager,
                    cast(Any, context),
                    plugin_modules=cast(Any, before_reasoning_modules),
                ),
            ),
            (
                "prompt_render",
                default_prompt_render_modules(
                    self.event_bus,
                    cast(Any, context),
                    plugin_modules=cast(Any, prompt_render_modules),
                ),
            ),
            (
                "before_step",
                default_before_step_modules(
                    self.event_bus,
                    plugin_modules=cast(Any, before_step_modules),
                ),
            ),
            (
                "after_step",
                default_after_step_modules(
                    self.event_bus,
                    plugin_modules=cast(Any, after_step_modules),
                ),
            ),
            (
                "after_reasoning",
                default_after_reasoning_modules(
                    self.event_bus,
                    cast(Any, getattr(pipeline, "_session", None)),
                    plugin_modules=cast(Any, after_reasoning_modules),
                ),
            ),
            (
                "after_turn",
                default_after_turn_modules(
                    self.event_bus,
                    cast(Any, getattr(pipeline, "_outbound_port", BusOutboundPort(self.bus))),
                    cast(Any, context),
                    cast(int, getattr(pipeline, "_history_window", 500)),
                    plugin_modules=cast(Any, after_turn_modules),
                ),
            ),
        ]

        parts: list[str] = []
        for phase_name, modules in phases:
            parts.append("=" * 60)
            parts.append(phase_name)
            parts.append("=" * 60)
            parts.append(inspect_phase(modules))
        return "\n".join(parts)

    async def stop(self) -> None:
        if self.plugin_manager is not None:
            await self.plugin_manager.terminate_all()
        await self.mcp_registry.shutdown()
        await self.event_bus.aclose()
        if self.peer_poller is not None:
            await self.peer_poller.stop()
        if self.peer_process_manager is not None:
            await self.peer_process_manager.shutdown_all()


def build_registered_tools(
    config: Config,
    workspace: Path,
    http_resources: SharedHttpResources,
    *,
    bus: MessageBus,
    provider,
    light_provider,
    vl_provider=None,
    session_store=None,
    tools: ToolRegistry | None = None,
    event_publisher=None,
    agent_loop_provider: Callable[[], Any] | None = None,
) -> tuple[
    ToolRegistry,
    MessagePushTool,
    SchedulerService,
    McpServerRegistry,
    MemoryRuntime,
    PeerProcessManager | None,
    PeerAgentPoller | None,
]:
    from session.store import SessionStore

    # ── 第一阶段：建服务（依赖无顺序陷阱）────────────────────────────────────
    wiring = getattr(config, "wiring", WiringConfig())
    tools = tools or ToolRegistry()
    multimodal = getattr(config, "multimodal", True)
    vl_available = (not multimodal) and bool(getattr(config, "vl_model", ""))
    readonly_tools = build_readonly_tools(
        http_resources, multimodal=multimodal, vl_available=vl_available
    )
    store = session_store or SessionStore(workspace / "sessions.db")
    push_tool = MessagePushTool()
    _configure_runtime_backed_message_push(config, push_tool)
    memory_result = resolve_memory_toolset_provider(wiring.memory).register(
        tools,
        ToolsetDeps(
            config=config,
            workspace=workspace,
            provider=provider,
            light_provider=light_provider,
            http_resources=http_resources,
            event_publisher=event_publisher,
        ),
    )
    memory_runtime = memory_result.extras["memory_runtime"]
    scheduler = build_scheduler(
        workspace,
        push_tool,
        agent_loop_provider=agent_loop_provider,
        runtime_config=getattr(config, "agent_runtime", None),
    )
    peer_process_manager, peer_poller = build_peer_agent_resources(
        config, bus, http_resources
    )

    # ── 第二阶段：注册工具（所有服务已就绪）──────────────────────────────────
    mcp_registry = None
    for name in wiring.toolsets:
        provider_obj = resolve_toolset_provider(
            name,
            readonly_tools=readonly_tools if name == "meta_common" else None,
        )
        result = provider_obj.register(
            tools,
            ToolsetDeps(
                config=config,
                workspace=workspace,
                session_store=store,
                push_tool=push_tool,
                http_resources=http_resources,
                provider=provider,
                light_provider=light_provider,
                vl_provider=vl_provider,
                vl_model=getattr(config, "vl_model", ""),
                bus=bus,
                memory_engine=memory_runtime.engine,
                scheduler=scheduler,
                event_publisher=event_publisher,
            ),
        )
        maybe_mcp = result.extras.get("mcp_registry")
        if maybe_mcp is not None:
            mcp_registry = maybe_mcp
    if mcp_registry is None:
        from agent.mcp.registry import McpServerRegistry

        mcp_registry = McpServerRegistry(
            config_path=workspace / "mcp_servers.json",
            tool_registry=tools,
        )
    _register_chatgpt_proxy_tools(config, workspace, http_resources, tools)
    _register_group_memory_tools(workspace, store, tools)
    _register_ragflow_tools(config, workspace, store, tools)

    return (
        tools,
        push_tool,
        scheduler,
        mcp_registry,
        memory_runtime,
        peer_process_manager,
        peer_poller,
    )


def _configure_runtime_backed_message_push(
    config: Config,
    push_tool: MessagePushTool,
) -> None:
    agent_runtime = getattr(config, "agent_runtime", None) or getattr(
        config,
        "agent_gateway",
        None,
    )
    if agent_runtime is None:
        return
    if not bool(getattr(agent_runtime, "enabled", False)):
        return
    if not bool(getattr(agent_runtime, "outbox_worker_enabled", False)):
        return
    if not str(getattr(agent_runtime, "base_url", "")).strip():
        return
    channels = [
        str(channel).strip()
        for channel in (getattr(agent_runtime, "outbound_channels", None) or [])
        if str(channel).strip()
    ]
    if not channels:
        return
    from integrations.agent_runtime import AgentRuntimeClient
    from integrations.agent_runtime_outbound import AgentRuntimeOutboundEnqueuer

    enqueuer = AgentRuntimeOutboundEnqueuer(
        AgentRuntimeClient(agent_runtime),
        account_id_by_channel=_runtime_outbound_account_ids_by_channel(config),
    )
    push_tool.configure_runtime_outbox(enqueuer.enqueue, channels=channels)


def _runtime_outbound_account_ids_by_channel(config: Config) -> dict[str, str]:
    channels_config = getattr(config, "channels", None)
    if channels_config is None:
        return {}
    result: dict[str, str] = {}
    telegram = getattr(channels_config, "telegram", None)
    if telegram is not None:
        channel_name = str(getattr(telegram, "channel_name", "")).strip()
        if channel_name:
            result[channel_name] = channel_name
    qq = getattr(channels_config, "qq", None)
    qq_accounts = list(getattr(channels_config, "qq_accounts", []) or [])
    if qq is not None:
        qq_accounts.insert(0, qq)
    for account in qq_accounts:
        channel_name = str(getattr(account, "channel_name", "")).strip()
        account_id = str(getattr(account, "bot_uin", "")).strip()
        if channel_name and account_id:
            result[channel_name] = account_id
    return result


def _register_chatgpt_proxy_tools(
    config: Config,
    workspace: Path,
    http_resources: SharedHttpResources,
    tools: ToolRegistry,
) -> None:
    proxy = getattr(config, "chatgpt_proxy", None)
    if proxy is None or not getattr(proxy, "enabled", False):
        return
    if not getattr(proxy, "base_url", ""):
        logger.warning("chatgpt_proxy 已启用但 base_url 为空，跳过图片工具注册")
        return
    from agent.tools.chatgpt_proxy import ChatGPTImageGenerateTool

    tools.register(
        ChatGPTImageGenerateTool(
            proxy,
            workspace,
            http_resources.external_default,
        ),
        risk="external-side-effect",
        always_on=False,
        search_hint="画图 图片生成 ChatGPT 反向代理 gpt-image image generation",
    )


def _register_group_memory_tools(
    workspace: Path,
    session_store,
    tools: ToolRegistry,
) -> None:
    from agent.tools.group_memory import (
        IngestGroupMemoryTool,
        ListGroupStrategiesTool,
        RecallGroupMemoryTool,
        ShowGroupStrategyEvidenceTool,
    )
    from group_memory import GroupMemoryService

    service = GroupMemoryService.from_workspace(
        workspace,
        session_store=session_store,
    )
    tools.register(
        IngestGroupMemoryTool(service),
        risk="write",
        always_on=False,
        search_hint="QQ群 游戏群 攻略 记忆 抽取 刷新 group memory ingest",
    )
    tools.register(
        RecallGroupMemoryTool(service),
        risk="read-only",
        always_on=False,
        search_hint="QQ群 游戏攻略 群记忆 RAG 查询 boss 配装 打法",
    )
    tools.register(
        ListGroupStrategiesTool(service),
        risk="read-only",
        always_on=False,
        search_hint="列出 群攻略 memory 状态 validated disputed pending",
    )
    tools.register(
        ShowGroupStrategyEvidenceTool(service),
        risk="read-only",
        always_on=False,
        search_hint="群攻略 证据 原始消息 来源 citation",
    )


def _register_ragflow_tools(
    config: Config,
    workspace: Path,
    session_store,
    tools: ToolRegistry,
) -> None:
    ragflow = getattr(config, "ragflow", None)
    if ragflow is None or not getattr(ragflow, "enabled", False):
        return
    if not getattr(ragflow, "base_url", ""):
        logger.warning("ragflow 已启用但 base_url 为空，跳过工具注册")
        return
    if not getattr(ragflow, "api_key", ""):
        logger.warning("ragflow 已启用但 api_key 为空，跳过工具注册")
        return

    from agent.tools.ragflow import (
        RAGFlowIndexQQGroupTool,
        RAGFlowListDatasetsTool,
        RAGFlowRetrieveTool,
        RAGFlowUploadFileTool,
        RAGFlowUploadTextTool,
    )
    from integrations.ragflow import RAGFlowClient

    client = RAGFlowClient(ragflow)
    runtime_message_source = None
    agent_runtime = getattr(config, "agent_runtime", None) or getattr(
        config,
        "agent_gateway",
        None,
    )
    if (
        agent_runtime is not None
        and bool(getattr(agent_runtime, "enabled", False))
        and str(getattr(agent_runtime, "base_url", "")).strip()
    ):
        from integrations.agent_runtime_inbox_source import (
            AgentRuntimeInboxGroupMessageSource,
        )

        runtime_message_source = AgentRuntimeInboxGroupMessageSource(agent_runtime)
    tools.register(
        RAGFlowRetrieveTool(client),
        risk="external-side-effect",
        always_on=False,
        search_hint="ragflow rag 检索 多来源 数据集 dataset retrieval graph kg",
    )
    tools.register(
        RAGFlowListDatasetsTool(client),
        risk="read-only",
        always_on=False,
        search_hint="ragflow dataset datasets list knowledge base",
    )
    tools.register(
        RAGFlowUploadTextTool(client),
        risk="external-side-effect",
        always_on=False,
        search_hint="ragflow 上传 文本 索引 parse ingest text",
    )
    tools.register(
        RAGFlowUploadFileTool(client, workspace),
        risk="external-side-effect",
        always_on=False,
        search_hint="ragflow 上传 文件 pdf docx txt excel parse ingest",
    )
    tools.register(
        RAGFlowIndexQQGroupTool(
            client,
            session_store,
            message_source=runtime_message_source,
        ),
        risk="external-side-effect",
        always_on=False,
        search_hint="ragflow qq 群消息 索引 多来源 数据源 group chat ingest",
    )


def _build_loop_deps(
    *,
    config: Config,
    workspace: Path,
    bus: MessageBus,
    provider: LLMProvider,
    light_provider: LLMProvider | None,
    tools: ToolRegistry,
    session_manager: SessionManager,
    presence: PresenceStore,
    processing_state: ProcessingState,
    event_bus: EventBus,
    memory_runtime: MemoryRuntime,
) -> AgentLoopDeps:
    wiring = getattr(config, "wiring", WiringConfig())
    context = resolve_context_factory(wiring.context)(
        workspace,
        memory_runtime.markdown.store,
    )
    if isinstance(context, ContextBuilder):
        context.set_media_capabilities(
            multimodal=bool(getattr(config, "multimodal", True)),
            vl_available=bool(getattr(config, "vl_model", "")),
        )
    memory_engine = memory_runtime.engine
    light = light_provider or provider
    llm_services = LLMServices(provider=provider, light_provider=light)
    memory_services = MemoryServices(engine=memory_engine)
    session_services = SessionServices(
        session_manager=session_manager, presence=presence
    )
    _bind_memory_lifecycle_if_supported(
        markdown=memory_runtime.markdown.maintenance,
        session_manager=session_manager,
    )
    retrieval_pipeline = DefaultMemoryRetrievalPipeline(
        memory=memory_services,
    )

    return AgentLoopDeps(
        bus=bus,
        event_bus=event_bus,
        provider=provider,
        tools=tools,
        session_manager=session_manager,
        workspace=workspace,
        presence=presence,
        light_provider=light_provider,
        processing_state=processing_state,
        memory_runtime=memory_runtime,
        retrieval_pipeline=retrieval_pipeline,
        context=context,
        llm_services=llm_services,
        memory_services=memory_services,
        session_services=session_services,
    )


def _bind_memory_lifecycle_if_supported(
    *,
    markdown: MarkdownMemoryMaintenance,
    session_manager: SessionManager,
) -> None:
    async def _save_session(session: object) -> None:
        await session_manager.save_async(cast(Session, session))

    markdown.bind_lifecycle(
        MemoryLifecycleBindRequest(
            get_session=session_manager.get_or_create,
            save_session=_save_session,
        )
    )


def build_core_runtime(
    config: Config,
    workspace: Path,
    http_resources: SharedHttpResources,
) -> CoreRuntime:
    bus = MessageBus()
    shadow_runtime = getattr(config, "shadow_runtime", config.shadow_gateway)
    shadow_observer = build_shadow_gateway_observer(
        settings=ShadowGatewaySettings(
            enabled=bool(getattr(shadow_runtime, "enabled", False)),
            endpoint=str(getattr(shadow_runtime, "endpoint", "")),
            log_path=str(getattr(shadow_runtime, "log_path", "")),
            request_timeout_seconds=float(
                getattr(shadow_runtime, "request_timeout_seconds", 2.0)
            ),
            agent_id=str(getattr(shadow_runtime, "agent_id", "shadow")),
        ),
        workspace=workspace,
        channel_account_ids=_shadow_channel_account_ids(config),
    )
    if shadow_observer is not None:
        bus.add_inbound_observer(shadow_observer)
    event_bus = EventBus()
    provider, light_provider, agent_provider = build_providers(config)
    vl_provider = build_vl_provider(config)
    # agent_provider is used for the AgentLoop (QA / tool calling).
    # provider (llm.main) is used for consolidation event extraction.
    loop_provider = agent_provider or provider
    loop_model = config.agent_model or config.model
    session_manager = SessionManager(workspace)
    if shadow_observer is not None:
        session_manager.add_message_observer(
            ObservedSessionShadowMirror(shadow_observer)
        )
    loop_ref: dict[str, AgentLoop] = {}
    tools, push_tool, scheduler, mcp_registry, memory_runtime, peer_pm, peer_poller = (
        build_registered_tools(
            config,
            workspace,
            http_resources,
            bus=bus,
            provider=provider,
            light_provider=light_provider,
            vl_provider=vl_provider,
            session_store=session_manager._store,
            event_publisher=event_bus,
            agent_loop_provider=lambda: loop_ref.get("loop"),
        )
    )
    presence = PresenceStore(session_manager._store)
    processing_state = ProcessingState()
    loop_deps = _build_loop_deps(
        config=config,
        workspace=workspace,
        bus=bus,
        provider=loop_provider,
        light_provider=light_provider,
        tools=tools,
        session_manager=session_manager,
        presence=presence,
        processing_state=processing_state,
        event_bus=event_bus,
        memory_runtime=memory_runtime,
    )
    loop = AgentLoop(
        loop_deps,
        AgentLoopConfig(
            llm=LLMConfig(
                model=loop_model,
                light_model=config.light_model,
                max_iterations=config.max_iterations,
                max_tokens=config.max_tokens,
                tool_search_enabled=config.tool_search_enabled,
                multimodal=bool(getattr(config, "multimodal", True)),
                vl_available=bool(getattr(config, "vl_model", "")),
            ),
            memory=MemoryConfig(
                window=config.memory_window,
            ),
        ),
    )
    loop_ref["loop"] = loop
    wire_turn_lifecycle(
        lifecycle=TurnLifecycle(event_bus),
        active_turn_states=loop.active_turn_states,
    )

    from agent.plugins.manager import PluginManager as _PluginManager
    plugin_manager = _PluginManager(
        plugin_dirs=_resolve_plugin_dirs(workspace),
        event_bus=event_bus,
        tool_registry=tools,
        workspace=workspace,
        session_manager=session_manager,
        memory_engine=memory_runtime.engine,
    )

    return CoreRuntime(
        config=config,
        http_resources=http_resources,
        loop=loop,
        bus=bus,
        event_bus=event_bus,
        tools=tools,
        push_tool=push_tool,
        session_manager=session_manager,
        scheduler=scheduler,
        provider=provider,
        light_provider=light_provider,
        vl_provider=vl_provider,
        agent_provider=agent_provider,
        mcp_registry=mcp_registry,
        memory_runtime=memory_runtime,
        presence=presence,
        peer_process_manager=peer_pm,
        peer_poller=peer_poller,
        plugin_manager=plugin_manager,
    )


def _resolve_plugin_dirs(workspace: Path) -> list[Path]:
    project_root = Path(__file__).resolve().parent.parent
    return [project_root / "plugins"]


def _shadow_channel_account_ids(config: Config) -> dict[str, str]:
    channels = getattr(config, "channels", None)
    result: dict[str, str] = {}
    if channels is None:
        return result

    qq = getattr(channels, "qq", None)
    if qq is not None and getattr(qq, "bot_uin", ""):
        result[str(getattr(qq, "channel_name", "qq"))] = str(qq.bot_uin)
    for account in getattr(channels, "qq_accounts", []) or []:
        if getattr(account, "bot_uin", ""):
            result[str(getattr(account, "channel_name", ""))] = str(account.bot_uin)
    return result
