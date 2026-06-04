# Spec 147: Dual-Account Rich Media Split Verification

## Summary

When one QQ account shows rich-media failures, the repo must allow verification
against a second NapCat / QQ account so the remaining blocker can be split by
account and media kind instead of being treated as a single opaque failure.

## Requirements

1. The verification workflow must be able to:
   - inspect whether a second QQ account has accessible groups;
   - run native NapCat rich-media comparison on that second account;
   - run Akashic manual delivery-dispatch on the same second-account group.
2. If second-account native and Akashic file send both succeed while image send
   still fails natively and through Akashic, the blocker classification must be
   narrowed as follows:
   - image failure is not caused by Akashic adapter logic and affects at least
     both current QQ sessions;
   - file failure on the first account is account-session-specific, not a
     universal rich-media failure.
3. Go outbox owner conclusions must stay precise:
   - `text_only` remains the only safe default execution-owner scope today;
   - broadening to `image` is blocked by cross-account native failure;
   - broadening to `file` globally is blocked because the current runtime has no
     per-account capability gate and the first account still fails.

## Non-Goals

- This spec does not implement per-account execution-owner gating.
- This spec does not solve Telegram backend configuration.
- This spec does not change QQ send semantics.
