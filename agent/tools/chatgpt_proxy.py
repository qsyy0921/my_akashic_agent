from __future__ import annotations

import base64
import json
import re
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any
from uuid import uuid4

from agent.config_models import ChatGPTProxyIntegrationConfig
from agent.tools.base import Tool
from agent.tools.filesystem import _detect_supported_image_mime_from_header
from core.net.http import HttpRequester, RequestBudget

_BASE64_RE = re.compile(r"^[A-Za-z0-9+/=\r\n]+$")
_DATA_URI_RE = re.compile(
    r"^data:(?P<mime>image/[a-zA-Z0-9.+-]+);base64,(?P<data>.+)$",
    re.DOTALL,
)
_MIME_SUFFIX = {
    "image/png": ".png",
    "image/jpeg": ".jpg",
    "image/jpg": ".jpg",
    "image/gif": ".gif",
    "image/webp": ".webp",
}


@dataclass
class _ImagePayload:
    kind: str
    value: str
    mime: str = ""


class ChatGPTImageGenerateTool(Tool):
    """Generate images through a separately configured OpenAI-compatible endpoint."""

    name = "chatgpt_image_generate"
    description = (
        "通过单独配置的 OpenAI-compatible 图片生成端点生成图片，"
        "不会改动或使用当前 MiMo 主模型配置。生成结果会保存到工作区，"
        "返回本地图片路径；需要发送给用户时再调用 message_push 的 image 参数。"
    )
    parameters = {
        "type": "object",
        "properties": {
            "prompt": {
                "type": "string",
                "description": "图片生成提示词，描述画面主体、风格、构图和细节。",
            },
            "size": {
                "type": "string",
                "description": "图片尺寸，例如 1024x1024、1536x1024、1024x1536。",
            },
            "n": {
                "type": "integer",
                "description": "生成图片数量，默认 1。",
                "minimum": 1,
                "maximum": 4,
            },
            "model": {
                "type": "string",
                "description": "覆盖默认图片模型；通常留空使用配置项。",
            },
            "quality": {
                "type": "string",
                "description": "可选质量参数，例如 low、medium、high 或 auto。",
            },
            "response_format": {
                "type": "string",
                "description": "可选返回格式，例如 b64_json 或 url；留空使用配置项。",
            },
            "output_format": {
                "type": "string",
                "description": "可选输出格式，例如 png、jpeg、webp。",
            },
            "timeout": {
                "type": "integer",
                "description": "请求总超时时间，单位秒，默认 180。",
                "minimum": 10,
                "maximum": 600,
            },
        },
        "required": ["prompt"],
    }

    def __init__(
        self,
        config: ChatGPTProxyIntegrationConfig,
        workspace: Path,
        requester: HttpRequester,
    ) -> None:
        self._config = config
        self._workspace = workspace
        self._requester = requester

    async def execute(
        self,
        prompt: str,
        size: str = "1024x1024",
        n: int = 1,
        model: str = "",
        quality: str = "",
        response_format: str | None = None,
        output_format: str = "",
        timeout: int = 180,
        **_: Any,
    ) -> str:
        if not self._config.enabled:
            return _json_error("chatgpt_proxy 未启用")
        if not self._config.base_url:
            return _json_error("chatgpt_proxy.base_url 未配置")
        prompt = str(prompt or "").strip()
        if not prompt:
            return _json_error("prompt 不能为空")

        timeout = max(10, min(int(timeout or 180), 600))
        payload = self._build_payload(
            prompt=prompt,
            size=size,
            n=n,
            model=model,
            quality=quality,
            response_format=response_format,
            output_format=output_format,
        )

        response, used_payload = await self._post_with_compat_retry(
            payload,
            timeout=timeout,
        )
        if response.status_code >= 400:
            return _json_error(
                f"图片端点返回 HTTP {response.status_code}",
                status=response.status_code,
                body=_response_preview(response.text),
            )

        try:
            body = response.json()
        except Exception:
            return _json_error("图片端点没有返回 JSON", body=_response_preview(response.text))

        images = _extract_image_payloads(body)
        if not images:
            return _json_error(
                "图片端点响应中没有找到 b64_json 或 url 图片",
                body=_response_preview(json.dumps(body, ensure_ascii=False)),
            )

        output_dir = _resolve_output_dir(self._workspace, self._config.output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)
        saved_paths: list[str] = []
        download_errors: list[str] = []
        for index, image in enumerate(images, start=1):
            try:
                saved_paths.append(
                    str(
                        await self._save_image_payload(
                            image,
                            output_dir=output_dir,
                            index=index,
                            timeout=timeout,
                        )
                    )
                )
            except Exception as exc:
                download_errors.append(str(exc))

        if not saved_paths:
            return _json_error("图片生成成功但保存失败", errors=download_errors)

        result: dict[str, Any] = {
            "ok": True,
            "model": used_payload.get("model"),
            "count": len(saved_paths),
            "paths": saved_paths,
            "note": "可用 message_push 的 image 参数发送这些本地路径。",
        }
        revised = _collect_revised_prompts(body)
        if revised:
            result["revised_prompts"] = revised
        if download_errors:
            result["warnings"] = download_errors
        return json.dumps(result, ensure_ascii=False)

    def _build_payload(
        self,
        *,
        prompt: str,
        size: str,
        n: int,
        model: str,
        quality: str,
        response_format: str | None,
        output_format: str,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "model": str(model or self._config.image_model or "gpt-image-1"),
            "prompt": prompt,
            "n": max(1, min(int(n or 1), 4)),
            "size": str(size or "1024x1024"),
        }
        fmt = self._config.response_format if response_format is None else response_format
        if fmt:
            payload["response_format"] = str(fmt)
        if quality:
            payload["quality"] = str(quality)
        if output_format:
            payload["output_format"] = str(output_format)
        return payload

    async def _post_with_compat_retry(
        self,
        payload: dict[str, Any],
        *,
        timeout: int,
    ) -> tuple[Any, dict[str, Any]]:
        response = await self._post(payload, timeout=timeout)
        if response.status_code not in (400, 422) or "response_format" not in payload:
            return response, payload
        retry_payload = dict(payload)
        retry_payload.pop("response_format", None)
        retry_response = await self._post(retry_payload, timeout=timeout)
        if retry_response.status_code < 400:
            return retry_response, retry_payload
        return response, payload

    async def _post(self, payload: dict[str, Any], *, timeout: int) -> Any:
        headers = {"Content-Type": "application/json"}
        if self._config.api_key:
            headers["Authorization"] = f"Bearer {self._config.api_key}"
        return await self._requester.post(
            _join_url(self._config.base_url, self._config.image_path),
            headers=headers,
            json=payload,
            timeout_s=float(timeout),
            budget=RequestBudget(total_timeout_s=float(timeout)),
        )

    async def _save_image_payload(
        self,
        image: _ImagePayload,
        *,
        output_dir: Path,
        index: int,
        timeout: int,
    ) -> Path:
        if image.kind == "url":
            response = await self._requester.get(
                image.value,
                follow_redirects=True,
                timeout_s=float(timeout),
                budget=RequestBudget(total_timeout_s=float(timeout)),
            )
            if response.status_code >= 400:
                raise RuntimeError(f"下载图片失败 HTTP {response.status_code}: {image.value}")
            data = response.content
            mime = response.headers.get("content-type", "").split(";")[0].strip()
        else:
            data = _decode_base64_image(image.value)
            mime = image.mime
        suffix = _suffix_for_image(data, mime)
        path = output_dir / _new_image_name(index, suffix)
        path.write_bytes(data)
        return path


