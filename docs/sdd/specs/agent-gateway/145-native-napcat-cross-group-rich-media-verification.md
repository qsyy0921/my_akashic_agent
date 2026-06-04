# Spec 145: Native NapCat Cross-Group Rich Media Verification

## Summary

When native NapCat rich-media parity already fails on one QQ group, the repo
must support repeating the same native comparison on additional observe-only
groups so the blocker can be classified as group-specific or session-wide.

## Requirements

1. The existing native comparison entrypoint
   `scripts/run-napcat-native-rich-media-smoke.ps1` must remain reusable with a
   caller-supplied `GroupId`.
2. The live verification workflow must run the native comparison on at least
   two additional enabled QQ observe-only groups beyond the original baseline
   group.
3. If image/file upload staging succeeds but final `send_group_msg` and
   `upload_group_file` fail with the same `rich media transfer failed` error on
   multiple groups under the same NapCat session, the blocker must be
   classified as session-wide or account-session-wide evidence, not a
   single-group anomaly.
4. The resulting execution-owner conclusion must stay narrow:
   - text-only Go outbox owner may remain enabled if it is already proven safe;
   - rich-media owner expansion must remain blocked until a fresh NapCat/QQ
     session or equivalent external change disproves the session-wide failure.

## Non-Goals

- This spec does not change QQ send semantics.
- This spec does not add automatic NapCat re-login or account rotation.
- This spec does not make Telegram backend ready.
