from __future__ import annotations

from typing import Any

import httpx

from agent.config_models import AgentGatewayIntegrationConfig
from integrations.agent_runtime_inbox_source import AgentRuntimeInboxGroupMessageSource


def test_agent_runtime_inbox_source_fetches_cursor_ordered_group_rows():
    requests: list[dict[str, str]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(dict(request.url.params))
        assert request.url.path == "/v1/inbox"
        return httpx.Response(
            200,
            json={
                "code": "OK",
                "data": [
                    _event(
                        seq=4,
                        event_id="qq:1049511700:group:27234224:platform-4",
                        content="Boss A 二阶段先处理左边小怪",
                    ),
                    _event(
                        seq=5,
                        event_id="qq:1049511700:group:27234224:platform-5",
                        content="[QQ群 27234224 | 2948770636] Build B 堆暴击更稳",
                    ),
                ],
            },
        )

    source = AgentRuntimeInboxGroupMessageSource(
        AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.local",
            request_timeout_seconds=3,
        ),
        transport=httpx.MockTransport(handler),
    )

    rows = source.fetch_new_messages(
        session_key="qq:gqq:27234224",
        group_id="27234224",
        after_seq=3,
        limit=20,
    )

    assert requests == [
        {
            "channel_kind": "qq",
            "conversation_id": "27234224",
            "conversation_type": "group",
            "observe_only": "true",
            "after_seq": "3",
            "order": "asc",
            "limit": "20",
        }
    ]
    assert [row["seq"] for row in rows] == [4, 5]
    assert rows[0]["content"].startswith("[QQ群 27234224 | 2948770636]")
    assert rows[1]["content"] == "[QQ群 27234224 | 2948770636] Build B 堆暴击更稳"
    assert rows[0]["id"] == "qq:gqq:27234224:4"


def test_agent_runtime_inbox_source_skips_unstable_events_without_seq():
    def handler(_request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={
                "code": "OK",
                "data": [
                    {
                        "event_id": "qq:1049511700:group:27234224:no-seq",
                        "sender": {"id": "2948770636"},
                        "content": "missing seq",
                        "timestamp": "2026-05-30T00:00:00Z",
                        "metadata": {},
                    }
                ],
            },
        )

    source = AgentRuntimeInboxGroupMessageSource(
        AgentGatewayIntegrationConfig(
            enabled=True,
            base_url="http://agent-runtime.local",
        ),
        transport=httpx.MockTransport(handler),
    )

    rows = source.fetch_new_messages(
        session_key="qq:gqq:27234224",
        group_id="27234224",
        after_seq=-1,
        limit=20,
    )

    assert rows == []


def _event(*, seq: int, event_id: str, content: str) -> dict[str, Any]:
    return {
        "event_id": event_id,
        "channel": {
            "kind": "qq",
            "account_id": "1049511700",
            "conversation_id": "27234224",
            "conversation_type": "group",
        },
        "sender": {"id": "2948770636", "kind": "human"},
        "content": content,
        "timestamp": f"2026-05-30T00:00:{seq:02d}Z",
        "metadata": {
            "observe_only": "true",
            "seq": str(seq),
            "session_key": "qq:gqq:27234224",
            "session_message_id": f"qq:gqq:27234224:{seq}",
        },
    }
