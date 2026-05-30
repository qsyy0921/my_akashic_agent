from __future__ import annotations

import logging
from typing import Any

from agent.config_models import Config
from agent.looping.interrupt import InterruptController
from agent.tools.message_push import MessagePushTool
from bus.event_bus import EventBus
from bus.queue import MessageBus
from core.net.http import SharedHttpResources
from session.manager import SessionManager

logger = logging.getLogger(__name__)


class _CompositeChannel:
    def __init__(self, channels: list[Any]) -> None:
        self.channels = channels

    async def stop(self) -> None:
        for channel in reversed(self.channels):
            stop = getattr(channel, "stop", None)
            if callable(stop):
                await stop()


async def start_channels(
    config: Config,
    *,
    bus: MessageBus,
    session_manager: SessionManager,
    push_tool: MessagePushTool,
    http_resources: SharedHttpResources,
    event_bus: EventBus,
    vl_provider: Any | None = None,
    vl_model: str = "",
    bot_commands: list[tuple[str, str]] | None = None,
    interrupt_controller: InterruptController | None = None,
) -> tuple[Any, Any, Any, Any]:
    from infra.channels.ipc_server import IPCServerChannel

    ipc = IPCServerChannel(bus, config.channels.socket)
    await ipc.start()
    push_tool.register_channel(
        "cli",
        text=ipc.send,
    )
    print(f"Agent 已启动  |  CLI 连接地址: {config.channels.socket}")

    send_ledger_client = None
    if config.agent_runtime.enabled and str(config.agent_runtime.base_url or "").strip():
        from integrations.agent_runtime import AgentRuntimeClient

        send_ledger_client = AgentRuntimeClient(config.agent_runtime)

    tg_channel = None
    if config.channels.telegram and config.channels.telegram.token:
        from infra.channels.telegram_channel import TelegramChannel

        tg = config.channels.telegram
        candidate = TelegramChannel(
            token=tg.token,
            bus=bus,
            session_manager=session_manager,
            allow_from=tg.allow_from,
            bot_commands=bot_commands,
            event_bus=event_bus,
            interrupt_controller=interrupt_controller,
            channel_name=tg.channel_name,
            send_ledger_client=send_ledger_client,
            receiver_status_client=send_ledger_client,
        )
        try:
            await candidate.start()
        except Exception as exc:
            logger.exception(
                "Telegram channel start failed; continuing without Telegram"
            )
            await _report_receiver_status(
                send_ledger_client,
                kind="telegram",
                channel_name=tg.channel_name,
                status="failed",
                reason="start_failed",
                last_error=str(exc),
            )
        else:
            tg_channel = candidate
            push_tool.register_channel(
                tg.channel_name,
                text=tg_channel.send,
                stream_text=tg_channel.send_stream,
                file=tg_channel.send_file,
                image=tg_channel.send_image,
            )
            print("Telegram Bot 已启动")

    if config.channels.feishu and config.channels.feishu.webhook_url:
        from infra.channels.feishu_channel import FeishuWebhookChannel

        fs = config.channels.feishu
        candidate = FeishuWebhookChannel(
            webhook_url=fs.webhook_url,
            secret=fs.secret,
            requester=http_resources.external_default,
        )
        try:
            await candidate.start()
        except Exception:
            logger.exception(
                "Feishu channel start failed; continuing without Feishu"
            )
        else:
            push_tool.register_channel(
                fs.channel_name,
                text=candidate.send,
                stream_text=candidate.send,
            )
            print(f"飞书 Bot 已启动  |  channel: {fs.channel_name}")

    if config.channels.wechat and config.channels.wechat.webhook_url:
        from infra.channels.wechat_channel import WechatWebhookChannel

        wc = config.channels.wechat
        candidate = WechatWebhookChannel(
            webhook_url=wc.webhook_url,
            mentioned_list=wc.mentioned_list,
            mentioned_mobile_list=wc.mentioned_mobile_list,
            requester=http_resources.external_default,
        )
        try:
            await candidate.start()
        except Exception:
            logger.exception(
                "Wechat channel start failed; continuing without Wechat"
            )
        else:
            push_tool.register_channel(
                wc.channel_name,
                text=candidate.send,
                stream_text=candidate.send,
            )
            print(f"微信 Bot 已启动  |  channel: {wc.channel_name}")

    qq_channel = None
    qq_configs = []
    if config.channels.qq and config.channels.qq.bot_uin:
        qq_configs.append(config.channels.qq)
    qq_configs.extend(config.channels.qq_accounts or [])
    if qq_configs:
        from infra.channels.qq_channel import QQChannel

        started_qq_channels = []
        for qq in qq_configs:
            candidate = QQChannel(
                bot_uin=qq.bot_uin,
                bus=bus,
                session_manager=session_manager,
                allow_from=qq.allow_from,
                bot_peer_ids=qq.bot_peer_ids,
                peer_trigger_prefixes=qq.peer_trigger_prefixes,
                groups=qq.groups,
                websocket_open_timeout_seconds=qq.websocket_open_timeout_seconds,
                http_requester=http_resources.external_default,
                event_bus=event_bus,
                interrupt_controller=interrupt_controller,
                vl_provider=vl_provider,
                vl_model=vl_model,
                channel_name=qq.channel_name,
                websocket_uri=qq.websocket_uri,
                websocket_token=qq.websocket_token,
                send_ledger_client=send_ledger_client,
            )
            try:
                await candidate.start()
            except Exception as exc:
                logger.exception(
                    "QQ channel start failed; continuing without QQ account %s",
                    qq.bot_uin,
                )
                await _report_receiver_status(
                    send_ledger_client,
                    kind="qq",
                    channel_name=qq.channel_name,
                    account_id=str(qq.bot_uin),
                    endpoint=str(qq.websocket_uri or ""),
                    status="failed",
                    reason="start_failed",
                    last_error=str(exc),
                    metadata={"websocket_uri": str(qq.websocket_uri or "")},
                )
            else:
                await _report_receiver_status(
                    send_ledger_client,
                    kind="qq",
                    channel_name=qq.channel_name,
                    account_id=str(qq.bot_uin),
                    endpoint=str(qq.websocket_uri or ""),
                    status="connected",
                    reason="ncatbot_started",
                    metadata={"websocket_uri": str(qq.websocket_uri or "")},
                )
                started_qq_channels.append(candidate)
                push_tool.register_channel(
                    qq.channel_name,
                    text=candidate.send,
                    file=candidate.send_file,
                    image=candidate.send_image,
                )
                print(f"QQ Bot 已启动  |  channel: {qq.channel_name} | QQ 号: {qq.bot_uin}")
        if len(started_qq_channels) == 1:
            qq_channel = started_qq_channels[0]
        elif len(started_qq_channels) > 1:
            qq_channel = _CompositeChannel(started_qq_channels)

    qqbot_channel = None
    if config.channels.qqbot and config.channels.qqbot.app_id:
        from infra.channels.qqbot_channel import QQBotChannel

        qqbot = config.channels.qqbot
        candidate = QQBotChannel(
            app_id=qqbot.app_id,
            client_secret=qqbot.client_secret,
            bus=bus,
            session_manager=session_manager,
            allow_from=qqbot.allow_from,
            groups=qqbot.groups,
            event_bus=event_bus,
            interrupt_controller=interrupt_controller,
        )
        try:
            await candidate.start()
        except Exception:
            logger.exception("Official QQBot channel start failed; continuing without QQBot")
        else:
            qqbot_channel = candidate
            push_tool.register_channel(
                "qqbot",
                text=qqbot_channel.send_proactive,
                stream_text=qqbot_channel.send_stream,
            )
            print(f"官方 QQBot 已启动  |  AppID: {qqbot.app_id}")

    return ipc, tg_channel, qq_channel, qqbot_channel


async def _report_receiver_status(
    client: Any | None,
    *,
    kind: str,
    channel_name: str,
    status: str,
    account_id: str = "",
    endpoint: str = "",
    reason: str = "",
    last_error: str = "",
    metadata: dict[str, str] | None = None,
) -> None:
    if client is None:
        return
    try:
        await client.report_receiver_status(
            kind=kind,
            channel_name=channel_name,
            account_id=account_id,
            endpoint=endpoint,
            status=status,
            reason=reason,
            last_error=last_error,
            source="python_channel",
            metadata=metadata or {},
        )
    except Exception as exc:
        logger.debug(
            "agent_runtime receiver status report failed channel=%s status=%s: %s",
            channel_name,
            status,
            exc,
        )
