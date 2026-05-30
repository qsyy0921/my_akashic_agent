from __future__ import annotations
from typing import Any, cast

import json
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest

from agent.config import Config, DEFAULT_SOCKET
from agent.config_models import Config as ConfigModel, WiringConfig
from agent.lifecycle.facade import TurnLifecycle
from agent.lifecycle.types import AfterStepCtx
from agent.looping.interrupt import TurnInterruptState
from agent.tools.registry import ToolRegistry
from bootstrap.tools import _build_loop_deps, build_registered_tools
from bootstrap.wiring import (
    wire_turn_lifecycle,
    register_memory_plugin,
    resolve_context_factory,
    resolve_memory_plugin,
    resolve_memory_toolset_provider,
    resolve_toolset_provider,
)
from bus.event_bus import EventBus


def _toml_value(value):
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, str):
        return json.dumps(value, ensure_ascii=False)
    if isinstance(value, list):
        return "[" + ", ".join(_toml_value(item) for item in value) + "]"
    return str(value)


def _dump_toml(data: dict, prefix: tuple[str, ...] = ()) -> list[str]:
    lines: list[str] = []
    scalar_lines: list[str] = []

    for key, value in data.items():
        if isinstance(value, dict):
            continue
        if (
            isinstance(value, list)
            and value
            and all(isinstance(item, dict) for item in value)
        ):
            continue
        scalar_lines.append(f"{key} = {_toml_value(value)}")

    if prefix:
        lines.append(f"[{'.'.join(prefix)}]")
    lines.extend(scalar_lines)
    if scalar_lines:
        lines.append("")

    for key, value in data.items():
        if isinstance(value, dict):
            lines.extend(_dump_toml(value, prefix + (key,)))
        elif (
            isinstance(value, list)
            and value
            and all(isinstance(item, dict) for item in value)
        ):
            for item in value:
                lines.append(f"[[{'.'.join(prefix + (key,))}]]")
                for item_key, item_value in item.items():
                    lines.append(f"{item_key} = {_toml_value(item_value)}")
                lines.append("")
    return lines


def _write_toml(path: Path, payload: dict) -> None:
    path.write_text("\n".join(_dump_toml(payload)).strip() + "\n", encoding="utf-8")


