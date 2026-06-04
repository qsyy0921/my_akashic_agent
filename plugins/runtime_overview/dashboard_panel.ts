/// <reference path="../../types/akashic-dashboard.d.ts" />

interface RuntimeOverviewCard {
  id: string;
  label: string;
  value: string | number | boolean;
  status: string;
  detail: Record<string, unknown>;
}

interface RuntimeOverviewResponse {
  summary: Record<string, string | number | boolean>;
  cards: RuntimeOverviewCard[];
  status: Record<string, unknown>;
  jobs_by_status: Record<string, number>;
  jobs_by_type: Record<string, number>;
  outbox_by_status: Record<string, number>;
  recent_events: Record<string, unknown>[];
  checkpoint_lag: Record<string, unknown>[];
}

interface DeliveryAdapterHealthResponse {
  items: Record<string, unknown>[];
  totals: Record<string, number>;
  status: Record<string, unknown>;
}

interface DeliverySmokeReadinessResponse {
  ready: boolean;
  reason: string;
  cases: Record<string, unknown>[];
  totals: Record<string, number>;
  status: Record<string, unknown>;
  side_effect: string;
}

function _runtimeStatusTag(status: string): string {
  const cls = `runtime-overview-status runtime-overview-${status || "muted"}`;
  return `<span class="${cls}">${escapeHtml(status || "muted")}</span>`;
}

function _formatJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value ?? "");
  }
}

function _short(value: unknown, limit: number): string {
  const text = String(value ?? "").replace(/\s+/g, " ").trim();
  return text.length > limit ? `${text.slice(0, limit)}...` : text;
}

function _record(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {};
}

function _array(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? value.filter((item): item is Record<string, unknown> => Boolean(item && typeof item === "object" && !Array.isArray(item))) : [];
}

function _link(value: unknown, label: string): string {
  const href = String(value ?? "").trim();
  if (!href) return `<span class="runtime-overview-muted">-</span>`;
  return `<a class="runtime-overview-link" href="${escapeHtml(href)}" target="_blank" rel="noreferrer">${escapeHtml(label)}</a>`;
}

function _target(kind: unknown, id: unknown): string {
  const parts = [kind, id].map((value) => String(value ?? "").trim()).filter(Boolean);
  return parts.length ? parts.join(" / ") : "-";
}

