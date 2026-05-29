"""
QQ Channel

通过 NcatBot（NapCat Python SDK）接入 QQ 私聊和群聊消息。
消息流向：QQ → NcatBot → MessageBus → AgentLoop → MessageBus → QQ

chat_id 约定：
  私聊："{user_id}"           （如 "987654321"）
  群聊："gqq:{group_id}"     （如 "gqq:111222333"）

摩擦点说明：
  1. run_backend() 是同步阻塞调用 → 用 run_in_executor 包裹
  2. NcatBot 事件回调运行在独立线程/loop → 用 run_coroutine_threadsafe 桥接到主 loop
  3. 出站消息需跨 loop 调用 API → 使用 run_coroutine_threadsafe 投递回 NcatBot loop
"""

import asyncio
import base64
from dataclasses import dataclass, field
import html
import importlib
import logging
import mimetypes
import os
import re
from collections.abc import Coroutine
from pathlib import Path
from typing import Any, cast

from agent.config_models import QQGroupConfig
from agent.looping.interrupt import InterruptController
from bus.event_bus import EventBus
from bus.events import InboundMessage, OutboundMessage
from bus.events_lifecycle import (
    ToolCallCompleted,
    ToolCallStarted,
    TurnStarted,
)
from bus.queue import MessageBus
from infra.channels.base import AttachmentStore, SessionIdentityIndex
from infra.channels.group_filter import (
    DefaultGroupFilter,
    GroupMessageFilter,
    strip_at_segments,
)
from infra.channels.private_loop_guard import (
    BotPeerTriggerPolicy,
    OUTBOUND_FILE_MARKER,
    OUTBOUND_FORWARD_MARKER,
    OUTBOUND_IMAGE_MARKER,
    RecentPrivateOutboundGuard,
    shared_private_outbound_guard,
)
from core.net.http import HttpRequester, RequestBudget, get_default_http_requester
from session.manager import SessionManager

# NcatBot 运行时产物（plugins、logs）放到用户目录，不污染项目目录
_NCATBOT_DIR = Path.home() / ".akashic" / "ncatbot"

logger = logging.getLogger(__name__)

_CHANNEL = "qq"
_GROUP_PREFIX = "gqq:"
_TRACE_THINKING_LIMIT = 500
_TRACE_TOOL_RESULT_LIMIT = 120
_TRACE_DEFAULT_ACTOR = "Akashic"
_NCATBOT_LOG_FORMAT = "%(asctime)s %(levelname)s %(name)s %(message)s"


@dataclass
class _QQTraceLine:
    tool_name: str
    status: str = "started"
    intent: str = ""
    target: str = ""
    result_preview: str = ""


@dataclass
class _QQTraceState:
    user_message: str = ""
    tool_lines: list[_QQTraceLine] = field(default_factory=list)


def _session_key_for_chat(chat_id: str, channel: str = _CHANNEL) -> str:
    return f"{channel}:{chat_id}"


def _session_key(channel: str, chat_id: str) -> str:
    return f"{channel}:{chat_id}"