def test_config_load_reads_wiring_block(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
                "wiring": {
                    "context": "default",
                    "memory": "default",
                    "memory_engine": "default",
                    "toolsets": ["schedule", "mcp"],
                },
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.wiring.context == "default"
    assert cfg.wiring.memory == "default"
    assert cfg.wiring.toolsets == ["schedule", "mcp"]


def test_config_load_reads_memory_engine_selector(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {"system_prompt": "s"},
            "memory": {
                "enabled": True,
                "engine": "memu",
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.memory.enabled is True
    assert cfg.memory.engine == "memu"


def test_config_load_ignores_wiring_memory_engine(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
                "wiring": {
                    "memory_engine": "memu",
                },
            },
            "memory": {
                "enabled": True,
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.memory.enabled is True
    assert cfg.memory.engine == ""


def test_config_load_ignores_legacy_memory_v2_enabled(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {"system_prompt": "s"},
            "memory_v2": {
                "enabled": True,
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert not hasattr(cfg, "memory_v2")
    assert cfg.memory.enabled is False
    assert cfg.memory.engine == ""


def test_config_load_reads_embedding_and_ignores_private_memory_sections(
    tmp_path: Path,
):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {"system_prompt": "s"},
            "memory": {
                "enabled": True,
                "engine": "",
                "embedding": {
                    "model": "legacy-embedding",
                    "api_key": "legacy-key",
                },
                "retrieval": {
                    "score_threshold": 0.99,
                    "thresholds": {"event": 0.99},
                },
                "hyde": {"enabled": True},
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.memory.enabled is True
    assert cfg.memory.engine == ""
    assert cfg.memory.embedding.model == "legacy-embedding"
    assert cfg.memory.embedding.api_key == "legacy-key"


def test_config_load_reads_memory_window_and_socket(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
                "context": {
                    "memory_window": 20,
                },
            },
            "channels": {
                "socket": "/tmp/dev-akashic.sock",
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.memory_window == 20


def test_config_load_reads_agent_dev_mode(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
                "dev_mode": True,
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.dev_mode is True


def test_config_load_accepts_dev_model_alias(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
                "dev_model": True,
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.dev_mode is True


def test_config_load_skips_unfilled_channels(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
):
    monkeypatch.delenv("TELEGRAM_BOT_TOKEN", raising=False)
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "channels": {
                "telegram": {
                    "token": "${TELEGRAM_BOT_TOKEN}",
                    "allow_from": ["user1"],
                },
                "qq": {
                    "bot_uin": "",
                    "allow_from": ["42"],
                },
                "qqbot": {
                    "app_id": "app",
                    "client_secret": "${QQBOT_SECRET}",
                    "allow_from": ["user-openid"],
                    "groups": [
                        {
                            "group_openid": "group-openid",
                            "allow_from": ["member-openid"],
                            "require_at": True,
                            "allow_proactive": True,
                        }
                    ],
                },
            },
        },
    )

    monkeypatch.setenv("QQBOT_SECRET", "secret")
    cfg = Config.load(cfg_path)

    assert cfg.channels.telegram is None
    assert cfg.channels.qq is None
    assert cfg.channels.qqbot is not None
    assert cfg.channels.qqbot.app_id == "app"
    assert cfg.channels.qqbot.client_secret == "secret"
    assert cfg.channels.qqbot.allow_from == ["user-openid"]
    assert cfg.channels.qqbot.groups[0].group_openid == "group-openid"
    assert cfg.channels.qqbot.groups[0].allow_from == ["member-openid"]
    assert cfg.channels.qqbot.groups[0].require_at is True
    assert cfg.channels.qqbot.groups[0].allow_proactive is True
    assert cfg.channels.socket == DEFAULT_SOCKET


def test_config_load_reads_fitbit_integration_block(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "integrations": {
                "fitbit": {
                    "enabled": True,
                }
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.fitbit.enabled is True


def test_config_load_reads_chatgpt_proxy_integration_block(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "integrations": {
                "chatgpt_proxy": {
                    "enabled": True,
                    "base_url": "${CHATGPT_PROXY_BASE_URL}",
                    "api_key": "${CHATGPT_PROXY_API_KEY}",
                    "image_model": "gpt-image-test",
                    "image_path": "/v1/images/generations",
                    "response_format": "url",
                    "output_dir": "images",
                }
            },
        },
    )
    monkeypatch.setenv("CHATGPT_PROXY_BASE_URL", "http://127.0.0.1:8000")
    monkeypatch.setenv("CHATGPT_PROXY_API_KEY", "proxy-key")

    cfg = Config.load(cfg_path)

    assert cfg.chatgpt_proxy.enabled is True
    assert cfg.chatgpt_proxy.base_url == "http://127.0.0.1:8000"
    assert cfg.chatgpt_proxy.api_key == "proxy-key"
    assert cfg.chatgpt_proxy.image_model == "gpt-image-test"
    assert cfg.chatgpt_proxy.image_path == "/v1/images/generations"
    assert cfg.chatgpt_proxy.response_format == "url"
    assert cfg.chatgpt_proxy.output_dir == "images"


def test_config_load_reads_ragflow_integration_block(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "integrations": {
                "ragflow": {
                    "enabled": True,
                    "base_url": "http://127.0.0.1:9380",
                    "api_key": "${RAGFLOW_API_KEY}",
                    "proxy_url": "http://127.0.0.1:7897",
                    "default_dataset_ids": "ds1,ds2",
                    "default_keyword": True,
                    "default_use_kg": True,
                    "default_top_k": 12,
                    "default_page_size": 6,
                    "default_similarity_threshold": 0.25,
                    "default_vector_similarity_weight": 0.45,
                    "request_timeout_seconds": 70,
                }
            },
        },
    )
    monkeypatch.setenv("RAGFLOW_API_KEY", "ragflow-key")

    cfg = Config.load(cfg_path)

    assert cfg.ragflow.enabled is True
    assert cfg.ragflow.base_url == "http://127.0.0.1:9380"
    assert cfg.ragflow.api_key == "ragflow-key"
    assert cfg.ragflow.proxy_url == "http://127.0.0.1:7897"
    assert cfg.ragflow.default_dataset_ids == ["ds1", "ds2"]
    assert cfg.ragflow.default_keyword is True
    assert cfg.ragflow.default_use_kg is True
    assert cfg.ragflow.default_top_k == 12
    assert cfg.ragflow.default_page_size == 6
    assert cfg.ragflow.default_similarity_threshold == 0.25
    assert cfg.ragflow.default_vector_similarity_weight == 0.45
    assert cfg.ragflow.request_timeout_seconds == 70


def test_config_load_reads_shadow_runtime_integration_block_with_compatibility(
    tmp_path: Path,
):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "integrations": {
                "shadow_runtime": {
                    "enabled": True,
                    "endpoint": "http://127.0.0.1:9898/v1/shadow/inbound",
                    "log_path": "shadow/runtime.jsonl",
                    "request_timeout_seconds": 7,
                    "agent_id": "runtime-shadow",
                }
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.shadow_gateway.enabled is True
    assert cfg.shadow_gateway.endpoint == "http://127.0.0.1:9898/v1/shadow/inbound"
    assert cfg.shadow_gateway.log_path == "shadow/runtime.jsonl"
    assert cfg.shadow_gateway.request_timeout_seconds == 7
    assert cfg.shadow_gateway.agent_id == "runtime-shadow"
    assert cfg.shadow_runtime is cfg.shadow_gateway


def test_config_load_reads_agent_gateway_integration_block(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "integrations": {
                "agent_gateway": {
                    "enabled": True,
                    "base_url": "http://127.0.0.1:8780",
                    "request_timeout_seconds": 9,
                    "worker_id": "worker-test",
                    "lease_ttl_seconds": 180,
                    "poll_interval_seconds": 3,
                    "knowledge_job_interval_seconds": 30,
                    "outbox_worker_enabled": True,
                }
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.agent_gateway.enabled is True
    assert cfg.agent_gateway.base_url == "http://127.0.0.1:8780"
    assert cfg.agent_gateway.request_timeout_seconds == 9
    assert cfg.agent_gateway.worker_id == "worker-test"
    assert cfg.agent_gateway.lease_ttl_seconds == 180
    assert cfg.agent_gateway.poll_interval_seconds == 3
    assert cfg.agent_gateway.knowledge_job_interval_seconds == 30
    assert cfg.agent_gateway.outbox_worker_enabled is True


def test_config_load_reads_agent_runtime_integration_block_with_compatibility(
    tmp_path: Path,
):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "integrations": {
                "agent_runtime": {
                    "enabled": True,
                    "base_url": "http://127.0.0.1:9898",
                    "request_timeout_seconds": 11,
                    "worker_id": "runtime-worker",
                    "lease_ttl_seconds": 99,
                    "poll_interval_seconds": 4,
                    "knowledge_job_interval_seconds": 45,
                    "outbox_worker_enabled": True,
                }
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.agent_gateway.enabled is True
    assert cfg.agent_gateway.base_url == "http://127.0.0.1:9898"
    assert cfg.agent_gateway.request_timeout_seconds == 11
    assert cfg.agent_gateway.worker_id == "runtime-worker"
    assert cfg.agent_gateway.lease_ttl_seconds == 99
    assert cfg.agent_gateway.poll_interval_seconds == 4
    assert cfg.agent_gateway.knowledge_job_interval_seconds == 45
    assert cfg.agent_gateway.outbox_worker_enabled is True
    assert cfg.agent_runtime is cfg.agent_gateway


def test_config_load_reads_toml_layout(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    cfg_path.write_text(
        """
[llm]
provider = "openai"

[llm.main]
model = "m"
api_key = "k"

[agent]
system_prompt = "s"
max_tokens = 256

[agent.context]
memory_window = 12

[channels]
socket = "/tmp/toml-akashic.sock"

[integrations.fitbit]
enabled = true
""".strip() + "\n",
        encoding="utf-8",
    )

    cfg = Config.load(cfg_path)

    assert cfg.provider == "openai"
    assert cfg.model == "m"
    assert cfg.max_tokens == 256
    assert cfg.memory_window == 12
    if sys.platform == "win32":
        assert cfg.channels.socket != "/tmp/toml-akashic.sock"
        assert cfg.channels.socket.startswith("127.0.0.1:")
    else:
        assert cfg.channels.socket == "/tmp/toml-akashic.sock"
    assert cfg.fitbit.enabled is True


def test_config_load_reads_qq_websocket_timeout(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "channels": {
                "qq": {
                    "bot_uin": "10001",
                    "allow_from": ["42"],
                    "websocket_open_timeout_seconds": 9.5,
                },
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.channels.qq is not None
    assert cfg.channels.qq.websocket_open_timeout_seconds == 9.5


def test_config_load_reads_qq_secondary_accounts(tmp_path: Path):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "channels": {
                "qq": {
                    "bot_uin": "1049511700",
                    "allow_from": ["1049511700"],
                    "channel_name": "qq",
                    "websocket_uri": "ws://localhost:3001",
                    "accounts": [
                        {
                            "bot_uin": "2365524513",
                            "allow_from": ["1049511700"],
                            "channel_name": "qq_2365524513",
                            "websocket_uri": "ws://localhost:3002",
                        }
                    ],
                },
            },
        },
    )

    cfg = Config.load(cfg_path)

    assert cfg.channels.qq is not None
    assert cfg.channels.qq.bot_uin == "1049511700"
    assert cfg.channels.qq.channel_name == "qq"
    assert cfg.channels.qq.websocket_uri == "ws://localhost:3001"
    assert len(cfg.channels.qq_accounts) == 1
    account = cfg.channels.qq_accounts[0]
    assert account.bot_uin == "2365524513"
    assert account.allow_from == ["1049511700"]
    assert account.channel_name == "qq_2365524513"
    assert account.websocket_uri == "ws://localhost:3002"


def test_channel_config_resolves_env_placeholders(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
):
    cfg_path = tmp_path / "config.toml"
    _write_toml(
        cfg_path,
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "channels": {
                "telegram": {
                    "token": "${TELEGRAM_BOT_TOKEN}",
                    "allow_from": ["${TELEGRAM_ALLOW_FROM}"],
                },
                "feishu": {
                    "webhook_url": "${FEISHU_WEBHOOK_URL}",
                    "secret": "${FEISHU_WEBHOOK_SECRET}",
                    "channel_name": "feishu_work",
                },
                "wechat": {
                    "webhook_url": "${WECHAT_WEBHOOK_URL}",
                    "mentioned_list": ["${WECHAT_MENTIONED_LIST}"],
                    "mentioned_mobile_list": ["${WECHAT_MENTIONED_MOBILE_LIST}"],
                    "channel_name": "wechat_work",
                },
                "qq": {
                    "bot_uin": "${QQ_BOT_UIN}",
                    "allow_from": ["${QQ_ALLOW_FROM}"],
                    "groups": [
                        {
                            "group_id": "${QQ_GROUP_ID}",
                            "allow_from": ["${QQ_GROUP_ALLOW_FROM}"],
                            "observe_only": True,
                        }
                    ],
                },
                "qqbot": {
                    "app_id": "${QQBOT_APP_ID}",
                    "client_secret": "${QQBOT_SECRET}",
                    "allow_from": ["${QQBOT_ALLOW_FROM}"],
                    "groups": [
                        {
                            "group_openid": "${QQBOT_GROUP_OPENID}",
                            "allow_from": ["${QQBOT_GROUP_ALLOW_FROM}"],
                        }
                    ],
                },
            },
        },
    )
    monkeypatch.setenv("TELEGRAM_BOT_TOKEN", "tg-token")
    monkeypatch.setenv("TELEGRAM_ALLOW_FROM", "alice,123")
    monkeypatch.setenv(
        "FEISHU_WEBHOOK_URL",
        "https://open.feishu.cn/open-apis/bot/v2/hook/test",
    )
    monkeypatch.setenv("FEISHU_WEBHOOK_SECRET", "fs-secret")
    monkeypatch.setenv(
        "WECHAT_WEBHOOK_URL",
        "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
    )
    monkeypatch.setenv("WECHAT_MENTIONED_LIST", "@all,user1")
    monkeypatch.setenv("WECHAT_MENTIONED_MOBILE_LIST", "13800138000")
    monkeypatch.setenv("QQ_BOT_UIN", "10001")
    monkeypatch.setenv("QQ_ALLOW_FROM", "42,43")
    monkeypatch.setenv("QQ_GROUP_ID", "20002")
    monkeypatch.setenv("QQ_GROUP_ALLOW_FROM", "44")
    monkeypatch.setenv("QQBOT_APP_ID", "app")
    monkeypatch.setenv("QQBOT_SECRET", "secret")
    monkeypatch.setenv("QQBOT_ALLOW_FROM", "openid-1,openid-2")
    monkeypatch.setenv("QQBOT_GROUP_OPENID", "group-openid")
    monkeypatch.setenv("QQBOT_GROUP_ALLOW_FROM", "member-openid")

    cfg = Config.load(cfg_path)

    assert cfg.channels.telegram is not None
    assert cfg.channels.telegram.token == "tg-token"
    assert cfg.channels.telegram.allow_from == ["alice", "123"]
    assert cfg.channels.feishu is not None
    assert (
        cfg.channels.feishu.webhook_url
        == "https://open.feishu.cn/open-apis/bot/v2/hook/test"
    )
    assert cfg.channels.feishu.secret == "fs-secret"
    assert cfg.channels.feishu.channel_name == "feishu_work"
    assert cfg.channels.wechat is not None
    assert (
        cfg.channels.wechat.webhook_url
        == "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test"
    )
    assert cfg.channels.wechat.mentioned_list == ["@all", "user1"]
    assert cfg.channels.wechat.mentioned_mobile_list == ["13800138000"]
    assert cfg.channels.wechat.channel_name == "wechat_work"
    assert cfg.channels.qq is not None
    assert cfg.channels.qq.bot_uin == "10001"
    assert cfg.channels.qq.allow_from == ["42", "43"]
    assert cfg.channels.qq.groups[0].group_id == "20002"
    assert cfg.channels.qq.groups[0].allow_from == ["44"]
    assert cfg.channels.qq.groups[0].observe_only is True
    assert cfg.channels.qqbot is not None
    assert cfg.channels.qqbot.app_id == "app"
    assert cfg.channels.qqbot.client_secret == "secret"
    assert cfg.channels.qqbot.allow_from == ["openid-1", "openid-2"]
    assert cfg.channels.qqbot.groups[0].group_openid == "group-openid"
    assert cfg.channels.qqbot.groups[0].allow_from == ["member-openid"]


def test_build_registered_tools_respects_toolset_order_and_subset(
    monkeypatch, tmp_path: Path
):
    calls: list[str] = []

    class _MemoryProvider:
        def register(self, registry, deps):
            calls.append("memory")
            runtime = SimpleNamespace(engine=object())
            return SimpleNamespace(extras={"memory_runtime": runtime})

    class _ToolsetProvider:
        def __init__(self, name: str) -> None:
            self._name = name

        def register(self, registry, deps):
            calls.append(self._name)
            extras = {"mcp_registry": object()} if self._name == "mcp" else {}
            return SimpleNamespace(extras=extras)

    monkeypatch.setattr(
        "bootstrap.tools.resolve_memory_toolset_provider",
        lambda name: _MemoryProvider(),
    )
    monkeypatch.setattr(
        "bootstrap.tools.resolve_toolset_provider",
        lambda name, readonly_tools=None: _ToolsetProvider(name),
    )
    monkeypatch.setattr("bootstrap.tools.build_readonly_tools", lambda *_, **__: {})
    monkeypatch.setattr(
        "bootstrap.tools.build_scheduler",
        lambda *_args, **_kwargs: SimpleNamespace(),
    )
    monkeypatch.setattr(
        "bootstrap.tools.build_peer_agent_resources",
        lambda *_args, **_kwargs: (None, None),
    )

    config = ConfigModel(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        wiring=WiringConfig(toolsets=["schedule", "mcp"]),
    )
    build_registered_tools(
        config=config,
        workspace=tmp_path,
        http_resources=cast(Any, SimpleNamespace()),
        bus=cast(Any, SimpleNamespace()),
        provider=object(),
        light_provider=object(),
        session_store=object(),
        tools=ToolRegistry(),
        event_publisher=EventBus(),
        agent_loop_provider=lambda: None,
    )

    assert calls == ["memory", "schedule", "mcp"]


def test_build_loop_deps_uses_context_factory(monkeypatch, tmp_path: Path):
    observed: dict[str, object] = {}
    fake_context = object()
    markdown_store = object()
    markdown_maintenance = SimpleNamespace(bind_lifecycle=lambda request: None)

    monkeypatch.setattr(
        "bootstrap.tools.resolve_context_factory",
        lambda name: (
            lambda workspace, memory_store: observed.update(
                {"name": name, "workspace": workspace, "memory_store": memory_store}
            )
            or fake_context
        ),
    )

    config = ConfigModel(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        wiring=WiringConfig(context="default"),
    )
    deps = _build_loop_deps(
        config=config,
        workspace=tmp_path,
        bus=cast(Any, SimpleNamespace()),
        provider=cast(Any, object()),
        light_provider=None,
        tools=ToolRegistry(),
        session_manager=cast(
            Any,
            SimpleNamespace(
                get_or_create=lambda key: None,
                save_async=lambda session: None,
            ),
        ),
        presence=cast(Any, None),
        processing_state=cast(Any, SimpleNamespace()),
        event_bus=EventBus(),
        memory_runtime=cast(
            Any,
            SimpleNamespace(
                engine=object(),
                markdown=SimpleNamespace(
                    store=markdown_store,
                    maintenance=markdown_maintenance,
                ),
            ),
        ),
    )

    assert observed["name"] == "default"
    assert observed["workspace"] == tmp_path
    assert observed["memory_store"] is markdown_store
    assert deps.context is fake_context


def test_wiring_error_messages_list_available_choices():
    try:
        resolve_context_factory("bad")
    except ValueError as exc:
        assert "可选值" in str(exc)
        assert "default" in str(exc)
    else:
        raise AssertionError("resolve_context_factory should fail for bad name")

    try:
        resolve_memory_toolset_provider("bad")
    except ValueError as exc:
        assert "可选值" in str(exc)
        assert "default" in str(exc)
    else:
        raise AssertionError("resolve_memory_toolset_provider should fail for bad name")

    try:
        resolve_memory_plugin("bad")
    except ValueError as exc:
        assert "可选值" in str(exc)
        assert "default" in str(exc)
    else:
        raise AssertionError("resolve_memory_plugin should fail for bad name")

    try:
        resolve_toolset_provider("bad")
    except ValueError as exc:
        assert "可选值" in str(exc)
        assert "meta_common" in str(exc)
    else:
        raise AssertionError("resolve_toolset_provider should fail for bad name")


def test_memory_plugin_registry_accepts_custom_engine(monkeypatch):
    class _Plugin:
        plugin_id = "custom"

        def build(self, deps):
            raise AssertionError("not used")

    register_memory_plugin("custom", lambda: _Plugin())

    assert resolve_memory_plugin("custom").plugin_id == "custom"


def test_memory_plugin_resolver_loads_plugin_directory(monkeypatch, tmp_path: Path):
    plugin_dir = tmp_path / "plugins" / "demo_memory"
    plugin_dir.mkdir(parents=True)
    (plugin_dir / "memory_plugin.py").write_text(
        "\n".join(
            [
                "from core.memory.plugin import MemoryPluginRuntime",
                "",
                "class MemoryPlugin:",
                "    plugin_id = 'demo_memory'",
                "    def build(self, deps):",
                "        raise AssertionError('not used')",
            ]
        ),
        encoding="utf-8",
    )
    import bootstrap.wiring as wiring

    monkeypatch.setattr(wiring, "_PROJECT_ROOT", tmp_path)

    assert resolve_memory_plugin("demo_memory").plugin_id == "demo_memory"


@pytest.mark.asyncio
async def test_wire_turn_lifecycle_registers_afterstep_progress_handler():
    bus = EventBus()
    states: dict[str, TurnInterruptState] = {
        "telegram:1": TurnInterruptState(
            session_key="telegram:1",
            original_user_message="hello",
        )
    }
    wire_turn_lifecycle(
        lifecycle=TurnLifecycle(bus),
        active_turn_states=states,
    )

    await bus.emit(
        AfterStepCtx(
            session_key="telegram:1",
            channel="telegram",
            chat_id="1",
            iteration=0,
            context_tokens_estimate=0,
            tools_called=("noop",),
            partial_reply="部分回复",
            tools_used_so_far=("a", "b"),
            tool_chain_partial=({"text": "tool", "calls": []},),
            partial_thinking="思考",
            has_more=True,
        )
    )

    state = states["telegram:1"]
    assert state.partial_reply == "部分回复"
    assert state.partial_thinking == "思考"
    assert state.tools_used == ["a", "b"]
    assert state.tool_chain_partial == [{"text": "tool", "calls": []}]


def test_build_registered_tools_without_mcp_toolset_still_returns_empty_registry(
    monkeypatch, tmp_path: Path
):
    monkeypatch.setattr(
        "bootstrap.tools.resolve_memory_toolset_provider",
        lambda name: SimpleNamespace(
            register=lambda registry, deps: SimpleNamespace(
                extras={"memory_runtime": SimpleNamespace(engine=object())}
            )
        ),
    )
    monkeypatch.setattr(
        "bootstrap.tools.resolve_toolset_provider",
        lambda name, readonly_tools=None: SimpleNamespace(
            register=lambda registry, deps: SimpleNamespace(extras={})
        ),
    )
    monkeypatch.setattr("bootstrap.tools.build_readonly_tools", lambda *_, **__: {})
    monkeypatch.setattr(
        "bootstrap.tools.build_scheduler",
        lambda *_args, **_kwargs: SimpleNamespace(),
    )
    monkeypatch.setattr(
        "bootstrap.tools.build_peer_agent_resources",
        lambda *_args, **_kwargs: (None, None),
    )

    config = ConfigModel(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        wiring=WiringConfig(toolsets=["schedule"]),
    )
    _, _, _, mcp_registry, _, _, _ = build_registered_tools(
        config=config,
        workspace=tmp_path,
        http_resources=cast(Any, SimpleNamespace()),
        bus=cast(Any, SimpleNamespace()),
        provider=object(),
        light_provider=object(),
        session_store=object(),
        tools=ToolRegistry(),
        event_publisher=EventBus(),
        agent_loop_provider=lambda: None,
    )

    assert mcp_registry is not None
    assert mcp_registry.list_servers() == "当前没有已注册的 MCP server。"


def test_bootstrap_disables_group_memory_loop_when_agent_gateway_enabled(
    tmp_path: Path,
):
    from bootstrap.app import (
        _build_agent_gateway_knowledge_worker_tasks,
        _build_group_memory_tasks,
    )
    from agent.config_models import (
        AgentGatewayIntegrationConfig,
        Config,
        ChannelsConfig,
        QQChannelConfig,
        QQGroupConfig,
    )

    config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        channels=ChannelsConfig(
            qq=QQChannelConfig(
                bot_uin="1049511700",
                groups=[QQGroupConfig(group_id="284331268", observe_only=True)],
            )
        ),
        agent_gateway=AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="http://127.0.0.1:8780",
            request_timeout_seconds=5,
            knowledge_job_interval_seconds=60,
        ),
    )

    legacy_tasks, legacy_loop = _build_group_memory_tasks(
        config,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    assert legacy_tasks == []
    assert legacy_loop is None

    knowledge_tasks, knowledge_worker = _build_agent_gateway_knowledge_worker_tasks(
        config,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    for task in knowledge_tasks:
        task.close()
    assert len(knowledge_tasks) == 1
    assert knowledge_worker is not None
    assert (
        type(getattr(knowledge_worker._group_memory, "_source")).__name__
        == "AgentRuntimeInboxGroupMessageSource"
    )


def test_bootstrap_disables_group_memory_loop_when_agent_runtime_enabled(
    tmp_path: Path,
):
    from bootstrap.app import (
        _build_agent_gateway_knowledge_worker_tasks,
        _build_group_memory_tasks,
    )
    from agent.config import Config

    _write_toml(
        tmp_path / "config.toml",
        {
            "llm": {
                "provider": "openai",
                "main": {
                    "model": "m",
                    "api_key": "k",
                },
            },
            "agent": {
                "system_prompt": "s",
            },
            "channels": {
                "qq": {
                    "bot_uin": "1049511700",
                    "groups": [
                        {
                            "group_id": "284331268",
                            "observe_only": True,
                        }
                    ],
                },
            },
            "integrations": {
                "agent_runtime": {
                    "enabled": True,
                    "base_url": "http://127.0.0.1:8780",
                    "request_timeout_seconds": 5,
                    "knowledge_job_interval_seconds": 60,
                }
            },
        },
    )
    cfg = Config.load(tmp_path / "config.toml")

    legacy_tasks, legacy_loop = _build_group_memory_tasks(
        cfg,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    assert legacy_tasks == []
    assert legacy_loop is None

    knowledge_tasks, knowledge_worker = _build_agent_gateway_knowledge_worker_tasks(
        cfg,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    for task in knowledge_tasks:
        task.close()
    assert len(knowledge_tasks) == 1
    assert knowledge_worker is not None
    assert (
        type(getattr(knowledge_worker._group_memory, "_source")).__name__
        == "AgentRuntimeInboxGroupMessageSource"
    )


def test_bootstrap_runtime_worker_entry_points_are_first_class(
    tmp_path: Path,
):
    from bootstrap.app import (
        _build_agent_runtime_knowledge_worker_tasks,
        _build_agent_gateway_knowledge_worker_tasks,
    )
    from agent.config_models import (
        AgentGatewayIntegrationConfig,
        Config,
        ChannelsConfig,
        QQChannelConfig,
        QQGroupConfig,
    )

    config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        channels=ChannelsConfig(
            qq=QQChannelConfig(
                bot_uin="1049511700",
                groups=[QQGroupConfig(group_id="284331268", observe_only=True)],
            )
        ),
        agent_gateway=AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="http://127.0.0.1:8780",
            request_timeout_seconds=5,
            knowledge_job_interval_seconds=60,
        ),
    )

    runtime_tasks, runtime_worker = _build_agent_runtime_knowledge_worker_tasks(
        config,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    legacy_tasks, legacy_worker = _build_agent_gateway_knowledge_worker_tasks(
        config,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    for task in runtime_tasks:
        task.close()
    for task in legacy_tasks:
        task.close()
    assert len(runtime_tasks) == 1
    assert len(legacy_tasks) == 1
    assert runtime_worker is not None
    assert legacy_worker is not None


def test_bootstrap_runtime_outbox_worker_is_opt_in():
    from bootstrap.app import (
        _build_agent_gateway_outbox_worker_tasks,
        _build_agent_runtime_outbox_worker_tasks,
    )
    from agent.config_models import (
        AgentGatewayIntegrationConfig,
        Config,
        ChannelsConfig,
        QQChannelConfig,
    )

    push_tool = SimpleNamespace()
    disabled_config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        channels=ChannelsConfig(qq=QQChannelConfig(bot_uin="2365524513")),
        agent_gateway=AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="http://127.0.0.1:8780",
            outbox_worker_enabled=False,
        ),
    )
    disabled_tasks, disabled_worker = _build_agent_runtime_outbox_worker_tasks(
        disabled_config,
        push_tool,
    )
    assert disabled_tasks == []
    assert disabled_worker is None

    enabled_config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        channels=ChannelsConfig(
            qq=QQChannelConfig(
                bot_uin="2365524513",
                channel_name="qq_2365524513",
            )
        ),
        agent_gateway=AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="http://127.0.0.1:8780",
            outbox_worker_enabled=True,
        ),
    )
    runtime_tasks, runtime_worker = _build_agent_runtime_outbox_worker_tasks(
        enabled_config,
        push_tool,
    )
    legacy_tasks, legacy_worker = _build_agent_gateway_outbox_worker_tasks(
        enabled_config,
        push_tool,
    )
    for task in runtime_tasks:
        task.close()
    for task in legacy_tasks:
        task.close()
    assert len(runtime_tasks) == 1
    assert len(legacy_tasks) == 1
    assert runtime_worker is not None
    assert legacy_worker is not None


def test_bootstrap_skips_knowledge_worker_when_agent_gateway_base_url_empty(
    tmp_path: Path,
):
    from bootstrap.app import (
        _build_agent_gateway_knowledge_worker_tasks,
    )
    from agent.config_models import (
        AgentGatewayIntegrationConfig,
        Config,
        ChannelsConfig,
        QQChannelConfig,
        QQGroupConfig,
    )

    config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        channels=ChannelsConfig(
            qq=QQChannelConfig(
                bot_uin="1049511700",
                groups=[QQGroupConfig(group_id="284331268", observe_only=True)],
            )
        ),
        agent_gateway=AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="",
            request_timeout_seconds=5,
            knowledge_job_interval_seconds=60,
        ),
    )

    knowledge_tasks, knowledge_worker = _build_agent_gateway_knowledge_worker_tasks(
        config,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    assert knowledge_tasks == []
    assert knowledge_worker is None


def test_bootstrap_runs_group_memory_loop_when_agent_gateway_disabled(
    tmp_path: Path,
):
    from bootstrap.app import (
        _build_agent_gateway_knowledge_worker_tasks,
        _build_group_memory_tasks,
    )
    from agent.config_models import (
        AgentGatewayIntegrationConfig,
        Config,
        ChannelsConfig,
        QQChannelConfig,
        QQGroupConfig,
    )

    config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        channels=ChannelsConfig(
            qq=QQChannelConfig(
                bot_uin="1049511700",
                groups=[QQGroupConfig(group_id="284331268", observe_only=True)],
            )
        ),
        agent_gateway=AgentGatewayIntegrationConfig(enabled=False),
    )

    legacy_tasks, legacy_loop = _build_group_memory_tasks(
        config,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    for task in legacy_tasks:
        task.close()
    assert len(legacy_tasks) == 1
    assert legacy_loop is not None

    knowledge_tasks, knowledge_worker = _build_agent_gateway_knowledge_worker_tasks(
        config,
        tmp_path,
        session_store=SimpleNamespace(),
    )
    assert knowledge_tasks == []
    assert knowledge_worker is None
