"""Tests for SchedulerService: tick, execution, misfire, rescheduling."""

import asyncio
import json
from dataclasses import asdict
from datetime import datetime, timezone, timedelta
from unittest.mock import AsyncMock, call

import httpx
import pytest

from agent.config_models import AgentRuntimeIntegrationConfig
from agent.scheduler import LatencyTracker, SchedulerService, ScheduledJob
from tests.conftest import drain_tasks, make_job

# ── Helpers ──────────────────────────────────────────────────────


def make_service(tmp_path, mock_push, mock_loop, now, tracker=None):
    return SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        tracker=tracker or LatencyTracker(default=25.0),
        _now_fn=lambda: now,
    )


def json_request(request: httpx.Request) -> dict:
    return json.loads(request.read().decode("utf-8"))


def runtime_job_payload(job: ScheduledJob) -> dict:
    payload = asdict(job)
    payload["fire_at"] = job.fire_at.isoformat()
    payload["created_at"] = job.created_at.isoformat()
    return payload


# ── Execution: INSTANT ───────────────────────────────────────────


async def test_instant_calls_push_not_ai(tmp_path, mock_push, mock_loop, fixed_now):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(tier="instant", fire_at=fixed_now - timedelta(seconds=1))
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_push.execute.assert_called_once()
    mock_loop.process_direct.assert_not_called()


async def test_instant_push_receives_correct_args(
    tmp_path, mock_push, mock_loop, fixed_now
):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=1),
        channel="telegram",
        chat_id="999",
        message="喝水了",
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_push.execute.assert_called_once_with(
        channel="telegram", chat_id="999", message="喝水了"
    )


# ── Execution: SOFT ──────────────────────────────────────────────


async def test_soft_calls_process_direct_not_push_directly(
    tmp_path, mock_push, mock_loop, fixed_now
):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    # fire_at - lead (25s) must be <= now; set fire_at far enough in past
    job = make_job(
        tier="soft",
        fire_at=fixed_now - timedelta(seconds=30),
        channel="telegram",
        chat_id="123",
        message=None,
        prompt="查询北京天气",
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_loop.process_direct.assert_called_once()
    call_kwargs = mock_loop.process_direct.call_args
    assert call_kwargs.kwargs["content"] == "查询北京天气"
    assert call_kwargs.kwargs["channel"] == "telegram"
    assert call_kwargs.kwargs["chat_id"] == "123"
    assert call_kwargs.kwargs["skip_post_memory"] is True
    assert call_kwargs.kwargs["disabled_tools"] == ["message_push"]


async def test_soft_sends_ai_response_via_push(
    tmp_path, mock_push, mock_loop, fixed_now
):
    mock_loop.process_direct = AsyncMock(return_value="北京今天晴，15°C")
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        tier="soft",
        fire_at=fixed_now - timedelta(seconds=30),
        prompt="查询北京天气",
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_push.execute.assert_called_once_with(
        channel=job.channel, chat_id=job.chat_id, message="北京今天晴，15°C"
    )


async def test_soft_records_latency(tmp_path, mock_push, mock_loop, fixed_now):
    tracker = LatencyTracker(default=25.0)
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now, tracker)
    job = make_job(
        tier="soft",
        fire_at=fixed_now - timedelta(seconds=30),
        prompt="天气",
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    assert len(tracker._samples) == 1


# ── Timing: pre-trigger ──────────────────────────────────────────


async def test_soft_not_fired_before_pretrigger(
    tmp_path, mock_push, mock_loop, fixed_now
):
    tracker = LatencyTracker(default=30.0)
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now, tracker)
    # fire_at is 60s in future; pretrigger = fire_at - 30s = now+30s, not yet due
    job = make_job(
        tier="soft",
        fire_at=fixed_now + timedelta(seconds=60),
        prompt="天气",
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_loop.process_direct.assert_not_called()


async def test_instant_not_fired_before_fire_at(
    tmp_path, mock_push, mock_loop, fixed_now
):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(tier="instant", fire_at=fixed_now + timedelta(seconds=10))
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_push.execute.assert_not_called()


# ── One-shot jobs removed after firing ───────────────────────────


async def test_at_job_removed_after_fire(tmp_path, mock_push, mock_loop, fixed_now):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        trigger="at", tier="instant", fire_at=fixed_now - timedelta(seconds=1)
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    assert job.id not in svc._jobs


async def test_after_job_removed_after_fire(tmp_path, mock_push, mock_loop, fixed_now):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        trigger="after", tier="instant", fire_at=fixed_now - timedelta(seconds=1)
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    assert job.id not in svc._jobs


# ── Every: rescheduling ───────────────────────────────────────────


async def test_every_job_rescheduled_after_fire(
    tmp_path, mock_push, mock_loop, fixed_now
):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        trigger="every",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=1),
        interval_seconds=3600,
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    # Job should still exist
    assert job.id in svc._jobs
    # fire_at should have advanced to approximately now + 1h
    new_fire_at = svc._jobs[job.id].fire_at
    assert new_fire_at > fixed_now


