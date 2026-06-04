from __future__ import annotations

import importlib.util
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

import httpx


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_receiver_status_cleanup_live_smoke.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_receiver_status_cleanup_live_smoke", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class FakeReceiverCleanupRuntime:
    def __init__(self, state_file: Path) -> None:
        self.state_file = state_file
        self.records: dict[str, dict] = {}
        self.requests: list[tuple[str, str]] = []
        self._write_state()

    def _write_state(self) -> None:
        payload = {
            "version": "2026-06-03.receiverstatusstore.v1",
            "receivers": list(self.records.values()),
        }
        self.state_file.parent.mkdir(parents=True, exist_ok=True)
        self.state_file.write_text(
            json.dumps(payload, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )

    @property
    def now(self) -> datetime:
        return datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    def _is_stale(self, item: dict, stale_after_seconds: int = 120) -> bool:
        updated_at = datetime.fromisoformat(str(item["UpdatedAt"])).astimezone(timezone.utc)
        return item["Status"] in {"starting", "connected"} and updated_at <= (
            self.now - timedelta(seconds=stale_after_seconds)
        )

    def _view_item(self, item: dict, stale_after_seconds: int = 120) -> dict:
        stale = self._is_stale(item, stale_after_seconds=stale_after_seconds)
        metadata = dict(item.get("Metadata") or {})
        if stale:
            metadata["last_status"] = item["Status"]
            metadata["stale_after_seconds"] = str(stale_after_seconds)
        return {
            "receiver_id": item["ReceiverID"],
            "kind": item["Kind"],
            "channel_name": item["ChannelName"],
            "account_id": item["AccountID"],
            "status": "stopped" if stale else item["Status"],
            "reason": "heartbeat_stale" if stale else item.get("Reason", ""),
            "source": item.get("Source", "python_channel"),
            "updated_at": item["UpdatedAt"],
            "metadata": metadata,
        }

    def handler(self, request: httpx.Request) -> httpx.Response:
        self.requests.append((request.method, request.url.path))
        if request.url.path == "/v1/receiver-statuses" and request.method == "GET":
            items = [self._view_item(item) for item in self.records.values()]
            stopped_total = sum(1 for item in items if item["status"] == "stopped")
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "receivers": sorted(items, key=lambda item: item["receiver_id"]),
                        "totals": {
                            "receivers": len(items),
                            "stopped": stopped_total,
                        },
                        "side_effect": "none",
                    },
                },
            )
        if request.url.path == "/v1/receiver-statuses/report" and request.method == "POST":
            body = json.loads(request.read().decode("utf-8"))
            kind = str(body["kind"])
            account_id = str(body["account_id"])
            channel_name = str(body["channel_name"])
            receiver_id = f"{kind}:{account_id}:{channel_name}"
            timestamp = datetime.fromisoformat(body["timestamp"]).astimezone(timezone.utc)
            self.records[receiver_id] = {
                "ReceiverID": receiver_id,
                "Kind": kind,
                "ChannelName": channel_name,
                "AccountID": account_id,
                "Status": str(body["status"]),
                "Reason": str(body.get("reason") or ""),
                "Source": str(body.get("source") or "python_channel"),
                "Metadata": dict(body.get("metadata") or {}),
                "UpdatedAt": timestamp.isoformat(),
            }
            self._write_state()
            return httpx.Response(
                200,
                json={"code": "OK", "data": {"accepted": True, "receiver_id": receiver_id}},
            )
        if request.url.path == "/v1/receiver-statuses/cleanup-stale" and request.method == "POST":
            body = json.loads(request.read().decode("utf-8"))
            stale_after_seconds = int(body.get("stale_after_seconds") or 120)
            receiver_id = str(body.get("receiver_id") or "")
            deleted = []
            for key, item in list(self.records.items()):
                if receiver_id and key != receiver_id:
                    continue
                if not self._is_stale(item, stale_after_seconds=stale_after_seconds):
                    continue
                deleted.append(self._view_item(item, stale_after_seconds=stale_after_seconds))
                del self.records[key]
            self._write_state()
            remaining = [self._view_item(item) for item in self.records.values()]
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "deleted": deleted,
                        "remaining": remaining,
                        "totals": {
                            "deleted": len(deleted),
                            "remaining": len(remaining),
                            "remaining_stopped": sum(
                                1 for item in remaining if item["status"] == "stopped"
                            ),
                        },
                        "side_effect": "runtime_state_only",
                    },
                },
            )
        raise AssertionError(f"unexpected request: {request.method} {request.url}")


def test_run_receiver_status_cleanup_smoke_returns_expected_evidence(tmp_path):
    module = _load_module()
    runtime = FakeReceiverCleanupRuntime(tmp_path / "receiver-statuses.json")
    client = module.HELPERS.WorkerStatusRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )
    now = datetime(2026, 6, 3, 12, 0, tzinfo=timezone.utc)

    result = module.run_receiver_status_cleanup_smoke(
        client,
        state_file=runtime.state_file,
        stale_receiver_id="qq:1049511700:qq",
        active_receiver_id="telegram:7689386159:telegram",
        now=now,
    )

    assert all(result["checks"].values())
    assert result["status_codes"]["cleanup"] == 200
    assert result["cleanup_totals"]["deleted"] == 1
    assert ("POST", "/v1/receiver-statuses/cleanup-stale") in runtime.requests


def test_inspect_live_runtime_receiver_status_cleanup_state_detects_stale_receivers(tmp_path):
    module = _load_module()
    runtime = FakeReceiverCleanupRuntime(tmp_path / "receiver-statuses.json")
    stale_time = runtime.now - timedelta(minutes=5)
    active_time = runtime.now
    runtime.records = {
        "qq:1049511700:qq": {
            "ReceiverID": "qq:1049511700:qq",
            "Kind": "qq",
            "ChannelName": "qq",
            "AccountID": "1049511700",
            "Status": "connected",
            "Reason": "",
            "Source": "python_channel",
            "Metadata": {},
            "UpdatedAt": stale_time.isoformat(),
        },
        "telegram:7689386159:telegram": {
            "ReceiverID": "telegram:7689386159:telegram",
            "Kind": "telegram",
            "ChannelName": "telegram",
            "AccountID": "7689386159",
            "Status": "connected",
            "Reason": "",
            "Source": "python_channel",
            "Metadata": {},
            "UpdatedAt": active_time.isoformat(),
        },
    }
    runtime._write_state()
    client = module.HELPERS.WorkerStatusRuntimeClient(
        "http://agent-runtime.test",
        transport=httpx.MockTransport(runtime.handler),
    )

    state = module.inspect_live_runtime_receiver_status_cleanup_state(client)

    assert state["receivers_total"] == 2
    assert state["receivers_stopped"] == 1
    assert state["stale_receivers"][0]["receiver_id"] == "qq:1049511700:qq"
