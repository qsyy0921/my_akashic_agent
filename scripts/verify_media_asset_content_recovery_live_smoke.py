from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
from contextlib import suppress
from dataclasses import dataclass
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

import httpx


REPO_ROOT = Path(__file__).resolve().parents[1]
SOURCE_BYTES = b"png bytes"
SOURCE_MIME = "image/png"


def _utcnow() -> datetime:
    return datetime.now(timezone.utc)


def _find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _find_go_exe() -> str:
    go_exe = shutil.which("go")
    if go_exe:
        return go_exe
    local_app_data = os.environ.get("LOCALAPPDATA", "")
    candidate = Path(local_app_data) / "Programs" / "Go" / "bin" / "go.exe"
    if candidate.exists():
        return str(candidate)
    raise RuntimeError("go executable not found in PATH or LOCALAPPDATA Programs\\Go\\bin")


def _is_within(path: Path, root: Path) -> bool:
    try:
        path.resolve().relative_to(root.resolve())
        return True
    except ValueError:
        return False


def _count_files(root: Path) -> int:
    if not root.exists():
        return 0
    return sum(1 for path in root.rglob("*") if path.is_file())


@dataclass
class TempRuntimeHandle:
    process: subprocess.Popen[str]
    base_url: str
    runtime_root: Path
    state_dir: Path
    cache_root: Path
    stdout_log: Path
    stderr_log: Path

    def stop(self) -> None:
        if self.process.poll() is None:
            with suppress(Exception):
                self.process.terminate()
            try:
                self.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                with suppress(Exception):
                    self.process.kill()
                with suppress(Exception):
                    self.process.wait(timeout=5)


@dataclass
class TempHttpSourceHandle:
    server: ThreadingHTTPServer
    thread: threading.Thread
    base_url: str
    requests: list[str]

    def stop(self) -> None:
        with suppress(Exception):
            self.server.shutdown()
        with suppress(Exception):
            self.server.server_close()
        with suppress(Exception):
            self.thread.join(timeout=5)