async def test_every_run_count_increments(tmp_path, mock_push, mock_loop, fixed_now):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        trigger="every",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=1),
        interval_seconds=60,
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    assert svc._jobs[job.id].run_count == 1


async def test_every_soft_p90_updates_affect_next_trigger(
    tmp_path, mock_push, mock_loop, fixed_now
):
    tracker = LatencyTracker(default=25.0, window=5)
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now, tracker)
    job = make_job(
        trigger="every",
        tier="soft",
        fire_at=fixed_now - timedelta(seconds=30),
        interval_seconds=3600,
        prompt="天气",
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    # P90 should now have a sample (from soft execution)
    assert len(tracker._samples) == 1


async def test_every_soft_cron_pretrigger_advances_past_current_boundary(
    tmp_path, mock_push, mock_loop
):
    """SOFT cron jobs should not re-fire the same nominal boundary."""

    now_ref = {"value": datetime(2025, 6, 1, 7, 59, 40, tzinfo=timezone.utc)}
    svc = SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        tracker=LatencyTracker(default=25.0),
        _now_fn=lambda: now_ref["value"],
    )
    fire_at = datetime(2025, 6, 1, 8, 0, 0, tzinfo=timezone.utc)
    job = make_job(
        trigger="every",
        tier="soft",
        fire_at=fire_at,
        cron_expr="0 8 * * *",
        timezone_="UTC",
        message=None,
        prompt="查询北京天气",
    )
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    assert mock_loop.process_direct.call_count == 1
    assert svc._jobs[job.id].fire_at > fire_at

    now_ref["value"] = datetime(2025, 6, 1, 7, 59, 46, tzinfo=timezone.utc)
    await svc._tick()
    await drain_tasks()

    assert mock_loop.process_direct.call_count == 1


async def test_runtime_execution_lease_acquired_and_completed(
    tmp_path, mock_push, mock_loop, fixed_now
):
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/v1/scheduler/leases/acquire":
            body = json_request(request)
            assert body["job_id"] == "lease-job"
            assert body["holder_id"].startswith("scheduler:python-worker:")
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": body["job_id"],
                        "holder_id": body["holder_id"],
                        "lease_token": "lease-token-1",
                        "lease_token_present": True,
                        "active": True,
                        "acquired": True,
                    },
                },
            )
        if request.url.path == "/v1/scheduler/jobs/lease-job/complete":
            body = json_request(request)
            assert body["action"] == "delete"
            assert body["lease_token"] == "lease-token-1"
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": "lease-job",
                        "action": "delete",
                        "deleted": True,
                        "lease_released": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        raise AssertionError(f"unexpected runtime request: {request.url.path}")

    svc = SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        _now_fn=lambda: fixed_now,
        runtime_config=AgentRuntimeIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.test",
            worker_id="python-worker",
            lease_ttl_seconds=300,
        ),
        runtime_transport=httpx.MockTransport(handler),
    )
    job = make_job(
        trigger="after",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=1),
        message="runtime lease",
    )
    job.id = "lease-job"
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_push.execute.assert_called_once_with(
        channel=job.channel, chat_id=job.chat_id, message="runtime lease"
    )
    assert [request.url.path for request in requests] == [
        "/v1/scheduler/leases/acquire",
        "/v1/scheduler/jobs/lease-job/complete",
    ]


async def test_runtime_execution_lease_not_released_when_completion_fails(
    tmp_path, mock_push, mock_loop, fixed_now
):
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/v1/scheduler/leases/acquire":
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": "snapshot-fails",
                        "lease_token": "lease-token-2",
                        "lease_token_present": True,
                        "active": True,
                        "acquired": True,
                    },
                },
            )
        if request.url.path == "/v1/scheduler/jobs/snapshot-fails/complete":
            return httpx.Response(500, text="complete failed")
        if request.url.path == "/v1/scheduler/leases/release":
            raise AssertionError("lease must not be released after completion failure")
        raise AssertionError(f"unexpected runtime request: {request.url.path}")

    svc = SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        _now_fn=lambda: fixed_now,
        runtime_config=AgentRuntimeIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.test",
            worker_id="python-worker",
            lease_ttl_seconds=300,
        ),
        runtime_transport=httpx.MockTransport(handler),
    )
    job = make_job(
        trigger="after",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=1),
        message="runtime lease",
    )
    job.id = "snapshot-fails"
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_push.execute.assert_called_once()
    assert job.id not in svc._jobs
    assert [request.url.path for request in requests] == [
        "/v1/scheduler/leases/acquire",
        "/v1/scheduler/jobs/snapshot-fails/complete",
    ]


