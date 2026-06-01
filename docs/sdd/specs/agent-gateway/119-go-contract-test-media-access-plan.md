# SPEC-119: Go Contract Test For Media Access Plan

## Status

Accepted for the current iteration.

## Context

`media_asset_content_access_plan.qq.image.json` now documents the shared
contract between Go runtime and Python dashboard, but the Go contract fixture
test still does not load or validate it. That means Python validates the new
fixture while Go can drift silently.

## Boundary Analysis

Go owns:

- loading and validating shared contract fixtures;
- asserting runtime-owned fields for media access plan.

Python owns:

- its own contract round-trip test and dashboard behavior.

Out of scope:

- changing runtime behavior;
- changing dashboard behavior;
- content streaming, OCR/VLM/RAG/AI, or media mutation.

## Decision

Update `services/agent-runtime/contract_fixtures_test.go` to:

- include `media_asset_content_access_plan.qq.image.json` in the required
  fixture list;
- require `content_access_plan` extra field;
- assert stable plan shape: `ready=true`, `reason=media_asset_content_ready`,
  `side_effect=none`, runtime/dashboard paths, content endpoint, and required
  step names.

## Acceptance

- Go contract fixture tests pass.
- Python contract fixture tests still pass.
- `go test ./...` passes.
- TODO is cleared at the end of the iteration.