class JsonRuntimeClient:
    def __init__(self, base_url: str, *, timeout: float = 30.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def request_json(
        self,
        method: str,
        path: str,
        *,
        json_body: Any | None = None,
        params: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        with httpx.Client(timeout=self.timeout, trust_env=False) as client:
            response = client.request(
                method.upper(),
                self.base_url + path,
                json=json_body,
                params=params,
                headers={"Content-Type": "application/json"},
            )
        response.raise_for_status()
        payload = response.json()
        if not isinstance(payload, dict):
            raise RuntimeError(f"unexpected non-object response for {method} {path}")
        code = payload.get("code")
        if code not in (None, "OK", 0):
            raise RuntimeError(str(payload.get("message") or payload))
        data = payload.get("data", payload)
        if isinstance(data, dict):
            return data
        return {"items": data}

    def get_bytes(self, path: str) -> tuple[bytes, dict[str, str]]:
        with httpx.Client(timeout=self.timeout, trust_env=False) as client:
            response = client.get(self.base_url + path)
        response.raise_for_status()
        return response.content, dict(response.headers)


def start_temp_http_source() -> TempHttpSourceHandle:
    port = _find_free_port()
    requests: list[str] = []

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:  # noqa: N802
            requests.append(self.path)
            if self.path != "/image.png":
                self.send_response(404)
                self.end_headers()
                return
            self.send_response(200)
            self.send_header("Content-Type", SOURCE_MIME)
            self.send_header("Content-Length", str(len(SOURCE_BYTES)))
            self.end_headers()
            self.wfile.write(SOURCE_BYTES)

        def log_message(self, format: str, *args: Any) -> None:  # noqa: A003
            return

    server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return TempHttpSourceHandle(
        server=server,
        thread=thread,
        base_url=f"http://127.0.0.1:{port}",
        requests=requests,
    )


def start_temp_runtime(repo_root: Path) -> TempRuntimeHandle:
    port = _find_free_port()
    runtime_root = Path(tempfile.mkdtemp(prefix="media-recovery-live-smoke-"))
    state_dir = runtime_root / "state"
    cache_root = runtime_root / "cache"
    state_dir.mkdir(parents=True, exist_ok=True)
    cache_root.mkdir(parents=True, exist_ok=True)
    stdout_log = runtime_root / "agent-runtime.stdout.log"
    stderr_log = runtime_root / "agent-runtime.stderr.log"
    stdout_handle = stdout_log.open("w", encoding="utf-8")
    stderr_handle = stderr_log.open("w", encoding="utf-8")
    env = os.environ.copy()
    env["AKASHIC_RUNTIME_ADDR"] = f"127.0.0.1:{port}"
    env["AKASHIC_RUNTIME_STATE_DIR"] = str(state_dir)
    env["AKASHIC_MEDIA_ASSET_ROOTS"] = str(cache_root)
    env["AKASHIC_MEDIA_CONTENT_RECOVERY_CACHE_ROOT"] = str(cache_root)
    env["AKASHIC_MEDIA_CONTENT_RECOVERY_MAX_BYTES"] = "1048576"
    for key in (
        "AKASHIC_ONEBOT_WS_URLS",
        "AKASHIC_ONEBOT_ACCESS_TOKEN",
        "AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT",
        "AKASHIC_BOT_IDS",
        "AKASHIC_QQ_GROUP_SEND_ENABLED",
        "AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED",
        "AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED",
        "AKASHIC_TELEGRAM_BOT_TOKEN",
        "TELEGRAM_BOT_TOKEN",
    ):
        env.pop(key, None)
    process = subprocess.Popen(
        [_find_go_exe(), "run", "./cmd/agent-runtime"],
        cwd=str(repo_root / "services" / "agent-runtime"),
        env=env,
        stdout=stdout_handle,
        stderr=stderr_handle,
        text=True,
    )
    base_url = f"http://127.0.0.1:{port}"
    deadline = time.time() + 60
    last_error = ""
    while time.time() < deadline:
        if process.poll() is not None:
            break
        try:
            with httpx.Client(timeout=2.0, trust_env=False) as client:
                response = client.get(base_url + "/healthz")
            if response.status_code == 200:
                stdout_handle.close()
                stderr_handle.close()
                return TempRuntimeHandle(
                    process=process,
                    base_url=base_url,
                    runtime_root=runtime_root,
                    state_dir=state_dir,
                    cache_root=cache_root,
                    stdout_log=stdout_log,
                    stderr_log=stderr_log,
                )
        except Exception as exc:  # pragma: no cover - startup wait
            last_error = str(exc)
        time.sleep(0.5)
    stdout_handle.close()
    stderr_handle.close()
    stderr_tail = ""
    if stderr_log.exists():
        stderr_tail = "\n".join(stderr_log.read_text(encoding="utf-8").splitlines()[-20:])
    process.kill()
    process.wait(timeout=5)
    raise RuntimeError(
        f"temp agent-runtime did not become healthy at {base_url}/healthz. "
        f"last_error={last_error!r}\n{stderr_tail}"
    )


def _register_asset(client: JsonRuntimeClient, *, asset_id: str, asset_url: str) -> dict[str, Any]:
    return client.request_json(
        "POST",
        "/v1/media-assets",
        json_body={
            "asset_id": asset_id,
            "channel": {
                "kind": "qq",
                "platform": "qq",
                "account_id": "1049511700",
                "conversation_id": "3219982",
                "conversation_type": "group",
            },
            "source_message_id": f"qq:gqq:3219982:{asset_id}",
            "sender_id": "2948770636",
            "kind": "image",
            "url": asset_url,
            "mime_type": SOURCE_MIME,
            "name": "image.png",
            "timestamp": _utcnow().isoformat(),
            "metadata": {
                "source": "verify_media_asset_content_recovery_live_smoke",
            },
        },
    )


def _record_approval(client: JsonRuntimeClient, *, asset_id: str) -> dict[str, Any]:
    return client.request_json(
        "POST",
        "/v1/operator-approvals",
        json_body={
            "target_kind": "media_asset_content",
            "target_id": asset_id,
            "decision": "approved",
            "operator_id": "qsyy",
            "timestamp": _utcnow().isoformat(),
            "metadata": {
                "source": "verify_media_asset_content_recovery_live_smoke",
            },
        },
    )


def _build_result(
    *,
    repo_root: Path,
    runtime: TempRuntimeHandle | None,
    source: TempHttpSourceHandle | None,
    verification: dict[str, Any] | None,
    error: str | None = None,
) -> dict[str, Any]:
    checks = dict((verification or {}).get("checks") or {})
    all_checks = list(checks.values())
    if error:
        status = "error"
        category = "media_asset_content_recovery_executor_live_smoke_error"
        reason = error
    elif all_checks and all(bool(value) for value in all_checks):
        status = "live_verified"
        category = "go_http_https_media_recovery_executor_live_verified"
        reason = (
            "temp Go runtime proved approval-bound preflight, dry-run, real HTTP/HTTPS "
            "recovery, registry update, local content readback, and control mutation audit."
        )
    else:
        status = "verification_failed"
        category = "media_asset_content_recovery_executor_live_smoke_failed"
        reason = "at least one HTTP/HTTPS media recovery executor live smoke check failed"
    return {
        "generated_at": _utcnow().isoformat(),
        "repo_root": str(repo_root),
        "verification_scope": "media_asset_content_recovery_http_https_executor",
        "runtime": {
            "base_url": None if runtime is None else runtime.base_url,
            "runtime_root": None if runtime is None else str(runtime.runtime_root),
            "state_dir": None if runtime is None else str(runtime.state_dir),
            "cache_root": None if runtime is None else str(runtime.cache_root),
            "stdout_log": None if runtime is None else str(runtime.stdout_log),
            "stderr_log": None if runtime is None else str(runtime.stderr_log),
        },
        "source": {
            "base_url": None if source is None else source.base_url,
            "requests": [] if source is None else list(source.requests),
        },
        "verification": verification,
        "conclusion": {
            "status": status,
            "category": category,
            "reason": reason,
        },
        **({"error": error} if error else {}),
    }


def run_media_asset_content_recovery_live_smoke(repo_root: Path) -> dict[str, Any]:
    runtime = None
    source = None
    try:
        source = start_temp_http_source()
        runtime = start_temp_runtime(repo_root)
        client = JsonRuntimeClient(runtime.base_url)
        asset_id = "asset-remote-http"
        asset_url = source.base_url + "/image.png"

        asset_registration = _register_asset(client, asset_id=asset_id, asset_url=asset_url)
        preflight_without_approval = client.request_json(
            "GET",
            "/v1/media-assets/content-recovery/preflight",
            params={
                "asset_id": asset_id,
                "operator_id": "qsyy",
            },
        )
        approval = _record_approval(client, asset_id=asset_id)
        approval_id = str(approval.get("approval_id") or "")
        preflight_with_approval = client.request_json(
            "GET",
            "/v1/media-assets/content-recovery/preflight",
            params={
                "asset_id": asset_id,
                "operator_id": "qsyy",
                "approval_id": approval_id,
            },
        )
        dry_run = client.request_json(
            "POST",
            "/v1/media-assets/content-recovery",
            json_body={
                "asset_id": asset_id,
                "operator_id": "qsyy",
                "approval_id": approval_id,
                "dry_run": True,
            },
        )
        source_requests_after_dry_run = len(source.requests)
        cache_files_after_dry_run = _count_files(runtime.cache_root)
        live_recovery = client.request_json(
            "POST",
            "/v1/media-assets/content-recovery",
            json_body={
                "asset_id": asset_id,
                "operator_id": "qsyy",
                "approval_id": approval_id,
                "mutation_id": "mutation-media-recover-http-1",
            },
        )
        recovered_asset = client.request_json("GET", f"/v1/media-assets/{asset_id}")
        content_access_plan = client.request_json(
            "GET",
            "/v1/media-assets/content-access-plan",
            params={"asset_id": asset_id},
        )
        content_bytes, content_headers = client.get_bytes(f"/v1/media-assets/{asset_id}/content")
        control_mutations = client.request_json(
            "GET",
            "/v1/control-mutations",
            params={
                "target_kind": "media_asset_content",
                "target_id": asset_id,
                "limit": 10,
            },
        )

        live_local_path = Path(str(live_recovery.get("local_path") or ""))
        recovered_metadata = dict(recovered_asset.get("metadata") or {})
        mutations = list(control_mutations.get("mutations") or [])
        applied_mutation = next(
            (
                item
                for item in mutations
                if isinstance(item, dict)
                and item.get("status") == "applied"
                and item.get("target_kind") == "media_asset_content"
                and item.get("target_id") == asset_id
                and item.get("action") == "recover_content"
            ),
            None,
        )
        checks = {
            "preflight_requires_approval_before_download": (
                preflight_without_approval.get("ready") is False
                and preflight_without_approval.get("reason") == "missing_approval_id"
                and preflight_without_approval.get("recovery_needed") is True
            ),
            "preflight_ready_after_approval": (
                preflight_with_approval.get("ready") is True
                and preflight_with_approval.get("reason")
                == "media_asset_content_recovery_preflight_ready"
                and isinstance(preflight_with_approval.get("suggested_audit"), dict)
            ),
            "dry_run_does_not_write_cache": (
                dry_run.get("ready") is True
                and dry_run.get("applied") is False
                and dry_run.get("dry_run") is True
                and dry_run.get("reason") == "media_asset_content_recovery_dry_run"
                and cache_files_after_dry_run == 0
                and source_requests_after_dry_run == 0
            ),
            "live_recovery_writes_cache_and_updates_registry": (
                live_recovery.get("ready") is True
                and live_recovery.get("applied") is True
                and live_recovery.get("reason") == "media_asset_content_recovery_applied"
                and live_local_path.exists()
                and _is_within(live_local_path, runtime.cache_root)
                and recovered_asset.get("url") == str(live_local_path)
                and recovered_metadata.get("local_path") == str(live_local_path)
                and recovered_metadata.get("recovered_from_url") == asset_url
                and recovered_metadata.get("recovery_scope") == "download_to_local_cache"
                and recovered_metadata.get("recovery_approval_id") == approval_id
                and recovered_metadata.get("recovery_mutation_id")
                == "mutation-media-recover-http-1"
            ),
            "content_endpoint_reads_recovered_bytes": (
                content_bytes == SOURCE_BYTES
                and content_headers.get("content-type", "").startswith(SOURCE_MIME)
                and content_access_plan.get("ready") is True
                and content_access_plan.get("reason") == "media_asset_content_ready"
            ),
            "control_mutation_audit_recorded": (
                applied_mutation is not None
                and int((control_mutations.get("totals") or {}).get("applied") or 0) >= 1
                and str(applied_mutation.get("approval_id")) == approval_id
                and str(applied_mutation.get("mutation_id"))
                == "mutation-media-recover-http-1"
            ),
            "remote_source_only_downloaded_on_live_recovery": len(source.requests) == 1,
        }
        verification = {
            "asset_id": asset_id,
            "asset_url": asset_url,
            "asset_registration": asset_registration,
            "preflight_without_approval": preflight_without_approval,
            "approval": approval,
            "preflight_with_approval": preflight_with_approval,
            "dry_run": dry_run,
            "live_recovery": live_recovery,
            "recovered_asset": recovered_asset,
            "content_access_plan": content_access_plan,
            "content_bytes_sha256": "sha256:" + hashlib.sha256(content_bytes).hexdigest(),
            "content_headers": content_headers,
            "control_mutations": control_mutations,
            "source_requests_after_dry_run": source_requests_after_dry_run,
            "cache_files_after_dry_run": cache_files_after_dry_run,
            "checks": checks,
        }
        return _build_result(
            repo_root=repo_root,
            runtime=runtime,
            source=source,
            verification=verification,
        )
    except Exception as exc:  # pragma: no cover - defensive live path
        return _build_result(
            repo_root=repo_root,
            runtime=runtime,
            source=source,
            verification=None,
            error=str(exc),
        )
    finally:
        if runtime is not None:
            runtime.stop()
        if source is not None:
            source.stop()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(REPO_ROOT))
    args = parser.parse_args(argv)
    result = run_media_asset_content_recovery_live_smoke(Path(args.repo_root).resolve())
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