async def test_runtime_execution_lease_completion_reschedules_every_job(
    tmp_path, mock_push, mock_loop, fixed_now
):
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/v1/scheduler/leases/acquire":
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": "every-complete",
                        "lease_token": "lease-token-3",
                        "lease_token_present": True,
                        "active": True,
                        "acquired": True,
                    },
                },
            )
        if request.url.path == "/v1/scheduler/jobs/every-complete/complete":
            body = json_request(request)
            assert body["action"] == "reschedule"
            assert body["lease_token"] == "lease-token-3"
            assert body["job"]["id"] == "every-complete"
            assert body["job"]["run_count"] == 1
            assert body["job"]["fire_at"] > fixed_now.isoformat()
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": "every-complete",
                        "action": "reschedule",
                        "deleted": False,
                        "lease_released": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        raise AssertionError(f"unexpected runtime request: {request.url.path}")

    svc = SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        _now_fn=lambda: fixed_now,
        runtime_config=AgentRuntimeIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.test",
            worker_id="python-worker",
            lease_ttl_seconds=300,
        ),
        runtime_transport=httpx.MockTransport(handler),
    )
    job = make_job(
        trigger="every",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=1),
        interval_seconds=60,
        message="runtime lease",
    )
    job.id = "every-complete"
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    assert svc._jobs[job.id].run_count == 1
    assert svc._jobs[job.id].fire_at > fixed_now
    assert [request.url.path for request in requests] == [
        "/v1/scheduler/leases/acquire",
        "/v1/scheduler/jobs/every-complete/complete",
    ]


async def test_runtime_execution_lease_denied_skips_job(
    tmp_path, mock_push, mock_loop, fixed_now
):
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/v1/scheduler/leases/acquire":
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": "lease-denied",
                        "lease_token_present": True,
                        "active": True,
                        "acquired": False,
                        "denied_reason": "active_lease_held",
                    },
                },
            )
        raise AssertionError(f"unexpected runtime request: {request.url.path}")

    svc = SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        _now_fn=lambda: fixed_now,
        runtime_config=AgentRuntimeIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.test",
            worker_id="python-worker",
        ),
        runtime_transport=httpx.MockTransport(handler),
    )
    job = make_job(tier="instant", fire_at=fixed_now - timedelta(seconds=1))
    job.id = "lease-denied"
    svc._jobs[job.id] = job

    await svc._tick()
    await drain_tasks()

    mock_push.execute.assert_not_called()
    assert job.id in svc._jobs
    assert svc._in_flight == set()
    assert [request.url.path for request in requests] == ["/v1/scheduler/leases/acquire"]


# ── Misfire handling ─────────────────────────────────────────────


def test_misfire_within_grace_loaded(tmp_path, mock_push, mock_loop, fixed_now):
    """Jobs missed within 5min grace period are retained for execution."""
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        trigger="at",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=100),  # 100s ago < 300s grace
    )
    # Persist and recover
    svc.store.save({job.id: job})
    svc.load_and_recover()

    assert job.id in svc._jobs


def test_misfire_beyond_grace_discarded(tmp_path, mock_push, mock_loop, fixed_now):
    """Jobs missed beyond 5min grace are discarded on startup."""
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        trigger="at",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=400),  # 400s ago > 300s grace
    )
    svc.store.save({job.id: job})
    svc.load_and_recover()

    assert job.id not in svc._jobs
    assert json.loads((tmp_path / "jobs.json").read_text(encoding="utf-8")) == []


def test_every_misfire_advances_to_future(tmp_path, mock_push, mock_loop, fixed_now):
    """Recurring jobs missed on restart are advanced to next future fire."""
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job(
        trigger="every",
        tier="instant",
        # Missed 3 hours ago, interval is 1h
        fire_at=fixed_now - timedelta(hours=3),
        interval_seconds=3600,
    )
    svc.store.save({job.id: job})
    svc.load_and_recover()

    assert job.id in svc._jobs
    assert svc._jobs[job.id].fire_at > fixed_now
    persisted = json.loads((tmp_path / "jobs.json").read_text(encoding="utf-8"))
    assert len(persisted) == 1
    assert persisted[0]["id"] == job.id
    assert persisted[0]["fire_at"] == svc._jobs[job.id].fire_at.isoformat()


