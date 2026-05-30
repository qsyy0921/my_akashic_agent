"use strict";
(() => {
  function _short(value, limit) {
    const text = String(value ?? "").replace(/\s+/g, " ").trim();
    return text.length > limit ? `${text.slice(0, limit)}...` : text;
  }
  function _formatJson(value) {
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value ?? "");
    }
  }
  function _renderFilters(container, dispatch) {
    container.innerHTML = `
    <div class="knowledge-checkpoint-filter-row">
      <label class="knowledge-checkpoint-filter">
        <span>\u641C\u7D22</span>
        <input data-knowledge-checkpoint-filter="q" value="${escapeHtml(dispatch.filters["q"] || "")}" placeholder="checkpoint / group / dataset" />
      </label>
      <label class="knowledge-checkpoint-filter">
        <span>\u524D\u7F00</span>
        <input data-knowledge-checkpoint-filter="prefix" value="${escapeHtml(dispatch.filters["prefix"] || "ragflow:qq:")}" placeholder="ragflow:qq:" />
      </label>
      <label class="knowledge-checkpoint-filter">
        <span>\u7FA4\u53F7</span>
        <input data-knowledge-checkpoint-filter="group_id" value="${escapeHtml(dispatch.filters["group_id"] || "")}" placeholder="27234224" />
      </label>
      <label class="knowledge-checkpoint-filter">
        <span>Dataset</span>
        <input data-knowledge-checkpoint-filter="dataset_id" value="${escapeHtml(dispatch.filters["dataset_id"] || "")}" placeholder="RAGFlow dataset id" />
      </label>
      <button class="ghost" type="button" data-knowledge-checkpoint-clear>\u6E05\u7A7A</button>
    </div>
  `;
    container.querySelectorAll("[data-knowledge-checkpoint-filter]").forEach((input) => {
      input.addEventListener("change", () => {
        const element = input;
        const key = element.getAttribute("data-knowledge-checkpoint-filter");
        if (!key) return;
        dispatch.setFilter(key, element.value.trim());
      });
    });
    container.querySelector("[data-knowledge-checkpoint-clear]")?.addEventListener("click", () => {
      dispatch.setFilters({
        q: "",
        prefix: "",
        group_id: "",
        dataset_id: ""
      });
    });
  }
  window.AkashicDashboard.registerPlugin({
    id: "knowledge_checkpoints",
    label: "Knowledge Checkpoints",
    viewLabel: "knowledge checkpoints",
    pageSize: 25,
    rowKey: "checkpoint_id",
    defaultSortBy: "updated_at",
    defaultSortOrder: "desc",
    countTitle(total) {
      return `${total} \u4E2A\u77E5\u8BC6\u6E38\u6807`;
    },
    columns: [
      { key: "checkpoint_id", label: "Checkpoint", flex: true, cellClass: "mono cell-id", rawTitle: true, renderCell: (value) => escapeHtml(_short(value, 42)) },
      { key: "cursor", label: "Cursor", width: 96, align: "right", sortable: true, fmt: "knowledge-cursor", cellClass: "mono cell-metric" },
      { key: "group_id", label: "Group", width: 120, cellClass: "mono cell-id", rawTitle: true },
      { key: "dataset_id", label: "Dataset", width: 180, cellClass: "mono cell-id", rawTitle: true, renderCell: (value) => escapeHtml(_short(value, 24)) },
      { key: "source", label: "Source", width: 86, cellClass: "cell-type" },
      { key: "updated_at", label: "Updated", width: 170, fmt: "mono-time", cellClass: "mono cell-time", sortable: true }
    ],
    async getCount() {
      const data = await api("/api/dashboard/knowledge-checkpoints?page_size=1&prefix=ragflow%3Aqq%3A");
      return data.total || 0;
    },
    async fetchPage({ page, pageSize, filters, sortBy, sortOrder }) {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      params.set("sort_by", sortBy || "updated_at");
      params.set("sort_order", sortOrder || "desc");
      params.set("prefix", filters?.prefix ?? "ragflow:qq:");
      if (filters?.q) params.set("q", filters.q);
      if (filters?.group_id) params.set("group_id", filters.group_id);
      if (filters?.dataset_id) params.set("dataset_id", filters.dataset_id);
      const payload = await api(`/api/dashboard/knowledge-checkpoints?${params.toString()}`);
      return {
        items: payload.items || [],
        total: payload.total || 0
      };
    },
    fetchDetail(item) {
      const checkpointId = String(item.checkpoint_id || "");
      if (!checkpointId) return Promise.resolve(item);
      return api(`/api/dashboard/knowledge-checkpoints/${encodePath(checkpointId)}`).catch(() => item);
    },
    renderFilters: _renderFilters,
    renderDetail(item, container) {
      if (!item) {
        container.innerHTML = `
        <div class="knowledge-checkpoint-empty">
          <div class="detail-title">Knowledge Checkpoint</div>
          <div class="detail-subtext">\u67E5\u770B Go agent-runtime \u6301\u4E45\u5316\u7684 RAG / memory ingest \u6E38\u6807\u3002</div>
        </div>
      `;
        return;
      }
      const checkpoint = item;
      container.innerHTML = `
      <div class="knowledge-checkpoint-detail">
        <div class="knowledge-checkpoint-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(checkpoint.checkpoint_id || "-")}</div>
            <div class="detail-subtext">${escapeHtml(checkpoint.kind || "knowledge")} \xB7 cursor ${escapeHtml(String(checkpoint.cursor ?? -1))}</div>
          </div>
        </div>

        <div class="knowledge-checkpoint-section">
          <div class="knowledge-checkpoint-grid">
            <div>
              <div class="knowledge-checkpoint-label">Source</div>
              <div class="knowledge-checkpoint-value mono">${escapeHtml(checkpoint.source || "-")}</div>
            </div>
            <div>
              <div class="knowledge-checkpoint-label">Group</div>
              <div class="knowledge-checkpoint-value mono">${escapeHtml(checkpoint.group_id || "-")}</div>
            </div>
            <div>
              <div class="knowledge-checkpoint-label">Dataset</div>
              <div class="knowledge-checkpoint-value mono">${escapeHtml(checkpoint.dataset_id || "-")}</div>
            </div>
            <div>
              <div class="knowledge-checkpoint-label">Updated</div>
              <div class="knowledge-checkpoint-value mono">${escapeHtml(checkpoint.updated_at || "-")}</div>
            </div>
          </div>
        </div>

        <div class="knowledge-checkpoint-section">
          <div class="detail-label">Metadata</div>
          <pre class="knowledge-checkpoint-json">${escapeHtml(_formatJson(checkpoint.metadata || {}))}</pre>
        </div>
      </div>
    `;
    },
    formatters: {
      "knowledge-cursor": (value) => escapeHtml(String(value ?? -1)),
      "mono-time": (value) => escapeHtml(String(value || "-"))
    }
  });
})();