def _truncate_trace_text(text: str, limit: int) -> str:
    raw = str(text or "").strip()
    if len(raw) <= limit:
        return raw
    omitted = len(raw) - limit
    head = max(0, limit // 2)
    tail = max(0, limit - head)
    return f"{raw[:head]} ...[{omitted} chars omitted]... {raw[-tail:]}"


def _format_tool_intent(arguments: dict[str, Any]) -> str:
    if not isinstance(arguments, dict):
        return ""
    for key in ("description", "query", "summary", "task", "action"):
        value = arguments.get(key)
        if isinstance(value, str) and value.strip():
            return _truncate_trace_text(value, 80)
    return ""


def _format_tool_target(arguments: dict[str, Any]) -> str:
    if not isinstance(arguments, dict):
        return ""
    if isinstance(arguments.get("path"), str) and arguments.get("path", "").strip():
        return _truncate_trace_text(str(arguments["path"]).strip(), 60)
    if isinstance(arguments.get("file_path"), str) and arguments.get("file_path", "").strip():
        return _truncate_trace_text(str(arguments["file_path"]).strip(), 60)
    for key in (
        "cmd",
        "command",
        "query",
        "url",
        "file",
        "text",
        "content",
        "prompt",
        "name",
    ):
        value = arguments.get(key)
        if isinstance(value, str | int | float) and str(value).strip():
            return _truncate_trace_text(str(value).strip(), 80)
    return ""


def _format_tool_trace_lines(lines: list[_QQTraceLine]) -> str:
    if not lines:
        return "No tool calls."
    rendered: list[str] = []
    for index, line in enumerate(lines, start=1):
        rendered.append(f"{index}. {_compress_tool_line(line)}")
    return "\n".join(rendered)


def _summarize_tool_result_preview(tool_name: str, preview: str) -> str:
    text = str(preview or "").strip()
    if not text:
        return ""
    name = tool_name.lower()
    if name == "fetch_messages":
        if '"matched_count"' in text or '"count"' in text:
            matched = re.search(r'"matched_count"\s*:\s*(\d+)', text)
            count = re.search(r'"count"\s*:\s*(\d+)', text)
            hit_text = matched.group(1) if matched else "?"
            total_text = count.group(1) if count else "?"
            return f"结果：命中 {hit_text} 条，返回上下文 {total_text} 条"
        return "结果：已返回消息上下文"
    if name == "list_dir":
        lines = [line.strip() for line in text.splitlines() if line.strip()]
        if lines:
            return f"结果：列出 {len(lines)} 项"
        return "结果：已列出目录内容"
    if name == "read_file":
        line_no = re.search(r"(\d+)→", text)
        if line_no:
            return f"结果：已读取第 {line_no.group(1)} 行附近内容"
        if "字节" in text:
            return "结果：已读取文件片段"
        return "结果：已读取文件"
    if name == "shell":
        exit_code = re.search(r'"exit_code"\s*:\s*(-?\d+)', text)
        if exit_code:
            code = exit_code.group(1)
            if code == "0":
                return "结果：命令执行成功"
            return f"结果：命令退出码 {code}"
        command = re.search(r'"command"\s*:\s*"([^"]+)"', text)
        if command:
            snippet = _truncate_trace_text(command.group(1), 50)
            return f"结果：已执行命令 {snippet}"
        if "（无输出）" in text or "(无输出)" in text:
            return "结果：命令已执行（无输出）"
        return "结果：命令已执行"
    if name == "list_schedules":
        matched = re.search(r"(\d+)\s*个", text)
        if matched:
            return f"结果：当前有 {matched.group(1)} 个提醒"
        return "结果：已列出提醒"
    if name == "cancel_schedule":
        matched = re.search(r"(\d+)\s*个", text)
        if matched:
            return f"结果：已取消 {matched.group(1)} 个提醒"
        return "结果：已执行取消"
    if name == "schedule":
        return "结果：已创建提醒"
    return f"结果：{_truncate_trace_text(text, _TRACE_TOOL_RESULT_LIMIT)}"


def _tool_emoji(tool_name: str) -> str:
    name = tool_name.lower()
    if name.startswith("mcp"):
        return "📡"
    if "search" in name or "fetch" in name:
        return "🔍"
    if "schedule" in name or "cancel" in name:
        return "⏰"
    if "shell" in name:
        return "⚙"
    if "file" in name or "read" in name or "write" in name:
        return "📄"
    return "🔧"


def _compress_tool_line(line: _QQTraceLine) -> str:
    status = "已完成" if line.status == "done" else "失败" if line.status == "error" else "进行中"
    parts = [f"{_tool_emoji(line.tool_name)} {line.tool_name}", status]
    if line.intent:
        parts.append(f"意图：{line.intent}")
    elif line.target:
        parts.append(f"目标：{line.target}")
    if line.result_preview:
        parts.append(line.result_preview)
    return " | ".join(parts)

# 匹配 CQ:image 码中的 url 字段
_CQ_IMAGE_RE = re.compile(r"\[CQ:image[^\]]*?(?:,|\b)url=([^,\]]+)[^\]]*\]")
_MAX_OBSERVE_IMAGE_DESCRIPTIONS = 3
_MAX_OBSERVE_FILE_BYTES = 20 * 1024 * 1024
_MAX_TEXT_PREVIEW_BYTES = 256 * 1024
_TEXT_PREVIEW_CHARS = 1600
_VISION_TIMEOUT_SECONDS = 90.0


@dataclass
class _ObservedFile:
    file_id: str = ""
    name: str = ""
    size: int | None = None
    busid: str = ""
    url: str = ""
    local_path: str = ""


def _safe_file_suffix(name: str) -> str:
    suffix = Path(str(name or "")).suffix
    if suffix and re.fullmatch(r"\.[A-Za-z0-9]{1,12}", suffix):
        return suffix
    return ".bin"


def _coerce_file_size(value: object) -> int | None:
    if value is None:
        return None
    try:
        size = int(str(value))
    except (TypeError, ValueError):
        return None
    return size if size >= 0 else None


def _format_file_size(size: int | None) -> str:
    if size is None:
        return "大小未知"
    units = ("B", "KB", "MB", "GB")
    value = float(size)
    unit = units[0]
    for unit in units:
        if value < 1024 or unit == units[-1]:
            break
        value /= 1024
    if unit == "B":
        return f"{int(value)}B"
    return f"{value:.1f}{unit}"


def _file_meta_from_notice(event: Any) -> _ObservedFile | None:
    raw = getattr(event, "file", None)
    if raw is None:
        return None
    if isinstance(raw, dict):
        getter = raw.get
    else:
        getter = lambda key, default=None: getattr(raw, key, default)
    file_id = str(getter("id", "") or getter("file_id", "") or "").strip()
    name = str(getter("name", "") or getter("file_name", "") or file_id or "群文件").strip()
    size = _coerce_file_size(getter("size", None))
    busid = str(getter("busid", "") or "").strip()
    return _ObservedFile(file_id=file_id, name=name, size=size, busid=busid)


def _preview_text_file(path: Path) -> str:
    raw = path.read_bytes()[:_MAX_TEXT_PREVIEW_BYTES]
    if b"\x00" in raw[:4096]:
        return ""
    text = ""
    for encoding in ("utf-8-sig", "utf-8", "gb18030", "gbk"):
        try:
            text = raw.decode(encoding)
            break
        except UnicodeDecodeError:
            continue
    if not text:
        return ""
    text = re.sub(r"\s+", " ", text).strip()
    if not text:
        return ""
    if len(text) > _TEXT_PREVIEW_CHARS:
        text = text[:_TEXT_PREVIEW_CHARS].rstrip() + f"...（截断 {len(text) - _TEXT_PREVIEW_CHARS} 字）"
    return text


def _summarize_local_file(path: str) -> str:
    p = Path(path)
    if not p.is_file():
        return "文件未能落盘，暂只能记录元信息。"
    mime, _ = mimetypes.guess_type(p.name)
    if mime and mime.startswith("text/"):
        preview = _preview_text_file(p)
        return f"文本预览：{preview}" if preview else "文本文件预览为空或无法解码。"
    if p.suffix.lower() in {".md", ".txt", ".csv", ".json", ".toml", ".yaml", ".yml", ".log"}:
        preview = _preview_text_file(p)
        return f"文本预览：{preview}" if preview else "文本文件预览为空或无法解码。"
    return f"已保存到本地：{path}"


def _path_is_relative_to(path: Path, base: Path) -> bool:
    try:
        path.relative_to(base)
        return True
    except ValueError:
        return False


def _resolve_existing_local_path(path: str, workspace: Path | None = None) -> Path:
    raw = Path(path)
    candidates = [raw] if raw.is_absolute() else [Path.cwd() / raw]
    if workspace is not None and not raw.is_absolute():
        candidates.append(workspace / raw)
    for candidate in candidates:
        resolved = candidate.resolve()
        if resolved.exists():
            return resolved
    return candidates[0].resolve()


def _patch_ncatbot_ws_open_timeout(timeout_seconds: float) -> None:
    """覆盖 ncatbot 进程内写死的 1 秒 WebSocket 握手超时。"""
    if timeout_seconds <= 0:
        return

    try:
        adapter_mod = importlib.import_module("ncatbot.core.adapter.adapter")
        original_connect = getattr(
            adapter_mod,
            "_akashic_original_websockets_connect",
            None,
        )
        if original_connect is None:
            original_connect = adapter_mod.websockets.connect
            adapter_mod._akashic_original_websockets_connect = original_connect

            def _patched_connect(*args, **kwargs):
                configured_timeout = getattr(
                    adapter_mod,
                    "_akashic_websocket_open_timeout_seconds",
                    None,
                )
                if configured_timeout is not None:
                    kwargs["open_timeout"] = configured_timeout
                return adapter_mod._akashic_original_websockets_connect(*args, **kwargs)

            adapter_mod.websockets.connect = _patched_connect

        adapter_mod._akashic_websocket_open_timeout_seconds = timeout_seconds
    except Exception as e:
        logger.warning("[qq] patch ncatbot WebSocket open_timeout 失败，沿用 SDK 默认值: %s", e)


def _ensure_ncatbot_log_format_env() -> None:
    """NcatBot imports its logger eagerly and crashes on values like LOG_FORMAT=json."""
    for env_name in ("LOG_FORMAT", "LOG_FILE_FORMAT"):
        value = os.environ.get(env_name)
        if not value:
            continue
        try:
            logging.Formatter(value)
        except ValueError:
            logger.warning(
                "[qq] 环境变量 %s=%r 不是 Python logging 格式，已为 NcatBot 临时改用默认格式",
                env_name,
                value,
            )
            os.environ[env_name] = _NCATBOT_LOG_FORMAT


def _extract_cq_images(raw: str) -> tuple[str, list[str]]:
    """从 CQ 码中提取图片 URL，返回 (纯文本, [url...])"""
    urls = _CQ_IMAGE_RE.findall(raw)
    text = re.sub(r"\[CQ:image[^\]]*\]", "", raw).strip()
    return text, urls


async def _download_to_temp(
    urls: list[str],
    requester: HttpRequester,
    attachments: AttachmentStore | None = None,
) -> list[str]:
    """下载图片到临时文件，返回本地路径列表"""
    if not urls:
        return []
    paths: list[str] = []
    attachment_store = attachments or AttachmentStore()
    ext_map = {
        "image/jpeg": ".jpg",
        "image/png": ".png",
        "image/gif": ".gif",
        "image/webp": ".webp",
    }
    for url in urls:
        try:
            url = html.unescape(url)  # 还原 &amp; 等 HTML 实体
            resp = await requester.get(
                url,
                follow_redirects=True,
                timeout_s=15.0,
                budget=RequestBudget(total_timeout_s=20.0),
            )
            resp.raise_for_status()
            ct = resp.headers.get("content-type", "image/jpeg").split(";")[0].strip()
            ext = ext_map.get(ct, ".jpg")
            path = attachment_store.write_bytes(
                resp.content,
                prefix="akashic_qq_",
                suffix=ext,
            )
            paths.append(str(path))
        except Exception as e:
            logger.warning(f"[qq] 图片下载失败  url={url[:80]}  错误: {e}")
    return paths


class QQChannel:

    def __init__(
        self,
        bot_uin: str,
        bus: MessageBus,
        session_manager: SessionManager,
        allow_from: list[str] | None = None,
        bot_peer_ids: list[str] | None = None,
        peer_trigger_prefixes: list[str] | None = None,
        groups: list[QQGroupConfig] | None = None,
        websocket_open_timeout_seconds: float = 5.0,
        group_filter: GroupMessageFilter | None = None,
        http_requester: HttpRequester | None = None,
        event_bus: EventBus | None = None,
        interrupt_controller: InterruptController | None = None,
        vl_provider: Any | None = None,
        vl_model: str = "",
        channel_name: str = _CHANNEL,
        websocket_uri: str = "",
        websocket_token: str = "NcatBot",
        private_outbound_guard: RecentPrivateOutboundGuard | None = None,
    ) -> None:
        _ensure_ncatbot_log_format_env()
        from ncatbot.core import BotClient
        from ncatbot.utils import ncatbot_config

        self._bus = bus
        self._session_manager = session_manager
        self._bot_uin = bot_uin
        self._channel = str(channel_name or _CHANNEL)
        allowed_users = [str(user_id) for user_id in (allow_from or [])]
        self._allow_from: set[str] = set(allowed_users)
        self._bot_peer_ids: set[str] = {
            str(user_id) for user_id in (bot_peer_ids or []) if str(user_id).strip()
        }
        self._peer_trigger_prefixes: tuple[str, ...] = tuple(
            str(prefix).strip()
            for prefix in (peer_trigger_prefixes or [])
            if str(prefix).strip()
        )
        self._bot_peer_policy = BotPeerTriggerPolicy.from_config(
            peer_ids=list(self._bot_peer_ids),
            prefixes=list(self._peer_trigger_prefixes),
        )
        self._private_outbound_guard = (
            private_outbound_guard or shared_private_outbound_guard
        )
        self._websocket_open_timeout_seconds = float(websocket_open_timeout_seconds)
        self._interrupt_controller = interrupt_controller
        ws = getattr(session_manager, "workspace", None)
        self._workspace = Path(ws) if ws else None
        self._attachments = AttachmentStore(Path(ws) / "uploads" if ws else None)
        self._trace_actor_name_cache: str | None = None
        self._identity_index = SessionIdentityIndex(
            session_manager,
            channel=self._channel,
            metadata_key="user_id",
        )

        # group_id → QQGroupConfig
        self._groups: dict[str, QQGroupConfig] = {g.group_id: g for g in (groups or [])}

        # 消息过滤器，默认使用 DefaultGroupFilter
        self._group_filter: GroupMessageFilter = group_filter or DefaultGroupFilter(
            bot_uin
        )
        self._http_requester = http_requester or get_default_http_requester(
            "external_default"
        )
        self._event_bus = event_bus
        self._trace_states: dict[str, _QQTraceState] = {}
        self._vl_provider = vl_provider
        self._vl_model = str(vl_model or "")
        self._vision_semaphore = asyncio.Semaphore(2)

        _patch_ncatbot_ws_open_timeout(self._websocket_open_timeout_seconds)
        ncatbot_config.bt_uin = bot_uin
        ncatbot_config.root = allowed_users[0] if allowed_users else bot_uin
        # NapCat 由 Docker 容器管理，NcatBot 只负责连接 WebSocket
        ncatbot_config.check_ncatbot_update = False
        ncatbot_config.skip_ncatbot_install_check = True
        ncatbot_config.napcat.remote_mode = True
        if websocket_uri:
            ncatbot_config.napcat.ws_uri = websocket_uri
        if websocket_token:
            ncatbot_config.napcat.ws_token = websocket_token
        # Akashic 只需要 NapCat 的 OneBot WebSocket，禁用 WebUI 避免启动时卡交互 token。
        ncatbot_config.napcat.enable_webui = False
        ncatbot_config.enable_webui_interaction = False
        # 运行时产物重定向到 ~/.akashic/ncatbot/，不污染项目目录
        _NCATBOT_DIR.mkdir(parents=True, exist_ok=True)
        (_NCATBOT_DIR / "plugins").mkdir(exist_ok=True)
        ncatbot_config.plugin.plugins_dir = str(_NCATBOT_DIR / "plugins")

        self._bot = BotClient()
        self._api = None
        self._main_loop: asyncio.AbstractEventLoop | None = None
        self._bot_loop: asyncio.AbstractEventLoop | None = None

        # username（QQ 号字符串）→ chat_id 映射，供主动推送工具使用
        self.user_map = self._identity_index.mapping

    def _is_allowed(self, user_id: str) -> bool:
        if not self._allow_from:
            return True
        return user_id in self._allow_from

    def _is_recent_akashic_private_echo(
        self,
        user_id: str,
        text: str,
        img_urls: list[str] | None,
    ) -> bool:
        return self._private_outbound_guard.is_echo(
            from_user=user_id,
            to_bot=self._bot_uin,
            text=text,
            has_image=bool(img_urls),
        )

    async def start(self) -> None:
        self._main_loop = asyncio.get_running_loop()
        self._identity_index.rebuild()
        if self._event_bus is not None:
            self._event_bus.on(TurnStarted, self._on_turn_started)
            self._event_bus.on(ToolCallStarted, self._on_tool_call_started)
            self._event_bus.on(ToolCallCompleted, self._on_tool_call_completed)

        @cast(Any, self._bot.on_private_message())
        async def _(event) -> None:
            if self._bot_loop is None:
                self._bot_loop = asyncio.get_running_loop()
            user_id = str(event.user_id)

            if not self._is_allowed(user_id):
                logger.warning(f"[qq] 拒绝未授权用户  user_id={user_id}")
                return

            raw: str = event.raw_message
            text, img_urls = _extract_cq_images(raw)
            if self._is_recent_akashic_private_echo(user_id, text, img_urls):
                logger.info(
                    "[qq] 忽略近期出站回流  channel=%s  from=%s  to_bot=%s",
                    self._channel,
                    user_id,
                    self._bot_uin,
                )
                return
            triggered = self._bot_peer_policy.normalize_private_text(
                user_id=user_id,
                text=text,
            )
            if triggered is None:
                logger.info(
                    "[qq] 忽略未带交互标签的机器人账号私聊  channel=%s  from=%s",
                    self._channel,
                    user_id,
                )
                return
            text = triggered
            if text.strip() == "/stop":
                self._submit_to_main_loop(self._handle_stop_private(user_id))
                return
            preview = text[:60] + "..." if len(text) > 60 else text
            logger.info(
                f"[qq] 私聊消息  user_id={user_id}  内容: {preview!r}  图片: {len(img_urls)}"
            )

            self.user_map[user_id] = user_id

            self._submit_to_main_loop(self._handle_private(user_id, text, img_urls))

        @cast(Any, self._bot.on_group_message())
        async def _(event) -> None:
            if self._bot_loop is None:
                self._bot_loop = asyncio.get_running_loop()

            group_id = str(event.group_id)
            user_id = str(event.user_id)

            group_cfg = self._groups.get(group_id)
            if group_cfg is None:
                logger.debug(f"[qq] 忽略未配置群  group_id={group_id}")
                return

            raw = strip_at_segments(event.raw_message)
            text, img_urls = _extract_cq_images(raw)
            if getattr(group_cfg, "observe_only", False):
                if group_cfg.allow_from and user_id not in group_cfg.allow_from:
                    logger.debug(
                        "[qq] 观察模式忽略非白名单群成员  group_id=%s  user_id=%s",
                        group_id,
                        user_id,
                    )
                    return
                preview = text[:60] + "..." if len(text) > 60 else text
                logger.info(
                    "[qq] 群观察消息  group_id=%s  user_id=%s  内容: %r  图片: %d",
                    group_id,
                    user_id,
                    preview,
                    len(img_urls),
                )
                self._submit_to_main_loop(
                    self._observe_group(group_id, user_id, text, img_urls, event)
                )
                return

            # 过滤判断（同步包装异步 filter，在 bot loop 里执行）
            future = asyncio.run_coroutine_threadsafe(
                self._group_filter.should_process(event, group_cfg),
                self._require_main_loop(),
            )
            if not future.result(timeout=5):
                return

            if text.strip() == "/stop":
                self._submit_to_main_loop(self._handle_stop_group(group_id, user_id))
                return
            preview = text[:60] + "..." if len(text) > 60 else text
            logger.info(
                f"[qq] 群聊消息  group_id={group_id}  user_id={user_id}  内容: {preview!r}  图片: {len(img_urls)}"
            )

            self._submit_to_main_loop(
                self._handle_group(group_id, user_id, text, img_urls)
            )

        @cast(Any, self._bot.on_notice())
        async def _(event) -> None:
            if self._bot_loop is None:
                self._bot_loop = asyncio.get_running_loop()
            if getattr(event, "notice_type", "") != "group_upload":
                return
            group_id = str(getattr(event, "group_id", "") or "")
            user_id = str(getattr(event, "user_id", "") or "")
            group_cfg = self._groups.get(group_id)
            if group_cfg is None or not getattr(group_cfg, "observe_only", False):
                return
            if group_cfg.allow_from and user_id not in group_cfg.allow_from:
                logger.debug(
                    "[qq] 观察模式忽略非白名单群文件  group_id=%s  user_id=%s",
                    group_id,
                    user_id,
                )
                return
            observed_file = _file_meta_from_notice(event)
            if observed_file is None:
                return
            logger.info(
                "[qq] 群文件观察  group_id=%s  user_id=%s  file=%r  size=%s",
                group_id,
                user_id,
                observed_file.name,
                _format_file_size(observed_file.size),
            )
            self._submit_to_main_loop(
                self._observe_group_file(group_id, user_id, observed_file, event)
            )

        @cast(Any, self._bot.on_startup())
        async def _(_event) -> None:
            self._bot_loop = asyncio.get_running_loop()

        logger.info("[qq] 正在启动 NcatBot（首次运行需要扫码登录）...")
        self._api = await self._main_loop.run_in_executor(None, self._bot.run_backend)
        logger.info("[qq] NcatBot 已启动")

        self._bus.subscribe_outbound(self._channel, self._on_response)

    async def stop(self) -> None:
        if self._api:
            loop = asyncio.get_running_loop()
            bot_exit = getattr(self._bot, "exit", None)
            if callable(bot_exit):
                await loop.run_in_executor(None, bot_exit)
            logger.info("[qq] QQChannel 已停止")

    async def _on_turn_started(self, event: TurnStarted) -> None:
        if event.channel != self._channel:
            return
        self._trace_states[event.session_key] = _QQTraceState(
            user_message=event.content,
        )

    async def _on_tool_call_started(self, event: ToolCallStarted) -> None:
        if event.channel != self._channel:
            return
        state = self._trace_states.setdefault(event.session_key, _QQTraceState())
        state.tool_lines.append(
            _QQTraceLine(
                tool_name=event.tool_name,
                intent=_format_tool_intent(event.arguments),
                target=_format_tool_target(event.arguments),
            )
        )

    async def _on_tool_call_completed(self, event: ToolCallCompleted) -> None:
        if event.channel != self._channel:
            return
        state = self._trace_states.setdefault(event.session_key, _QQTraceState())
        line = next(
            (
                item
                for item in reversed(state.tool_lines)
                if item.tool_name == event.tool_name and item.status == "started"
            ),
            None,
        )
        if line is None:
            line = _QQTraceLine(
                tool_name=event.tool_name,
                intent=_format_tool_intent(event.final_arguments or event.arguments),
                target=_format_tool_target(event.final_arguments or event.arguments),
            )
            state.tool_lines.append(line)
        line.status = "error" if event.status == "error" else "done"
        preview = str(event.result_preview or "").strip()
        if preview:
            line.result_preview = _summarize_tool_result_preview(
                event.tool_name,
                preview,
            )

    # ── 入站处理 ──────────────────────────────────────────────────────

    async def _handle_private(
        self, user_id: str, content: str, img_urls: list[str] | None = None
    ) -> None:
        """私聊入站：chat_id = user_id"""
        await self._identity_index.remember(user_id, user_id)
        media = await _download_to_temp(
            img_urls or [],
            self._http_requester,
            self._attachments,
        )
        await self._bus.publish_inbound(
            InboundMessage(
                channel=self._channel,
                sender=user_id,
                chat_id=user_id,
                content=content,
                media=media,
                metadata={"chat_type": "private"},
            )
        )

    async def _handle_stop_private(self, user_id: str) -> None:
        if self._interrupt_controller is None:
            await self.send(user_id, "当前未启用中断功能。")
            return
        result = self._interrupt_controller.request_interrupt(
            session_key=f"{self._channel}:{user_id}",
            sender=user_id,
            command="/stop",
        )
        await self.send(user_id, result.message)

    async def _handle_group(
        self,
        group_id: str,
        user_id: str,
        content: str,
        img_urls: list[str] | None = None,
    ) -> None:
        """群聊入站：chat_id = gqq:{group_id}，session 按群共享"""
        chat_id = f"{_GROUP_PREFIX}{group_id}"
        session = self._session_manager.get_or_create(f"{self._channel}:{chat_id}")
        if "group_id" not in session.metadata:
            session.metadata["group_id"] = group_id
            await self._session_manager.save_async(session)
        media = await _download_to_temp(
            img_urls or [],
            self._http_requester,
            self._attachments,
        )
        await self._bus.publish_inbound(
            InboundMessage(
                channel=self._channel,
                sender=user_id,
                chat_id=chat_id,
                content=content,
                media=media,
                metadata={
                    "chat_type": "group",
                    "group_id": group_id,
                    "sender_id": user_id,
                },
            )
        )

    async def _observe_group(
        self,
        group_id: str,
        user_id: str,
        content: str,
        img_urls: list[str] | None = None,
        event: Any | None = None,
    ) -> None:
        """群观察模式：只落库，不进入 agent 回复链路。"""
        chat_id = f"{_GROUP_PREFIX}{group_id}"
        session = self._session_manager.get_or_create(f"{self._channel}:{chat_id}")
        metadata_changed = False
        for key, value in {
            "group_id": group_id,
            "chat_type": "group",
            "observe_only": True,
        }.items():
            if session.metadata.get(key) != value:
                session.metadata[key] = value
                metadata_changed = True

        media = await _download_to_temp(
            img_urls or [],
            self._http_requester,
            self._attachments,
        )
        attachment_notes = await self._describe_observed_images(media)
        text = str(content or "").strip()
        image_suffix = f" [图片 x {len(media)}]" if media else ""
        body = f"{text}{image_suffix}".strip() or "[空消息]"
        if attachment_notes:
            body = f"{body}\n" + "\n".join(attachment_notes)
        observed_content = f"[QQ群 {group_id} | {user_id}] {body}"
        extra: dict[str, object] = {
            "chat_type": "group",
            "group_id": group_id,
            "sender_id": user_id,
            "observe_only": True,
        }
        if attachment_notes:
            extra["attachment_summaries"] = attachment_notes
        message_id = getattr(event, "message_id", None) if event is not None else None
        if message_id is not None:
            extra["platform_message_id"] = str(message_id)
        session.add_message(
            "user",
            observed_content,
            media=media if media else None,
            **extra,
        )
        await self._session_manager.save_async(session)
        logger.debug(
            "[qq] 群观察消息已落库  session=%s  metadata_changed=%s",
            session.key,
            metadata_changed,
        )

    async def _observe_group_file(
        self,
        group_id: str,
        user_id: str,
        observed_file: _ObservedFile,
        event: Any | None = None,
    ) -> None:
        """群文件观察模式：记录文件元信息、可下载文件和可检索摘要。"""
        chat_id = f"{_GROUP_PREFIX}{group_id}"
        session = self._session_manager.get_or_create(f"{self._channel}:{chat_id}")
        metadata_changed = False
        for key, value in {
            "group_id": group_id,
            "chat_type": "group",
            "observe_only": True,
        }.items():
            if session.metadata.get(key) != value:
                session.metadata[key] = value
                metadata_changed = True

        if observed_file.file_id:
            try:
                api = self._api
                if api is not None:
                    observed_file.url = str(
                        await self._run_on_bot_loop(
                            api.get_group_file_url(int(group_id), observed_file.file_id)
                        )
                        or ""
                    )
            except Exception as exc:
                logger.warning(
                    "[qq] 获取群文件 URL 失败  group_id=%s  file_id=%s  err=%s",
                    group_id,
                    observed_file.file_id,
                    exc,
                )

        if observed_file.url and (
            observed_file.size is None or observed_file.size <= _MAX_OBSERVE_FILE_BYTES
        ):
            observed_file.local_path = await self._download_observed_file(observed_file)

        summaries: list[str] = []
        if observed_file.local_path:
            if self._is_image_file(observed_file.local_path):
                summaries.extend(await self._describe_observed_images([observed_file.local_path]))
            else:
                summaries.append(_summarize_local_file(observed_file.local_path))
        elif observed_file.size and observed_file.size > _MAX_OBSERVE_FILE_BYTES:
            summaries.append(
                f"文件超过自动下载上限 {_format_file_size(_MAX_OBSERVE_FILE_BYTES)}，暂只记录元信息。"
            )
        elif observed_file.url:
            summaries.append("文件下载失败，暂只记录文件 URL 与元信息。")

        file_line = (
            f"[群文件] {observed_file.name} "
            f"({_format_file_size(observed_file.size)})"
        )
        if observed_file.local_path:
            file_line += f" 本地路径: {observed_file.local_path}"
        if summaries:
            file_line += "\n" + "\n".join(summaries)
        observed_content = f"[QQ群 {group_id} | {user_id}] {file_line}"

        extra: dict[str, object] = {
            "chat_type": "group",
            "group_id": group_id,
            "sender_id": user_id,
            "observe_only": True,
            "attachment_type": "file",
            "file_id": observed_file.file_id,
            "file_name": observed_file.name,
            "file_size": observed_file.size,
            "file_url": observed_file.url,
            "attachment_summaries": summaries,
        }
        message_id = getattr(event, "message_id", None) if event is not None else None
        if message_id is not None:
            extra["platform_message_id"] = str(message_id)
        media = [observed_file.local_path] if observed_file.local_path else None
        session.add_message("user", observed_content, media=media, **extra)
        await self._session_manager.save_async(session)
        logger.debug(
            "[qq] 群文件观察已落库  session=%s  metadata_changed=%s",
            session.key,
            metadata_changed,
        )

    async def _describe_observed_images(self, media: list[str]) -> list[str]:
        if not media or self._vl_provider is None or not self._vl_model:
            return []
        notes: list[str] = []
        for index, path in enumerate(media[:_MAX_OBSERVE_IMAGE_DESCRIPTIONS], start=1):
            if not self._is_image_file(path):
                continue
            try:
                async with self._vision_semaphore:
                    note = await asyncio.wait_for(
                        self._describe_image(path),
                        timeout=_VISION_TIMEOUT_SECONDS,
                    )
            except Exception as exc:
                note = f"图片识别失败：{exc}"
            note = str(note or "").strip()
            if note:
                notes.append(f"[图片{index}识别] {note}")
        omitted = len(media) - _MAX_OBSERVE_IMAGE_DESCRIPTIONS
        if omitted > 0:
            notes.append(f"[图片识别] 另有 {omitted} 张图片未自动识别。")
        return notes

    async def _describe_image(self, path: str) -> str:
        from agent.tools.vision import ReadImageVisionTool

        resolved_path = _resolve_existing_local_path(path, self._workspace)
        workspace = self._workspace.resolve() if self._workspace is not None else None
        allowed_dir = (
            workspace
            if workspace is not None and _path_is_relative_to(resolved_path, workspace)
            else resolved_path.parent
        )
        tool = ReadImageVisionTool(self._vl_provider, self._vl_model, allowed_dir)
        prompt = (
            "请用中文提取这张 QQ 群图片中对长期记忆和 RAG 有价值的信息。"
            "重点识别截图文字、配置、价格、游戏攻略、任务步骤、报错和结论；"
            "如果只是表情包或闲聊图片，简短说明画面即可。控制在 120 字以内。"
        )
        return await tool.execute(str(resolved_path), prompt)

    def _is_image_file(self, path: str) -> bool:
        mime, _ = mimetypes.guess_type(str(path))
        return bool(mime and mime.startswith("image/"))

    async def _download_observed_file(self, observed_file: _ObservedFile) -> str:
        try:
            resp = await self._http_requester.get(
                html.unescape(observed_file.url),
                follow_redirects=True,
                timeout_s=30.0,
                budget=RequestBudget(total_timeout_s=45.0),
            )
            resp.raise_for_status()
            if len(resp.content) > _MAX_OBSERVE_FILE_BYTES:
                logger.info(
                    "[qq] 群文件超过自动保存上限  file=%r  bytes=%d",
                    observed_file.name,
                    len(resp.content),
                )
                return ""
            path = self._attachments.write_bytes(
                resp.content,
                prefix="akashic_qq_file_",
                suffix=_safe_file_suffix(observed_file.name),
            )
            return str(path)
        except Exception as exc:
            logger.warning("[qq] 群文件下载失败  file=%r  err=%s", observed_file.name, exc)
            return ""

    async def _handle_stop_group(self, group_id: str, user_id: str) -> None:
        chat_id = f"{_GROUP_PREFIX}{group_id}"
        if self._interrupt_controller is None:
            await self.send(chat_id, "当前未启用中断功能。")
            return
        result = self._interrupt_controller.request_interrupt(
            session_key=f"{self._channel}:{chat_id}",
            sender=user_id,
            command="/stop",
        )
        await self.send(chat_id, result.message)

    # ── 出站路由 ──────────────────────────────────────────────────────

    async def _on_response(self, msg: OutboundMessage) -> None:
        preview = msg.content[:60] + "..." if len(msg.content) > 60 else msg.content
        api = self._api
        if api is None:
            raise RuntimeError("QQChannel 尚未启动")
        session_key = _session_key_for_chat(msg.chat_id, self._channel)
        if not msg.chat_id.startswith(_GROUP_PREFIX):
            try:
                await self._send_private_trace(msg.chat_id, session_key, msg)
            except Exception as e:
                logger.warning(f"[qq] 私聊 tracing 合并转发失败  chat_id={msg.chat_id}  错误: {e}")
        if msg.content.strip():
            try:
                if msg.chat_id.startswith(_GROUP_PREFIX):
                    group_id = msg.chat_id[len(_GROUP_PREFIX) :]
                    logger.info(f"[qq] 群聊回复  group_id={group_id}  内容: {preview!r}")
                    await self._run_on_bot_loop(
                        api.send_group_text(int(group_id), msg.content)
                    )
                else:
                    logger.info(f"[qq] 私聊回复  user_id={msg.chat_id}  内容: {preview!r}")
                    await self._run_on_bot_loop(
                        api.send_private_text(int(msg.chat_id), msg.content)
                    )
                    self._private_outbound_guard.record(
                        self._bot_uin, msg.chat_id, msg.content
                    )
            except Exception as e:
                logger.error(f"[qq] 发送失败  chat_id={msg.chat_id}  错误: {e}")
        for image in (msg.media or []):
            try:
                await self.send_image(msg.chat_id, image)
            except Exception as e:
                logger.error(f"[qq] meme 图片发送失败  chat_id={msg.chat_id}  path={image}  err={e}")
        self._trace_states.pop(session_key, None)

    async def _send_private_trace(
        self,
        chat_id: str,
        session_key: str,
        msg: OutboundMessage,
    ) -> None:
        api = self._api
        if api is None:
            raise RuntimeError("QQChannel 尚未启动")
        trace = self._trace_states.get(session_key)
        if trace is None:
            return
        thinking_source = str(msg.thinking or "")
        thinking = _truncate_trace_text(thinking_source, _TRACE_THINKING_LIMIT)
        tool_text = _format_tool_trace_lines(trace.tool_lines)
        if not thinking and not trace.tool_lines:
            return
        from ncatbot.core import ForwardConstructor

        info = await self._run_on_bot_loop(api.get_login_info())
        actor_name = self._trace_actor_name()
        constructor = ForwardConstructor(str(info.user_id), actor_name)
        constructor.attach_text(
            f"【模型思路】\n{thinking or '（无 thinking）'}",
            nickname=actor_name,
        )
        constructor.attach_text(
            f"【工具链】\n{tool_text}",
            nickname=actor_name,
        )
        forward = constructor.to_forward()
        payload = forward.to_forward_dict()
        payload["source"] = f"{actor_name} 的过程记录"
        payload["summary"] = "查看本轮过程记录"
        payload["prompt"] = f"{actor_name} 过程记录"
        payload["news"] = [
            {"text": f"{actor_name}：【模型思路】"},
            {"text": f"{actor_name}：【工具链】"},
        ]
        await self._run_on_bot_loop(
            api.send_private_forward_msg(int(chat_id), **payload)
        )
        self._private_outbound_guard.record(
            self._bot_uin, chat_id, OUTBOUND_FORWARD_MARKER
        )

    def _trace_actor_name(self) -> str:
        cached = self._trace_actor_name_cache
        if cached:
            return cached
        workspace = self._workspace
        if workspace is None:
            self._trace_actor_name_cache = _TRACE_DEFAULT_ACTOR
            return _TRACE_DEFAULT_ACTOR
        self_path = workspace / "memory" / "SELF.md"
        try:
            text = self_path.read_text(encoding="utf-8")
        except Exception:
            self._trace_actor_name_cache = _TRACE_DEFAULT_ACTOR
            return _TRACE_DEFAULT_ACTOR
        body_match = re.search(
            r"(?m)^-\s*我是\s+([A-Za-z][A-Za-z0-9_-]{1,40})\b",
            text,
        )
        if body_match:
            name = body_match.group(1).strip()
            if name:
                self._trace_actor_name_cache = name
                return name
        match = re.search(r"(?m)^#\s*(.+?)\s+的自我认知\s*$", text)
        if match:
            name = match.group(1).strip()
            if name:
                self._trace_actor_name_cache = name
                return name
        self._trace_actor_name_cache = _TRACE_DEFAULT_ACTOR
        return _TRACE_DEFAULT_ACTOR

    # ── 主动推送（供 MessagePushTool 使用）────────────────────────────

    async def send(self, chat_id: str, message: str) -> None:
        """发送文本消息，自动区分私聊/群聊"""
        api = self._api
        if api is None:
            raise RuntimeError("QQChannel 尚未启动")
        if chat_id.startswith(_GROUP_PREFIX):
            group_id = chat_id[len(_GROUP_PREFIX) :]
            await self._run_on_bot_loop(api.send_group_text(int(group_id), message))
        else:
            await self._run_on_bot_loop(api.send_private_text(int(chat_id), message))
            self._private_outbound_guard.record(self._bot_uin, chat_id, message)

    async def send_file(
        self, chat_id: str, file_path: str, name: str | None = None
    ) -> None:
        """发送文件，自动区分私聊/群聊"""
        api = self._api
        if api is None:
            raise RuntimeError("QQChannel 尚未启动")
        uri = _local_to_base64(file_path) if _is_local(file_path) else file_path
        if chat_id.startswith(_GROUP_PREFIX):
            group_id = chat_id[len(_GROUP_PREFIX) :]
            await self._run_on_bot_loop(api.send_group_file(int(group_id), uri, name))
        else:
            await self._run_on_bot_loop(api.send_private_file(int(chat_id), uri, name))
            self._private_outbound_guard.record(
                self._bot_uin, chat_id, OUTBOUND_FILE_MARKER
            )

    async def send_image(self, chat_id: str, image: str) -> None:
        """发送图片，自动区分私聊/群聊"""
        api = self._api
        if api is None:
            raise RuntimeError("QQChannel 尚未启动")
        uri = _local_to_base64(image) if _is_local(image) else image
        if chat_id.startswith(_GROUP_PREFIX):
            group_id = chat_id[len(_GROUP_PREFIX) :]
            await self._run_on_bot_loop(api.send_group_image(int(group_id), uri))
        else:
            await self._run_on_bot_loop(api.send_private_image(int(chat_id), uri))
            self._private_outbound_guard.record(
                self._bot_uin, chat_id, OUTBOUND_IMAGE_MARKER
            )

    def _require_main_loop(self) -> asyncio.AbstractEventLoop:
        if self._main_loop is None:
            raise RuntimeError("QQ main loop 未就绪")
        return self._main_loop

    def _submit_to_main_loop(self, coro: Coroutine[object, object, None]) -> None:
        asyncio.run_coroutine_threadsafe(coro, self._require_main_loop())

    async def _run_on_bot_loop(
        self, coro: Coroutine[object, object, object]
    ) -> object:
        if self._bot_loop is None:
            raise RuntimeError("QQ bot loop 未就绪")
        future = asyncio.run_coroutine_threadsafe(coro, self._bot_loop)
        return await asyncio.wrap_future(future)


def _is_local(path: str) -> bool:
    """判断是否为本地文件路径（非 URL、非 base64）"""
    return not path.startswith(("http://", "https://", "base64://", "file://"))


def _local_to_base64(path: str) -> str:
    """将本地文件编码为 NapCat 接受的 base64:// URI"""
    data = Path(path).read_bytes()
    return "base64://" + base64.b64encode(data).decode()
