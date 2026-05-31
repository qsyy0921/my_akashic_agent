# 042 Proactive Drift State

Date: 2026-05-31

## 背景

Proactive drift 的运行入口和 LLM/tool 执行仍在 Python，但 drift 完成后的确定性状态
仍保存在 Python workspace 文件中：

- `drift/drift.json`：近期 run 摘要和 note。
- `drift/skills/<skill>/state.json`：单个 skill 的 `last_run_at`、`run_count`、
  `status` 和 `next`。

这些状态不是模型推理本身，而是 runtime 控制面状态，适合交给 Go 统一持久化和诊断。
当前 Go proactive state 已经持久化 delivery、seen、cooldown、context-only、
drift_last_at、AnyAction quota 和 bg-context global mark，本切片复用同一个
`ProactiveStateService` 和 `proactive-state.json`，不新增服务。

## 范围

新增到 Go：

- drift skill state domain model：`skill_name`、`last_run_at`、`run_count`、
  `status`、`next`。
- drift recent run domain model：`skill`、`run_at`、`one_line`、
  `message_result`。
- `POST /v1/proactive/drift/finish`
- `GET /v1/proactive/drift/summary`
- `GET /v1/proactive/drift/skills/{skill_name}`
- file-backed `proactive-state.json` 和 in-memory store 同步支持。

接入到 Python：

- `DriftStateStore.save_finish()` 成功完成 drift 后优先写 Go runtime，同时继续写本地
  JSON mirror 作为 fallback。
- `DriftStateStore.load_drift()` 优先读取 Go summary，并与本地 fallback 合并，避免
  迁移期间丢失旧 recent runs。
- `scan_skills()` 读取单 skill state 时优先读取 Go skill state，再与本地 fallback 取
  较新的状态。

暂不迁移：

- skill 文件扫描和 frontmatter 解析。
- `read_file` / `write_file` drift tool 的文件读写能力。
- LLM 决策、MCP tool 调用、prompt 构造。
- tick log 和 semantic candidate SQLite 表。

## API

```text
POST /v1/proactive/drift/finish
GET  /v1/proactive/drift/summary?limit=10
GET  /v1/proactive/drift/skills/{skill_name}
```

Finish request:

```json
{
  "skill_used": "explore-curiosity",
  "one_line": "整理了今日群里出现的攻略线索",
  "next": "继续核验来源",
  "message_result": "silent",
  "note": "可选短备注",
  "timestamp": "2026-05-31T10:00:00Z"
}
```

Summary response:

```json
{
  "version": 1,
  "recent_runs": [
    {
      "skill": "explore-curiosity",
      "run_at": "2026-05-31T10:00:00Z",
      "one_line": "整理了今日群里出现的攻略线索",
      "message_result": "silent"
    }
  ],
  "note": "可选短备注",
  "side_effect": "none"
}
```

## 分层

```text
trigger/http
  -> app/port/in/ProactiveStateManager
  -> app/service/ProactiveStateService
  -> app/port/out/ProactiveStateRepository
  -> domain/model/ProactiveDriftSkillState + ProactiveDriftRecentRun
  -> infrastructure/proactivestate + infrastructure/memory
```

## 规则

- `skill_used` 必须非空，并裁剪到 80 字符。
- `one_line` 裁剪到 150 字符。
- `next` 裁剪到 100 字符。
- `note` 裁剪到 150 字符；空 note 不覆盖已有 note。
- `message_result` 只允许 `sent` / `silent`，其它值归一为 `silent`。
- `run_count` 在 Go 中按 skill 原子递增。
- `status` 与旧 Python 行为保持一致，finish 后写为 `in_progress`。
- summary 只读接口必须 `side_effect=none`。
- Python fallback 文件继续写，直到 live 观察确认 Go state 足够稳定。

## 验收

- Go domain/service/http/store tests 覆盖 finish、summary、skill state 查询和文件重载。
- Python bridge tests 覆盖 `DriftStateStore.save_finish()` 调用 Go 后仍写本地 mirror。
- Runtime 不可用时 Python drift state 保持旧 JSON fallback 行为。
- `go test ./...`、`go vet ./...`、Go build、相关 Python pytest 和 py_compile 通过。
