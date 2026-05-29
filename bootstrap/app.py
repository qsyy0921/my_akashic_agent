from __future__ import annotations

import asyncio
import logging
import sys
from pathlib import Path
from typing import Awaitable, Callable

from agent.config_models import Config
from bootstrap.channels import start_channels
from bootstrap.dashboard_api import build_dashboard_server
from bootstrap.memory import build_memory_runtime
from bootstrap.proactive import build_memory_optimizer_task, build_proactive_runtime
from bootstrap.providers import build_providers
from bootstrap.tools import CoreRuntime, build_core_runtime
from bus.event_bus import EventBus
from core.net.http import (
    SharedHttpResources,
    clear_default_shared_http_resources,
    configure_default_shared_http_resources,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s  %(levelname)-8s  %(name)s  %(message)s",
    datefmt="%H:%M:%S",
    stream=sys.stdout,
    force=True,
)
logging.getLogger("httpx").setLevel(logging.WARNING)
logging.getLogger("telegram").setLevel(logging.WARNING)
logging.getLogger("apscheduler").setLevel(logging.WARNING)
logging.getLogger("openai").setLevel(logging.WARNING)

logger = logging.getLogger(__name__)


async def _run_cleanup_steps(*steps: tuple[str, Callable[[], Awaitable[None]]]) -> None:
    first_error: Exception | None = None
    for name, step in steps:
        try:
            await step()
        except Exception as exc:
            if first_error is None:
                first_error = exc
            logger.warning("shutdown step failed: %s: %s", name, exc)
    if first_error is not None:
        raise first_error


async def _noop_async() -> None:
    return None


