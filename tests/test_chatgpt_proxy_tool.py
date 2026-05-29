from __future__ import annotations

import asyncio
import json
from pathlib import Path
from types import SimpleNamespace
from typing import Any, cast

import httpx
import pytest

from agent.config_models import ChatGPTProxyIntegrationConfig, Config
from agent.tools.chatgpt_proxy import ChatGPTImageGenerateTool
from agent.tools.registry import ToolRegistry
from bootstrap.tools import _register_chatgpt_proxy_tools
from core.net.http import HttpRequester, RequestBudget, RetryPolicy

_PNG_B64 = (
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk"
    "+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
)


def _requester(handler) -> HttpRequester:
    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    return HttpRequester(
        client=client,
        retry_policy=RetryPolicy(max_attempts=1),
        default_timeout_s=1.0,
        default_budget=RequestBudget(total_timeout_s=2.0),
        sleep=lambda _: asyncio.sleep(0),
    )


@pytest.mark.asyncio
async def test_chatgpt_image_generate_saves_b64_image(tmp_path: Path):
    seen: dict[str, Any] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen["url"] = str(request.url)
        seen["auth"] = request.headers.get("authorization")
        seen["payload"] = json.loads(request.content.decode("utf-8"))
        return httpx.Response(
            200,
            request=request,
            json={
                "data": [
                    {
                        "b64_json": _PNG_B64,
                        "revised_prompt": "a tiny generated image",
                    }
                ]
            },
        )

    requester = _requester(handler)
    tool = ChatGPTImageGenerateTool(
        ChatGPTProxyIntegrationConfig(
            enabled=True,
            base_url="https://proxy.example/v1",
            api_key="proxy-key",
            image_model="gpt-image-test",
            output_dir="images",
        ),
        tmp_path,
        requester,
    )

    raw = await tool.execute(prompt="画一只杯子", size="1024x1024")
    result = json.loads(raw)

    assert result["ok"] is True
    assert seen["url"] == "https://proxy.example/v1/images/generations"
    assert seen["auth"] == "Bearer proxy-key"
    assert seen["payload"]["model"] == "gpt-image-test"
    assert seen["payload"]["response_format"] == "b64_json"
    saved = Path(result["paths"][0])
    assert saved.exists()
    assert saved.parent == tmp_path / "images"
    assert saved.suffix == ".png"
    assert result["revised_prompts"] == ["a tiny generated image"]
    await requester.client.aclose()


@pytest.mark.asyncio
async def test_chatgpt_image_generate_retries_without_response_format(
    tmp_path: Path,
):
    payloads: list[dict[str, Any]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        payload = json.loads(request.content.decode("utf-8"))
        payloads.append(payload)
        if len(payloads) == 1:
            return httpx.Response(
                400,
                request=request,
                json={"error": "unsupported response_format"},
            )
        return httpx.Response(
            200,
            request=request,
            json={"data": [{"b64_json": _PNG_B64}]},
        )

    requester = _requester(handler)
    tool = ChatGPTImageGenerateTool(
        ChatGPTProxyIntegrationConfig(
            enabled=True,
            base_url="https://proxy.example/v1",
        ),
        tmp_path,
        requester,
    )

    result = json.loads(await tool.execute(prompt="test"))

    assert result["ok"] is True
    assert "response_format" in payloads[0]
    assert "response_format" not in payloads[1]
    await requester.client.aclose()


def test_register_chatgpt_proxy_tool_requires_enabled_and_base_url(tmp_path: Path):
    config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        chatgpt_proxy=ChatGPTProxyIntegrationConfig(
            enabled=True,
            base_url="http://127.0.0.1:8000/v1",
        ),
    )
    registry = ToolRegistry()

    _register_chatgpt_proxy_tools(
        config,
        tmp_path,
        cast(Any, SimpleNamespace(external_default=object())),
        registry,
    )

    assert registry.has_tool("chatgpt_image_generate")


def test_register_chatgpt_proxy_tool_skips_missing_base_url(tmp_path: Path):
    config = Config(
        provider="openai",
        model="m",
        api_key="k",
        system_prompt="s",
        chatgpt_proxy=ChatGPTProxyIntegrationConfig(enabled=True),
    )
    registry = ToolRegistry()

    _register_chatgpt_proxy_tools(
        config,
        tmp_path,
        cast(Any, SimpleNamespace(external_default=object())),
        registry,
    )

    assert not registry.has_tool("chatgpt_image_generate")
