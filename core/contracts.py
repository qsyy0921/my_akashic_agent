from __future__ import annotations

import json
from copy import deepcopy
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any, Mapping


JSONMap = dict[str, Any]


class ContractValidationError(ValueError):
    pass


@dataclass(frozen=True)
class ChannelRef:
    account_id: str
    conversation_id: str
    conversation_type: str
    platform: str = ""
    kind: str = ""

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any]) -> "ChannelRef":
        return cls(
            platform=str(data.get("platform", "")),
            kind=str(data.get("kind", "")),
            account_id=str(data.get("account_id", "")),
            conversation_id=str(data.get("conversation_id", "")),
            conversation_type=str(data.get("conversation_type", "")),
        )

    @property
    def route_platform(self) -> str:
        return self.platform or self.kind

    def to_dict(self) -> JSONMap:
        data: JSONMap = {}
        if self.kind:
            data["kind"] = self.kind
        if self.platform:
            data["platform"] = self.platform
        data["account_id"] = self.account_id
        data["conversation_id"] = self.conversation_id
        data["conversation_type"] = self.conversation_type
        return data


@dataclass(frozen=True)
class SenderRef:
    id: str
    display_name: str = ""
    kind: str = ""
    extra: JSONMap = field(default_factory=dict)

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any]) -> "SenderRef":
        known = {"id", "display_name", "kind"}
        return cls(
            id=str(data.get("id", "")),
            display_name=str(data.get("display_name", "")),
            kind=str(data.get("kind", "")),
            extra={key: deepcopy(value) for key, value in data.items() if key not in known},
        )

    def to_dict(self) -> JSONMap:
        data: JSONMap = {"id": self.id}
        if self.display_name:
            data["display_name"] = self.display_name
        if self.kind:
            data["kind"] = self.kind
        data.update(deepcopy(self.extra))
        return data


@dataclass(frozen=True)
class AttachmentRef:
    id: str
    kind: str
    url: str = ""
    mime_type: str = ""
    name: str = ""
    size_bytes: int | None = None
    extra: JSONMap = field(default_factory=dict)

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any]) -> "AttachmentRef":
        known = {"id", "kind", "url", "mime_type", "name", "size_bytes"}
        size = data.get("size_bytes")
        return cls(
            id=str(data.get("id", "")),
            kind=str(data.get("kind", "")),
            url=str(data.get("url", "")),
            mime_type=str(data.get("mime_type", "")),
            name=str(data.get("name", "")),
            size_bytes=int(size) if isinstance(size, int | float) else None,
            extra={key: deepcopy(value) for key, value in data.items() if key not in known},
        )

    def to_dict(self) -> JSONMap:
        data: JSONMap = {}
        if self.id:
            data["id"] = self.id
        data["kind"] = self.kind
        if self.url:
            data["url"] = self.url
        if self.mime_type:
            data["mime_type"] = self.mime_type
        if self.name:
            data["name"] = self.name
        if self.size_bytes is not None:
            data["size_bytes"] = self.size_bytes
        data.update(deepcopy(self.extra))
        return data


@dataclass(frozen=True)
class CitationRef:
    citation_id: str
    source_type: str
    source_id: str
    quote: str = ""
    timestamp: str = ""
    extra: JSONMap = field(default_factory=dict)

    @classmethod
    def from_mapping(cls, data: Mapping[str, Any]) -> "CitationRef":
        known = {"citation_id", "source_type", "source_id", "quote", "timestamp"}
        return cls(
            citation_id=str(data.get("citation_id", "")),
            source_type=str(data.get("source_type", "")),
            source_id=str(data.get("source_id", "")),
            quote=str(data.get("quote", "")),
            timestamp=str(data.get("timestamp", "")),
            extra={key: deepcopy(value) for key, value in data.items() if key not in known},
        )

    def to_dict(self) -> JSONMap:
        data: JSONMap = {
            "citation_id": self.citation_id,
            "source_type": self.source_type,
            "source_id": self.source_id,
        }
        if self.quote:
            data["quote"] = self.quote
        if self.timestamp:
            data["timestamp"] = self.timestamp
        data.update(deepcopy(self.extra))
        return data