class AppRuntime:
    def __init__(self, config: Config, workspace: Path) -> None:
        self.config = config
        self.workspace = workspace
        self.http_resources = SharedHttpResources()
        self.ipc = None
        self.tg_channel = None
        self.qq_channel = None
        self.qqbot_channel = None
        self.core: CoreRuntime | None = None
        self.agent_loop = None
        self.bus = None
        self.event_bus: EventBus | None = None
        self.tools = None
        self.push_tool = None
        self.session_manager = None
        self.scheduler = None
        self.provider = None
        self.light_provider = None
        self.mcp_registry = None
        self.memory_runtime = None
        self.presence = None
        self.proactive_loop = None
        self.group_memory_loop = None
        self.agent_gateway_image_worker = None
        self.agent_gateway_knowledge_worker = None
        self.peer_process_manager = None
        self.peer_poller = None
        self.dashboard_server = None
        self.dashboard_task: asyncio.Task[None] | None = None
        self.tasks: list[Awaitable[None]] = []
        self._memory_optimizer = None
        self._shutdown = False
        self._started = False

    async def start(self) -> None:
        if self._started:
            return
        configure_default_shared_http_resources(self.http_resources)
        try:
            self.core = build_core_runtime(
                self.config,
                self.workspace,
                self.http_resources,
            )
            self.agent_loop = self.core.loop
            self.bus = self.core.bus
            event_bus = self.core.event_bus
            self.event_bus = event_bus
            self.tools = self.core.tools
            self.push_tool = self.core.push_tool
            self.session_manager = self.core.session_manager
            self.scheduler = self.core.scheduler
            self.provider = self.core.provider
            self.light_provider = self.core.light_provider
            self.mcp_registry = self.core.mcp_registry
            self.memory_runtime = self.core.memory_runtime
            self.presence = self.core.presence
            self.peer_process_manager = self.core.peer_process_manager
            self.peer_poller = self.core.peer_poller
            await self.core.start()

            plugin_manager = getattr(self.core, "plugin_manager", None)
            self.ipc, self.tg_channel, self.qq_channel, self.qqbot_channel = await start_channels(
                self.config,
                bus=self.bus,
                session_manager=self.session_manager,
                push_tool=self.push_tool,
                http_resources=self.http_resources,
                event_bus=event_bus,
                vl_provider=getattr(self.core, "vl_provider", None),
                vl_model=getattr(self.config, "vl_model", ""),
                bot_commands=(
                    plugin_manager.telegram_bot_commands
                    if plugin_manager
                    else None
                ),
                interrupt_controller=self.agent_loop,
            )

            self.tasks = [
                self.agent_loop.run(),
                self.bus.dispatch_outbound(),
                self.scheduler.run(),
            ]
            optimizer_tasks, self._memory_optimizer = build_memory_optimizer_task(
                self.config,
                provider=self.provider,
                memory_store=self.memory_runtime.markdown.store,
            )
            self.tasks.extend(optimizer_tasks)
            self.dashboard_server = build_dashboard_server(
                workspace=self.workspace,
                manual_consolidator=self.agent_loop,
                manual_memory_optimizer=self._memory_optimizer,
                memory_admin=self.memory_runtime.engine,
                memory_store=self.memory_runtime.markdown.store,
            )
            self.dashboard_task = asyncio.create_task(
                self.dashboard_server.serve(),
                name="dashboard_server",
            )
            proactive_tasks, self.proactive_loop = build_proactive_runtime(
                self.config,
                self.workspace,
                session_manager=self.session_manager,
                provider=self.provider,
                light_provider=self.light_provider,
                push_tool=self.push_tool,
                memory_store=self.memory_runtime,
                presence=self.presence,
                agent_loop=self.agent_loop,
                tool_hooks=list(plugin_manager.tool_hooks) if plugin_manager else None,
            )
            self.tasks.extend(proactive_tasks)
            if self.proactive_loop is not None:
                self.ipc.set_proactive_loop(self.proactive_loop)
            group_memory_tasks, self.group_memory_loop = _build_group_memory_tasks(
                self.config,
                self.workspace,
                self.session_manager._store,
            )
            self.tasks.extend(group_memory_tasks)
            knowledge_worker_tasks, self.agent_gateway_knowledge_worker = (
                _build_agent_gateway_knowledge_worker_tasks(
                    self.config,
                    self.workspace,
                    self.session_manager._store,
                )
            )
            self.tasks.extend(knowledge_worker_tasks)
            image_worker_tasks, self.agent_gateway_image_worker = (
                _build_agent_gateway_image_worker_tasks(
                    self.config,
                    self.workspace,
                    self.http_resources,
                )
            )
            self.tasks.extend(image_worker_tasks)

            self._started = True
        except Exception:
            await self.shutdown()
            raise

    async def run(self) -> None:
        try:
            await self.start()
            await asyncio.gather(*self.tasks)
        finally:
            await self.shutdown()

    async def shutdown(self) -> None:
        if self._shutdown:
            return
        self._shutdown = True
        try:
            if self.dashboard_server is not None:
                self.dashboard_server.should_exit = True
            if self.dashboard_task is not None:
                try:
                    await self.dashboard_task
                except asyncio.CancelledError:
                    pass
            await _run_cleanup_steps(
                ("core.stop", self.core.stop if self.core else _noop_async),
                ("ipc.stop", self.ipc.stop if self.ipc else _noop_async),
                (
                    "telegram.stop",
                    self.tg_channel.stop if self.tg_channel else _noop_async,
                ),
                ("qq.stop", self.qq_channel.stop if self.qq_channel else _noop_async),
                (
                    "qqbot.stop",
                    self.qqbot_channel.stop if self.qqbot_channel else _noop_async,
                ),
                (
                    "memory_runtime.aclose",
                    self.memory_runtime.aclose if self.memory_runtime else _noop_async,
                ),
                (
                    "group_memory.stop",
                    _group_memory_stop(self.group_memory_loop),
                ),
                (
                    "agent_gateway_image_worker.stop",
                    _loop_stop(self.agent_gateway_image_worker),
                ),
                (
                    "agent_gateway_knowledge_worker.stop",
                    _loop_stop(self.agent_gateway_knowledge_worker),
                ),
                ("http_resources.aclose", self.http_resources.aclose),
            )
        finally:
            clear_default_shared_http_resources(self.http_resources)


def build_app_runtime(config: Config, workspace: Path | None = None) -> AppRuntime:
    return AppRuntime(config, workspace or (Path.home() / ".akashic" / "workspace"))


def _build_group_memory_tasks(
    config: Config,
    workspace: Path,
    session_store,
) -> tuple[list[Awaitable[None]], object | None]:
    if bool(getattr(getattr(config, "agent_gateway", None), "enabled", False)):
        return [], None
    groups = sorted(_observe_only_qq_group_accounts(config))
    if not groups:
        return [], None
    from group_memory import GroupMemoryService
    from group_memory.service import GroupMemoryLoop

    service = GroupMemoryService.from_workspace(workspace, session_store=session_store)
    loop = GroupMemoryLoop(service=service, group_ids=groups)
    return [loop.run()], loop


def _group_memory_stop(loop: object | None):
    return _loop_stop(loop)


