# Spec 143: Native NapCat Rich Media Comparison

## Summary

When QQ rich-media live smoke fails through the Go OneBot adapter, the repo
must provide a native NapCat/OneBot comparison path that replays the same
`upload_file_stream -> send_group_msg/upload_group_file` protocol without
Akashic outbox or delivery-dispatch layers.

## Requirements

1. The repo must include a runnable local script that:
   - connects directly to the configured NapCat WebSocket endpoint;
   - uploads a local image and a local file using the same streamed payload
     shape as `services/agent-runtime/infrastructure/onebotdelivery/adapter.go`;
   - sends the image to a real QQ group with `send_group_msg`;
   - sends the file to the same group with `upload_group_file`;
   - returns raw OneBot responses as JSON.
2. The comparison path must be able to distinguish:
   - native NapCat/QQ failure;
   - Akashic-only failure;
   - insufficient evidence.
3. If native NapCat returns the same platform error as Akashic, the blocker
   must be classified as NapCat/QQ session capability, not Go adapter logic.
4. Outbox default execution-owner conclusions must respect this evidence:
   - if rich media is still blocked natively and the local outbox worker is a
     global on/off switch, the repo must not claim that a full default Go
     cutover is safe yet.

## Non-Goals

- This spec does not change QQ message send semantics.
- This spec does not add per-message-kind execution-owner routing.
- This spec does not attempt to solve Telegram backend configuration.
