# SDD Review: Go Message Gateway Structure

## Scope

Review the first Go-side message gateway skeleton and confirm it follows the
selected DDD plus hexagonal package structure.

## Reviewed Artifacts

- `docs/sdd/adr/0001-message-gateway-architecture.md`
- `docs/sdd/specs/message-gateway/005-go-package-structure.md`
- `services/message-gateway`

## Architecture Decision

Go-side backend infrastructure will use:

```text
api
app
domain
infrastructure
trigger
types
```

This matches the Java-style package taxonomy the project wants to present in
resume and interview contexts, while still preserving hexagonal dependency
direction.

## Dependency Check

Expected dependency direction:

```text
trigger        -> api, app
app            -> domain, types
infrastructure -> app, domain, types
api            -> types
domain         -> types only when unavoidable
types          -> none
```

Observed in the initial skeleton:

- `trigger/http` depends on `api/dto` and `app` ports/commands.
- `app/service` depends on `app/assembler`, `app/command`, and outbound ports.
- `app/assembler` depends on `domain/model`.
- `infrastructure/memory` implements an app outbound port contract through the
  `domain/model` envelope type.
- `domain` does not depend on `app`, `trigger`, `api`, or `infrastructure`.
- `types` has no project-internal dependencies.

## Verification

Completed:

- SDD docs were added for the package structure.
- Go message gateway files were added under `services/message-gateway`.
- Git ignore rules were adjusted so `docs/sdd` is tracked.
- Go `go1.26.3` was downloaded from `go.dev/dl`, installed locally under
  `%USERPROFILE%\.codex\tools\go1.26.3`, and verified by SHA256.
- `gofmt` completed for all Go gateway packages.
- `go test ./...` passed.
- `go build ./cmd/message-gateway` passed.
- A Go architecture test now enforces the selected layer dependency rules.

## Follow-Up

- Keep the local Go toolchain available in the shell PATH before running
  commands:

```powershell
$goRoot = "$env:USERPROFILE\.codex\tools\go1.26.3"
$env:PATH = "$goRoot\bin;$env:PATH"
cd E:\agent\akashic\services\message-gateway
gofmt -w architecture_test.go api app cmd domain infrastructure trigger types
go test ./...
```

- Add the same checks to CI before adding NATS/NapCat adapters.