def _build_agent_gateway_image_worker_tasks(
    config: Config,
    workspace: Path,
    http_resources: SharedHttpResources,
) -> tuple[list[Awaitable[None]], object | None]:
    agent_gateway = getattr(config, "agent_gateway", None)
    chatgpt_proxy = getattr(config, "chatgpt_proxy", None)
    if agent_gateway is None or not bool(getattr(agent_gateway, "enabled", False)):
        return [], None
    if chatgpt_proxy is None or not bool(getattr(chatgpt_proxy, "enabled", False)):
        logger.warning("agent_gateway 已启用但 chatgpt_proxy 未启用，跳过 image worker")
        return [], None
    if not getattr(chatgpt_proxy, "base_url", ""):
        logger.warning("agent_gateway 已启用但 chatgpt_proxy.base_url 为空，跳过 image worker")
        return [], None

    from agent.tools.chatgpt_proxy import ChatGPTImageGenerateTool
    from integrations.agent_gateway import AgentGatewayClient
    from integrations.agent_gateway_image_worker import AgentGatewayImageWorker

    client = AgentGatewayClient(agent_gateway)
    image_tool = ChatGPTImageGenerateTool(
        chatgpt_proxy,
        workspace,
        http_resources.external_default,
    )
    worker = AgentGatewayImageWorker(
        client=client,
        image_tool=image_tool,
        worker_id=str(getattr(agent_gateway, "worker_id", "akashic-python-worker")),
        lease_ttl_seconds=int(getattr(agent_gateway, "lease_ttl_seconds", 300)),
        poll_interval_seconds=float(
            getattr(agent_gateway, "poll_interval_seconds", 2.0)
        ),
    )
    return [worker.run()], worker


def _build_agent_gateway_knowledge_worker_tasks(
    config: Config,
    workspace: Path,
    session_store,
) -> tuple[list[Awaitable[None]], object | None]:
    agent_gateway = getattr(config, "agent_gateway", None)
    if agent_gateway is None or not bool(getattr(agent_gateway, "enabled", False)):
        return [], None
    if not str(getattr(agent_gateway, "base_url", "")).strip():
        logger.warning("agent_gateway 已启用但 base_url 为空，跳过 knowledge worker")
        return [], None
    group_accounts = _observe_only_qq_group_accounts(config)
    if not group_accounts:
        return [], None

    from agent.tools.ragflow import RAGFlowIndexQQGroupTool
    from group_memory import GroupMemoryService
    from integrations.agent_gateway import AgentGatewayClient
    from integrations.agent_gateway_knowledge_worker import AgentGatewayKnowledgeWorker
    from integrations.ragflow import RAGFlowClient

    ragflow = getattr(config, "ragflow", None)
    ragflow_indexer = None
    ragflow_dataset_ids: list[str] = []
    if (
        ragflow is not None
        and bool(getattr(ragflow, "enabled", False))
        and getattr(ragflow, "base_url", "")
        and getattr(ragflow, "api_key", "")
    ):
        ragflow_dataset_ids = [
            str(value).strip()
            for value in getattr(ragflow, "default_dataset_ids", [])
            if str(value).strip()
        ]
        if ragflow_dataset_ids:
            ragflow_indexer = RAGFlowIndexQQGroupTool(
                RAGFlowClient(ragflow),
                session_store,
            )

    worker = AgentGatewayKnowledgeWorker(
        client=AgentGatewayClient(agent_gateway),
        group_memory=GroupMemoryService.from_workspace(
            workspace,
            session_store=session_store,
        ),
        worker_id=str(getattr(agent_gateway, "worker_id", "akashic-python-worker")),
        group_accounts=group_accounts,
        ragflow_indexer=ragflow_indexer,
        ragflow_dataset_ids=ragflow_dataset_ids,
        lease_ttl_seconds=int(getattr(agent_gateway, "lease_ttl_seconds", 300)),
        poll_interval_seconds=float(
            getattr(agent_gateway, "poll_interval_seconds", 2.0)
        ),
        enqueue_interval_seconds=float(
            getattr(agent_gateway, "knowledge_job_interval_seconds", 60.0)
        ),
    )
    return [worker.run()], worker


def _observe_only_qq_group_accounts(config: Config) -> dict[str, str]:
    channels = getattr(config, "channels", None)
    if channels is None:
        return {}
    accounts = []
    qq = getattr(channels, "qq", None)
    if qq is not None:
        accounts.append(qq)
    accounts.extend(getattr(channels, "qq_accounts", []) or [])

    result: dict[str, str] = {}
    for account in accounts:
        account_id = str(getattr(account, "bot_uin", "")).strip()
        if not account_id:
            continue
        for group in getattr(account, "groups", []) or []:
            if not bool(getattr(group, "observe_only", False)):
                continue
            group_id = str(getattr(group, "group_id", "")).strip()
            if group_id and group_id not in result:
                result[group_id] = account_id
    return result


def _loop_stop(loop: object | None):
    async def _stop() -> None:
        if loop is not None and hasattr(loop, "stop"):
            loop.stop()

    return _stop
