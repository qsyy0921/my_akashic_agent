/// <reference path="../../types/akashic-dashboard.d.ts" />

interface RagEvalItem {
  job_id: string;
  lifecycle_status: string;
  quality_status: string;
  passed: boolean | null;
  questions: number | null;
  top1_accuracy: number | null;
  evidence_coverage: number | null;
  min_top1_accuracy: number | null;
  min_evidence_coverage: number | null;
  attempts: number;
  max_attempts: number;
  error_message: string;
  payload: Record<string, unknown>;
  result: Record<string, unknown>;
  results: Record<string, unknown>[];
  created_at: string;
  updated_at: string;
}

interface RagEvalListResponse {
  items: RagEvalItem[];
  total: number;
  summary: Record<string, unknown>;
  trend: Record<string, unknown>[];
}

function _formatJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value ?? "");
  }
}

function _metric(value: unknown): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "-";
  return `${(value * 100).toFixed(1)}%`;
}

function _nullable(value: unknown): string {
  if (value === null || value === undefined || value === "") return "-";
  return String(value);
}

function _statusTag(status: string): string {
  const cls = `rag-eval-status rag-eval-${status || "unknown"}`;
  return `<span class="${cls}">${escapeHtml(status || "unknown")}</span>`;
}

function _renderFilters(container: HTMLElement, dispatch: PluginDispatch): void {
  container.innerHTML = `
    <div class="rag-eval-filter-row">
      <label class="rag-eval-filter">
        <span>搜索</span>
        <input data-rag-eval-filter="q" value="${escapeHtml(dispatch.filters["q"] || "")}" placeholder="job_id / payload / result" />
      </label>
      <label class="rag-eval-filter">
        <span>质量状态</span>
        <input data-rag-eval-filter="quality_status" value="${escapeHtml(dispatch.filters["quality_status"] || "")}" placeholder="passed / failed_quality" />
      </label>
      <button class="ghost" type="button" data-rag-eval-clear>清空</button>
    </div>
  `;
  container.querySelectorAll("[data-rag-eval-filter]").forEach((input) => {
    input.addEventListener("change", () => {
      const element = input as HTMLInputElement;
      const key = element.getAttribute("data-rag-eval-filter");
      if (!key) return;
      dispatch.setFilter(key, element.value.trim());
    });
  });
  container.querySelector("[data-rag-eval-clear]")?.addEventListener("click", () => {
    dispatch.setFilters({ q: "", quality_status: "" });
  });
}

window.AkashicDashboard.registerPlugin({
  id: "rag_eval",
  label: "RAG Eval",
  viewLabel: "rag eval",
  pageSize: 25,
  rowKey: "job_id",
  defaultSortBy: "updated_at",
  defaultSortOrder: "desc",

  countTitle(total: number): string {
    return `${total} 个评测任务`;
  },

  columns: [
    { key: "job_id", label: "Job ID", flex: true, cellClass: "mono cell-id", rawTitle: true },
    { key: "quality_status", label: "Quality", width: 130, renderCell: (value) => _statusTag(String(value || "")) },
    { key: "top1_accuracy", label: "Top1", width: 92, align: "right", fmt: "rag-eval-percent", sortable: true },
    { key: "evidence_coverage", label: "Evidence", width: 98, align: "right", fmt: "rag-eval-percent", sortable: true },
    { key: "questions", label: "Q", width: 56, align: "right", fmt: "rag-eval-value", sortable: true },
    { key: "lifecycle_status", label: "Lifecycle", width: 112, cellClass: "cell-status" },
    { key: "updated_at", label: "Updated", width: 150, fmt: "rag-eval-value", cellClass: "mono cell-time", sortable: true },
  ],

  async getCount(): Promise<number | null> {
    const data = await api<RagEvalListResponse>("/api/dashboard/rag-eval?page_size=1");
    return data.total || 0;
  },

  async fetchPage({ page, pageSize, filters, sortBy, sortOrder }: FetchPageOpts): Promise<FetchPageResult> {
    const params = new URLSearchParams();
    params.set("page", String(page));
    params.set("page_size", String(pageSize));
    params.set("sort_by", sortBy || "updated_at");
    params.set("sort_order", sortOrder || "desc");
    if (filters?.q) params.set("q", filters.q);
    if (filters?.quality_status) params.set("quality_status", filters.quality_status);
    const payload = await api<RagEvalListResponse>(`/api/dashboard/rag-eval?${params.toString()}`);
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
        <div class="rag-eval-empty">
          <div class="detail-title">RAG Eval</div>
          <div class="detail-subtext">查看 Go generic job 中的 RAG 评测质量门和指标结果。</div>
        </div>
      `;
      return;
    }
    const evalJob = item as unknown as RagEvalItem;
    container.innerHTML = `
      <div class="rag-eval-detail">
        <div class="rag-eval-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(evalJob.job_id || "-")}</div>
            <div class="detail-subtext">${_statusTag(evalJob.quality_status || "unknown")} · ${escapeHtml(evalJob.lifecycle_status || "-")}</div>
          </div>
        </div>
        <div class="rag-eval-section">
          <div class="rag-eval-grid">
            <div>
              <div class="rag-eval-label">Top1</div>
              <div class="rag-eval-value">${escapeHtml(_metric(evalJob.top1_accuracy))}</div>
            </div>
            <div>
              <div class="rag-eval-label">Evidence</div>
              <div class="rag-eval-value">${escapeHtml(_metric(evalJob.evidence_coverage))}</div>
            </div>
            <div>
              <div class="rag-eval-label">Questions</div>
              <div class="rag-eval-value">${escapeHtml(_nullable(evalJob.questions))}</div>
            </div>
            <div>
              <div class="rag-eval-label">Gate</div>
              <div class="rag-eval-value">${escapeHtml(_metric(evalJob.min_top1_accuracy))} / ${escapeHtml(_metric(evalJob.min_evidence_coverage))}</div>
            </div>
          </div>
        </div>
        <div class="rag-eval-section">
          <div class="detail-label">Result</div>
          <pre class="rag-eval-json">${escapeHtml(_formatJson(evalJob.result || {}))}</pre>
        </div>
        <div class="rag-eval-section">
          <div class="detail-label">Per Question</div>
          <pre class="rag-eval-json">${escapeHtml(_formatJson(evalJob.results || []))}</pre>
        </div>
        <div class="rag-eval-section">
          <div class="detail-label">Payload</div>
          <pre class="rag-eval-json">${escapeHtml(_formatJson(evalJob.payload || {}))}</pre>
        </div>
        <div class="rag-eval-section">
          <div class="detail-label">Error</div>
          <div class="rag-eval-error">${escapeHtml(evalJob.error_message || "-")}</div>
        </div>
      </div>
    `;
  },

  formatters: {
    "rag-eval-percent": (value: unknown) => escapeHtml(_metric(value)),
    "rag-eval-value": (value: unknown) => escapeHtml(_nullable(value)),
  },
});

export {};
