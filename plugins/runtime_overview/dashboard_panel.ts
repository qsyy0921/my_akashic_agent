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
    container.innerHTML = `
      <div class="runtime-overview-detail">
        <div class="runtime-overview-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(card.label || "-")}</div>
            <div class="detail-subtext">${_runtimeStatusTag(card.status || "muted")} · ${escapeHtml(String(card.value ?? "-"))}</div>
          </div>
        </div>
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