def _join_url(base_url: str, path: str) -> str:
    return f"{base_url.rstrip('/')}/{path.lstrip('/')}"


def _resolve_output_dir(workspace: Path, configured: str) -> Path:
    raw = Path(str(configured or "generated_images")).expanduser()
    if raw.is_absolute():
        return raw.resolve()
    root = workspace.resolve()
    resolved = (root / raw).resolve()
    try:
        resolved.relative_to(root)
    except ValueError as exc:
        raise ValueError(f"output_dir 不能逃逸工作区: {configured}") from exc
    return resolved


def _extract_image_payloads(value: Any) -> list[_ImagePayload]:
    images: list[_ImagePayload] = []
    _walk_for_images(value, images)
    return images


def _walk_for_images(value: Any, images: list[_ImagePayload]) -> None:
    if isinstance(value, dict):
        for key, item in value.items():
            if isinstance(item, str):
                maybe = _image_payload_from_field(str(key), item)
                if maybe is not None:
                    images.append(maybe)
                    continue
            _walk_for_images(item, images)
    elif isinstance(value, list):
        for item in value:
            _walk_for_images(item, images)


def _image_payload_from_field(key: str, value: str) -> _ImagePayload | None:
    text = value.strip()
    if not text:
        return None
    if key in {"url", "image_url"} and text.startswith(("http://", "https://")):
        return _ImagePayload(kind="url", value=text)
    data_uri = _DATA_URI_RE.match(text)
    if data_uri is not None:
        return _ImagePayload(
            kind="base64",
            value=data_uri.group("data"),
            mime=data_uri.group("mime"),
        )
    if key in {
        "b64_json",
        "image_base64",
        "base64",
        "image",
        "result",
        "partial_image_b64",
    } and _looks_base64(text):
        return _ImagePayload(kind="base64", value=text)
    return None


def _looks_base64(value: str) -> bool:
    compact = "".join(value.split())
    if len(compact) < 32 or not _BASE64_RE.fullmatch(compact):
        return False
    try:
        base64.b64decode(compact, validate=True)
    except Exception:
        return False
    return True


def _decode_base64_image(value: str) -> bytes:
    return base64.b64decode("".join(value.split()), validate=True)


def _suffix_for_image(data: bytes, mime: str) -> str:
    if mime in _MIME_SUFFIX:
        return _MIME_SUFFIX[mime]
    detected = _detect_supported_image_mime_from_header(data[:4096])
    if detected in _MIME_SUFFIX:
        return _MIME_SUFFIX[detected]
    return ".png"


def _new_image_name(index: int, suffix: str) -> str:
    stamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    return f"chatgpt_{stamp}_{index}_{uuid4().hex[:8]}{suffix}"


def _collect_revised_prompts(value: Any) -> list[str]:
    prompts: list[str] = []
    if isinstance(value, dict):
        for key, item in value.items():
            if key == "revised_prompt" and isinstance(item, str) and item.strip():
                prompts.append(item.strip())
            else:
                prompts.extend(_collect_revised_prompts(item))
    elif isinstance(value, list):
        for item in value:
            prompts.extend(_collect_revised_prompts(item))
    return prompts


def _response_preview(text: str, limit: int = 2000) -> str:
    if len(text) <= limit:
        return text
    return text[:limit] + "...[truncated]"


def _json_error(message: str, **extra: Any) -> str:
    return json.dumps({"ok": False, "error": message, **extra}, ensure_ascii=False)
