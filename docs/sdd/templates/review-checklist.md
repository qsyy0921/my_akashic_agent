# SDD Review Checklist

Use this checklist for AI-generated or architecture-sensitive changes.

## Spec Alignment

- [ ] The change references a spec.
- [ ] All goals are implemented.
- [ ] Non-goals were not implemented accidentally.
- [ ] Public schema changes are documented.

## Architecture

- [ ] Domain logic is not embedded in IO adapters.
- [ ] Bootstrap code only wires dependencies.
- [ ] Cross-language boundaries use DTO/schema, not internal classes.
- [ ] No file grows past the project size threshold without justification.

## Messaging Safety

- [ ] Events have stable IDs.
- [ ] Consumers are idempotent.
- [ ] Retries cannot duplicate side effects.
- [ ] Dead-letter behavior is defined.
- [ ] Loop prevention is tested.

## Bot-to-Bot Safety

- [ ] Peer bot messages require a valid protocol tag.
- [ ] Signatures are verified before processing.
- [ ] TTL, budget, nonce, and duplicate-content checks are enforced.
- [ ] Unknown or invalid peer messages are observe-only.

## Tests

- [ ] Unit tests cover pure domain logic.
- [ ] Integration tests cover queue and adapter behavior.
- [ ] Failure paths are tested.
- [ ] Regression tests cover previous bugs.

## Review Result

- Decision: Accept | Request changes | Block
- Notes:

