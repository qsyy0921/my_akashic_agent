"use strict";
(() => {
  function _jobRoute(item) {
    const route = item.route || {};
    const kind = route.kind || item.route_kind || "-";
    const account = route.account_id || item.route_account_id || "-";
    const conv = route.conversation_id || item.route_conversation_id || "-";
    const type = route.conversation_type || item.route_conversation_type || "-";
    return `${kind}:${account}:${type}:${conv}`;
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
  function _jobStatusTag(status) {
    const cls = `agent-job-status agent-status-${status || "unknown"}`;
    const text = status || "unknown";
    return `<span class="${cls}">${escapeHtml(text)}</span>`;
  }
  function _renderFilters(container, dispatch) {
    container.innerHTML = `
    <div class="agent-job-filter-row">
      <label class="agent-job-filter">
        <span>\u641C\u7D22</span>
        <input data-agent-job-filter="q" value="${escapeHtml(dispatch.filters["q"] || "")}" placeholder="job_id / route / payload / metadata" />
      </label>
      <label class="agent-job-filter">
        <span>\u7C7B\u578B</span>
        <input data-agent-job-filter="job_type" value="${escapeHtml(dispatch.filters["job_type"] || "")}" placeholder="\u4F8B\u5982 rag_ingest" />
      </label>
      <label class="agent-job-filter">
        <span>\u72B6\u6001</span>
        <input data-agent-job-filter="status" value="${escapeHtml(dispatch.filters["status"] || "")}" placeholder="pending / running" />
      </label>
      <label class="agent-job-filter">
        <span>\u4F1A\u8BDD</span>
        <input data-agent-job-filter="route_conversation_id" value="${escapeHtml(dispatch.filters["route_conversation_id"] || "")}" placeholder="conversation_id" />
      </label>
      <button class="ghost" type="button" data-agent-job-clear>\u6E05\u7A7A</button>
    </div>
  `;
    container.querySelectorAll("[data-agent-job-filter]").forEach((input) => {
      input.addEventListener("change", () => {
        const element = input;
        const key = element.getAttribute("data-agent-job-filter");
        if (!key) return;
        dispatch.setFilter(key, element.value.trim());
      });
    });
    container.querySelector("[data-agent-job-clear]")?.addEventListener("click", () => {
      dispatch.setFilters({
        q: "",
        job_type: "",
        status: "",
        route_conversation_id: "",
        route_account_id: "",
        route_kind: ""
      });
    });
  }
  async function _runJobAction(jobId, action) {
    await api(`/api/dashboard/agent-jobs/${encodePath(jobId)}/${action}`, {
      method: "POST"
    });
  }
  function _scheduleRefresh() {
    window.dispatchEvent(new CustomEvent("akashic-dashboard-refresh"));
  }
  async function _retryBatch(ids) {
    for (const jobId of ids) {
      await _runJobAction(jobId, "retry");
    }
    _scheduleRefresh();
  }
  async function _cancelBatch(ids) {
    for (const jobId of ids) {
      await _runJobAction(jobId, "cancel");
    }
    _scheduleRefresh();
  }
  window.AkashicDashboard.registerPlugin({
    id: "agent_jobs",
    label: "Agent Jobs",
    viewLabel: "agent jobs",
    pageSize: 25,
    rowKey: "job_id",
    defaultSortBy: "updated_at",
    defaultSortOrder: "desc",
    countTitle(total) {
      return `${total} \u4E2A\u4EFB\u52A1`;
    },
    columns: [
      { key: "job_id", label: "Job ID", width: 220, cellClass: "mono cell-id", rawTitle: true },
      { key: "job_type", label: "Type", width: 150, cellClass: "cell-type", rawTitle: true },
      { key: "status", label: "Status", width: 120, cellClass: "cell-status", renderCell: (v) => _jobStatusTag(String(v || "")) },
      { key: "route", label: "Route", flex: true, renderCell: (_value, row) => escapeHtml(_jobRoute(row)) },
      { key: "attempts", label: "Attempts", width: 88, fmt: "agent-metric", cellClass: "mono cell-metric", align: "right", sortable: true },
      { key: "max_attempts", label: "Max", width: 60, fmt: "agent-metric", cellClass: "mono cell-metric", align: "right", sortable: true },
      { key: "lease_owner", label: "Lease", width: 112, cellClass: "mono cell-owner", rawTitle: true },
      { key: "error_message", label: "Error", width: 150, renderCell: (value) => _short(value, 64), cellClass: "content-preview" },
      { key: "updated_at", label: "Updated", width: 140, fmt: "mono-time", cellClass: "mono cell-time", sortable: true }
    ],
    batchActions: [
      {
        label: "\u6279\u91CF\u91CD\u8BD5",
        className: "ghost",
        async run(ids) {
          await _retryBatch(ids);
        }
      },
      {
        label: "\u6279\u91CF\u53D6\u6D88",
        className: "danger-ghost",
        async run(ids) {
          await _cancelBatch(ids);
        }
      }
    ],
    async getCount() {
      const data = await api("/api/dashboard/agent-jobs?page_size=1");
      return data.total || 0;
    },
    async fetchPage({ page, pageSize, filters, sortBy, sortOrder }) {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      params.set("sort_by", sortBy || "updated_at");
      params.set("sort_order", sortOrder || "desc");
      if (filters?.q) params.set("q", filters.q);
      if (filters?.job_type) params.set("job_type", filters.job_type);
      if (filters?.status) params.set("status", filters.status);
      if (filters?.route_conversation_id) params.set("route_conversation_id", filters.route_conversation_id);
      if (filters?.route_account_id) params.set("route_account_id", filters.route_account_id);
      if (filters?.route_kind) params.set("route_kind", filters.route_kind);
      const payload = await api(`/api/dashboard/agent-jobs?${params.toString()}`);
      return {
        items: payload.items || [],
        total: payload.total || 0
      };
    },
    fetchDetail(item) {
      return Promise.resolve(item);
    },
    renderFilters: _renderFilters,
    renderDetail(item, container) {
      if (!item) {
        container.innerHTML = `
        <div class="agent-job-empty">
          <div class="detail-title">Agent Job</div>
          <div class="detail-subtext">Go \u63A7\u5236\u9762\u677F\u91CC\u7684\u4EFB\u52A1\u53EF\u89C1\u6027\u4E0E\u751F\u547D\u5468\u671F\u5165\u53E3\u3002</div>
        </div>
      `;
        return;
      }
      const job = item;
      container.innerHTML = `
      <div class="agent-job-detail">
        <div class="agent-job-detail-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(job.job_id || "-")}</div>
            <div class="detail-subtext">${escapeHtml(job.job_type || "-")} \xB7 ${escapeHtml(job.status || "-")}</div>
          </div>
          <div class="agent-job-detail-actions">
            <button class="primary" type="button" data-agent-job-refresh-run>\u91CD\u8BD5</button>
            <button class="danger-ghost" type="button" data-agent-job-refresh-cancel>\u53D6\u6D88</button>
            <span class="muted-text" data-agent-job-action-result></span>
          </div>
        </div>

        <div class="agent-job-section">
          <div class="agent-job-grid">
            <div>
              <div class="agent-job-label">Route</div>
              <div class="agent-job-value mono"><code>${escapeHtml(_jobRoute(job))}</code></div>
            </div>
            <div>
              <div class="agent-job-label">Attempts</div>
              <div class="agent-job-value">${escapeHtml(String(job.attempts || 0))}/${escapeHtml(String(job.max_attempts || 0))}</div>
            </div>
            <div>
              <div class="agent-job-label">Lease</div>
              <div class="agent-job-value mono">${escapeHtml(job.lease_owner || "-")}</div>
            </div>
            <div>
              <div class="agent-job-label">Expires</div>
              <div class="agent-job-value">${escapeHtml(job.lease_expires_at || "-")}</div>
            </div>
            <div>
              <div class="agent-job-label">Updated</div>
              <div class="agent-job-value mono">${escapeHtml(job.updated_at || "-")}</div>
            </div>
            <div>
              <div class="agent-job-label">Created</div>
              <div class="agent-job-value mono">${escapeHtml(job.created_at || "-")}</div>
            </div>
          </div>
        </div>

        <div class="agent-job-section">
          <div class="detail-label">Payload</div>
          <pre class="agent-job-json">${escapeHtml(_formatJson(job.payload || {}))}</pre>
        </div>

        <div class="agent-job-section">
          <div class="detail-label">Metadata</div>
          <pre class="agent-job-json">${escapeHtml(_formatJson(job.metadata || {}))}</pre>
        </div>

        <div class="agent-job-section">
          <div class="detail-label">Result</div>
          <pre class="agent-job-json">${escapeHtml(_formatJson(job.result || {}))}</pre>
        </div>

        <div class="agent-job-section">
          <div class="detail-label">Error</div>
          <div class="agent-job-error">${escapeHtml(job.error_message || "-")}</div>
        </div>
      </div>
    `;
      const actionResult = container.querySelector("[data-agent-job-action-result]");
      const retryButton = container.querySelector("[data-agent-job-refresh-run]");
      const cancelButton = container.querySelector("[data-agent-job-refresh-cancel]");
      const jobId = String(job.job_id || "");
      if (actionResult) actionResult.textContent = "";
      if (retryButton) {
        retryButton.addEventListener("click", async () => {
          if (!jobId) return;
          retryButton.disabled = true;
          try {
            await _runJobAction(jobId, "retry");
            _scheduleRefresh();
            if (actionResult) actionResult.textContent = `\u91CD\u8BD5\u5DF2\u63D0\u4EA4: ${jobId}`;
          } catch (error) {
            if (actionResult) actionResult.textContent = error instanceof Error ? error.message : String(error);
          } finally {
            retryButton.disabled = false;
          }
        });
      }
      if (cancelButton) {
        cancelButton.addEventListener("click", async () => {
          if (!jobId) return;
          cancelButton.disabled = true;
          try {
            await _runJobAction(jobId, "cancel");
            _scheduleRefresh();
            if (actionResult) actionResult.textContent = `\u5DF2\u63D0\u4EA4\u53D6\u6D88: ${jobId}`;
          } catch (error) {
            if (actionResult) actionResult.textContent = error instanceof Error ? error.message : String(error);
          } finally {
            cancelButton.disabled = false;
          }
        });
      }
    },
    formatters: {
      "agent-metric": (value) => escapeHtml(String(value ?? 0)),
      "mono-time": (value) => escapeHtml(String(value || "-"))
    }
  });
})();
