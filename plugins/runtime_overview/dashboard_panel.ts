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
          <tbody>${rows || `<tr><td colspan="6" class="runtime-overview-muted">No work kinds sampled</td></tr>`}</tbody>
        </table>
      </div>
    </div>
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
    const specializedDetail = card.id === "media_asset_content"
      ? _renderMediaAssetContentDetail(_record(card.detail))
      : card.id === "queue_topology"
        ? _renderQueueTopologyDetail(_record(card.detail))
      : card.id === "control_audit"
        ? _renderControlAuditDetail(_record(card.detail))
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
