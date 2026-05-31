# 031 Agent Worker Status

日期：2026-05-31

## 目标

将 Python AI worker 的确定性运行态交给 Go `agent-runtime` 管理，覆盖 image generation、knowledge、rag eval 和兼容 outbox worker 的 liveness、当前任务、累计成功/失败、最近错误和 stale 判断。

Python 仍负责模型调用、RAGFlow 上传、图片生成工具和实际 AI pipeline；Go 只负责状态登记、持久化、只读诊断和 runtime overview 聚合。

## 边界

- Go `domain`：`AgentWorkerStatus` 领域模型，定义 `starting|idle|running|failed|stopped` 状态和 stale heartbeat 规则。
- Go `app`：`AgentWorkerStatusService` 提供 `ReportAgentWorkerStatus` 和 `ListAgentWorkerStatuses` 用例。
- Go `infrastructure`：默认文件态 `agent-worker-statuses.json`，支持 `AKASHIC_AGENT_WORKER_STATUSES_DSN/PATH`，`memory` 可回到内存态。
- Go `trigger`：`POST /v1/agent-worker-statuses/report` 写入状态，`GET /v1/agent-worker-statuses` 只读查询。
- Python：worker 在启动、空闲、执行任务、成功、失败、停止时 best-effort 上报；上报失败不影响任务执行。

## 不做

- 不把 LLM、OCR/VLM、图片生成、RAGFlow indexer 执行迁移到 Go。
- 不用 worker status 驱动调度或发送，只做诊断和运行态可观测。
- 不泄漏平台 token、lease token 或模型密钥。

## API

```http
POST /v1/agent-worker-statuses/report
```

```json
{
  "worker_id": "akashic-python-worker",
  "worker_type": "knowledge",
  "status": "running",
  "current_job_id": "group_memory_extract:qq:284331268:bucket",
  "last_job_id": "",
  "last_error": "",
  "processed_total": 12,
  "failed_total": 1,
  "source": "python",
  "metadata": {"reason": "no_job"}
}
```

```http
GET /v1/agent-worker-statuses?stale_after_seconds=180
```

响应包含 `workers`、`totals`、`side_effect=none`。超过 stale 阈值的 `starting|idle|running` worker 在读时降级为 `stopped`，并带 `stale=true` 与 `metadata.last_status`。

## Runtime Overview

`GET /v1/runtime-overview` 增加：

- `agent_workers`
- `summary.agent_workers`
- `summary.agent_workers_running`
- `summary.agent_workers_idle`
- `summary.agent_workers_failed`
- `summary.agent_workers_stale`
- `cards[id=agent_workers]`

## 验收

- Go service 单测覆盖状态上报和 stale 降级。
- Go store 单测覆盖默认 JSON 文件持久化。
- HTTP handler 单测覆盖 report/list。
- Python client 单测覆盖 report payload。
- 至少一个 Python worker 单测覆盖 no-job idle 上报。
