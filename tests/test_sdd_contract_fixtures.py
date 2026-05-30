from __future__ import annotations

import json
from datetime import datetime
from pathlib import Path
from typing import Any

from core.contracts import ContractFixture

ROOT = Path(__file__).resolve().parents[1]
CONTRACT_DIR = ROOT / "tests" / "fixtures" / "contracts"
REPLAY_DIR = ROOT / "tests" / "fixtures" / "group_message_replay"

REQUIRED_CONTRACTS = {
    "message_envelope.qq.private.text.json",
    "message_envelope.qq.group.image.json",
    "agent_inbound.allowed.private.json",
    "agent_inbound.observe_only.group.json",
    "agent_decision.reply.private.json",
    "agent_decision.create_image_job.json",
    "media_asset.qq.image.json",
    "media_asset.qq.file.json",
    "image_job.lifecycle.json",
    "memory_extract_job.group_thread.json",
    "rag_ingest_job.thread_summary.json",
    "knowledge_checkpoint.ragflow.qq.json",
    "inbox_replay.observe_only.qq.json",
    "outbox_delivery.qq.private.text.json",
    "media_asset_content.qq.image.json",
    "agent_job_event_stream.rag_ingest.json",
    "group_thread.hardware.json",
    "group_thread.game_guide.json",
    "source_citation.message_asset.json",
}

REQUIRED_REPLAYS = {
    "hardware_recommendation_001.json",
    "game_guide_001.json",
    "image_evidence_001.json",
    "file_attachment_001.json",
    "conflicting_claims_001.json",
    "version_sensitive_001.json",
    "unresolved_question_001.json",
    "observe_only_no_reply_001.json",
}


def test_contract_fixture_manifest_is_complete() -> None:
    assert {path.name for path in CONTRACT_DIR.glob("*.json")} == REQUIRED_CONTRACTS


def test_contract_fixtures_have_shared_required_fields() -> None:
    for path in sorted(CONTRACT_DIR.glob("*.json")):
        data = _load_json(path)
        _assert_non_empty(data, "schema_version", path)
        _assert_non_empty(data, "kind", path)
        assert data.get("event_id") or data.get("job_id"), f"{path} needs event_id or job_id"
        _assert_route(data, path)
        _assert_timestamp(data, path)
        assert isinstance(data.get("metadata"), dict), f"{path} metadata must be object"
        assert isinstance(data.get("source_message_ids", []), list), path
        assert isinstance(data.get("source_asset_ids", []), list), path
        if not path.name.startswith("message_envelope."):
            assert (
                data.get("source_message_ids") is not None
            ), f"{path} derived contract needs source_message_ids"


def test_contract_fixtures_round_trip_through_python_contract_model() -> None:
    for path in sorted(CONTRACT_DIR.glob("*.json")):
        data = _load_json(path)
        fixture = ContractFixture.from_dict(data)
        round_tripped = ContractFixture.from_json(fixture.to_json())
        assert round_tripped.stable_id == fixture.stable_id
        assert round_tripped.platform == data["platform"]
        assert round_tripped.account_id == data["account_id"]
        assert round_tripped.conversation_id == data["conversation_id"]
        assert round_tripped.conversation_type == data["conversation_type"]
        assert round_tripped.source_message_ids == data["source_message_ids"]
        assert round_tripped.source_asset_ids == data["source_asset_ids"]
        assert round_tripped.metadata == data["metadata"]
        assert round_tripped.extra == fixture.extra


def test_contract_channel_fields_match_top_level_route() -> None:
    for path in sorted(CONTRACT_DIR.glob("*.json")):
        data = _load_json(path)
        channel = data.get("channel")
        if isinstance(channel, dict):
            assert channel.get("platform", channel.get("kind")) == data["platform"]
            assert channel["account_id"] == data["account_id"]
            assert channel["conversation_id"] == data["conversation_id"]
            assert channel["conversation_type"] == data["conversation_type"]


