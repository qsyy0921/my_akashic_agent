# TDD: QQ Cutover Route Matrix Live Verifier

## Scope

- 保护 QQ cutover route matrix 的构造逻辑与 unified goal verifier 的消费边界。

## Target Code Paths

- `scripts/verify_qq_cutover_route_matrix.py`
- `scripts/verify-qq-cutover-route-matrix.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| current partial cutover | unit | route matrix 正确列出 Go scope、policy block 和 platform blocker |
| first-account group file drift | unit | 若 first-account group file 混入 Go scope，则 verifier failed |
| unified goal verifier wiring | integration | goal verifier 能输出 `qq_cutover_route_matrix` 相关字段 |

## Required Automated Tests

- `uv run pytest tests/test_verify_qq_cutover_route_matrix.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-qq-cutover-route-matrix.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Deferred Coverage

- rich-media native probe 本身仍留给独立 live smoke / 人工对照；
  本 verifier 只消费当前 runtime gate 和既定 blocker taxonomy。
