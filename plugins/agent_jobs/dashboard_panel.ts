/// <reference path="../../types/akashic-dashboard.d.ts" />

interface AgentJobRoute {
  kind?: string;
  account_id?: string;
  conversation_id?: string;
  conversation_type?: string;
}

interface AgentJob {
  job_id: string;
  job_type: string;
  agent_id: string;
  route?: AgentJobRoute;
  status: string;
  attempts: number;
  max_attempts: number;
  lease_owner: string;
  lease_token_present?: boolean;
  lease_expires_at: string;
  error_message: string;
  source_event_ids: string[];
  source_asset_ids: string[];
  payload: Record<string, string | number | boolean>;
  result: Record<string, string>;
  metadata: Record<string, string>;
  created_at: string;
  updated_at: string;
  route_kind?: string;
  route_account_id?: string;
  route_conversation_id?: string;
  route_conversation_type?: string;
}

interface AgentJobListResponse {
  items: AgentJob[];
  total: number;
}

interface AgentJobRecoveryResponse {
  timestamp: string;
  scanned: number;
  recovered: number;
  dead_lettered: number;
  items: Array<{ action: string; job: AgentJob }>;
}

function _jobRoute(item: AgentJob): string {
  const route = item.route || {};
  const kind = route.kind || item.route_kind || "-";
  const account = route.account_id || item.route_account_id || "-";
  const conv = route.conversation_id || item.route_conversation_id || "-";
  const type = route.conversation_type || item.route_conversation_type || "-";
  return `${kind}:${account}:${type}:${conv}`;
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

function _jobStatusTag(status: string): string {
  const cls = `agent-job-status agent-status-${status || "unknown"}`;
  const text = status || "unknown";
  return `<span class="${cls}">${escapeHtml(text)}</span>`;
}

function _renderFilters(container: HTMLElement, dispatch: PluginDispatch): void {
  container.innerHTML = `
    <div class="agent-job-filter-row">
      <label class="agent-job-filter">
        <span>搜索</span>
        <input data-agent-job-filter="q" value="${escapeHtml(dispatch.filters["q"] || "")}" placeholder="job_id / route / payload / metadata" />
      </label>
      <label class="agent-job-filter">
        <span>类型</span>
        <input data-agent-job-filter="job_type" value="${escapeHtml(dispatch.filters["job_type"] || "")}" placeholder="例如 rag_ingest" />
      </label>
      <label class="agent-job-filter">
        <span>状态</span>
        <input data-agent-job-filter="status" value="${escapeHtml(dispatch.filters["status"] || "")}" placeholder="pending / running" />
      </label>
      <label class="agent-job-filter">
        <span>会话</span>
        <input data-agent-job-filter="route_conversation_id" value="${escapeHtml(dispatch.filters["route_conversation_id"] || "")}" placeholder="conversation_id" />
      </label>
      <button class="ghost" type="button" data-agent-job-clear>清空</button>
    </div>
  `;
  container.querySelectorAll("[data-agent-job-filter]").forEach((input) => {
    input.addEventListener("change", () => {
      const element = input as HTMLInputElement;
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
      route_kind: "",
    });
  });
}

async function _runJobAction(jobId: string, action: "retry" | "cancel"): Promise<void> {
  await api(`/api/dashboard/agent-jobs/${encodePath(jobId)}/${action}`, {
    method: "POST",
  });
}

async function _recoverExpiredJobs(limit = 50): Promise<AgentJobRecoveryResponse> {
  return api<AgentJobRecoveryResponse>("/api/dashboard/agent-jobs/recover-expired", {
    method: "POST",
    body: JSON.stringify({ limit }),
    headers: { "Content-Type": "application/json" },
  });
}

function _scheduleRefresh(): void {
  window.dispatchEvent(new CustomEvent("akashic-dashboard-refresh"));
}

async function _retryBatch(ids: string[]): Promise<void> {
  for (const jobId of ids) {
    await _runJobAction(jobId, "retry");
  }
  _scheduleRefresh();
}

async function _cancelBatch(ids: string[]): Promise<void> {
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

  countTitle(total: number): string {
    return `${total} 个任务`;
  },

  columns: [
    { key: "job_id", label: "Job ID", width: 220, cellClass: "mono cell-id", rawTitle: true },
    { key: "job_type", label: "Type", width: 150, cellClass: "cell-type", rawTitle: true },
    { key: "status", label: "Status", width: 120, cellClass: "cell-status", renderCell: (v) => _jobStatusTag(String(v || "")) },
    { key: "route", label: "Route", flex: true, renderCell: (_value, row) => escapeHtml(_jobRoute(row as unknown as AgentJob)) },
    { key: "attempts", label: "Attempts", width: 88, fmt: "agent-metric", cellClass: "mono cell-metric", align: "right", sortable: true },
    { key: "max_attempts", label: "Max", width: 60, fmt: "agent-metric", cellClass: "mono cell-metric", align: "right", sortable: true },
    { key: "lease_owner", label: "Lease", width: 112, cellClass: "mono cell-owner", rawTitle: true },
    { key: "error_message", label: "Error", width: 150, renderCell: (value) => _short(value, 64), cellClass: "content-preview" },
    { key: "updated_at", label: "Updated", width: 140, fmt: "mono-time", cellClass: "mono cell-time", sortable: true },
  ],

  batchActions: [
    {
      label: "批量重试",
      className: "ghost",
      async run(ids: string[]): Promise<void> {
        await _retryBatch(ids);
      },
    },
    {
      label: "批量取消",
      className: "danger-ghost",
      async run(ids: string[]): Promise<void> {
        await _cancelBatch(ids);
      },
    },
  ],

  async getCount(): Promise<number | null> {
    const data = await api<AgentJobListResponse>("/api/dashboard/agent-jobs?page_size=1");
    return data.total || 0;
  },

  async fetchPage({ page, pageSize, filters, sortBy, sortOrder }: FetchPageOpts): Promise<FetchPageResult> {
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

    const payload = await api<AgentJobListResponse>(`/api/dashboard/agent-jobs?${params.toString()}`);
    return {
      items: (payload.items || []) as unknown as Record<string, unknown>[],
      total: payload.total || 0,
    };
  },

  fetchDetail(item: Record<string, unknown>): Promise<Record<string, unknown>> {
    return Promise.resolve(item);
  },

  renderFilters: _renderFilters,

  renderDetail(item: Record<string, unknown> | null, container: HTMLElement): void {
    if (!item) {
      container.innerHTML = `
        <div class="agent-job-empty">
          <div class="detail-title">Agent Job</div>
          <div class="detail-subtext">Go 控制面板里的任务可见性与生命周期入口。</div>
        </div>
      `;
      return;
    }
    const job = item as unknown as AgentJob;
    container.innerHTML = `
      <div class="agent-job-detail">
        <div class="agent-job-detail-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(job.job_id || "-")}</div>
            <div class="detail-subtext">${escapeHtml(job.job_type || "-")} · ${escapeHtml(job.status || "-")}</div>
          </div>
          <div class="agent-job-detail-actions">
            <button class="ghost" type="button" data-agent-job-recover-expired>恢复过期租约</button>
            <button class="primary" type="button" data-agent-job-refresh-run>重试</button>
            <button class="danger-ghost" type="button" data-agent-job-refresh-cancel>取消</button>
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
              <div class="agent-job-label">Lease Token</div>
              <div class="agent-job-value mono">${escapeHtml(job.lease_token_present ? "present" : "-")}</div>
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
    const actionResult = container.querySelector<HTMLElement>("[data-agent-job-action-result]");
    const retryButton = container.querySelector<HTMLButtonElement>("[data-agent-job-refresh-run]");
    const cancelButton = container.querySelector<HTMLButtonElement>("[data-agent-job-refresh-cancel]");
    const recoverButton = container.querySelector<HTMLButtonElement>("[data-agent-job-recover-expired]");
    const jobId = String(job.job_id || "");
    if (actionResult) actionResult.textContent = "";

    if (recoverButton) {
      recoverButton.addEventListener("click", async () => {
        recoverButton.disabled = true;
        try {
          const result = await _recoverExpiredJobs(50);
          _scheduleRefresh();
          if (actionResult) {
            actionResult.textContent = `已扫描 ${result.scanned || 0}，恢复 ${result.recovered || 0}，死信 ${result.dead_lettered || 0}`;
          }
        } catch (error) {
          if (actionResult) actionResult.textContent = error instanceof Error ? error.message : String(error);
        } finally {
          recoverButton.disabled = false;
        }
      });
    }

    if (retryButton) {
      retryButton.addEventListener("click", async () => {
        if (!jobId) return;
        retryButton.disabled = true;
        try {
          await _runJobAction(jobId, "retry");
          _scheduleRefresh();
          if (actionResult) actionResult.textContent = `重试已提交: ${jobId}`;
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
          if (actionResult) actionResult.textContent = `已提交取消: ${jobId}`;
        } catch (error) {
          if (actionResult) actionResult.textContent = error instanceof Error ? error.message : String(error);
        } finally {
          cancelButton.disabled = false;
        }
      });
    }
  },

  formatters: {
    "agent-metric": (value: unknown) => escapeHtml(String(value ?? 0)),
    "mono-time": (value: unknown) => escapeHtml(String(value || "-")),
  },
});

export {};
