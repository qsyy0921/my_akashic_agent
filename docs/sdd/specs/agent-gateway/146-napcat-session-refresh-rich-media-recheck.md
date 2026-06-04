# Spec 146: NapCat Session Refresh Rich Media Recheck

## Summary

When cross-group native NapCat rich-media parity still fails, the repo must
record whether a plain container restart is sufficient to recover the session
or whether manual QQ re-login / session replacement is required.

## Requirements

1. The live verification workflow must be allowed to restart the relevant
   NapCat container and then re-run the native rich-media comparison on a real
   QQ group.
2. The workflow must re-check receiver status after restart so transport
   connectivity is separated from rich-media capability.
3. If restart preserves connected receiver status but native rich-media still
   fails with the same `rich media transfer failed` error, the blocker must be
   tightened from generic session suspicion to "manual re-login or replacement
   of the current NapCat / QQ rich-media session is required".
4. The resulting Go execution-owner conclusion must remain narrow:
   - `text_only` may stay enabled when already proven safe;
   - rich-media owner expansion remains blocked until a fresh session disproves
     the failure.

## Non-Goals

- This spec does not implement automatic QQ login.
- This spec does not claim Telegram backend readiness.
- This spec does not change QQ send semantics.