@dataclass(frozen=True)
class ContractFixture:
    schema_version: str
    kind: str
    platform: str
    account_id: str
    conversation_id: str
    conversation_type: str
    timestamp: str
    event_id: str = ""
    job_id: str = ""
    agent_id: str = ""
    channel: ChannelRef | None = None
    sender: SenderRef | None = None
    content: str | None = None
    attachments: list[AttachmentRef] = field(default_factory=list)
    citations: list[CitationRef] = field(default_factory=list)
    source_message_ids: list[str] = field(default_factory=list)
    source_asset_ids: list[str] = field(default_factory=list)
    metadata: JSONMap = field(default_factory=dict)
    extra: JSONMap = field(default_factory=dict)

    @classmethod
    def from_dict(cls, data: Mapping[str, Any]) -> "ContractFixture":
        known = {
            "schema_version",
            "kind",
            "event_id",
            "job_id",
            "platform",
            "account_id",
            "conversation_id",
            "conversation_type",
            "agent_id",
            "channel",
            "sender",
            "content",
            "attachments",
            "citations",
            "timestamp",
            "source_message_ids",
            "source_asset_ids",
            "metadata",
        }
        channel_raw = data.get("channel")
        sender_raw = data.get("sender")
        fixture = cls(
            schema_version=str(data.get("schema_version", "")),
            kind=str(data.get("kind", "")),
            event_id=str(data.get("event_id", "")),
            job_id=str(data.get("job_id", "")),
            platform=str(data.get("platform", "")),
            account_id=str(data.get("account_id", "")),
            conversation_id=str(data.get("conversation_id", "")),
            conversation_type=str(data.get("conversation_type", "")),
            agent_id=str(data.get("agent_id", "")),
            channel=(
                ChannelRef.from_mapping(channel_raw)
                if isinstance(channel_raw, Mapping)
                else None
            ),
            sender=(
                SenderRef.from_mapping(sender_raw)
                if isinstance(sender_raw, Mapping)
                else None
            ),
            content=(
                str(data["content"])
                if "content" in data and data.get("content") is not None
                else None
            ),
            attachments=_parse_attachment_list(data.get("attachments", [])),
            citations=_parse_citation_list(data.get("citations", [])),
            timestamp=str(data.get("timestamp", "")),
            source_message_ids=_parse_string_list(data.get("source_message_ids", [])),
            source_asset_ids=_parse_string_list(data.get("source_asset_ids", [])),
            metadata=_parse_metadata(data.get("metadata")),
            extra={key: deepcopy(value) for key, value in data.items() if key not in known},
        )
        fixture.validate()
        return fixture

    @classmethod
    def from_json(cls, value: str) -> "ContractFixture":
        data = json.loads(value)
        if not isinstance(data, Mapping):
            raise ContractValidationError("contract fixture must be a JSON object")
        return cls.from_dict(data)

    @classmethod
    def from_path(cls, path: str | Path) -> "ContractFixture":
        return cls.from_json(Path(path).read_text(encoding="utf-8"))

    @property
    def stable_id(self) -> str:
        return self.event_id or self.job_id

    def validate(self) -> None:
        _require_text(self.schema_version, "schema_version")
        _require_text(self.kind, "kind")
        if not self.event_id and not self.job_id:
            raise ContractValidationError("event_id or job_id is required")
        _require_text(self.platform, "platform")
        _require_text(self.account_id, "account_id")
        _require_text(self.conversation_id, "conversation_id")
        _require_text(self.conversation_type, "conversation_type")
        if self.kind != "MediaAsset":
            _require_text(self.agent_id, "agent_id")
        _parse_timestamp(self.timestamp, "timestamp")
        if self.channel is not None:
            self._validate_channel_route()
        if not isinstance(self.metadata, dict):
            raise ContractValidationError("metadata must be an object")
        for citation in self.citations:
            _require_text(citation.citation_id, "citations[].citation_id")
            _require_text(citation.source_type, "citations[].source_type")
            _require_text(citation.source_id, "citations[].source_id")
            if citation.timestamp:
                _parse_timestamp(citation.timestamp, "citations[].timestamp")

    def _validate_channel_route(self) -> None:
        assert self.channel is not None
        if self.channel.route_platform != self.platform:
            raise ContractValidationError("channel platform must match top-level route")
        if self.channel.account_id != self.account_id:
            raise ContractValidationError("channel account_id must match top-level route")
        if self.channel.conversation_id != self.conversation_id:
            raise ContractValidationError(
                "channel conversation_id must match top-level route"
            )
        if self.channel.conversation_type != self.conversation_type:
            raise ContractValidationError(
                "channel conversation_type must match top-level route"
            )

    def to_dict(self) -> JSONMap:
        data: JSONMap = {
            "schema_version": self.schema_version,
            "kind": self.kind,
        }
        if self.event_id:
            data["event_id"] = self.event_id
        if self.job_id:
            data["job_id"] = self.job_id
        data.update(
            {
                "platform": self.platform,
                "account_id": self.account_id,
                "conversation_id": self.conversation_id,
                "conversation_type": self.conversation_type,
            }
        )
        if self.agent_id:
            data["agent_id"] = self.agent_id
        if self.channel is not None:
            data["channel"] = self.channel.to_dict()
        if self.sender is not None:
            data["sender"] = self.sender.to_dict()
        if self.content is not None:
            data["content"] = self.content
        if self.attachments:
            data["attachments"] = [item.to_dict() for item in self.attachments]
        elif "attachments" in self.extra:
            data["attachments"] = []
        if self.citations:
            data["citations"] = [item.to_dict() for item in self.citations]
        data.update(deepcopy(self.extra))
        data["timestamp"] = self.timestamp
        data["source_message_ids"] = list(self.source_message_ids)
        data["source_asset_ids"] = list(self.source_asset_ids)
        data["metadata"] = deepcopy(self.metadata)
        return data

    def to_json(self) -> str:
        return json.dumps(self.to_dict(), ensure_ascii=False, sort_keys=True)


def _parse_string_list(value: Any) -> list[str]:
    if not isinstance(value, list):
        raise ContractValidationError("source id fields must be lists")
    return [str(item) for item in value]


def _parse_attachment_list(value: Any) -> list[AttachmentRef]:
    if not isinstance(value, list):
        raise ContractValidationError("attachments must be a list")
    return [
        AttachmentRef.from_mapping(item)
        for item in value
        if isinstance(item, Mapping)
    ]


def _parse_citation_list(value: Any) -> list[CitationRef]:
    if not isinstance(value, list):
        raise ContractValidationError("citations must be a list")
    return [CitationRef.from_mapping(item) for item in value if isinstance(item, Mapping)]


def _parse_metadata(value: Any) -> JSONMap:
    if value is None:
        raise ContractValidationError("metadata must be an object")
    if not isinstance(value, Mapping):
        raise ContractValidationError("metadata must be an object")
    return {str(key): deepcopy(item) for key, item in value.items()}


def _require_text(value: str, field_name: str) -> None:
    if not value.strip():
        raise ContractValidationError(f"{field_name} is required")


def _parse_timestamp(value: str, field_name: str) -> datetime:
    if not value:
        raise ContractValidationError(f"{field_name} is required")
    parsed = datetime.fromisoformat(value)
    if parsed.tzinfo is None:
        raise ContractValidationError(f"{field_name} must include timezone")
    return parsed