def test_runtime_boundary_fixtures_cover_current_go_owned_contracts() -> None:
    expectations = {
        "knowledge_checkpoint.ragflow.qq.json": {
            "kind": "KnowledgeCheckpoint",
            "required_keys": {"checkpoint_id", "cursor"},
        },
        "inbox_replay.observe_only.qq.json": {
            "kind": "InboxReplay",
            "required_keys": {"replay"},
        },
        "outbox_delivery.qq.private.text.json": {
            "kind": "OutboxDelivery",
            "required_keys": {"delivery", "content"},
        },
        "media_asset_content.qq.image.json": {
            "kind": "MediaAssetContent",
            "required_keys": {"asset_id", "content_access"},
        },
        "agent_job_event_stream.rag_ingest.json": {
            "kind": "AgentJobEventStream",
            "required_keys": {"job_type", "events"},
        },
    }

    for name, expectation in expectations.items():
        data = _load_json(CONTRACT_DIR / name)
        assert data["kind"] == expectation["kind"]
        for key in expectation["required_keys"]:
            assert key in data, f"{name} missing {key}"

    checkpoint = _load_json(CONTRACT_DIR / "knowledge_checkpoint.ragflow.qq.json")
    assert isinstance(checkpoint["cursor"], int)
    assert checkpoint["checkpoint_id"].startswith("ragflow:qq:")
    assert checkpoint["metadata"]["last_message_id"] in checkpoint["source_message_ids"]

    replay = _load_json(CONTRACT_DIR / "inbox_replay.observe_only.qq.json")
    assert replay["replay"]["observe_only"] is True
    assert replay["replay"]["expected_outbound"] == "none"

    outbox = _load_json(CONTRACT_DIR / "outbox_delivery.qq.private.text.json")
    assert outbox["delivery"]["status"] == "succeeded"
    assert outbox["delivery"]["echo_loop_guard"]["ttl_seconds"] > 0

    media_content = _load_json(CONTRACT_DIR / "media_asset_content.qq.image.json")
    assert media_content["content_access"]["access_path"].startswith(
        "/api/dashboard/media-assets/content"
    )
    assert media_content["content_access"]["runtime_path"].startswith("/v1/media-assets/")
    assert media_content["content_access"]["preview_ready"] is True

    event_stream = _load_json(CONTRACT_DIR / "agent_job_event_stream.rag_ingest.json")
    assert [event["sequence"] for event in event_stream["events"]] == [1, 2, 3]
    assert event_stream["events"][-1]["status"] == "succeeded"


def test_group_replay_fixture_manifest_is_complete() -> None:
    assert {path.name for path in REPLAY_DIR.glob("*.json")} == REQUIRED_REPLAYS


def test_group_replay_fixtures_define_quality_expectations() -> None:
    for path in sorted(REPLAY_DIR.glob("*.json")):
        data = _load_json(path)
        _assert_non_empty(data, "schema_version", path)
        _assert_non_empty(data, "case_id", path)
        profile = data.get("group_profile")
        assert isinstance(profile, dict), f"{path} group_profile must be object"
        assert profile.get("observe_only") is True, f"{path} must be observe-only"
        assert data.get("messages"), f"{path} needs replay messages"
        expected = data.get("expected")
        assert isinstance(expected, dict), f"{path} expected must be object"
        assert isinstance(expected.get("memory_types"), list), path
        assert isinstance(expected.get("source_message_ids"), list), path
        assert isinstance(expected.get("source_asset_ids"), list), path
        assert expected.get("observe_only_outbound") == "none", path
        for message in data["messages"]:
            _assert_non_empty(message, "message_id", path)
            _assert_non_empty(message, "sender_id", path)
            _assert_non_empty(message, "content", path)
            _parse_timestamp(str(message["timestamp"]), path)


def _load_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def _assert_non_empty(data: dict[str, Any], key: str, path: Path) -> None:
    value = data.get(key)
    assert isinstance(value, str) and value.strip(), f"{path} missing {key}"


def _assert_route(data: dict[str, Any], path: Path) -> None:
    for key in ("platform", "account_id", "conversation_id", "conversation_type"):
        _assert_non_empty(data, key, path)
    if data["kind"] not in {"MediaAsset"}:
        _assert_non_empty(data, "agent_id", path)


def _assert_timestamp(data: dict[str, Any], path: Path) -> None:
    _parse_timestamp(str(data.get("timestamp", "")), path)


def _parse_timestamp(value: str, path: Path) -> None:
    assert value, f"{path} missing timestamp"
    parsed = datetime.fromisoformat(value)
    assert parsed.tzinfo is not None, f"{path} timestamp must include timezone"
