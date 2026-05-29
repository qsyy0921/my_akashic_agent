# SPEC-002: Reference Architecture Analysis

## Status

Draft

## Context

Akashic should not copy another agent project wholesale. The useful goal is to
extract stable engineering patterns from mature agent systems and adapt them to
Akashic's QQ/Telegram/group-memory use case.

Reviewed references:

- Local Hermes source: `E:\agent\Hermes`
- Local Claude Code restoration/analysis: `E:\agent\ccb`
- Local CyberClaw source: `E:\agent\cyberclaw`
- OpenClaw docs and source documentation:
  - `https://openclawdoc.com/docs/agents/overview/`
  - `https://github.com/openclaw/openclaw/blob/main/docs/tools/skills.md`
  - `https://openclawlab.com/en/docs/concepts/multi-agent/`

## Claude Code / CCB Patterns

Observed from `E:\agent\ccb\docs\introduction\architecture-overview.mdx`,
`E:\agent\ccb\docs\conversation\the-loop.mdx`, and
`E:\agent\ccb\docs\design\tool-search-design-guide.md`.

Useful patterns:

- Five-layer split: interaction, orchestration, core loop, tools, provider
  communication.
- A turn is a state machine, not a recursive blob: preprocess context, stream
  model, execute tools, decide continue/stop/recover.
- Context processing is a pipeline: tool-result budgeting, compression, collapse,
  auto-compact, then provider call.
- Tool arrays should stay stable. Use core tools plus deferred tool discovery
  and proxy execution to avoid prompt-cache churn.
- Permission checks are part of the tool boundary: validate input, check policy,
  execute, render result.
- Recovery transitions must be explicit: prompt-too-long, max-output recovery,
  stop-hook blocking, abort, and max-turns.

Akashic adaptation:

- Python should own the agent turn state machine and context pipeline.
- Go should not implement model loops, but it should record turn/job state and
  expose durable events.
- Tool search should become a first-class Python service with a stable core set
  and deferred catalog, rather than ad hoc dynamic injection.

## Hermes Patterns

Observed from `E:\agent\Hermes\README.md`,
`E:\agent\Hermes\AGENTS.md`, and gateway/tool modules.

Useful patterns:

- Separate CLI and messaging gateway surfaces. Messaging platforms are gateway
  concerns; model/tool execution remains agent runtime.
- Toolsets are configurable by platform/session. A Telegram session does not need
  the same tool surface as a CLI coding session.
- Sessions, profiles, logs, approval state, and gateway context are explicit
  data, not implicit global state.
- Skills are operational procedures loaded with progressive disclosure.
- Approval and safety state is session-scoped, not process-global.
- Large monolithic files are a cautionary example: they work, but Akashic should
  avoid adding more behavior into single files like `passive_turn.py` or
  `qq_channel.py`.

Akashic adaptation:

- Use Go for gateway/session routing and Python for agent/toolset selection.
- Keep per-channel tool policies: QQ group observation should not see the same
  tools as a private owner chat.
- Move platform approvals, outbound ledgers, and pending delivery to Go; keep
  shell/file/code permission logic in Python tool hooks.

## OpenClaw Patterns

Observed from public docs.

Useful patterns:

- Agent = scoped brain with model, memory, tools, and channel layer.
- Multi-agent routing distinguishes `agentId`, `accountId`, and binding rules.
- Each agent can have its own workspace, state directory, sessions, and auth.
- Skill precedence and allowlists are explicit: workspace, project-agent,
  personal-agent, managed/local, bundled, extra dirs.
- Skills are snapshotted per session and can refresh through controlled watchers.
- Environment injection is scoped to a run, then restored.

Akashic adaptation:

- Introduce `AgentId`, `AccountId`, `ConversationId`, and `Binding` as Go domain
  concepts.
- Treat the two QQ accounts as channel accounts, not separate Python channels.
- Allow future specialized agents, such as `diy-hardware-observer` and
  `game-guide-curator`, with separate memory/workspace policies.
- Skill visibility should be configured by agent/session/channel instead of
  globally enabled.

## CyberClaw Patterns

Observed from `E:\agent\cyberclaw\README.md`,
`E:\agent\cyberclaw\cyberclaw\core\agent.py`, and
`E:\agent\cyberclaw\docs\LAZY_LOADING_GUIDE.md`.

Useful patterns:

- Transparent event audit: LLM input, tool call, tool result, AI message, system
  action.
- Two-phase skill use: inspect/help first, then run with explicit task input.
- Lazy skill loading: scan metadata cheaply, load full instructions only on use.
- Sandbox-first tools: path and shell restrictions are designed as core
  behavior, not optional UI warnings.

Akashic adaptation:

- Add a durable Go audit stream for platform and job events.
- Keep Python lifecycle event hooks, but mirror important events to Go for
  replay and dashboard inspection.
- Convert heavy skills/tools to progressive disclosure: short catalog in prompt,
  full instruction loaded only when selected.
- Use policy gates before side effects, with tests for shell/file/message/image
  actions.

## Resulting Principles

- Go owns infrastructure truth; Python owns intelligence truth.
- Every side effect has an event id, owner account, session, policy decision, and
  audit record.
- Every agent turn is a state machine with named transitions and bounded
  retries.
- Skills/tools are discoverable and allowlisted; full instructions are loaded
  lazily.
- Dashboard reads typed APIs, not implementation internals.

