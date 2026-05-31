# Phase 8.85 Review: QQ Group File Notice Idempotency

## Scope

- Extended QQ/NapCat observe-only group upload handling to call the Go-owned
  `/v1/inbound-dedupe/check` runtime state before writing a session row,
  fetching a group file URL, downloading the file, or generating a preview.
- Added a separate file-event key instead of reusing ordinary message keys.
- Kept fallback behavior unchanged when no stable file-event identity is
  available.

## Key Design

File upload notice keys are selected in priority order:

```text
group_file_message:{group_id}:{message_id}
group_file_id:{group_id}:{file_id}
group_file_fingerprint:{group_id}:{sha256(group_id|user_id|busid|file_name|file_size)[:32]}
```

The fingerprint fallback is used only when `busid`, `file_name`, and
`file_size` are all present. This avoids silently deduping vague file notices
that only say "some group file".

The dedupe scope remains account-isolated:

```text
qq:{channel}:{bot_uin}
```

## Design Review

- This stays within the existing Go inbound dedupe bounded context; no new
  service is needed.
- Go owns deterministic TTL state and duplicate decisions.
- Python keeps platform SDK parsing, file download, preview generation, and
  observe-only session writes.
- Duplicate file notices do not trigger platform sends, model calls, or RAG
  jobs.

## Validation

- `uv run pytest tests\test_channel_clients.py::test_qq_group_upload_notice_records_file_preview tests\test_channel_clients.py::test_qq_group_upload_notice_uses_runtime_inbound_dedupe tests\test_channel_clients.py::test_qq_observe_group_uses_runtime_inbound_dedupe -q --basetemp .tmp\pytest-qq-file-notice-dedupe-target`
- `uv run pytest tests\test_channel_clients.py tests\test_agent_gateway_client.py -q --basetemp .tmp\pytest-qq-file-notice-dedupe-full-2`
- `uv run python -m py_compile infra\channels\qq_channel.py`
- `go test ./...`

## Residual Risk

- Real NapCat deployments may expose different file notice field names. The
  parser already supports dict/object `id`, `file_id`, `name`, `file_name`,
  `size`, and `busid`; new variants should be added as observed.
- The fingerprint fallback can theoretically suppress two identical file
  notices from the same user/group/busid/name/size inside the TTL. The fallback
  is deliberately conservative and only used when enough fields exist.

## Decision

Accept this slice as completing the QQ observe-only file notice idempotency
gap identified after the QQ message dedupe migration.
