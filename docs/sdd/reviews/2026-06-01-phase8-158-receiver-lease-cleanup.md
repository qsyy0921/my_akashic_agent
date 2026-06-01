# Phase 8.158 Review: Receiver Lease Cleanup

日期：2026-06-01

## 范围

- 为 Go `agent-runtime` receiver lease 增加显式过期清理用例。
- 新增 `POST /v1/receiver-leases/cleanup-expired`。
- 补 service / HTTP 单元测试。
- 更新 SDD TODO / DONE / LIVE_CHECKS。

## 架构检查

- DDD 分层保持一致：DTO、command、query、inport、app service、HTTP trigger 分别承载各自职责。
- Go 只处理确定性 runtime lease 生命周期，不接管 Python receiver 进程，不触发 QQ/Telegram 登录、采集、回复或 AI pipeline。
- cleanup endpoint 是显式 operator/supervisor 动作，没有新增后台 scheduler，避免和现有 receiver lease acquire/renew/release 行为产生隐藏副作用。

## 行为

- 以请求 timestamp 或当前时间判断 lease 是否过期。
- 删除过期 lease 的内存态和 repository 记录。
- 保留活跃 lease。
- 返回 deleted / remaining 列表、summary totals、`side_effect=runtime_state_only`。

## 验证

- `go test ./app/service ./trigger/http`
- `go test ./...`
- `git diff --check`

## 风险

- cleanup 是显式接口，不会自动清理。后续如需定时清理，应另做 runtime worker/scheduler 设计，并保留 operator 可见性。
- 当前 repository 删除逐条执行；如果未来 repository 变为远程存储，可再增加批量删除 port，但本轮不为未出现的存储形态扩展接口。
