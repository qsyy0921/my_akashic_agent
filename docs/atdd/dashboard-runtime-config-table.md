# ATDD - Dashboard Runtime Config Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `runtime_config` detail 时，应能
直接看到 runtime 地址、QQ group send toggle、Telegram token 配置、OneBot
endpoint 和 worker flags，而不是只能读取 raw JSON。

## Acceptance

1. panel 显示 `Runtime Config`。
2. panel 显示 `QQ Group Send`、`OneBot Endpoints`、`Strict Lease Token` 和
   `Environment Keys`。
3. old-payload fallback case 仍能生成 `runtime_config` card，且 `value` 使用
   `host:port` 而不是完整 URL。
4. unified goal verifier 返回 `dashboard_read_models.runtime_config_table=true`。
