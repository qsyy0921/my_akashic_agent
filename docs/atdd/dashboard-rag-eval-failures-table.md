# ATDD - Dashboard RAG Eval Failures Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `rag_eval_failures`
detail 时，应能直接看到 rag-eval failure totals 与 sampled rows，而不是只能
读取 raw JSON。

## Acceptance

1. panel 显示 `RAG Eval Failures`。
2. panel 显示 `No rag-eval failures sampled`。
3. panel 显示 `Quality`、`Passed`、`Updated At`。
4. unified goal verifier 返回：
   - `dashboard_read_models.rag_eval_failures_table=true`
