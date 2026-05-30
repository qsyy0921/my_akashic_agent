"use strict";
(() => {
  function _runtimeStatusTag(status) {
    const cls = `runtime-overview-status runtime-overview-${status || "muted"}`;
    return `<span class="${cls}">${escapeHtml(status || "muted")}</span>`;
  }
  function _formatJson(value) {
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value ?? "");
    }
  }
  function _short(value, limit) {
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
    countTitle(total) {
      return `${total} \u4E2A\u8FD0\u884C\u6307\u6807`;
    },
    columns: [
      { key: "label", label: "Metric", flex: true, cellClass: "cell-type", rawTitle: true },
      { key: "value", label: "Value", width: 120, cellClass: "mono cell-metric", align: "right" },
      { key: "status", label: "Status", width: 120, renderCell: (value) => _runtimeStatusTag(String(value || "")) },
      { key: "detail", label: "Detail", width: 260, renderCell: (value) => escapeHtml(_short(_formatJson(value), 96)), cellClass: "content-preview" }
    ],
    async getCount() {
      const payload = await api("/api/dashboard/runtime-overview?limit=1&event_limit=1");
      return payload.cards?.length || 0;
    },
    async fetchPage({ page, pageSize }) {
      const payload = await api("/api/dashboard/runtime-overview");
      const cards = payload.cards || [];
      const start = Math.max(0, (page - 1) * pageSize);
      return {
        items: cards.slice(start, start + pageSize),
        total: cards.length
      };
    },
    fetchDetail(item) {
      return Promise.resolve(item);
    },
    renderDetail(item, container) {
      if (!item) {
        container.innerHTML = `
        <div class="runtime-overview-empty">
          <div class="detail-title">Runtime Overview</div>
          <div class="detail-subtext">Go agent-runtime \u7684\u5065\u5EB7\u72B6\u6001\u3001\u4EFB\u52A1\u79DF\u7EA6\u3001\u6B7B\u4FE1\u3001\u6E38\u6807\u5EF6\u8FDF\u548C\u4E8B\u4EF6\u6D41\u6458\u8981\u3002</div>
        </div>
      `;
        return;
      }
      const card = item;
      const adapterHealthSection = card.id === "delivery_adapters" ? `
        <div class="runtime-overview-section">
          <div class="runtime-overview-section-header">
            <div class="detail-label">Adapter Health</div>
            <button class="runtime-overview-action" type="button" data-runtime-adapter-health>
              Probe Health
            </button>
          </div>
          <pre class="runtime-overview-json" data-runtime-adapter-health-output>${escapeHtml("Not checked")}</pre>
        </div>
      ` : "";
      container.innerHTML = `
      <div class="runtime-overview-detail">
        <div class="runtime-overview-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(card.label || "-")}</div>
            <div class="detail-subtext">${_runtimeStatusTag(card.status || "muted")} \xB7 ${escapeHtml(String(card.value ?? "-"))}</div>
          </div>
        </div>
        <div class="runtime-overview-section">
          <div class="detail-label">Detail</div>
          <pre class="runtime-overview-json">${escapeHtml(_formatJson(card.detail || {}))}</pre>
        </div>
        ${adapterHealthSection}
      </div>
    `;
      const button = container.querySelector("[data-runtime-adapter-health]");
      const output = container.querySelector("[data-runtime-adapter-health-output]");
      if (button && output) {
        button.addEventListener("click", async () => {
          button.disabled = true;
          output.textContent = "Checking...";
          try {
            const payload = await api(
              "/api/dashboard/runtime-overview/delivery-adapter-health?timeout_seconds=3"
            );
            output.textContent = _formatJson(payload);
          } catch (error) {
            output.textContent = _formatJson({
              error: error instanceof Error ? error.message : String(error),
              side_effect: "none"
            });
          } finally {
            button.disabled = false;
          }
        });
      }
    }
  });
})();