def test_load_and_recover_reconciles_runtime_scheduler_crud(
    tmp_path, mock_push, mock_loop, fixed_now
):
    recurring = make_job(
        trigger="every",
        tier="instant",
        fire_at=fixed_now - timedelta(hours=3),
        interval_seconds=3600,
    )
    recurring.id = "runtime-recurring"
    expired = make_job(
        trigger="after",
        tier="instant",
        fire_at=fixed_now - timedelta(seconds=400),
    )
    expired.id = "runtime-expired"
    future = make_job(
        trigger="at",
        tier="instant",
        fire_at=fixed_now + timedelta(hours=1),
    )
    future.id = "runtime-future"
    runtime_jobs = [
        runtime_job_payload(recurring),
        runtime_job_payload(expired),
        runtime_job_payload(future),
    ]
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/v1/scheduler/jobs" and request.method == "GET":
            return httpx.Response(200, json={"code": "OK", "data": runtime_jobs})
        if request.url.path == "/v1/scheduler/jobs/upsert":
            body = json_request(request)
            assert body["source"] == "python_scheduler"
            assert body["job"]["id"] == "runtime-recurring"
            assert datetime.fromisoformat(body["job"]["fire_at"]) > fixed_now
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": "runtime-recurring",
                        "created": False,
                        "deleted": False,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        if request.url.path == "/v1/scheduler/jobs/runtime-expired":
            assert request.url.params["source"] == "python_scheduler"
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": "runtime-expired",
                        "found": True,
                        "deleted": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        raise AssertionError(f"unexpected runtime request: {request.method} {request.url}")

    svc = SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        _now_fn=lambda: fixed_now,
        runtime_config=AgentRuntimeIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.test",
            worker_id="python-worker",
        ),
        runtime_transport=httpx.MockTransport(handler),
    )

    svc.load_and_recover()

    assert "runtime-recurring" in svc._jobs
    assert "runtime-expired" not in svc._jobs
    assert "runtime-future" in svc._jobs
    assert svc._jobs["runtime-recurring"].fire_at > fixed_now
    assert [request.url.path for request in requests] == [
        "/v1/scheduler/jobs",
        "/v1/scheduler/jobs/upsert",
        "/v1/scheduler/jobs/runtime-expired",
    ]
    persisted = json.loads((tmp_path / "jobs.json").read_text(encoding="utf-8"))
    assert {item["id"] for item in persisted} == {"runtime-recurring", "runtime-future"}


# ── Cancel ───────────────────────────────────────────────────────


def test_cancel_job_by_id(tmp_path, mock_push, mock_loop, fixed_now):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    job = make_job()
    svc._jobs[job.id] = job

    result = svc.cancel_job(job.id)

    assert result is True
    assert job.id not in svc._jobs


def test_add_and_cancel_job_use_runtime_crud_when_enabled(
    tmp_path, mock_push, mock_loop, fixed_now
):
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/v1/scheduler/jobs/upsert":
            body = json_request(request)
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": body["job"]["id"],
                        "created": True,
                        "deleted": False,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        if request.url.path.startswith("/v1/scheduler/jobs/"):
            return httpx.Response(
                200,
                json={
                    "code": "OK",
                    "data": {
                        "job_id": request.url.path.rsplit("/", 1)[-1],
                        "found": True,
                        "deleted": True,
                        "side_effect": "runtime_state_write",
                    },
                },
            )
        raise AssertionError(f"unexpected runtime request: {request.url.path}")

    svc = SchedulerService(
        store_path=tmp_path / "jobs.json",
        push_tool=mock_push,
        agent_loop=mock_loop,
        _now_fn=lambda: fixed_now,
        runtime_config=AgentRuntimeIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.test",
            worker_id="python-worker",
        ),
        runtime_transport=httpx.MockTransport(handler),
    )
    job = make_job()

    svc.add_job(job)
    assert svc.cancel_job(job.id) is True

    assert [request.url.path for request in requests] == [
        "/v1/scheduler/jobs/upsert",
        f"/v1/scheduler/jobs/{job.id}",
    ]


def test_cancel_nonexistent_returns_false(tmp_path, mock_push, mock_loop, fixed_now):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    assert svc.cancel_job("nonexistent-id") is False


def test_cancel_by_name(tmp_path, mock_push, mock_loop, fixed_now):
    svc = make_service(tmp_path, mock_push, mock_loop, fixed_now)
    j1 = make_job(name="daily-weather")
    j2 = make_job(name="other")
    svc._jobs[j1.id] = j1
    svc._jobs[j2.id] = j2

    cancelled = svc.cancel_job_by_name("daily-weather")

    assert len(cancelled) == 1
    assert j1.id not in svc._jobs
    assert j2.id in svc._jobs