function _renderMediaAssetContentDetail(detail: Record<string, unknown>): string {
  const totals = _record(detail.totals);
  const items = _array(detail.items).slice(0, 20);
  const totalCells = ["assets", "ready", "forbidden", "unavailable", "disabled", "error"]
    .map((key) => `
      <div class="runtime-overview-kpi">
        <span>${escapeHtml(key)}</span>
        <strong>${escapeHtml(String(totals[key] ?? 0))}</strong>
      </div>
    `)
    .join("");
  const rows = items.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.asset_id, 36))}</td>
      <td>${escapeHtml(_short(item.name || item.kind || "-", 32))}</td>
      <td>${_runtimeStatusTag(String(item.content_status || "muted"))}</td>
      <td>${escapeHtml(_short(item.content_reason || "-", 42))}</td>
      <td class="runtime-overview-links">
        ${_link(item.content_endpoint, "content")}
        ${_link(item.content_access_plan_endpoint, "access")}
        ${_link(item.content_recovery_plan_endpoint, "recovery")}
        ${_link(item.content_recovery_preflight_endpoint, "preflight")}
      </td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Media Content Diagnostics</div>
        <div class="detail-subtext">${escapeHtml(String(detail.side_effect || "none"))}</div>
      </div>
      <div class="runtime-overview-kpis">${totalCells}</div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Asset</th>
              <th>Name</th>
              <th>Status</th>
              <th>Reason</th>
              <th>Links</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="5" class="runtime-overview-muted">No media assets sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderMediaAssetContentRecoveryDetail(detail: Record<string, unknown>): string {
  const recovery = _record(detail.media_asset_content_recovery || detail);
  const totals = _record(recovery.totals);
  const endpoints = _record(recovery.endpoints);
  const auditRows = _array(recovery.recent_audits).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.mutation_id, 34))}</td>
      <td>${escapeHtml(_short(`${_target(item.target_kind, item.target_id)} / ${String(item.action || "-")}`, 52))}</td>
      <td>${_runtimeStatusTag(String(item.status || "muted"))}</td>
      <td class="mono">${escapeHtml(_short(item.approval_id || "-", 30))}</td>
      <td>${escapeHtml(_short(item.operator_id || "-", 28))}</td>
      <td>${escapeHtml(_short(item.reason || "-", 44))}</td>
      <td>${escapeHtml(_short(item.created_at || item.timestamp || "-", 32))}</td>
    </tr>
  `).join("");
  const totalCells = ["audits", "planned", "applied", "failed", "rolled_back"]
    .map((key) => `
      <div class="runtime-overview-kpi">
        <span>${escapeHtml(key)}</span>
        <strong>${escapeHtml(String(totals[key] ?? 0))}</strong>
      </div>
    `)
    .join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Media Content Recovery</div>
          <div class="detail-subtext">${escapeHtml(String(recovery.side_effect || "none"))} · ${escapeHtml(_short(recovery.reason || "-", 44))}</div>
        </div>
        <div class="runtime-overview-links">
          ${_link(endpoints.plan, "plan")}
          ${_link(endpoints.preflight, "preflight")}
          ${_link(endpoints.recovery, "recovery")}
        </div>
      </div>
      <div class="runtime-overview-kpis">${totalCells}</div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Mutation</th>
              <th>Target / Action</th>
              <th>Status</th>
              <th>Approval</th>
              <th>Operator</th>
              <th>Reason</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>${auditRows || `<tr><td colspan="7" class="runtime-overview-muted">No media content recovery audits sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderMediaAssetRetentionDetail(detail: Record<string, unknown>): string {
  const diagnostics = _record(detail.media_asset_retention_diagnostics || detail);
  const totals = _record(diagnostics.totals);
  const items = _array(diagnostics.items).slice(0, 20);
  const notes = _array(diagnostics.notes);
  const rows = items.map((item) => {
    const channel = _record(item.channel);
    const target = _target(channel.kind, channel.conversation_id || channel.account_id);
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.asset_id, 36))}</td>
        <td>${escapeHtml(_short(item.name || item.kind || "-", 28))}</td>
        <td>${escapeHtml(_short(String(item.retention_class || item.retention || "-"), 16))}</td>
        <td>${_runtimeStatusTag(item.cleanup_due ? "warn" : "ok")}</td>
        <td>${escapeHtml(String(item.age_seconds ?? 0))}</td>
        <td>${escapeHtml(_short(item.cleanup_after || "-", 28))}</td>
        <td>${escapeHtml(_short(item.cleanup_reason || "-", 42))}</td>
        <td>${escapeHtml(_short(target || "-", 32))}</td>
      </tr>
    `;
  }).join("");
  const noteRows = notes.slice(0, 10).map((item) => `<tr><td>${escapeHtml(_short(String(item), 160))}</td></tr>`).join("");
  const totalCells = ["assets", "cleanup_due", "default", "ephemeral", "permanent", "unknown"]
    .map((key) => `
      <div class="runtime-overview-kpi">
        <span>${escapeHtml(key)}</span>
        <strong>${escapeHtml(String(totals[key] ?? 0))}</strong>
      </div>
    `)
    .join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Media Asset Retention</div>
          <div class="detail-subtext">${escapeHtml(String(diagnostics.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">${totalCells}</div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Asset</th>
              <th>Name</th>
              <th>Retention Class</th>
              <th>Cleanup Due</th>
              <th>Age (s)</th>
              <th>Cleanup After</th>
              <th>Reason</th>
              <th>Target</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No media asset retention assets sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr><th>Notes</th></tr>
          </thead>
          <tbody>${noteRows || `<tr><td class="runtime-overview-muted">No media asset retention notes</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderMediaAssetRetentionSteps(
  title: string,
  steps: Record<string, unknown>[],
  emptyLabel: string,
): string {
  const rows = steps.slice(0, 20).map((item) => {
    const metadata = _record(item.metadata);
    const metadataText = Object.keys(metadata).length
      ? Object.entries(metadata).map(([key, value]) => `${key}=${String(value ?? "")}`).join("\n")
      : "-";
    return `
      <tr>
        <td>${escapeHtml(_short(item.name || "-", 32))}</td>
        <td class="mono">${escapeHtml(_short(item.method || "-", 10))}</td>
        <td class="mono">${escapeHtml(_short(item.endpoint || "-", 40))}</td>
        <td>${escapeHtml(_short(item.description || "-", 80))}</td>
        <td><pre class="runtime-overview-json runtime-overview-json-inline">${escapeHtml(metadataText)}</pre></td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">${escapeHtml(title)}</div>
        <div class="detail-subtext">${escapeHtml(String(steps.length))} steps</div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Method</th>
              <th>Endpoint</th>
              <th>Description</th>
              <th>Metadata</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="5" class="runtime-overview-muted">${escapeHtml(emptyLabel)}</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderMediaAssetRetentionPlanDetail(detail: Record<string, unknown>): string {
  const plan = _record(detail.media_asset_retention_plan || detail);
  const requiredSteps = _array(plan.required_steps);
  const verifySteps = _array(plan.verify_steps);
  const rollbackSteps = _array(plan.rollback_steps);
  const diagnostics = _record(plan.diagnostics);
  const blockers = Array.isArray(plan.blockers) ? plan.blockers : [];
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Media Asset Retention Plan</div>
          <div class="detail-subtext">${escapeHtml(_short(plan.reason || "-", 52))} · ${escapeHtml(String(plan.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>Ready</span><strong>${escapeHtml(String(Boolean(plan.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>Assets</span><strong>${escapeHtml(String(plan.asset_count ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Candidates</span><strong>${escapeHtml(String(plan.candidate_count ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Required Steps</span><strong>${escapeHtml(String(requiredSteps.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>Verify Steps</span><strong>${escapeHtml(String(verifySteps.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>Rollback Steps</span><strong>${escapeHtml(String(rollbackSteps.length))}</strong></div>
      </div>
      <div class="runtime-overview-detail-grid">
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Blockers</div>
          <div class="detail-subtext">${escapeHtml(_short(blockers.join(", ") || "-", 220))}</div>
        </div>
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Diagnostics Assets</div>
          <div class="detail-subtext">${escapeHtml(String(_record(diagnostics.totals).assets ?? plan.asset_count ?? 0))}</div>
        </div>
      </div>
    </div>
    ${_renderMediaAssetRetentionSteps("Required Steps", requiredSteps, "No required steps sampled")}
    ${_renderMediaAssetRetentionSteps("Verification Steps", verifySteps, "No verification steps sampled")}
    ${_renderMediaAssetRetentionSteps("Rollback Steps", rollbackSteps, "No rollback steps sampled")}
  `;
}

function _renderMediaAssetRetentionCleanupDetail(detail: Record<string, unknown>): string {
  const cleanup = _record(detail.media_asset_retention_cleanup || detail);
  const endpoints = _record(cleanup.endpoints);
  const totals = _record(cleanup.totals);
  const blockers = Array.isArray(cleanup.blockers) ? cleanup.blockers : [];
  const notes = Array.isArray(cleanup.notes) ? cleanup.notes : [];
  const noteRows = notes.slice(0, 10).map((item) => `<tr><td>${escapeHtml(_short(item, 140))}</td></tr>`).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Media Asset Retention Cleanup</div>
          <div class="detail-subtext">${escapeHtml(_short(cleanup.reason || "-", 52))} · ${escapeHtml(String(cleanup.side_effect || "none"))}</div>
        </div>
        <div class="runtime-overview-links">
          ${_link(endpoints.plan, "plan")}
          ${_link(endpoints.preflight, "preflight")}
          ${_link(endpoints.cleanup, "cleanup")}
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>Ready</span><strong>${escapeHtml(String(Boolean(cleanup.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>Assets</span><strong>${escapeHtml(String(cleanup.asset_count ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Candidates</span><strong>${escapeHtml(String(cleanup.candidate_count ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Planned</span><strong>${escapeHtml(String(totals.planned ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Applied</span><strong>${escapeHtml(String(totals.applied ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Failed</span><strong>${escapeHtml(String(totals.failed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Audits</span><strong>${escapeHtml(String(totals.audits ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Rolled Back</span><strong>${escapeHtml(String(totals.rolled_back ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-detail-grid">
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Blockers</div>
          <div class="detail-subtext">${escapeHtml(_short(blockers.join(", ") || "-", 220))}</div>
        </div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr><th>Notes</th></tr>
          </thead>
          <tbody>${noteRows || `<tr><td class="runtime-overview-muted">No media asset retention cleanup notes</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderDeadLettersDetail(detail: Record<string, unknown>): string {
  const agentMetrics = _record(detail.agent_job_metrics);
  const outboxMetrics = _record(detail.outbox_metrics);
  const agentDeadLetters = _record(agentMetrics.dead_letters);
  const outboxDeadLetters = _record(outboxMetrics.dead_letters);
  const agentByType = _record(agentDeadLetters.by_type);
  const outboxByChannelKind = _record(outboxDeadLetters.by_channel_kind);
  const agentRows = _array(agentDeadLetters.recent).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_id || "-", 32))}</td>
      <td>${escapeHtml(String(item.job_type ?? "-"))}</td>
      <td>${escapeHtml(String(item.event_type ?? "-"))}</td>
      <td>${escapeHtml(String(item.status ?? "-"))}</td>
      <td>${escapeHtml(String(item.attempt ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
      <td>${escapeHtml(_short(item.occurred_at || item.updated_at || "-", 24))}</td>
    </tr>
  `).join("");
  const outboxRows = _array(outboxDeadLetters.recent).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.delivery_id || "-", 32))}</td>
      <td>${escapeHtml(String(item.channel_kind ?? "-"))}</td>
      <td>${escapeHtml(String(item.error_kind ?? "-"))}</td>
      <td>${escapeHtml(_short(item.error_message || "-", 40))}</td>
      <td>${escapeHtml(String(item.attempt ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
      <td>${escapeHtml(_short(item.occurred_at || "-", 24))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Dead Letters</div>
          <div class="detail-subtext">Combined read-only view of AgentJob and outbox dead letters</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>agent job dead letters</span><strong>${escapeHtml(String(agentDeadLetters.current_total ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>outbox dead letters</span><strong>${escapeHtml(String(outboxDeadLetters.current_total ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>agent job types</span><strong>${escapeHtml(String(Object.keys(agentByType).length))}</strong></div>
        <div class="runtime-overview-kpi"><span>outbox channel kinds</span><strong>${escapeHtml(String(Object.keys(outboxByChannelKind).length))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job ID</th>
              <th>Job Type</th>
              <th>Event</th>
              <th>Status</th>
              <th>Attempt</th>
              <th>Occurred</th>
            </tr>
          </thead>
          <tbody>${agentRows || `<tr><td colspan="6" class="runtime-overview-muted">No agent-job dead letters sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Delivery ID</th>
              <th>Channel</th>
              <th>Error Kind</th>
              <th>Error</th>
              <th>Attempt</th>
              <th>Occurred</th>
            </tr>
          </thead>
          <tbody>${outboxRows || `<tr><td colspan="6" class="runtime-overview-muted">No outbox dead letters sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">Agent Job Dead Letter Totals</div>
        <pre class="runtime-overview-json">${escapeHtml(JSON.stringify(agentByType, null, 2))}</pre>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">Outbox Dead Letter Totals</div>
        <pre class="runtime-overview-json">${escapeHtml(JSON.stringify(outboxByChannelKind, null, 2))}</pre>
      </div>
    </div>
  `;
}

function _renderCheckpointLagDetail(detail: Record<string, unknown>): string {
  const diagnostics = _record(detail.diagnostics);
  const provided = _array((detail as Record<string, unknown>).checkpoints || (detail as Record<string, unknown>).checkpoint_lag);
  const derived = _array(diagnostics.workers).flatMap((worker) => {
    const record = _record(worker);
    return _array(record.checkpoints).map((item) => ({
      ..._record(item),
      job_type: _record(item).job_type || record.job_type,
      checkpoint_prefix: _record(item).checkpoint_prefix || record.checkpoint_prefix,
    }));
  });
  const checkpoints = (provided.length ? provided : derived).slice(0, 20);
  const rows = checkpoints.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.session_key || item.dataset_id || "-", 28))}</td>
      <td>${escapeHtml(String(item.job_type ?? "-"))}</td>
      <td>${escapeHtml(String(item.dataset_id ?? "-"))}</td>
      <td>${escapeHtml(String(item.latest_source_seq ?? "-"))}</td>
      <td>${escapeHtml(String(item.checkpoint_seq ?? "-"))}</td>
      <td>${escapeHtml(String(item.checkpoint_lag_messages ?? "-"))}</td>
      <td>${escapeHtml(String(item.freshness_status ?? "-"))}</td>
      <td>${escapeHtml(_short(item.freshness_reason || "-", 42))}</td>
    </tr>
  `).join("");
  const lags = checkpoints
    .map((item) => Number(item.checkpoint_lag_messages ?? 0))
    .filter((value) => Number.isFinite(value));
  const maxLag = lags.length ? Math.max(...lags) : 0;
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Checkpoint Lag</div>
          <div class="detail-subtext">${escapeHtml(String(diagnostics.generated_at || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>checkpoints</span><strong>${escapeHtml(String(checkpoints.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>max lag</span><strong>${escapeHtml(String(maxLag))}</strong></div>
        <div class="runtime-overview-kpi"><span>workers</span><strong>${escapeHtml(String(_array(diagnostics.workers).length))}</strong></div>
        <div class="runtime-overview-kpi"><span>stale after</span><strong>${escapeHtml(String(diagnostics.stale_after_seconds ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Session</th>
              <th>Job Type</th>
              <th>Dataset</th>
              <th>Latest Seq</th>
              <th>Checkpoint Seq</th>
              <th>Lag Messages</th>
              <th>Freshness</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No lagged checkpoints sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderRuntimeHealthDetail(detail: Record<string, unknown>): string {
  const health = _record(detail.health || detail);
  const errors = Array.isArray(detail.errors) ? detail.errors : [];
  const healthRows = Object.entries(health)
    .filter(([key]) => key !== "errors")
    .map(([key, value]) => `
      <tr>
        <td>${escapeHtml(key)}</td>
        <td>${escapeHtml(_short(_formatJson(value), 120))}</td>
      </tr>
    `)
    .join("");
  const errorRows = errors.slice(0, 20).map((value) => `
    <tr>
      <td>${escapeHtml(_short(value, 200))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Runtime Health</div>
          <div class="detail-subtext">${escapeHtml(String(health.status || "ok"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>Status</span><strong>${escapeHtml(String(health.status || "ok"))}</strong></div>
        <div class="runtime-overview-kpi"><span>Errors</span><strong>${escapeHtml(String(errors.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>Health Fields</span><strong>${escapeHtml(String(Object.keys(health).length))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Field</th>
              <th>Value</th>
            </tr>
          </thead>
          <tbody>${healthRows || `<tr><td colspan="2" class="runtime-overview-muted">No runtime health snapshot fields</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr><th>Health Errors</th></tr>
          </thead>
          <tbody>${errorRows || `<tr><td class="runtime-overview-muted">No runtime health errors</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderStaleJobsDetail(detail: Record<string, unknown>): string {
  const diagnostics = _record(detail.diagnostics || detail);
  const totals = _record(diagnostics.totals);
  const staleItems = _array(detail.items).slice(0, 20);
  const workers = _array(diagnostics.workers).slice(0, 20);
  const workerRows = workers.map((item) => {
    const recentJobs = _array(item.recent_jobs);
    const latestJob = _record(item.latest_job || recentJobs[0]);
    const route = _record(latestJob.route);
    const latestTarget = _target(route.kind, route.conversation_id || route.account_id);
    return `
      <tr>
        <td>${escapeHtml(String(item.job_type || "-"))}</td>
        <td class="mono">${escapeHtml(_short(item.checkpoint_prefix || "-", 18))}</td>
        <td><pre class="runtime-overview-json runtime-overview-json-inline">${escapeHtml(_formatJson(item.status_counts || {}))}</pre></td>
        <td>${escapeHtml(String(item.stale_lease_count ?? 0))}</td>
        <td>${escapeHtml(String(item.leaseable_count ?? 0))}</td>
        <td class="mono">${escapeHtml(_short(latestJob.job_id || latestTarget || "-", 48))}</td>
        <td>${escapeHtml(_short(latestJob.updated_at || latestJob.created_at || "-", 24))}</td>
      </tr>
    `;
  }).join("");
  const staleRows = staleItems.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_id || "-", 42))}</td>
      <td>${escapeHtml(String(item.job_type || "-"))}</td>
      <td>${_runtimeStatusTag(String(item.status || "muted"))}</td>
      <td class="mono">${escapeHtml(_short(item.lease_owner || "-", 24))}</td>
      <td>${escapeHtml(String(item.attempts ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
      <td>${escapeHtml(_short(item.updated_at || item.created_at || "-", 24))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Stale Jobs</div>
          <div class="detail-subtext">${escapeHtml(_short(String(diagnostics.generated_at || "-"), 40))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>Jobs</span><strong>${escapeHtml(String(totals.jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Checkpoints</span><strong>${escapeHtml(String(totals.checkpoints ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Leaseable Jobs</span><strong>${escapeHtml(String(totals.leaseable_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Stale Leases</span><strong>${escapeHtml(String(totals.stale_leases ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Workers</span><strong>${escapeHtml(String(workers.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>Stale After</span><strong>${escapeHtml(String(diagnostics.stale_after_seconds ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Checkpoint Prefix</th>
              <th>Status Counts</th>
              <th>Stale Leases</th>
              <th>Leaseable</th>
              <th>Latest Job</th>
              <th>Latest Updated</th>
            </tr>
          </thead>
          <tbody>${workerRows || `<tr><td colspan="7" class="runtime-overview-muted">No worker stale-job diagnostics sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job</th>
              <th>Job Type</th>
              <th>Status</th>
              <th>Lease Owner</th>
              <th>Attempts</th>
              <th>Updated</th>
            </tr>
          </thead>
          <tbody>${staleRows || `<tr><td colspan="6" class="runtime-overview-muted">No stale jobs sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderControlAuditDetail(detail: Record<string, unknown>): string {
  const approvals = _record(detail.operator_approvals);
  const mutations = _record(detail.control_mutations);
  const approvalTotals = _record(approvals.totals);
  const mutationTotals = _record(mutations.totals);
  const approvalRows = _array(approvals.approvals).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.approval_id, 34))}</td>
      <td>${escapeHtml(_short(_target(item.target_kind, item.target_id), 42))}</td>
      <td>${_runtimeStatusTag(String(item.decision || "muted"))}</td>
      <td>${_runtimeStatusTag(item.active === false ? "muted" : "ok")}</td>
      <td>${escapeHtml(_short(item.operator_id || "-", 28))}</td>
      <td>${escapeHtml(_short(item.created_at || "-", 32))}</td>
    </tr>
  `).join("");
  const mutationRows = _array(mutations.mutations).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.mutation_id, 34))}</td>
      <td>${escapeHtml(_short(`${_target(item.target_kind, item.target_id)} / ${String(item.action || "-")}`, 52))}</td>
      <td>${_runtimeStatusTag(String(item.status || "muted"))}</td>
      <td class="mono">${escapeHtml(_short(item.approval_id || "-", 30))}</td>
      <td>${escapeHtml(_short(item.operator_id || "-", 28))}</td>
      <td>${escapeHtml(_short(item.rollback_ref || item.reason || "-", 44))}</td>
      <td>${escapeHtml(_short(item.created_at || "-", 32))}</td>
    </tr>
  `).join("");
  const approvalKpis = ["approvals", "active", "approved", "rejected", "revoked"]
    .map((key) => `
      <div class="runtime-overview-kpi">
        <span>${escapeHtml(key)}</span>
        <strong>${escapeHtml(String(approvalTotals[key] ?? 0))}</strong>
      </div>
    `)
    .join("");
  const mutationKpis = ["mutations", "planned", "applied", "failed", "rolled_back"]
    .map((key) => `
      <div class="runtime-overview-kpi">
        <span>${escapeHtml(key)}</span>
        <strong>${escapeHtml(String(mutationTotals[key] ?? 0))}</strong>
      </div>
    `)
    .join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Operator Approvals</div>
        <div class="detail-subtext">${escapeHtml(String(approvals.side_effect || "runtime_state_only"))}</div>
      </div>
      <div class="runtime-overview-kpis">${approvalKpis}</div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Approval</th>
              <th>Target</th>
              <th>Decision</th>
              <th>Active</th>
              <th>Operator</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>${approvalRows || `<tr><td colspan="6" class="runtime-overview-muted">No operator approvals sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Control Mutations</div>
        <div class="detail-subtext">${escapeHtml(String(mutations.side_effect || "runtime_state_only"))}</div>
      </div>
      <div class="runtime-overview-kpis">${mutationKpis}</div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Mutation</th>
              <th>Target / Action</th>
              <th>Status</th>
              <th>Approval</th>
              <th>Operator</th>
              <th>Reason / Rollback</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>${mutationRows || `<tr><td colspan="7" class="runtime-overview-muted">No control mutations sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderControlMutationPolicyDetail(detail: Record<string, unknown>): string {
  const policy = _record(detail.control_mutation_policy || detail);
  const intents = _array(policy.intents);
  const actionCount = intents.reduce((total, item) => {
    const actions = Array.isArray(item.actions) ? item.actions.length : 0;
    return total + actions;
  }, 0);
  const rows = intents.slice(0, 30).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.target_kind, 34))}</td>
      <td>${escapeHtml(Array.isArray(item.actions) ? item.actions.join(", ") : "-")}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Control Mutation Policy</div>
        <div class="detail-subtext">${escapeHtml(String(policy.side_effect || "none"))}</div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>allowed</span><strong>${escapeHtml(String(Boolean(policy.allowed)))}</strong></div>
        <div class="runtime-overview-kpi"><span>reason</span><strong>${escapeHtml(_short(policy.reason || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>targets</span><strong>${escapeHtml(String(intents.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>actions</span><strong>${escapeHtml(String(actionCount))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Target Kind</th>
              <th>Allowed Actions</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="2" class="runtime-overview-muted">No control mutation policy intents sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderJobEventsDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.agent_job_metrics || detail);
  const throughput = _record(metrics.throughput);
  const eventTotals = _record(throughput.events_by_type);
  const recentEvents = _array(detail.recent_events).slice(0, 20);
  const totalRows = Object.entries(eventTotals).slice(0, 20).map(([eventType, count]) => `
    <tr>
      <td>${escapeHtml(_short(eventType, 24))}</td>
      <td>${escapeHtml(String(count ?? 0))}</td>
    </tr>
  `).join("");
  const recentRows = recentEvents.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.event_id || "-", 34))}</td>
      <td class="mono">${escapeHtml(_short(item.job_id || "-", 34))}</td>
      <td>${escapeHtml(String(item.job_type ?? "-"))}</td>
      <td>${escapeHtml(String(item.event_type ?? "-"))}</td>
      <td>${escapeHtml(String(item.status ?? "-"))}</td>
      <td>${escapeHtml(String(item.attempt ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
      <td class="mono">${escapeHtml(_short(item.lease_owner || "-", 24))}</td>
      <td>${escapeHtml(_short(item.occurred_at || item.updated_at || "-", 24))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Job Events</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_events ?? recentEvents.length ?? 0} events sampled`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>sampled events</span><strong>${escapeHtml(String(metrics.sampled_events ?? recentEvents.length ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>created</span><strong>${escapeHtml(String(throughput.created ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>leased</span><strong>${escapeHtml(String(throughput.leased ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>running</span><strong>${escapeHtml(String(throughput.running ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>succeeded</span><strong>${escapeHtml(String(throughput.succeeded ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>failed</span><strong>${escapeHtml(String(throughput.failed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>terminal events</span><strong>${escapeHtml(String(throughput.terminal_events ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Event Type</th>
              <th>Count</th>
            </tr>
          </thead>
          <tbody>${totalRows || `<tr><td colspan="2" class="runtime-overview-muted">No job event totals sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Event ID</th>
              <th>Job</th>
              <th>Job Type</th>
              <th>Event</th>
              <th>Status</th>
              <th>Attempt</th>
              <th>Lease Owner</th>
              <th>Occurred At</th>
            </tr>
          </thead>
          <tbody>${recentRows || `<tr><td colspan="8" class="runtime-overview-muted">No recent job events sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderOutboxEventsDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.outbox_metrics || detail);
  const throughput = _record(metrics.throughput);
  const eventTotals = _record(throughput.events_by_type);
  const deadLetters = _record(metrics.dead_letters);
  const recentEvents = _array(detail.recent_events);
  const fallbackRecent = _array(deadLetters.recent);
  const rowsSource = (recentEvents.length ? recentEvents : fallbackRecent).slice(0, 20);
  const totalRows = Object.entries(eventTotals).slice(0, 20).map(([eventType, count]) => `
    <tr>
      <td>${escapeHtml(_short(eventType, 24))}</td>
      <td>${escapeHtml(String(count ?? 0))}</td>
    </tr>
  `).join("");
  const recentRows = rowsSource.map((item) => {
    const channel = _record(item.channel);
    const channelLabel = [item.channel_kind || channel.kind, channel.account_id, channel.conversation_type]
      .filter(Boolean)
      .join(" / ");
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.delivery_id || item.event_id || "-", 34))}</td>
        <td>${escapeHtml(_short(channelLabel || "-", 28))}</td>
        <td>${escapeHtml(String(item.event_type ?? "-"))}</td>
        <td>${escapeHtml(String(item.status ?? "-"))}</td>
        <td>${escapeHtml(String(item.attempt ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
        <td class="mono">${escapeHtml(_short(item.lease_owner || "-", 24))}</td>
        <td>${escapeHtml(_short(item.error_message || "-", 36))}</td>
        <td>${escapeHtml(_short(item.occurred_at || item.updated_at || "-", 24))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Outbox Events</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_events ?? rowsSource.length ?? 0} events sampled`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>sampled events</span><strong>${escapeHtml(String(metrics.sampled_events ?? rowsSource.length ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>queued</span><strong>${escapeHtml(String(throughput.queued ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>leased</span><strong>${escapeHtml(String(throughput.leased ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dispatching</span><strong>${escapeHtml(String(throughput.dispatching ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>succeeded</span><strong>${escapeHtml(String(throughput.succeeded ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>failed</span><strong>${escapeHtml(String(throughput.failed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dead lettered</span><strong>${escapeHtml(String(throughput.dead_lettered ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>terminal events</span><strong>${escapeHtml(String(throughput.terminal_events ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Event Type</th>
              <th>Count</th>
            </tr>
          </thead>
          <tbody>${totalRows || `<tr><td colspan="2" class="runtime-overview-muted">No outbox event totals sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Delivery / Event</th>
              <th>Channel</th>
              <th>Event</th>
              <th>Status</th>
              <th>Attempt</th>
              <th>Lease Owner</th>
              <th>Error</th>
              <th>Occurred At</th>
            </tr>
          </thead>
          <tbody>${recentRows || `<tr><td colspan="8" class="runtime-overview-muted">No recent outbox events sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderRagEvalFailuresDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.agent_job_metrics || detail);
  const deadLetters = _record(metrics.dead_letters);
  const byType = _record(deadLetters.by_type);
  const throughput = _record(metrics.throughput);
  const directItems = _array(detail.items);
  const fallbackItems = _array(deadLetters.recent).filter(
    (item) => String(item.job_type || "") === "rag_eval",
  );
  const rowsSource = (directItems.length ? directItems : fallbackItems).slice(0, 20);
  const notes = Array.isArray(detail.notes)
    ? detail.notes
    : Array.isArray(metrics.notes)
      ? metrics.notes
      : [];
  const rows = rowsSource.map((item) => {
    const result = _record(item.result);
    const passed = typeof item.passed === "boolean"
      ? item.passed
      : typeof result.passed === "boolean"
        ? result.passed
        : null;
    const lifecycleStatus = String(item.lifecycle_status || item.status || "-");
    const qualityStatus = String(
      item.quality_status || (passed === false ? "failed_quality" : "-"),
    );
    const errorLabel = String(
      item.error_message ||
      item.coverage_reason ||
      qualityStatus ||
      "-",
    );
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.job_id || "-", 36))}</td>
        <td>${escapeHtml(lifecycleStatus)}</td>
        <td>${escapeHtml(qualityStatus)}</td>
        <td>${escapeHtml(passed === null ? "-" : String(passed))}</td>
        <td>${escapeHtml(String(item.attempts ?? item.attempt ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
        <td class="mono">${escapeHtml(_short(item.lease_owner || "-", 24))}</td>
        <td>${escapeHtml(_short(errorLabel, 36))}</td>
        <td>${escapeHtml(_short(item.updated_at || item.occurred_at || item.created_at || "-", 24))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">RAG Eval Failures</div>
          <div class="detail-subtext">${escapeHtml(_short(`${rowsSource.length} failure rows sampled`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>sampled jobs</span><strong>${escapeHtml(String(metrics.sampled_jobs ?? rowsSource.length ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>rag eval dead letters</span><strong>${escapeHtml(String(byType.rag_eval ?? rowsSource.length ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>total dead letters</span><strong>${escapeHtml(String(deadLetters.current_total ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>failed events</span><strong>${escapeHtml(String(throughput.failed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>terminal events</span><strong>${escapeHtml(String(throughput.terminal_events ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job</th>
              <th>Lifecycle</th>
              <th>Quality</th>
              <th>Passed</th>
              <th>Attempt</th>
              <th>Lease Owner</th>
              <th>Error</th>
              <th>Updated At</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No rag-eval failures sampled</td></tr>`}</tbody>
        </table>
      </div>
      ${_renderRuntimeOverviewNotesSection("Notes", notes, "No rag-eval failure notes")}
    </div>
  `;
}

function _renderRuntimeConfigDetail(detail: Record<string, unknown>): string {
  const config = _record(detail.runtime_config || detail);
  const runtime = _record(config.runtime);
  const delivery = _record(config.delivery);
  const workers = _record(config.workers);
  const environment = _array(config.environment).slice(0, 30);
  const onebotEndpoints = _array(delivery.onebot_endpoints).slice(0, 20);
  const expectedChannels = Array.isArray(delivery.onebot_expected_channels) ? delivery.onebot_expected_channels : [];
  const missingChannels = Array.isArray(delivery.onebot_missing_channels) ? delivery.onebot_missing_channels : [];
  const telegramChannels = Array.isArray(delivery.telegram_channels) ? delivery.telegram_channels : [];
  const botIds = Array.isArray(runtime.bot_ids) ? runtime.bot_ids : [];
  const endpointRows = onebotEndpoints.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.channel || "-", 24))}</td>
      <td>${escapeHtml(_short(item.transport || "-", 18))}</td>
      <td>${escapeHtml(String(Boolean(item.websocket_configured)))}</td>
      <td>${escapeHtml(String(Boolean(item.http_configured)))}</td>
      <td>${escapeHtml(String(Boolean(item.access_token_configured)))}</td>
      <td class="mono">${escapeHtml(_short(item.endpoint || "-", 40))}</td>
    </tr>
  `).join("");
  const envRows = environment.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.key || "-", 36))}</td>
      <td>${escapeHtml(String(Boolean(item.present)))}</td>
      <td>${escapeHtml(String(Boolean(item.secret)))}</td>
      <td class="mono">${escapeHtml(_short(item.value_redacted || "-", 52))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Runtime Config</div>
          <div class="detail-subtext">${escapeHtml(_short(runtime.address || "-", 48))} · ${escapeHtml(_short(runtime.address_source || "-", 28))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>QQ Group Send</span><strong>${escapeHtml(String(Boolean(delivery.qq_group_send_enabled)))}</strong></div>
        <div class="runtime-overview-kpi"><span>Telegram Token</span><strong>${escapeHtml(String(Boolean(delivery.telegram_token_configured)))}</strong></div>
        <div class="runtime-overview-kpi"><span>OneBot Endpoints</span><strong>${escapeHtml(String(onebotEndpoints.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>Expected Channels</span><strong>${escapeHtml(String(expectedChannels.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>Strict Lease Token</span><strong>${escapeHtml(String(Boolean(workers.agent_job_strict_lease_token)))}</strong></div>
        <div class="runtime-overview-kpi"><span>AgentJob External Lease</span><strong>${escapeHtml(String(Boolean(workers.queue_external_lease_agent_job_enabled)))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Runtime Address</th>
              <th>Bot IDs</th>
              <th>Telegram Channels</th>
              <th>Expected Channels</th>
              <th>Missing Channels</th>
              <th>Allowed Kinds</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td class="mono">${escapeHtml(_short(runtime.address || "-", 36))}</td>
              <td>${escapeHtml(botIds.length ? botIds.join(", ") : "-")}</td>
              <td>${escapeHtml(telegramChannels.length ? telegramChannels.join(", ") : "-")}</td>
              <td>${escapeHtml(expectedChannels.length ? expectedChannels.join(", ") : "-")}</td>
              <td>${escapeHtml(missingChannels.length ? missingChannels.join(", ") : "-")}</td>
              <td>${escapeHtml(Array.isArray(workers.outbox_delivery_allowed_kinds) ? workers.outbox_delivery_allowed_kinds.join(", ") : "-")}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">OneBot Endpoints</div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Channel</th>
              <th>Transport</th>
              <th>WebSocket</th>
              <th>HTTP</th>
              <th>Access Token</th>
              <th>Endpoint</th>
            </tr>
          </thead>
          <tbody>${endpointRows || `<tr><td colspan="6" class="runtime-overview-muted">No OneBot endpoints sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Worker Flags</div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Outbox Worker</th>
              <th>Knowledge Planner</th>
              <th>Agent Job Recovery</th>
              <th>By Account</th>
              <th>By Conversation Type</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>${escapeHtml(String(Boolean(workers.outbox_delivery_worker_enabled)))}</td>
              <td>${escapeHtml(String(Boolean(workers.knowledge_job_planner_enabled)))}</td>
              <td>${escapeHtml(String(Boolean(workers.agent_job_recovery_enabled)))}</td>
              <td><pre class="runtime-overview-json runtime-overview-json-inline">${escapeHtml(_formatJson(workers.outbox_delivery_allowed_kinds_by_account || {}))}</pre></td>
              <td><pre class="runtime-overview-json runtime-overview-json-inline">${escapeHtml(_formatJson(workers.outbox_delivery_allowed_kinds_by_account_conversation_type || {}))}</pre></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Environment Keys</div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Key</th>
              <th>Present</th>
              <th>Secret</th>
              <th>Value</th>
            </tr>
          </thead>
          <tbody>${envRows || `<tr><td colspan="4" class="runtime-overview-muted">No environment keys sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderWorkerLeasesDetail(detail: Record<string, unknown>): string {
  const diagnostics = _record(detail.diagnostics || detail.worker_leases || detail);
  const totals = _record(diagnostics.totals);
  const workers = _array(diagnostics.workers).slice(0, 20);
  const rows = workers.map((item) => {
    const statusCounts = _record(item.status_counts);
    const latestJob = _record(item.latest_job);
    const latestJobId = latestJob.job_id || (_array(item.recent_jobs)[0] || {}).job_id || "-";
    const latestUpdatedAt = latestJob.updated_at || (_array(item.recent_jobs)[0] || {}).updated_at || "-";
    const statusSummary = Object.entries(statusCounts).map(([status, count]) => `${status}:${count}`).join(", ") || "-";
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.job_type || "-", 28))}</td>
        <td>${escapeHtml(String(item.checkpoint_prefix ?? "-"))}</td>
        <td>${escapeHtml(statusSummary)}</td>
        <td>${escapeHtml(String(item.stale_lease_count ?? 0))}</td>
        <td>${escapeHtml(String(item.leaseable_count ?? 0))}</td>
        <td>${escapeHtml(String(_array(item.recent_jobs).length))}</td>
        <td class="mono">${escapeHtml(_short(latestJobId, 36))}</td>
        <td>${escapeHtml(_short(latestUpdatedAt, 24))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Worker Leases</div>
          <div class="detail-subtext">${escapeHtml(_short(String(diagnostics.generated_at || "-"), 40))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>jobs</span><strong>${escapeHtml(String(totals.jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>checkpoints</span><strong>${escapeHtml(String(totals.checkpoints ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>leaseable jobs</span><strong>${escapeHtml(String(totals.leaseable_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>stale leases</span><strong>${escapeHtml(String(totals.stale_leases ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>workers</span><strong>${escapeHtml(String(workers.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>stale after(s)</span><strong>${escapeHtml(String(diagnostics.stale_after_seconds ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Checkpoint Prefix</th>
              <th>Status Counts</th>
              <th>Stale Leases</th>
              <th>Leaseable</th>
              <th>Recent Jobs</th>
              <th>Latest Job</th>
              <th>Latest Updated</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No worker lease diagnostics sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderQueueTopologyDetail(detail: Record<string, unknown>): string {
  const topology = _record(detail.queue_topology || detail);
  const workKinds = _array(topology.work_kinds);
  const nodes = _array(topology.nodes);
  const edges = _array(topology.edges);
  const rows = workKinds.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.work_kind, 28))}</td>
      <td>${escapeHtml(_short(item.queue_source || "-", 34))}</td>
      <td>${escapeHtml(_short(item.execution_owner || "-", 34))}</td>
      <td>${escapeHtml(_short(item.ack_owner || "-", 34))}</td>
      <td>${_runtimeStatusTag(item.allowed === false ? "blocked" : "ok")}</td>
      <td>${escapeHtml(_short(Array.isArray(item.blockers) ? item.blockers.join(", ") : "-", 52))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Queue Topology</div>
        <div class="detail-subtext">${escapeHtml(String(topology.side_effect || "none"))}</div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>provider</span><strong>${escapeHtml(String(topology.provider || "-"))}</strong></div>
        <div class="runtime-overview-kpi"><span>mode</span><strong>${escapeHtml(String(topology.mode || "-"))}</strong></div>
        <div class="runtime-overview-kpi"><span>phase</span><strong>${escapeHtml(_short(topology.migration_phase || "-", 24))}</strong></div>
        <div class="runtime-overview-kpi"><span>external lease</span><strong>${escapeHtml(String(Boolean(topology.external_lease_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>nodes</span><strong>${escapeHtml(String(nodes.length))}</strong></div>
        <div class="runtime-overview-kpi"><span>edges</span><strong>${escapeHtml(String(edges.length))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Work Kind</th>
              <th>Queue Source</th>
              <th>Execution Owner</th>
              <th>Ack Owner</th>
              <th>Allowed</th>
              <th>Blockers</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="6" class="runtime-overview-muted">No queue topology work kinds sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderQqCutoverRouteTable(
  title: string,
  routes: Record<string, unknown>[],
  emptyLabel: string,
): string {
  const rows = routes.slice(0, 20).map((item) => `
    <tr>
      <td>${escapeHtml(String(item.account_id || "-"))}</td>
      <td>${escapeHtml(String(item.conversation_type || "-"))}</td>
      <td>${escapeHtml(String(item.conversation_id || "-"))}</td>
      <td>${escapeHtml(String(item.kind || "-"))}</td>
      <td>${escapeHtml(String(item.source || "-"))}</td>
      <td>${escapeHtml(_short(item.reason || "-", 48))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">${escapeHtml(title)}</div>
        <div class="detail-subtext">${escapeHtml(String(routes.length))} routes</div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Account</th>
              <th>Conversation</th>
              <th>ID</th>
              <th>Kind</th>
              <th>Source</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="6" class="runtime-overview-muted">${escapeHtml(emptyLabel)}</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderQqCutoverRouteMatrixDetail(detail: Record<string, unknown>): string {
  const matrix = _record(detail.qq_cutover_route_matrix || detail);
  const totals = _record(matrix.totals);
  const qqAccounts = _record(matrix.qq_accounts);
  const configuredBotIds = Array.isArray(qqAccounts.configured_bot_ids)
    ? qqAccounts.configured_bot_ids.map((item) => String(item ?? ""))
    : [];
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">QQ Cutover Route Matrix</div>
          <div class="detail-subtext">${escapeHtml(String(matrix.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>go scope</span><strong>${escapeHtml(String(totals.go_execution_owner_scope ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>currently sendable</span><strong>${escapeHtml(String(totals.currently_sendable_routes ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>policy blocked</span><strong>${escapeHtml(String(totals.policy_blocked_routes ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>platform blockers</span><strong>${escapeHtml(String(totals.platform_blocker_routes ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>group send enabled</span><strong>${escapeHtml(String(Boolean(matrix.qq_group_send_enabled)))}</strong></div>
        <div class="runtime-overview-kpi"><span>execution scope</span><strong>${escapeHtml(_short(matrix.outbox_execution_scope || "-", 28))}</strong></div>
      </div>
      <div class="runtime-overview-detail-grid">
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Execution Owner</div>
          <div class="detail-subtext">${escapeHtml(_short(matrix.outbox_execution_owner || "-", 40))}</div>
        </div>
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">QQ Accounts</div>
          <div class="detail-subtext">${escapeHtml(_short(configuredBotIds.join(", ") || "-", 80))}</div>
        </div>
      </div>
    </div>
    ${_renderQqCutoverRouteTable("Go Execution Owner Scope", _array(matrix.go_execution_owner_scope), "No go-owned routes sampled")}
    ${_renderQqCutoverRouteTable("Currently Sendable Routes", _array(matrix.currently_sendable_routes), "No currently sendable routes sampled")}
    ${_renderQqCutoverRouteTable("Policy Blocked Routes", _array(matrix.policy_blocked_routes), "No policy-blocked routes sampled")}
    ${_renderQqCutoverRouteTable("Platform Blocker Routes", _array(matrix.platform_blocker_routes), "No platform-blocker routes sampled")}
    ${_renderRuntimeOverviewNotesSection("Notes", matrix.notes, "No QQ cutover route-matrix notes")}
  `;
}

function _renderKnowledgeJobPlannerPreviewDetail(detail: Record<string, unknown>): string {
  const preview = _record(detail.knowledge_job_planner_preview || detail.preview || detail);
  const groups = Array.isArray(preview.groups) ? preview.groups : [];
  const plans = _array(preview.plans);
  const rows = plans.slice(0, 20).map((item) => {
    const channel = _record(item.channel);
    const jobs = _array(item.jobs);
    const firstJob = jobs[0] || {};
    const payload = _record(firstJob.payload);
    const metadata = _record(firstJob.metadata);
    const accountId = String(channel.account_id || "-");
    const conversationId = String(channel.conversation_id || payload.group_id || "-");
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.target_id || "-", 40))}</td>
        <td>${escapeHtml(_short(`${String(channel.kind || "-")} / ${accountId} / ${conversationId}`, 42))}</td>
        <td>${escapeHtml(String(jobs.length))}</td>
        <td>${escapeHtml(_short(firstJob.job_type || "-", 26))}</td>
        <td>${escapeHtml(String(payload.observe_only ?? "-"))}</td>
        <td>${escapeHtml(_short(metadata.scheduler || "-", 40))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Knowledge Planner Preview</div>
          <div class="detail-subtext">${escapeHtml(_short(String(preview.timestamp || "-"), 44))} · bucket ${escapeHtml(String(preview.bucket ?? "-"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>Targets</span><strong>${escapeHtml(String(preview.targets ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Skipped</span><strong>${escapeHtml(String(preview.skipped_targets ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Total Jobs</span><strong>${escapeHtml(String(preview.total_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Group Memory Jobs</span><strong>${escapeHtml(String(preview.group_memory_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>RAG Ingest Jobs</span><strong>${escapeHtml(String(preview.rag_ingest_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Interval Seconds</span><strong>${escapeHtml(String(preview.interval_seconds ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-detail-grid">
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Groups</div>
          <div class="detail-subtext">${escapeHtml(_short(groups.join(", ") || "-", 220))}</div>
        </div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Target</th>
              <th>Channel</th>
              <th>Jobs</th>
              <th>First Job Type</th>
              <th>Observe Only</th>
              <th>Scheduler</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="6" class="runtime-overview-muted">No knowledge planner preview sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderKnowledgeJobPlannerReadinessDetail(detail: Record<string, unknown>): string {
  const readiness = _record(detail.knowledge_job_planner_readiness || detail.readiness || detail);
  const preview = _record(readiness.preview);
  const notes = Array.isArray(readiness.notes) ? readiness.notes : [];
  const noteRows = notes.slice(0, 10).map((item) => `
    <tr><td>${escapeHtml(_short(item, 140))}</td></tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Knowledge Planner Readiness</div>
          <div class="detail-subtext">${escapeHtml(_short(readiness.reason || "-", 44))} · ${escapeHtml(String(readiness.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>Ready</span><strong>${escapeHtml(String(Boolean(readiness.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>Planner Enabled</span><strong>${escapeHtml(String(Boolean(readiness.planner_enabled)))}</strong></div>
        <div class="runtime-overview-kpi"><span>Planner Running</span><strong>${escapeHtml(String(Boolean(readiness.planner_running)))}</strong></div>
        <div class="runtime-overview-kpi"><span>Worker Ready</span><strong>${escapeHtml(String(Boolean(readiness.knowledge_worker_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>Worker Active</span><strong>${escapeHtml(String(readiness.knowledge_worker_active ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Worker Stale</span><strong>${escapeHtml(String(readiness.knowledge_worker_stale ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Worker Failed</span><strong>${escapeHtml(String(readiness.knowledge_worker_failed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>Worker Stopped</span><strong>${escapeHtml(String(readiness.knowledge_worker_stopped ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-detail-grid">
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Preview Bucket</div>
          <div class="detail-subtext">${escapeHtml(String(preview.bucket ?? "-"))}</div>
        </div>
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Preview Targets</div>
          <div class="detail-subtext">${escapeHtml(String(preview.targets ?? 0))}</div>
        </div>
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Preview Total Jobs</div>
          <div class="detail-subtext">${escapeHtml(String(preview.total_jobs ?? 0))}</div>
        </div>
        <div class="runtime-overview-detail-pair">
          <div class="detail-label">Preview Timestamp</div>
          <div class="detail-subtext">${escapeHtml(_short(preview.timestamp || "-", 44))}</div>
        </div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr><th>Notes</th></tr>
          </thead>
          <tbody>${noteRows || `<tr><td class="runtime-overview-muted">No knowledge planner readiness notes</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderAgentJobExternalLeaseSteps(
  title: string,
  steps: Record<string, unknown>[],
): string {
  const rows = steps.slice(0, 20).map((item) => {
    const env = _record(item.env);
    const envText = Object.keys(env).length
      ? Object.entries(env)
          .map(([key, value]) => `${key}=${String(value ?? "")}`)
          .join("\n")
      : "-";
    return `
      <tr>
        <td>${escapeHtml(String(item.step_index ?? "-"))}</td>
        <td>${escapeHtml(_short(item.phase || "-", 18))}</td>
        <td>${escapeHtml(_short(item.action || "-", 40))}</td>
        <td class="mono">${escapeHtml(_short(item.method || "-", 12))}</td>
        <td class="mono">${escapeHtml(_short(item.endpoint || "-", 40))}</td>
        <td>${escapeHtml(_short(item.detail || "-", 72))}</td>
        <td><pre class="runtime-overview-json runtime-overview-json-inline">${escapeHtml(envText)}</pre></td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">${escapeHtml(title)}</div>
        <div class="detail-subtext">${escapeHtml(String(steps.length))} steps</div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>#</th>
              <th>Phase</th>
              <th>Action</th>
              <th>Method</th>
              <th>Endpoint</th>
              <th>Detail</th>
              <th>Env</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="7" class="runtime-overview-muted">No ${escapeHtml(title.toLowerCase())} sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderExternalLeaseDiagnosticsDetail(detail: Record<string, unknown>): string {
  const queue = _record(detail.queue_backend || detail.external_lease_diagnostics || detail);
  const selectedCapability = _record(queue.selected_provider_capability);
  const notes = Array.isArray(queue.notes) ? queue.notes : [];
  const blockers = Array.isArray(selectedCapability.blockers) ? selectedCapability.blockers : [];
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">External Lease Diagnostics</div>
          <div class="detail-subtext">${escapeHtml(_short(queue.migration_phase || "-", 24))} · ${escapeHtml(_short(queue.lease_owner || "-", 28))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>provider</span><strong>${escapeHtml(_short(queue.provider || "-", 18))}</strong></div>
        <div class="runtime-overview-kpi"><span>mode</span><strong>${escapeHtml(_short(queue.mode || "-", 18))}</strong></div>
        <div class="runtime-overview-kpi"><span>external queue</span><strong>${escapeHtml(String(Boolean(queue.external_queue_configured)))}</strong></div>
        <div class="runtime-overview-kpi"><span>external active</span><strong>${escapeHtml(String(Boolean(queue.external_queue_active)))}</strong></div>
        <div class="runtime-overview-kpi"><span>consumer concurrency</span><strong>${escapeHtml(String(queue.consumer_concurrency ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>max in flight</span><strong>${escapeHtml(String(queue.max_in_flight ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>outbox owner</span><strong>${escapeHtml(_short(queue.outbox_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>agent_job owner</span><strong>${escapeHtml(_short(queue.agent_job_execution_owner || "-", 28))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Selected Provider</th>
              <th>Consumer Model</th>
              <th>Adapter Boundary</th>
              <th>Recommended First Backend</th>
              <th>Supports External Lease</th>
              <th>Supports Result Ack</th>
              <th>Blockers</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>${escapeHtml(_short(selectedCapability.provider || queue.provider || "-", 20))}</td>
              <td>${escapeHtml(_short(selectedCapability.consumer_model || queue.consumer_model || "-", 24))}</td>
              <td>${escapeHtml(_short(selectedCapability.adapter_boundary || "-", 40))}</td>
              <td>${escapeHtml(_short(queue.recommended_first_backend || "-", 22))}</td>
              <td>${escapeHtml(String(Boolean(selectedCapability.supports_external_lease)))}</td>
              <td>${escapeHtml(String(Boolean(selectedCapability.supports_agent_job_result_ack)))}</td>
              <td>${escapeHtml(_short((blockers.join(", ") || "-"), 56))}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    ${_renderRuntimeOverviewNotesSection("Notes", notes, "None")}
  `;
}

function _renderSchedulerJobsDetail(detail: Record<string, unknown>): string {
  const scheduler = _record(detail.scheduler_jobs || detail);
  const jobsByTrigger = _record(scheduler.jobs_by_trigger);
  const jobsByTier = _record(scheduler.jobs_by_tier);
  const jobsByStatus = _record(scheduler.jobs_by_status);
  const recent = _array(scheduler.recent).slice(0, 20);
  const rows = recent.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_id || "-", 34))}</td>
      <td>${escapeHtml(_short(item.trigger_type || item.trigger || "-", 16))}</td>
      <td>${escapeHtml(_short(item.tier || "-", 12))}</td>
      <td>${_runtimeStatusTag(String(item.status || "muted"))}</td>
      <td>${escapeHtml(_short(item.channel || "-", 18))}</td>
      <td>${escapeHtml(_short(item.next_run_at || item.run_at || "-", 28))}</td>
      <td>${escapeHtml(_short(item.reason || "-", 36))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Scheduler Jobs</div>
          <div class="detail-subtext">${escapeHtml(String(scheduler.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>sampled</span><strong>${escapeHtml(String(scheduler.sampled_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>enabled</span><strong>${escapeHtml(String(scheduler.enabled_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>disabled</span><strong>${escapeHtml(String(scheduler.disabled_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>overdue</span><strong>${escapeHtml(String(scheduler.overdue_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>due soon</span><strong>${escapeHtml(String(scheduler.due_soon_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>instant</span><strong>${escapeHtml(String(scheduler.instant_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>soft</span><strong>${escapeHtml(String(scheduler.soft_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>due soon(s)</span><strong>${escapeHtml(String(scheduler.due_soon_seconds ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Trigger</th>
              <th>Count</th>
              <th>Tier</th>
              <th>Count</th>
              <th>Status</th>
              <th>Count</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>after</td>
              <td>${escapeHtml(String(jobsByTrigger.after ?? 0))}</td>
              <td>instant</td>
              <td>${escapeHtml(String(jobsByTier.instant ?? 0))}</td>
              <td>future</td>
              <td>${escapeHtml(String(jobsByStatus.future ?? 0))}</td>
            </tr>
            <tr>
              <td>at</td>
              <td>${escapeHtml(String(jobsByTrigger.at ?? 0))}</td>
              <td>soft</td>
              <td>${escapeHtml(String(jobsByTier.soft ?? 0))}</td>
              <td>due_soon</td>
              <td>${escapeHtml(String(jobsByStatus.due_soon ?? 0))}</td>
            </tr>
            <tr>
              <td>every</td>
              <td>${escapeHtml(String(jobsByTrigger.every ?? 0))}</td>
              <td>-</td>
              <td>-</td>
              <td>overdue</td>
              <td>${escapeHtml(String(jobsByStatus.overdue ?? 0))}</td>
            </tr>
            <tr>
              <td>-</td>
              <td>-</td>
              <td>-</td>
              <td>-</td>
              <td>disabled</td>
              <td>${escapeHtml(String(jobsByStatus.disabled ?? 0))}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job</th>
              <th>Trigger</th>
              <th>Tier</th>
              <th>Status</th>
              <th>Channel</th>
              <th>Next Run</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="7" class="runtime-overview-muted">No scheduler jobs sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderAgentWorkersDetail(detail: Record<string, unknown>): string {
  const workersDetail = _record(detail.agent_workers || detail);
  const totals = _record(workersDetail.totals);
  const workers = _array(workersDetail.workers).slice(0, 20);
  const rows = workers.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.worker_id || "-", 32))}</td>
      <td>${escapeHtml(_short(item.worker_type || "-", 18))}</td>
      <td>${_runtimeStatusTag(String(item.status || "muted"))}</td>
      <td>${escapeHtml(String(Boolean(item.lease_active)))}</td>
      <td>${escapeHtml(String(Boolean(item.stale)))}</td>
      <td>${escapeHtml(String(item.processed_total ?? 0))}</td>
      <td>${escapeHtml(String(item.failed_total ?? 0))}</td>
      <td>${escapeHtml(_short(_record(item.metadata).reason || item.last_error || "-", 42))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Workers</div>
          <div class="detail-subtext">${escapeHtml(String(workersDetail.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>workers</span><strong>${escapeHtml(String(totals.workers ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>idle</span><strong>${escapeHtml(String(totals.idle ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>running</span><strong>${escapeHtml(String(totals.running ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>stopped</span><strong>${escapeHtml(String(totals.stopped ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>stale</span><strong>${escapeHtml(String(totals.stale ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>failed</span><strong>${escapeHtml(String(totals.failed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>knowledge</span><strong>${escapeHtml(String(totals.knowledge ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>outbox</span><strong>${escapeHtml(String(totals.outbox_delivery ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Worker</th>
              <th>Type</th>
              <th>Status</th>
              <th>Lease Active</th>
              <th>Stale</th>
              <th>Processed</th>
              <th>Failed</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No agent workers sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderRuntimeOverviewNotesSection("Notes", workersDetail.notes, "None")}
  `;
}

function _renderRuntimeWorkersDetail(detail: Record<string, unknown>): string {
  const workersDetail = _record(detail.runtime_workers || detail);
  const totals = _record(workersDetail.totals);
  const workers = _array(workersDetail.workers).slice(0, 20);
  const rows = workers.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.name || "-", 28))}</td>
      <td>${escapeHtml(_short(item.kind || "-", 22))}</td>
      <td>${escapeHtml(String(Boolean(item.enabled)))}</td>
      <td>${escapeHtml(String(Boolean(item.running)))}</td>
      <td class="mono">${escapeHtml(_short(item.worker_id || "-", 32))}</td>
      <td>${escapeHtml(String(item.interval_seconds ?? item.consumer_concurrency ?? "-"))}</td>
      <td>${escapeHtml(String(item.batch_size ?? item.max_in_flight ?? "-"))}</td>
      <td>${escapeHtml(_short((_array(item.notes).map((note) => _short(note, 24)).join(", ") || "-"), 56))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Runtime Workers</div>
          <div class="detail-subtext">${escapeHtml(_short((_array(workersDetail.notes).map((note) => _short(note, 32)).join(", ") || "none"), 80))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>workers</span><strong>${escapeHtml(String(totals.workers ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>enabled</span><strong>${escapeHtml(String(totals.enabled ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>disabled</span><strong>${escapeHtml(String(totals.disabled ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>running</span><strong>${escapeHtml(String(totals.running ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Kind</th>
              <th>Enabled</th>
              <th>Running</th>
              <th>Worker ID</th>
              <th>Interval / Concurrency</th>
              <th>Batch / Max In Flight</th>
              <th>Notes</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No runtime workers sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderAgentJobWorkerCoverageDetail(detail: Record<string, unknown>): string {
  const items = _array(detail.agent_job_worker_coverage || detail).slice(0, 20);
  const rows = items.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_type || "-", 28))}</td>
      <td>${escapeHtml(_short((Array.isArray(item.expected_worker_types) ? item.expected_worker_types.join(", ") : "-"), 24))}</td>
      <td>${_runtimeStatusTag(String(item.coverage_status || "muted"))}</td>
      <td>${escapeHtml(String(item.worker_count ?? 0))}</td>
      <td>${escapeHtml(String(item.active_workers ?? 0))}</td>
      <td>${escapeHtml(String(item.running_workers ?? 0))}</td>
      <td>${escapeHtml(String(item.failed_workers ?? 0))}</td>
      <td>${escapeHtml(String(item.stale_workers ?? 0))}</td>
      <td>${escapeHtml(String(Boolean(item.high_pressure)))}</td>
      <td>${escapeHtml(_short(item.coverage_reason || "-", 36))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Job Worker Coverage</div>
          <div class="detail-subtext">${escapeHtml(_short(`${items.length} sampled job types`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Expected Workers</th>
              <th>Coverage</th>
              <th>Workers</th>
              <th>Active</th>
              <th>Running</th>
              <th>Failed</th>
              <th>Stale</th>
              <th>High Pressure</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="10" class="runtime-overview-muted">No worker coverage sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderAgentJobPressureDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.agent_job_metrics || detail.agent_job_pressure || detail);
  const throughput = _record(metrics.throughput);
  const deadLetters = _record(metrics.dead_letters);
  const pressure = _record(metrics.pressure);
  const items = _array(pressure.by_type).slice(0, 20);
  const rows = items.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_type || "-", 28))}</td>
      <td>${escapeHtml(String(item.pending ?? 0))}</td>
      <td>${escapeHtml(String(item.leased ?? 0))}</td>
      <td>${escapeHtml(String(item.running ?? 0))}</td>
      <td>${escapeHtml(String(item.active ?? 0))}</td>
      <td>${escapeHtml(String(item.oldest_pending_age_seconds ?? 0))}</td>
      <td>${escapeHtml(String(Boolean(item.high_pressure)))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Job Pressure</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_jobs ?? 0} jobs / ${metrics.sampled_events ?? 0} events sampled`, 44))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>job types</span><strong>${escapeHtml(String(pressure.job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>high pressure</span><strong>${escapeHtml(String(pressure.high_pressure_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>max pending</span><strong>${escapeHtml(String(pressure.max_pending ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>max active</span><strong>${escapeHtml(String(pressure.max_active ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>oldest pending(s)</span><strong>${escapeHtml(String(pressure.oldest_pending_age_seconds ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>created</span><strong>${escapeHtml(String(throughput.created ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>succeeded</span><strong>${escapeHtml(String(throughput.succeeded ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dead letters</span><strong>${escapeHtml(String(deadLetters.current_total ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Pending</th>
              <th>Leased</th>
              <th>Running</th>
              <th>Active</th>
              <th>Oldest Pending</th>
              <th>High Pressure</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="7" class="runtime-overview-muted">No pressure items sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderAgentJobMetricsDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.agent_job_metrics || detail);
  const throughput = _record(metrics.throughput);
  const deadLetters = _record(metrics.dead_letters);
  const jobsByType = _record(metrics.jobs_by_type);
  const pressure = _record(metrics.pressure);
  const typeRows = Object.entries(jobsByType).slice(0, 20).map(([jobType, value]) => {
    const item = _record(value);
    const byStatus = _record(item.by_status);
    const statusSummary = Object.entries(byStatus).map(([status, count]) => `${status}:${count}`).join(", ") || "-";
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(jobType, 28))}</td>
        <td>${escapeHtml(String(item.total ?? 0))}</td>
        <td>${escapeHtml(statusSummary)}</td>
      </tr>
    `;
  }).join("");
  const pressureRows = _array(pressure.by_type).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_type || "-", 28))}</td>
      <td>${escapeHtml(String(item.pending ?? 0))}</td>
      <td>${escapeHtml(String(item.leased ?? 0))}</td>
      <td>${escapeHtml(String(item.running ?? 0))}</td>
      <td>${escapeHtml(String(item.active ?? 0))}</td>
      <td>${escapeHtml(String(item.oldest_pending_age_seconds ?? 0))}</td>
      <td>${escapeHtml(String(Boolean(item.high_pressure)))}</td>
    </tr>
  `).join("");
  const deadLetterRows = _array(deadLetters.recent).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_id || "-", 32))}</td>
      <td>${escapeHtml(String(item.job_type ?? "-"))}</td>
      <td>${escapeHtml(String(item.status ?? "-"))}</td>
      <td>${escapeHtml(String(item.event_type ?? "-"))}</td>
      <td>${escapeHtml(_short(item.error_message || "-", 36))}</td>
      <td>${escapeHtml(String(item.attempt ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
      <td>${escapeHtml(_short(item.updated_at || item.occurred_at || "-", 24))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Job Metrics</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_jobs ?? 0} jobs / ${metrics.sampled_events ?? 0} events sampled`, 44))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>sampled jobs</span><strong>${escapeHtml(String(metrics.sampled_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>sampled events</span><strong>${escapeHtml(String(metrics.sampled_events ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>job types</span><strong>${escapeHtml(String(Object.keys(jobsByType).length))}</strong></div>
        <div class="runtime-overview-kpi"><span>created</span><strong>${escapeHtml(String(throughput.created ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>leased</span><strong>${escapeHtml(String(throughput.leased ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>running</span><strong>${escapeHtml(String(throughput.running ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>succeeded</span><strong>${escapeHtml(String(throughput.succeeded ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dead letters current</span><strong>${escapeHtml(String(deadLetters.current_total ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Total</th>
              <th>Statuses</th>
            </tr>
          </thead>
          <tbody>${typeRows || `<tr><td colspan="3" class="runtime-overview-muted">No job types sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Pending</th>
              <th>Leased</th>
              <th>Running</th>
              <th>Active</th>
              <th>Oldest Pending</th>
              <th>High Pressure</th>
            </tr>
          </thead>
          <tbody>${pressureRows || `<tr><td colspan="7" class="runtime-overview-muted">No agent-job pressure sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job ID</th>
              <th>Job Type</th>
              <th>Status</th>
              <th>Event</th>
              <th>Error</th>
              <th>Attempt</th>
              <th>Updated</th>
            </tr>
          </thead>
          <tbody>${deadLetterRows || `<tr><td colspan="7" class="runtime-overview-muted">No recent agent-job dead letters sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderOutboxPressureDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.outbox_metrics || detail.outbox_pressure || detail);
  const throughput = _record(metrics.throughput);
  const deadLetters = _record(metrics.dead_letters);
  const pressure = _record(metrics.pressure);
  const deliveriesByStatus = _record(metrics.deliveries_by_status);
  const byChannelKind = _record(metrics.deliveries_by_channel_kind);
  const items = _array(pressure.by_account).slice(0, 20);
  const rows = items.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.account_key || "-", 28))}</td>
      <td>${escapeHtml(String(item.channel_kind ?? "-"))}</td>
      <td>${escapeHtml(String(item.account_id ?? "-"))}</td>
      <td>${escapeHtml(String(item.queued ?? 0))}</td>
      <td>${escapeHtml(String(item.dispatching ?? 0))}</td>
      <td>${escapeHtml(String(item.active ?? 0))}</td>
      <td>${escapeHtml(String(item.dead_lettered ?? 0))}</td>
      <td>${escapeHtml(String(Boolean(item.high_pressure)))}</td>
      <td>${escapeHtml(_short(item.pressure_reason || "-", 32))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Outbox Pressure</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_deliveries ?? 0} deliveries / ${metrics.sampled_events ?? 0} events sampled`, 48))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>accounts</span><strong>${escapeHtml(String(pressure.accounts ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>high pressure</span><strong>${escapeHtml(String(pressure.high_pressure_accounts ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>max queued</span><strong>${escapeHtml(String(pressure.max_queued ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>max active</span><strong>${escapeHtml(String(pressure.max_active ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>queued</span><strong>${escapeHtml(String(throughput.queued ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>leased</span><strong>${escapeHtml(String(throughput.leased ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>succeeded</span><strong>${escapeHtml(String(throughput.succeeded ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dead letters</span><strong>${escapeHtml(String(deadLetters.current_total ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Account Key</th>
              <th>Channel</th>
              <th>Account ID</th>
              <th>Queued</th>
              <th>Dispatching</th>
              <th>Active</th>
              <th>Dead Lettered</th>
              <th>High Pressure</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="9" class="runtime-overview-muted">No outbox pressure sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">Delivery Status</div>
        <pre class="runtime-overview-json">${escapeHtml(JSON.stringify(deliveriesByStatus, null, 2))}</pre>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">By Channel Kind</div>
        <pre class="runtime-overview-json">${escapeHtml(JSON.stringify(byChannelKind, null, 2))}</pre>
      </div>
    </div>
  `;
}

function _renderOutboxMetricsDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.outbox_metrics || detail.outbox_pressure || detail);
  const throughput = _record(metrics.throughput);
  const deadLetters = _record(metrics.dead_letters);
  const byChannelKind = _record(metrics.deliveries_by_channel_kind);
  const recent = _array(deadLetters.recent).slice(0, 20);
  const rows = recent.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.delivery_id || "-", 32))}</td>
      <td>${escapeHtml(String(item.channel_kind ?? "-"))}</td>
      <td>${escapeHtml(String(item.event_type ?? "-"))}</td>
      <td>${escapeHtml(String(item.status ?? "-"))}</td>
      <td>${escapeHtml(String(item.error_kind ?? "-"))}</td>
      <td>${escapeHtml(_short(item.error_message || "-", 32))}</td>
      <td>${escapeHtml(String(item.attempt ?? 0))}/${escapeHtml(String(item.max_attempts ?? 0))}</td>
      <td>${escapeHtml(_short(item.occurred_at || "-", 24))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Outbox Metrics</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_deliveries ?? 0} deliveries / ${metrics.sampled_events ?? 0} events sampled`, 48))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>queued</span><strong>${escapeHtml(String(throughput.queued ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>leased</span><strong>${escapeHtml(String(throughput.leased ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dispatching</span><strong>${escapeHtml(String(throughput.dispatching ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>succeeded</span><strong>${escapeHtml(String(throughput.succeeded ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>failed</span><strong>${escapeHtml(String(throughput.failed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dead lettered</span><strong>${escapeHtml(String(throughput.dead_lettered ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>terminal events</span><strong>${escapeHtml(String(throughput.terminal_events ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>dead letters current</span><strong>${escapeHtml(String(deadLetters.current_total ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">By Channel Kind</div>
        <pre class="runtime-overview-json">${escapeHtml(JSON.stringify(byChannelKind, null, 2))}</pre>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Delivery ID</th>
              <th>Channel</th>
              <th>Event</th>
              <th>Status</th>
              <th>Error Kind</th>
              <th>Error</th>
              <th>Attempt</th>
              <th>Occurred At</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No recent dead letters sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderSendLedgerMetricsDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.send_ledger_metrics || detail);
  const repeatedHashes = _array(metrics.repeated_hashes).slice(0, 20);
  const recent = _array(metrics.recent).slice(0, 20);
  const repeatedRows = repeatedHashes.map((item) => `
    <tr>
      <td>${escapeHtml(String(item.from_bot_id ?? "-"))}</td>
      <td>${escapeHtml(String(item.conversation_id ?? "-"))}</td>
      <td class="mono">${escapeHtml(_short(item.content_hash || "-", 28))}</td>
      <td>${escapeHtml(String(item.count ?? 0))}</td>
      <td>${escapeHtml(_short(item.latest_timestamp || "-", 24))}</td>
    </tr>
  `).join("");
  const recentRows = recent.map((item) => `
    <tr>
      <td>${escapeHtml(String(item.from_bot_id ?? "-"))}</td>
      <td>${escapeHtml(String(item.conversation_id ?? "-"))}</td>
      <td class="mono">${escapeHtml(_short(item.content_hash || "-", 28))}</td>
      <td>${escapeHtml(_short(item.timestamp || "-", 24))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Send Ledger Metrics</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_records ?? 0} records sampled`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>records</span><strong>${escapeHtml(String(metrics.sampled_records ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>bots</span><strong>${escapeHtml(String(metrics.unique_bots ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>conversations</span><strong>${escapeHtml(String(metrics.unique_conversations ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>content hashes</span><strong>${escapeHtml(String(metrics.unique_content_hashes ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>repeated hashes</span><strong>${escapeHtml(String(metrics.repeated_content_hashes ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Bot</th>
              <th>Conversation</th>
              <th>Content Hash</th>
              <th>Count</th>
              <th>Latest</th>
            </tr>
          </thead>
          <tbody>${repeatedRows || `<tr><td colspan="5" class="runtime-overview-muted">No repeated hashes sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Bot</th>
              <th>Conversation</th>
              <th>Content Hash</th>
              <th>Timestamp</th>
            </tr>
          </thead>
          <tbody>${recentRows || `<tr><td colspan="4" class="runtime-overview-muted">No recent send ledger records sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderInboxMetricsDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.inbox_metrics || detail);
  const byChannelKind = _record(metrics.events_by_channel_kind);
  const byConversation = Object.entries(_record(metrics.events_by_conversation))
    .slice(0, 20)
    .map(([conversationKey, rawValue]) => {
      const item = _record(rawValue);
      const channel = _record(item.channel);
      const channelLabel = [channel.kind, channel.account_id, channel.conversation_type]
        .filter(Boolean)
        .join(" / ");
      return `
        <tr>
          <td class="mono">${escapeHtml(_short(conversationKey, 40))}</td>
          <td>${escapeHtml(channelLabel || "-")}</td>
          <td>${escapeHtml(String(item.total ?? 0))}</td>
          <td>${escapeHtml(String(item.observe_only ?? 0))}</td>
          <td>${escapeHtml(String(item.reply_eligible ?? 0))}</td>
          <td>${escapeHtml(String(item.with_attachments ?? 0))}</td>
          <td>${escapeHtml(String(item.unique_senders ?? 0))}</td>
          <td>${escapeHtml(String(item.latest_seq ?? "-"))}</td>
          <td>${escapeHtml(_short(item.latest_received_at || "-", 24))}</td>
        </tr>
      `;
    })
    .join("");
  const recentRows = _array(metrics.recent).slice(0, 20).map((item) => {
    const channel = _record(item.channel);
    const channelLabel = [channel.kind, channel.account_id, channel.conversation_id]
      .filter(Boolean)
      .join(" / ");
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.event_id || "-", 36))}</td>
        <td>${escapeHtml(channelLabel || "-")}</td>
        <td>${escapeHtml(String(item.sender_kind ?? "-"))}</td>
        <td>${escapeHtml(String(item.decision_action ?? "-"))}</td>
        <td>${escapeHtml(item.observe_only === true ? "true" : "false")}</td>
        <td>${escapeHtml(String(item.attachment_count ?? 0))}</td>
        <td>${escapeHtml(String(item.seq ?? "-"))}</td>
        <td>${escapeHtml(_short(item.received_at || "-", 24))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Inbox Metrics</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_events ?? 0} events sampled`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>events</span><strong>${escapeHtml(String(metrics.sampled_events ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>observe only</span><strong>${escapeHtml(String(metrics.observe_only_total ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>reply eligible</span><strong>${escapeHtml(String(metrics.reply_eligible_total ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>attachments</span><strong>${escapeHtml(String(metrics.with_attachments ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>attachment count</span><strong>${escapeHtml(String(metrics.attachment_count ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>unique senders</span><strong>${escapeHtml(String(metrics.unique_senders ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">By Channel Kind</div>
        <pre class="runtime-overview-json">${escapeHtml(JSON.stringify(byChannelKind, null, 2))}</pre>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Conversation</th>
              <th>Channel</th>
              <th>Total</th>
              <th>Observe Only</th>
              <th>Reply Eligible</th>
              <th>With Attachments</th>
              <th>Unique Senders</th>
              <th>Latest Seq</th>
              <th>Latest Received</th>
            </tr>
          </thead>
          <tbody>${byConversation || `<tr><td colspan="9" class="runtime-overview-muted">No conversation metrics sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Event ID</th>
              <th>Channel</th>
              <th>Sender Kind</th>
              <th>Decision</th>
              <th>Observe Only</th>
              <th>Attachments</th>
              <th>Seq</th>
              <th>Received At</th>
            </tr>
          </thead>
          <tbody>${recentRows || `<tr><td colspan="8" class="runtime-overview-muted">No recent inbox events sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderInboundDedupeMetricsDetail(detail: Record<string, unknown>): string {
  const metrics = _record(detail.inbound_dedupe_metrics || detail);
  const scopeRows = _array(metrics.scopes).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.scope || "-", 40))}</td>
      <td>${escapeHtml(String(item.records ?? 0))}</td>
      <td>${escapeHtml(String(item.active_records ?? 0))}</td>
      <td>${escapeHtml(String(item.expired_records ?? 0))}</td>
      <td>${escapeHtml(String(item.duplicate_records ?? 0))}</td>
      <td>${escapeHtml(String(item.seen_total ?? 0))}</td>
      <td>${escapeHtml(String(item.duplicate_seen_total ?? 0))}</td>
      <td>${escapeHtml(_short(item.latest_seen_at || "-", 24))}</td>
    </tr>
  `).join("");
  const notes = _array(metrics.notes).slice(0, 10).map((item) => `
    <li>${escapeHtml(String(item ?? "-"))}</li>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Inbound Dedupe</div>
          <div class="detail-subtext">${escapeHtml(_short(`${metrics.sampled_records ?? 0} records sampled`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>records</span><strong>${escapeHtml(String(metrics.sampled_records ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>active</span><strong>${escapeHtml(String(metrics.active_records ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>expired</span><strong>${escapeHtml(String(metrics.expired_records ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>duplicates</span><strong>${escapeHtml(String(metrics.duplicate_records ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>seen total</span><strong>${escapeHtml(String(metrics.seen_total ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>duplicate seen</span><strong>${escapeHtml(String(metrics.duplicate_seen_total ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Scope</th>
              <th>Records</th>
              <th>Active</th>
              <th>Expired</th>
              <th>Duplicates</th>
              <th>Seen Total</th>
              <th>Duplicate Seen</th>
              <th>Latest Seen</th>
            </tr>
          </thead>
          <tbody>${scopeRows || `<tr><td colspan="8" class="runtime-overview-muted">No dedupe scopes sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">Notes</div>
        <ul class="runtime-overview-list">${notes || `<li class="runtime-overview-muted">No dedupe notes sampled</li>`}</ul>
      </div>
    </div>
  `;
}

function _renderObserveTargetsDetail(detail: Record<string, unknown>): string {
  const observeTargets = _record(detail.observe_targets || detail);
  const totals = _record(observeTargets.totals);
  const targetRows = _array(observeTargets.targets).slice(0, 20).map((item) => {
    const channel = _record(item.channel);
    const channelLabel = [channel.kind, channel.account_id, channel.conversation_id]
      .filter(Boolean)
      .join(" / ");
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.target_id || "-", 40))}</td>
        <td>${escapeHtml(channelLabel || "-")}</td>
        <td>${escapeHtml(channel.conversation_type || "-")}</td>
        <td>${escapeHtml(item.observe_only === true ? "true" : "false")}</td>
        <td>${escapeHtml(item.reply_allowed === true ? "true" : "false")}</td>
        <td>${escapeHtml(item.require_at === true ? "true" : "false")}</td>
        <td>${escapeHtml(item.enabled === true ? "true" : "false")}</td>
        <td>${escapeHtml(String(item.source ?? "-"))}</td>
        <td>${escapeHtml(_short(item.updated_at || "-", 24))}</td>
      </tr>
    `;
  }).join("");
  const notes = _array(observeTargets.notes).slice(0, 10).map((item) => `
    <li>${escapeHtml(String(item ?? "-"))}</li>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Observe Targets</div>
          <div class="detail-subtext">${escapeHtml(_short(`${totals.targets ?? 0} targets configured`, 36))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>targets</span><strong>${escapeHtml(String(totals.targets ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>enabled</span><strong>${escapeHtml(String(totals.enabled ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>observe only</span><strong>${escapeHtml(String(totals.observe_only ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>reply allowed</span><strong>${escapeHtml(String(totals.reply_allowed ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>groups</span><strong>${escapeHtml(String(totals.groups ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>qq</span><strong>${escapeHtml(String(totals.qq ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Target ID</th>
              <th>Channel</th>
              <th>Conversation Type</th>
              <th>Observe Only</th>
              <th>Reply Allowed</th>
              <th>Require @</th>
              <th>Enabled</th>
              <th>Source</th>
              <th>Updated At</th>
            </tr>
          </thead>
          <tbody>${targetRows || `<tr><td colspan="9" class="runtime-overview-muted">No observe targets sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">Notes</div>
        <ul class="runtime-overview-list">${notes || `<li class="runtime-overview-muted">No observe-target notes sampled</li>`}</ul>
      </div>
    </div>
  `;
}

function _renderObserveCaptureDetail(detail: Record<string, unknown>): string {
  const observeCapture = _record(detail.observe_capture || detail);
  const totals = _record(observeCapture.totals);
  const targetRows = _array(observeCapture.targets).slice(0, 20).map((item) => {
    const channel = _record(item.channel);
    const coverage = _record(item.coverage);
    const channelLabel = [channel.kind, channel.account_id, channel.conversation_id]
      .filter(Boolean)
      .join(" / ");
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.target_id || "-", 40))}</td>
        <td>${escapeHtml(channelLabel || "-")}</td>
        <td>${escapeHtml(String(item.status ?? "-"))}</td>
        <td>${escapeHtml(String(item.receiver_status ?? "-"))}</td>
        <td>${escapeHtml(item.receiver_activity_recent === true ? "true" : "false")}</td>
        <td>${escapeHtml(String(item.inbox_events ?? 0))}</td>
        <td>${escapeHtml(String(item.media_assets ?? 0))}</td>
        <td>${escapeHtml(String(item.content_forbidden_assets ?? 0))}</td>
        <td>${escapeHtml(coverage.file_seen === true ? "true" : "false")}</td>
        <td>${escapeHtml(coverage.media_content_ready === true ? "true" : "false")}</td>
      </tr>
    `;
  }).join("");
  const notes = _array(observeCapture.notes).slice(0, 10).map((item) => `
    <li>${escapeHtml(String(item ?? "-"))}</li>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Observe Capture</div>
          <div class="detail-subtext">${escapeHtml(_short(`${totals.targets ?? 0} targets audited`, 32))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>targets</span><strong>${escapeHtml(String(totals.targets ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(totals.ready ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>warning</span><strong>${escapeHtml(String(totals.warning ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>blocked</span><strong>${escapeHtml(String(totals.blocked ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>image covered</span><strong>${escapeHtml(String(totals.image_covered ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>file covered</span><strong>${escapeHtml(String(totals.file_covered ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>content ready</span><strong>${escapeHtml(String(totals.content_ready_assets ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>receiver recent</span><strong>${escapeHtml(String(totals.receiver_activity_recent ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Target ID</th>
              <th>Channel</th>
              <th>Status</th>
              <th>Receiver</th>
              <th>Recent</th>
              <th>Inbox Events</th>
              <th>Media Assets</th>
              <th>Forbidden</th>
              <th>File Seen</th>
              <th>Content Ready</th>
            </tr>
          </thead>
          <tbody>${targetRows || `<tr><td colspan="10" class="runtime-overview-muted">No observe capture targets sampled</td></tr>`}</tbody>
        </table>
      </div>
      <div class="runtime-overview-section runtime-overview-section-tight">
        <div class="detail-label">Notes</div>
        <ul class="runtime-overview-list">${notes || `<li class="runtime-overview-muted">No observe-capture notes sampled</li>`}</ul>
      </div>
    </div>
  `;
}

function _renderDeliveryAdaptersDetail(detail: Record<string, unknown>): string {
  const items = _array(detail.items || detail.delivery_adapters || detail).slice(0, 20);
  const rows = items.map((item) => `
    <tr>
      <td>${escapeHtml(String(item.provider ?? "-"))}</td>
      <td>${escapeHtml(String(item.channel ?? "-"))}</td>
      <td>${escapeHtml(String(item.transport ?? "-"))}</td>
      <td>${escapeHtml(item.enabled === true ? "true" : "false")}</td>
      <td>${escapeHtml(item.endpoint_configured === true ? "true" : "false")}</td>
      <td>${escapeHtml(item.access_token_configured === true ? "true" : "false")}</td>
      <td class="mono">${escapeHtml(_short(item.endpoint || "-", 42))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Delivery Adapters</div>
          <div class="detail-subtext">${escapeHtml(_short(`${items.length} adapters configured`, 30))}</div>
        </div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Provider</th>
              <th>Channel</th>
              <th>Transport</th>
              <th>Enabled</th>
              <th>Endpoint</th>
              <th>Access Token</th>
              <th>Endpoint URL</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="7" class="runtime-overview-muted">No delivery adapters sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
  `;
}

function _renderAgentJobExternalLeaseReadinessDetail(detail: Record<string, unknown>): string {
  const readiness = _record(detail.agent_job_external_lease_readiness || detail);
  const coverageRows = _array(readiness.worker_coverage).slice(0, 20).map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.job_type || "-", 28))}</td>
      <td>${escapeHtml(Array.isArray(item.expected_worker_types) ? item.expected_worker_types.join(", ") : "-")}</td>
      <td>${escapeHtml(String(item.worker_count ?? 0))}</td>
      <td>${escapeHtml(String(item.active_workers ?? 0))}</td>
      <td>${escapeHtml(String(item.running_workers ?? 0))}</td>
      <td>${escapeHtml(String(item.failed_workers ?? 0))}</td>
      <td>${escapeHtml(String(item.stale_workers ?? 0))}</td>
      <td>${_runtimeStatusTag(String(item.coverage_status || "muted"))}</td>
      <td>${escapeHtml(_short(item.coverage_reason || "-", 44))}</td>
    </tr>
  `).join("");
  const blockers = Array.isArray(readiness.blockers) ? readiness.blockers : [];
  const notes = Array.isArray(readiness.notes) ? readiness.notes : [];
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Job External Lease</div>
          <div class="detail-subtext">${escapeHtml(_short(readiness.reason || "-", 44))} · ${escapeHtml(String(readiness.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(Boolean(readiness.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>execution owner</span><strong>${escapeHtml(_short(readiness.execution_owner || "-", 26))}</strong></div>
        <div class="runtime-overview-kpi"><span>queue provider</span><strong>${escapeHtml(_short(readiness.queue_provider || "-", 18))}</strong></div>
        <div class="runtime-overview-kpi"><span>queue mode</span><strong>${escapeHtml(_short(readiness.queue_mode || "-", 20))}</strong></div>
        <div class="runtime-overview-kpi"><span>strict token</span><strong>${escapeHtml(String(Boolean(readiness.strict_lease_token_enabled)))}</strong></div>
        <div class="runtime-overview-kpi"><span>result ack</span><strong>${escapeHtml(String(Boolean(readiness.agent_job_result_ack_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>external lease</span><strong>${escapeHtml(String(Boolean(readiness.external_lease_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>worker ready</span><strong>${escapeHtml(String(Boolean(readiness.agent_job_worker_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>blockers</span><strong>${escapeHtml(String(blockers.length))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Expected Workers</th>
              <th>Workers</th>
              <th>Active</th>
              <th>Running</th>
              <th>Failed</th>
              <th>Stale</th>
              <th>Status</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>${coverageRows || `<tr><td colspan="9" class="runtime-overview-muted">No worker coverage sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Blockers</div>
      </div>
      <pre class="runtime-overview-json">${escapeHtml(blockers.length ? blockers.join("\n") : "None")}</pre>
    </div>
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Notes</div>
      </div>
      <pre class="runtime-overview-json">${escapeHtml(notes.length ? notes.join("\n") : "None")}</pre>
    </div>
  `;
}

function _renderAgentJobExternalLeasePlanDetail(detail: Record<string, unknown>): string {
  const plan = _record(detail.agent_job_external_lease_plan || detail);
  const readiness = _record(plan.readiness);
  const blockers = Array.isArray(plan.blockers) ? plan.blockers : [];
  const notes = Array.isArray(plan.notes) ? plan.notes : [];
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Job External Lease Plan</div>
          <div class="detail-subtext">${escapeHtml(_short(plan.decision || "-", 24))} · ${escapeHtml(String(plan.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(Boolean(plan.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>decision</span><strong>${escapeHtml(_short(plan.decision || "-", 20))}</strong></div>
        <div class="runtime-overview-kpi"><span>current owner</span><strong>${escapeHtml(_short(plan.current_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>desired owner</span><strong>${escapeHtml(_short(plan.desired_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>recommended owner</span><strong>${escapeHtml(_short(plan.recommended_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>readiness reason</span><strong>${escapeHtml(_short(readiness.reason || "-", 24))}</strong></div>
        <div class="runtime-overview-kpi"><span>blockers</span><strong>${escapeHtml(String(blockers.length))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Current Owner</th>
              <th>Desired Owner</th>
              <th>Recommended Owner</th>
              <th>Queue Provider</th>
              <th>Queue Mode</th>
              <th>Worker Ready</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>${escapeHtml(_short(plan.current_execution_owner || "-", 40))}</td>
              <td>${escapeHtml(_short(plan.desired_execution_owner || "-", 40))}</td>
              <td>${escapeHtml(_short(plan.recommended_execution_owner || "-", 40))}</td>
              <td>${escapeHtml(_short(readiness.queue_provider || "-", 24))}</td>
              <td>${escapeHtml(_short(readiness.queue_mode || "-", 24))}</td>
              <td>${escapeHtml(String(Boolean(readiness.agent_job_worker_ready)))}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    ${_renderAgentJobExternalLeaseSteps("Required Checks", _array(plan.required_checks))}
    ${_renderAgentJobExternalLeaseSteps("Enable Steps", _array(plan.enable_steps))}
    ${_renderAgentJobExternalLeaseSteps("Verification Steps", _array(plan.verification_steps))}
    ${_renderAgentJobExternalLeaseSteps("Rollback Steps", _array(plan.rollback_steps))}
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Blockers</div>
      </div>
      <pre class="runtime-overview-json">${escapeHtml(blockers.length ? blockers.join("\n") : "None")}</pre>
    </div>
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">Notes</div>
      </div>
      <pre class="runtime-overview-json">${escapeHtml(notes.length ? notes.join("\n") : "None")}</pre>
    </div>
  `;
}

function _renderRuntimeOverviewNotesSection(
  title: string,
  lines: unknown,
  emptyLabel: string,
): string {
  const values = Array.isArray(lines) ? lines : [];
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div class="detail-label">${escapeHtml(title)}</div>
      </div>
      <pre class="runtime-overview-json">${escapeHtml(values.length ? values.join("\n") : emptyLabel)}</pre>
    </div>
  `;
}

function _renderDeliverySmokeReadinessDetail(detail: Record<string, unknown>): string {
  const readiness = _record(detail.delivery_smoke_readiness || detail);
  const totals = _record(readiness.totals);
  const cases = _array(readiness.cases).slice(0, 20);
  const rows = cases.map((item) => {
    const plan = _record(item.plan);
    const steps = _array(plan.steps);
    const firstStep = _record(steps[0]);
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.name || "-", 40))}</td>
        <td>${escapeHtml(String(Boolean(item.ready)))}</td>
        <td>${escapeHtml(_short(item.reason || "-", 30))}</td>
        <td class="mono">${escapeHtml(_short(plan.channel || "-", 20))}</td>
        <td class="mono">${escapeHtml(_short(plan.chat_id || "-", 18))}</td>
        <td>${escapeHtml(_short(firstStep.conversation_type || "-", 12))}</td>
        <td>${escapeHtml(_short(firstStep.kind || "-", 12))}</td>
        <td>${escapeHtml(String(plan.step_count ?? steps.length ?? 0))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Delivery Smoke Readiness</div>
          <div class="detail-subtext">${escapeHtml(_short(readiness.reason || "-", 44))} · ${escapeHtml(String(readiness.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(Boolean(readiness.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>cases</span><strong>${escapeHtml(String(totals.cases ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>ready cases</span><strong>${escapeHtml(String(totals.ready ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>not ready</span><strong>${escapeHtml(String(totals.not_ready ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Case</th>
              <th>Ready</th>
              <th>Reason</th>
              <th>Channel</th>
              <th>Chat ID</th>
              <th>Conversation</th>
              <th>Kind</th>
              <th>Steps</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No smoke cases sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderRuntimeOverviewNotesSection("Notes", readiness.notes, "None")}
  `;
}

function _renderQueueBackendDetail(detail: Record<string, unknown>): string {
  const backend = _record(detail.queue_backend || detail);
  const providerCaps = _array(backend.provider_capabilities).slice(0, 20);
  const selected = _record(backend.selected_provider_capability);
  const rows = providerCaps.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.provider || "-", 18))}</td>
      <td>${escapeHtml(_short(item.status || "-", 14))}</td>
      <td>${escapeHtml(String(Boolean(item.implemented)))}</td>
      <td>${escapeHtml(String(Boolean(item.recommended)))}</td>
      <td>${escapeHtml(String(Boolean(item.supports_external_lease)))}</td>
      <td>${escapeHtml(String(Boolean(item.supports_agent_job_result_ack)))}</td>
      <td>${escapeHtml(String(Boolean(item.supports_concurrent_consumers)))}</td>
      <td>${escapeHtml(_short(item.consumer_model || "-", 22))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Queue Backend</div>
          <div class="detail-subtext">${escapeHtml(_short(`${String(backend.provider || "-")} / ${String(backend.mode || "-")}`, 40))} · ${escapeHtml(_short(backend.migration_phase || "-", 24))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>provider</span><strong>${escapeHtml(_short(backend.provider || "-", 18))}</strong></div>
        <div class="runtime-overview-kpi"><span>mode</span><strong>${escapeHtml(_short(backend.mode || "-", 22))}</strong></div>
        <div class="runtime-overview-kpi"><span>outbox owner</span><strong>${escapeHtml(_short(backend.outbox_execution_owner || "-", 24))}</strong></div>
        <div class="runtime-overview-kpi"><span>outbox scope</span><strong>${escapeHtml(_short(backend.outbox_execution_scope || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>agent_job owner</span><strong>${escapeHtml(_short(backend.agent_job_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>recommended backend</span><strong>${escapeHtml(_short(backend.recommended_first_backend || "-", 20))}</strong></div>
        <div class="runtime-overview-kpi"><span>selected external lease</span><strong>${escapeHtml(String(Boolean(selected.supports_external_lease)))}</strong></div>
        <div class="runtime-overview-kpi"><span>selected result-ack</span><strong>${escapeHtml(String(Boolean(selected.supports_agent_job_result_ack)))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Provider</th>
              <th>Status</th>
              <th>Implemented</th>
              <th>Recommended</th>
              <th>External Lease</th>
              <th>Result Ack</th>
              <th>Concurrent Consumers</th>
              <th>Consumer Model</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No provider capabilities sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderRuntimeOverviewNotesSection("Notes", backend.notes, "None")}
  `;
}

function _renderReceiverStatusesDetail(detail: Record<string, unknown>): string {
  const statuses = _record(detail.receiver_statuses || detail);
  const totals = _record(statuses.totals);
  const receivers = _array(statuses.receivers).slice(0, 20);
  const rows = receivers.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.receiver_id || "-", 34))}</td>
      <td>${escapeHtml(_short(item.kind || "-", 12))}</td>
      <td class="mono">${escapeHtml(_short(item.channel_name || "-", 18))}</td>
      <td class="mono">${escapeHtml(_short(item.account_id || "-", 18))}</td>
      <td>${_runtimeStatusTag(String(item.status || "muted"))}</td>
      <td>${escapeHtml(_short(item.reason || "-", 20))}</td>
      <td>${escapeHtml(_short(item.endpoint || "-", 36))}</td>
      <td>${escapeHtml(_short(item.updated_at || "-", 28))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Receiver Statuses</div>
          <div class="detail-subtext">${escapeHtml(String(statuses.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>receivers</span><strong>${escapeHtml(String(totals.receivers ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>connected</span><strong>${escapeHtml(String(totals.connected ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>qq</span><strong>${escapeHtml(String(totals.qq ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>telegram</span><strong>${escapeHtml(String(totals.telegram ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>suspended</span><strong>${escapeHtml(String(totals.suspended ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>failed</span><strong>${escapeHtml(String(totals.failed ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Receiver</th>
              <th>Kind</th>
              <th>Channel</th>
              <th>Account</th>
              <th>Status</th>
              <th>Reason</th>
              <th>Endpoint</th>
              <th>Updated</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No receiver statuses sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderRuntimeOverviewNotesSection("Notes", statuses.notes, "None")}
  `;
}

function _renderReceiverLeasesDetail(detail: Record<string, unknown>): string {
  const leasesDetail = _record(detail.receiver_leases || detail);
  const totals = _record(leasesDetail.totals);
  const leases = _array(leasesDetail.leases).slice(0, 20);
  const rows = leases.map((item) => `
    <tr>
      <td class="mono">${escapeHtml(_short(item.receiver_id || "-", 34))}</td>
      <td>${escapeHtml(_short(item.kind || "-", 12))}</td>
      <td class="mono">${escapeHtml(_short(item.account_id || "-", 18))}</td>
      <td class="mono">${escapeHtml(_short(item.owner_instance_id || "-", 28))}</td>
      <td>${escapeHtml(_short(item.status || "-", 16))}</td>
      <td>${escapeHtml(_short(item.expires_at || "-", 28))}</td>
      <td>${escapeHtml(_short(item.updated_at || "-", 28))}</td>
    </tr>
  `).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Receiver Leases</div>
          <div class="detail-subtext">${escapeHtml(String(leasesDetail.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>leases</span><strong>${escapeHtml(String(totals.leases ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>active</span><strong>${escapeHtml(String(totals.active ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>expired</span><strong>${escapeHtml(String(totals.expired ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>qq</span><strong>${escapeHtml(String(totals.qq ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>telegram</span><strong>${escapeHtml(String(totals.telegram ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Receiver</th>
              <th>Kind</th>
              <th>Account</th>
              <th>Owner Instance</th>
              <th>Status</th>
              <th>Expires</th>
              <th>Updated</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="7" class="runtime-overview-muted">No receiver leases sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderRuntimeOverviewNotesSection("Notes", leasesDetail.notes, "None")}
  `;
}

function _renderKnowledgePipelinesDetail(detail: Record<string, unknown>): string {
  const pipelinesDetail = _record(detail.knowledge_pipelines || detail);
  const totals = _record(pipelinesDetail.totals);
  const pipelines = _array(pipelinesDetail.pipelines).slice(0, 20);
  const rows = pipelines.map((item) => {
    const channel = _record(item.channel);
    const groupMemory = _record(item.group_memory);
    const ragIngest = _record(item.rag_ingest);
    const coverage = _array(item.worker_coverage);
    const rawReasons = Array.isArray(item.reasons) ? item.reasons : [];
    const rawCaptureBlockers = Array.isArray(item.capture_blockers) ? item.capture_blockers : [];
    const coverageStatuses = coverage
      .map((coverageItem) => _short(coverageItem.coverage_status || coverageItem.coverage_reason || "-", 18))
      .filter(Boolean)
      .join(", ");
    const reasons = [
      ...rawReasons.map((reason) => _short(reason, 24)),
      ...rawCaptureBlockers.map((reason) => _short(reason, 24)),
    ].filter(Boolean).join(", ");
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.target_id || "-", 34))}</td>
        <td class="mono">${escapeHtml(_short(channel.account_id || "-", 18))}</td>
        <td>${_runtimeStatusTag(String(item.capture_status || "muted"))}</td>
        <td>${_runtimeStatusTag(String(groupMemory.freshness_status || "muted"))}</td>
        <td>${_runtimeStatusTag(String(ragIngest.freshness_status || "muted"))}</td>
        <td>${escapeHtml(_short(coverageStatuses || "-", 24))}</td>
        <td>${escapeHtml(String(item.latest_source_seq ?? "-"))}</td>
        <td>${escapeHtml(_short(reasons || "-", 56))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Knowledge Pipelines</div>
          <div class="detail-subtext">${escapeHtml(String(pipelinesDetail.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>targets</span><strong>${escapeHtml(String(totals.targets ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>enabled</span><strong>${escapeHtml(String(totals.enabled ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(totals.ready ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>warning</span><strong>${escapeHtml(String(totals.warning ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>blocked</span><strong>${escapeHtml(String(totals.blocked ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>lagging</span><strong>${escapeHtml(String(totals.lagging ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>stale checkpoints</span><strong>${escapeHtml(String(totals.stale_checkpoints ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>receiver connected</span><strong>${escapeHtml(String(totals.receiver_connected ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Target</th>
              <th>Account</th>
              <th>Capture Status</th>
              <th>Group Memory</th>
              <th>RAG Ingest</th>
              <th>Coverage</th>
              <th>Latest Source Seq</th>
              <th>Reasons</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="8" class="runtime-overview-muted">No pipeline targets sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderRuntimeOverviewNotesSection("Notes", pipelinesDetail.notes, "None")}
  `;
}

function _renderAgentJobCapacityPlanDetail(detail: Record<string, unknown>): string {
  const plan = _record(detail.agent_job_capacity_plan || detail);
  const summary = _record(plan.summary);
  const items = _array(plan.items).slice(0, 20);
  const rows = items.map((item) => {
    const coverage = _record(item.coverage);
    return `
      <tr>
        <td class="mono">${escapeHtml(_short(item.job_type || "-", 28))}</td>
        <td>${_runtimeStatusTag(String(item.severity || "muted"))}</td>
        <td>${escapeHtml(_short(item.action || "-", 22))}</td>
        <td>${escapeHtml(String(item.pending ?? 0))}</td>
        <td>${escapeHtml(String(item.active ?? 0))}</td>
        <td>${escapeHtml(String(item.oldest_pending_age_seconds ?? 0))}</td>
        <td>${escapeHtml(String(Boolean(item.high_pressure)))}</td>
        <td>${_runtimeStatusTag(String(coverage.coverage_status || "muted"))}</td>
        <td>${escapeHtml(_short(item.recommendation || "-", 56))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Job Capacity</div>
          <div class="detail-subtext">${escapeHtml(_short(plan.reason || "-", 44))} · ${escapeHtml(String(plan.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(Boolean(plan.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>job types</span><strong>${escapeHtml(String(summary.job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>mapped</span><strong>${escapeHtml(String(summary.mapped_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>high pressure</span><strong>${escapeHtml(String(summary.high_pressure_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>blocked</span><strong>${escapeHtml(String(summary.capacity_blocked_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>worker warnings</span><strong>${escapeHtml(String(summary.worker_warning_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>max pending</span><strong>${escapeHtml(String(summary.max_pending ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>oldest pending(s)</span><strong>${escapeHtml(String(summary.oldest_pending_age_seconds ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Job Type</th>
              <th>Severity</th>
              <th>Action</th>
              <th>Pending</th>
              <th>Active</th>
              <th>Oldest Pending</th>
              <th>High Pressure</th>
              <th>Coverage</th>
              <th>Recommendation</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="9" class="runtime-overview-muted">No capacity items sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderAgentJobExternalLeaseSteps("Verification Steps", _array(plan.verification_steps))}
    ${_renderRuntimeOverviewNotesSection("Notes", plan.notes, "None")}
  `;
}

function _renderAgentJobPriorityPlanDetail(detail: Record<string, unknown>): string {
  const plan = _record(detail.agent_job_priority_plan || detail);
  const summary = _record(plan.summary);
  const items = _array(plan.items).slice(0, 20);
  const rows = items.map((item) => {
    const coverage = _record(item.coverage);
    return `
      <tr>
        <td>${escapeHtml(String(item.rank ?? "-"))}</td>
        <td class="mono">${escapeHtml(_short(item.job_type || "-", 28))}</td>
        <td>${escapeHtml(_short(item.priority_class || "-", 18))}</td>
        <td>${escapeHtml(String(item.priority_score ?? 0))}</td>
        <td>${escapeHtml(_short(item.action || "-", 22))}</td>
        <td>${escapeHtml(String(item.pending ?? 0))}</td>
        <td>${escapeHtml(String(item.active ?? 0))}</td>
        <td>${_runtimeStatusTag(String(coverage.coverage_status || "muted"))}</td>
        <td>${escapeHtml(_short(item.recommendation || "-", 56))}</td>
      </tr>
    `;
  }).join("");
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Agent Job Priority</div>
          <div class="detail-subtext">${escapeHtml(_short(plan.reason || "-", 44))} · ${escapeHtml(String(plan.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(Boolean(plan.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>job types</span><strong>${escapeHtml(String(summary.job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>high priority</span><strong>${escapeHtml(String(summary.high_priority_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>blocked</span><strong>${escapeHtml(String(summary.blocked_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>warning</span><strong>${escapeHtml(String(summary.warning_job_types ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>max score</span><strong>${escapeHtml(String(summary.max_priority_score ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Rank</th>
              <th>Job Type</th>
              <th>Priority Class</th>
              <th>Score</th>
              <th>Action</th>
              <th>Pending</th>
              <th>Active</th>
              <th>Coverage</th>
              <th>Recommendation</th>
            </tr>
          </thead>
          <tbody>${rows || `<tr><td colspan="9" class="runtime-overview-muted">No priority items sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
    ${_renderAgentJobExternalLeaseSteps("Verification Steps", _array(plan.verification_steps))}
    ${_renderRuntimeOverviewNotesSection("Notes", plan.notes, "None")}
  `;
}

function _renderKnowledgePlannerCutoverPlanDetail(detail: Record<string, unknown>): string {
  const plan = _record(detail.knowledge_job_planner_cutover_plan || detail);
  const readiness = _record(plan.readiness);
  const preview = _record(readiness.preview);
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Knowledge Planner Cutover</div>
          <div class="detail-subtext">${escapeHtml(_short(plan.decision || "-", 24))} · ${escapeHtml(String(plan.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(Boolean(plan.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>current owner</span><strong>${escapeHtml(_short(plan.current_admission_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>desired owner</span><strong>${escapeHtml(_short(plan.desired_admission_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>recommended owner</span><strong>${escapeHtml(_short(plan.recommended_admission_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>planner running</span><strong>${escapeHtml(String(Boolean(readiness.planner_running)))}</strong></div>
        <div class="runtime-overview-kpi"><span>worker ready</span><strong>${escapeHtml(String(Boolean(readiness.knowledge_worker_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>preview jobs</span><strong>${escapeHtml(String(preview.total_jobs ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>targets</span><strong>${escapeHtml(String(preview.targets ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Current Owner</th>
              <th>Desired Owner</th>
              <th>Recommended Owner</th>
              <th>Planner Enabled</th>
              <th>Planner Running</th>
              <th>Worker Ready</th>
              <th>Preview Bucket</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>${escapeHtml(_short(plan.current_admission_owner || "-", 40))}</td>
              <td>${escapeHtml(_short(plan.desired_admission_owner || "-", 40))}</td>
              <td>${escapeHtml(_short(plan.recommended_admission_owner || "-", 40))}</td>
              <td>${escapeHtml(String(Boolean(readiness.planner_enabled)))}</td>
              <td>${escapeHtml(String(Boolean(readiness.planner_running)))}</td>
              <td>${escapeHtml(String(Boolean(readiness.knowledge_worker_ready)))}</td>
              <td>${escapeHtml(String(preview.bucket ?? "-"))}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    ${_renderAgentJobExternalLeaseSteps("Required Checks", _array(plan.required_checks))}
    ${_renderAgentJobExternalLeaseSteps("Enable Steps", _array(plan.enable_steps))}
    ${_renderAgentJobExternalLeaseSteps("Verification Steps", _array(plan.verification_steps))}
    ${_renderAgentJobExternalLeaseSteps("Rollback Steps", _array(plan.rollback_steps))}
    ${_renderRuntimeOverviewNotesSection("Notes", plan.notes, "None")}
  `;
}

function _renderOutboundCutoverPlanDetail(detail: Record<string, unknown>): string {
  const plan = _record(detail.outbound_cutover_plan || detail);
  const readiness = _record(plan.readiness);
  const smokeReadiness = _record(readiness.delivery_smoke_readiness);
  const totals = _record(smokeReadiness.totals);
  return `
    <div class="runtime-overview-section">
      <div class="runtime-overview-section-header">
        <div>
          <div class="detail-label">Outbound Cutover Plan</div>
          <div class="detail-subtext">${escapeHtml(_short(plan.decision || "-", 24))} · ${escapeHtml(String(plan.side_effect || "none"))}</div>
        </div>
      </div>
      <div class="runtime-overview-kpis">
        <div class="runtime-overview-kpi"><span>ready</span><strong>${escapeHtml(String(Boolean(plan.ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>current owner</span><strong>${escapeHtml(_short(plan.current_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>desired owner</span><strong>${escapeHtml(_short(plan.desired_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>recommended owner</span><strong>${escapeHtml(_short(plan.recommended_execution_owner || "-", 28))}</strong></div>
        <div class="runtime-overview-kpi"><span>execution ready</span><strong>${escapeHtml(String(Boolean(readiness.execution_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>smoke ready</span><strong>${escapeHtml(String(Boolean(readiness.smoke_ready)))}</strong></div>
        <div class="runtime-overview-kpi"><span>smoke cases</span><strong>${escapeHtml(String(totals.cases ?? 0))}</strong></div>
        <div class="runtime-overview-kpi"><span>not ready</span><strong>${escapeHtml(String(totals.not_ready ?? 0))}</strong></div>
      </div>
      <div class="runtime-overview-table-wrap">
        <table class="runtime-overview-table">
          <thead>
            <tr>
              <th>Current Owner</th>
              <th>Desired Owner</th>
              <th>Recommended Owner</th>
              <th>Execution Ready</th>
              <th>Smoke Ready</th>
              <th>Expected OneBot Channels</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>${escapeHtml(_short(plan.current_execution_owner || "-", 40))}</td>
              <td>${escapeHtml(_short(plan.desired_execution_owner || "-", 40))}</td>
              <td>${escapeHtml(_short(plan.recommended_execution_owner || "-", 40))}</td>
              <td>${escapeHtml(String(Boolean(readiness.execution_ready)))}</td>
              <td>${escapeHtml(String(Boolean(readiness.smoke_ready)))}</td>
              <td>${escapeHtml(Array.isArray(readiness.expected_onebot_channels) ? readiness.expected_onebot_channels.join(", ") : "-")}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    ${_renderAgentJobExternalLeaseSteps("Required Checks", _array(plan.required_checks))}
    ${_renderAgentJobExternalLeaseSteps("Enable Steps", _array(plan.enable_steps))}
    ${_renderAgentJobExternalLeaseSteps("Verification Steps", _array(plan.verification_steps))}
    ${_renderAgentJobExternalLeaseSteps("Rollback Steps", _array(plan.rollback_steps))}
    ${_renderRuntimeOverviewNotesSection("Notes", plan.notes, "None")}
  `;
}

window.AkashicDashboard.registerPlugin({
  id: "runtime_overview",
  label: "Runtime Overview",
  viewLabel: "runtime overview",
  pageSize: 25,
  rowKey: "id",
  defaultSortBy: "label",
  defaultSortOrder: "asc",

  countTitle(total: number): string {
    return `${total} 个运行指标`;
  },

  columns: [
    { key: "label", label: "Metric", flex: true, cellClass: "cell-type", rawTitle: true },
    { key: "value", label: "Value", width: 120, cellClass: "mono cell-metric", align: "right" },
    { key: "status", label: "Status", width: 120, renderCell: (value) => _runtimeStatusTag(String(value || "")) },
    { key: "detail", label: "Detail", width: 260, renderCell: (value) => escapeHtml(_short(_formatJson(value), 96)), cellClass: "content-preview" },
  ],

  async getCount(): Promise<number | null> {
    const payload = await api<RuntimeOverviewResponse>("/api/dashboard/runtime-overview?limit=1&event_limit=1");
    return payload.cards?.length || 0;
  },

  async fetchPage({ page, pageSize }: FetchPageOpts): Promise<FetchPageResult> {
    const payload = await api<RuntimeOverviewResponse>("/api/dashboard/runtime-overview");
    const cards = payload.cards || [];
    const start = Math.max(0, (page - 1) * pageSize);
    return {
      items: cards.slice(start, start + pageSize) as unknown as Record<string, unknown>[],
      total: cards.length,
    };
  },

  fetchDetail(item: Record<string, unknown>): Promise<Record<string, unknown>> {
    return Promise.resolve(item);
  },

  renderDetail(item: Record<string, unknown> | null, container: HTMLElement): void {
    if (!item) {
      container.innerHTML = `
        <div class="runtime-overview-empty">
          <div class="detail-title">Runtime Overview</div>
          <div class="detail-subtext">Go agent-runtime 的健康状态、任务租约、死信、游标延迟和事件流摘要。</div>
        </div>
      `;
      return;
    }
    const card = item as unknown as RuntimeOverviewCard;
    const adapterHealthSection = card.id === "delivery_adapters"
      ? `
        <div class="runtime-overview-section">
          <div class="runtime-overview-section-header">
            <div class="detail-label">Adapter Health</div>
            <button class="runtime-overview-action" type="button" data-runtime-adapter-health>
              Probe Health
            </button>
          </div>
          <pre class="runtime-overview-json" data-runtime-adapter-health-output>${escapeHtml("Not checked")}</pre>
        </div>
        <div class="runtime-overview-section">
          <div class="runtime-overview-section-header">
            <div class="detail-label">Delivery Smoke</div>
            <div class="runtime-overview-actions">
              <input class="runtime-overview-input" type="text" data-runtime-delivery-smoke-groups placeholder="Group IDs" />
              <button class="runtime-overview-action" type="button" data-runtime-delivery-smoke>
                Smoke Readiness
              </button>
            </div>
          </div>
          <pre class="runtime-overview-json" data-runtime-delivery-smoke-output>${escapeHtml("Not checked")}</pre>
        </div>
      `
      : "";
    const specializedDetail = card.id === "delivery_smoke"
        ? _renderDeliverySmokeReadinessDetail(_record(card.detail))
      : card.id === "delivery_adapters"
        ? _renderDeliveryAdaptersDetail(_record(card.detail))
      : card.id === "queue_backend"
        ? _renderQueueBackendDetail(_record(card.detail))
      : card.id === "receiver_statuses"
        ? _renderReceiverStatusesDetail(_record(card.detail))
      : card.id === "receiver_leases"
        ? _renderReceiverLeasesDetail(_record(card.detail))
      : card.id === "knowledge_pipelines"
        ? _renderKnowledgePipelinesDetail(_record(card.detail))
      : card.id === "media_asset_retention_plan"
        ? _renderMediaAssetRetentionPlanDetail(_record(card.detail))
      : card.id === "media_asset_retention_cleanup"
        ? _renderMediaAssetRetentionCleanupDetail(_record(card.detail))
      : card.id === "dead_letters"
        ? _renderDeadLettersDetail(_record(card.detail))
      : card.id === "checkpoint_lag"
        ? _renderCheckpointLagDetail(_record(card.detail))
      : card.id === "job_events"
        ? _renderJobEventsDetail(_record(card.detail))
      : card.id === "outbox_events"
        ? _renderOutboxEventsDetail(_record(card.detail))
      : card.id === "rag_eval_failures"
        ? _renderRagEvalFailuresDetail(_record(card.detail))
      : card.id === "runtime_health"
        ? _renderRuntimeHealthDetail(_record(card.detail))
      : card.id === "stale_jobs"
        ? _renderStaleJobsDetail(_record(card.detail))
      : card.id === "knowledge_job_planner_preview"
        ? _renderKnowledgeJobPlannerPreviewDetail(_record(card.detail))
      : card.id === "knowledge_job_planner_readiness"
        ? _renderKnowledgeJobPlannerReadinessDetail(_record(card.detail))
      : card.id === "media_asset_content"
        ? _renderMediaAssetContentDetail(_record(card.detail))
      : card.id === "media_asset_content_recovery"
        ? _renderMediaAssetContentRecoveryDetail(_record(card.detail))
      : card.id === "media_asset_retention"
        ? _renderMediaAssetRetentionDetail(_record(card.detail))
      : card.id === "worker_leases"
        ? _renderWorkerLeasesDetail(_record(card.detail))
      : card.id === "queue_topology"
        ? _renderQueueTopologyDetail(_record(card.detail))
      : card.id === "qq_cutover_route_matrix"
        ? _renderQqCutoverRouteMatrixDetail(_record(card.detail))
      : card.id === "external_lease_diagnostics"
        ? _renderExternalLeaseDiagnosticsDetail(_record(card.detail))
      : card.id === "scheduler_jobs"
        ? _renderSchedulerJobsDetail(_record(card.detail))
      : card.id === "agent_workers"
        ? _renderAgentWorkersDetail(_record(card.detail))
      : card.id === "runtime_workers"
        ? _renderRuntimeWorkersDetail(_record(card.detail))
      : card.id === "observe_targets"
        ? _renderObserveTargetsDetail(_record(card.detail))
      : card.id === "observe_capture"
        ? _renderObserveCaptureDetail(_record(card.detail))
      : card.id === "agent_job_worker_coverage"
        ? _renderAgentJobWorkerCoverageDetail(_record(card.detail))
      : card.id === "agent_job_metrics"
        ? _renderAgentJobMetricsDetail(_record(card.detail))
      : card.id === "agent_job_pressure"
        ? _renderAgentJobPressureDetail(_record(card.detail))
      : card.id === "send_ledger_metrics"
        ? _renderSendLedgerMetricsDetail(_record(card.detail))
      : card.id === "inbox_metrics"
        ? _renderInboxMetricsDetail(_record(card.detail))
      : card.id === "inbound_dedupe_metrics"
        ? _renderInboundDedupeMetricsDetail(_record(card.detail))
      : card.id === "outbox_metrics"
        ? _renderOutboxMetricsDetail(_record(card.detail))
      : card.id === "outbox_pressure"
        ? _renderOutboxPressureDetail(_record(card.detail))
      : card.id === "agent_job_external_lease_readiness"
        ? _renderAgentJobExternalLeaseReadinessDetail(_record(card.detail))
      : card.id === "agent_job_external_lease_plan"
        ? _renderAgentJobExternalLeasePlanDetail(_record(card.detail))
      : card.id === "agent_job_capacity_plan"
        ? _renderAgentJobCapacityPlanDetail(_record(card.detail))
      : card.id === "agent_job_priority_plan"
        ? _renderAgentJobPriorityPlanDetail(_record(card.detail))
      : card.id === "knowledge_job_planner_cutover_plan"
        ? _renderKnowledgePlannerCutoverPlanDetail(_record(card.detail))
      : card.id === "outbound_cutover_plan"
        ? _renderOutboundCutoverPlanDetail(_record(card.detail))
      : card.id === "control_audit"
        ? _renderControlAuditDetail(_record(card.detail))
      : card.id === "control_mutation_policy"
        ? _renderControlMutationPolicyDetail(_record(card.detail))
      : card.id === "runtime_config"
        ? _renderRuntimeConfigDetail(_record(card.detail))
      : "";
    container.innerHTML = `
      <div class="runtime-overview-detail">
        <div class="runtime-overview-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(card.label || "-")}</div>
            <div class="detail-subtext">${_runtimeStatusTag(card.status || "muted")} · ${escapeHtml(String(card.value ?? "-"))}</div>
          </div>
        </div>
        ${specializedDetail}
        <div class="runtime-overview-section">
          <div class="detail-label">Detail</div>
          <pre class="runtime-overview-json">${escapeHtml(_formatJson(card.detail || {}))}</pre>
        </div>
        ${adapterHealthSection}
      </div>
    `;
    const button = container.querySelector<HTMLButtonElement>("[data-runtime-adapter-health]");
    const output = container.querySelector<HTMLElement>("[data-runtime-adapter-health-output]");
    if (button && output) {
      button.addEventListener("click", async () => {
        button.disabled = true;
        output.textContent = "Checking...";
        try {
          const payload = await api<DeliveryAdapterHealthResponse>(
            "/api/dashboard/runtime-overview/delivery-adapter-health?timeout_seconds=3",
          );
          output.textContent = _formatJson(payload);
        } catch (error) {
          output.textContent = _formatJson({
            error: error instanceof Error ? error.message : String(error),
            side_effect: "none",
          });
        } finally {
          button.disabled = false;
        }
      });
    }
    const smokeButton = container.querySelector<HTMLButtonElement>("[data-runtime-delivery-smoke]");
    const smokeOutput = container.querySelector<HTMLElement>("[data-runtime-delivery-smoke-output]");
    const smokeGroups = container.querySelector<HTMLInputElement>("[data-runtime-delivery-smoke-groups]");
    if (smokeButton && smokeOutput) {
      smokeButton.addEventListener("click", async () => {
        smokeButton.disabled = true;
        smokeOutput.textContent = "Checking...";
        try {
          const groups = encodeURIComponent(smokeGroups?.value || "");
          const payload = await api<DeliverySmokeReadinessResponse>(
            `/api/dashboard/runtime-overview/delivery-smoke-readiness?include_synthetic_media=true&group_ids=${groups}`,
          );
          smokeOutput.textContent = _formatJson(payload);
        } catch (error) {
          smokeOutput.textContent = _formatJson({
            error: error instanceof Error ? error.message : String(error),
            side_effect: "none",
          });
        } finally {
          smokeButton.disabled = false;
        }
      });
    }
  },
});

export {};
