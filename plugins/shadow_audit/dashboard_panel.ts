/// <reference path="../../types/akashic-dashboard.d.ts" />

interface ShadowAttachment {
  id?: string;
  kind?: string;
  url?: string;
  content_url?: string;
  mime_type?: string;
  name?: string;
}

interface ShadowAuditRecord {
  event_id: string;
  platform: string;
  account_id: string;
  conversation_id: string;
  conversation_type: string;
  sender_id: string;
  content: string;
  timestamp: string;
  attachment_count: number;
  decision_action?: string;
  decision_reason?: string;
  attachments?: ShadowAttachment[];
  metadata?: Record<string, unknown>;
  source?: string;
}

interface ShadowAuditResponse {
  items: ShadowAuditRecord[];
  total: number;
}

function _short(value: unknown, limit: number): string {
  const text = String(value ?? "").replace(/\s+/g, " ").trim();
  return text.length > limit ? `${text.slice(0, limit).trim()}...` : text;
}

function _route(row: ShadowAuditRecord): string {
  const platform = row.platform || "-";
  const account = row.account_id || "-";
  const type = row.conversation_type || "-";
  const conversation = row.conversation_id || "-";
  return `${platform}:${account}:${type}:${conversation}`;
}

function _decision(row: ShadowAuditRecord): string {
  if (!row.decision_action) {
    return row.source === "jsonl" ? "logged" : "-";
  }
  return row.decision_reason
    ? `${row.decision_action} (${row.decision_reason})`
    : row.decision_action;
}

function _sourceClass(row: ShadowAuditRecord): string {
  return row.source === "gateway" ? "shadow-source-gateway" : "shadow-source-jsonl";
}

function _renderAttachments(item: ShadowAuditRecord): string {
  const attachments = Array.isArray(item.attachments) ? item.attachments : [];
  if (!attachments.length && !item.attachment_count) {
    return '<div class="muted-text">No attachments recorded.</div>';
  }
  if (!attachments.length) {
    return `<div class="muted-text">${escapeHtml(String(item.attachment_count))} attachment(s), link metadata is not exposed by the gateway query yet.</div>`;
  }
  return `
    <div class="shadow-attachment-list">
      ${attachments.map((attachment, index) => {
        const name = attachment.name || attachment.id || `attachment-${index + 1}`;
        const url = attachment.content_url || attachment.url || "";
        const mime = attachment.mime_type || attachment.kind || "file";
        const label = `${name} (${mime})`;
        if (!url) {
          return `<div class="shadow-attachment-row">${escapeHtml(label)}</div>`;
        }
        return `<a class="shadow-attachment-row" href="${escapeHtml(url)}" target="_blank" rel="noreferrer">${escapeHtml(label)}</a>`;
      }).join("")}
    </div>
  `;
}

function _renderFilterInput(
  dispatch: PluginDispatch,
  key: string,
  label: string,
  placeholder: string,
): string {
  const value = String(dispatch.filters[key] || "");
  return `
    <label class="shadow-filter">
      <span>${escapeHtml(label)}</span>
      <input data-shadow-filter="${escapeHtml(key)}" value="${escapeHtml(value)}" placeholder="${escapeHtml(placeholder)}" />
    </label>
  `;
}

function _wireFilters(container: ParentNode, dispatch: PluginDispatch): void {
  container.querySelectorAll("[data-shadow-filter]").forEach((input) => {
    input.addEventListener("change", () => {
      const element = input as HTMLInputElement;
      const key = element.getAttribute("data-shadow-filter");
      if (!key) return;
      dispatch.setFilter(key, element.value.trim());
    });
  });
  container.querySelector("[data-shadow-source]")?.addEventListener("change", (event) => {
    dispatch.setFilter("source", (event.target as HTMLSelectElement).value);
  });
  container.querySelector("[data-shadow-clear]")?.addEventListener("click", () => {
    dispatch.setFilters({
      q: "",
      source: "auto",
      platform: "",
      account_id: "",
      conversation_id: "",
      conversation_type: "",
    });
  });
}

window.AkashicDashboard.registerPlugin({
  id: "shadow_audit",
  label: "Shadow Audit",
  viewLabel: "shadow audit",
  pageSize: 25,
  rowKey: "event_id",
  defaultSortBy: "timestamp",
  defaultSortOrder: "desc",
  countTitle(total: number): string {
    return `${total} shadow events`;
  },
  columns: [
    { key: "timestamp", label: "Time", width: 96, fmt: "mono-time", cellClass: "mono cell-time", rawTitle: true, sortable: true },
    { key: "platform", label: "Route", width: 188, renderCell: (_value: unknown, row: Record<string, unknown>) => escapeHtml(_route(row as unknown as ShadowAuditRecord)), cellClass: "mono cell-session", rawTitle: true },
    { key: "sender_id", label: "Sender", width: 108, cellClass: "mono cell-session", rawTitle: true, sortable: true },
    { key: "decision_action", label: "Decision", width: 120, renderCell: (_value: unknown, row: Record<string, unknown>) => {
      const item = row as unknown as ShadowAuditRecord;
      return `<span class="shadow-source-pill ${_sourceClass(item)}">${escapeHtml(_decision(item))}</span>`;
    } },
    { key: "attachment_count", label: "Files", width: 60, fmt: "metric", cellClass: "mono cell-metric", align: "right", sortable: true },
    { key: "content", label: "Content", flex: true, renderCell: (value: unknown) => escapeHtml(_short(value, 120)), cellClass: "content-preview" },
  ],
  async getCount(): Promise<number | null> {
    try {
      const data = await api<ShadowAuditResponse>("/api/dashboard/shadow-audit/observed?page_size=1");
      return data.total || 0;
    } catch {
      return null;
    }
  },
  async fetchPage({ page, pageSize, filters, sortBy, sortOrder }: FetchPageOpts): Promise<FetchPageResult> {
    const params = new URLSearchParams();
    params.set("page", String(page));
    params.set("page_size", String(pageSize));
    params.set("sort_by", sortBy || "timestamp");
    params.set("sort_order", sortOrder || "desc");
    for (const key of ["source", "q", "platform", "account_id", "conversation_id", "conversation_type"]) {
      const value = filters?.[key];
      if (value) params.set(key, String(value));
    }
    const data = await api<ShadowAuditResponse>(`/api/dashboard/shadow-audit/observed?${params.toString()}`);
    return {
      items: data.items || [],
      total: data.total || 0,
    };
  },
  fetchDetail(item: ShadowAuditRecord): Promise<ShadowAuditRecord> {
    return Promise.resolve(item);
  },
  renderFilters(container: HTMLElement, dispatch: PluginDispatch): void {
    container.innerHTML = `
      <div class="shadow-filter-row">
        ${_renderFilterInput(dispatch, "q", "Search", "content / sender / metadata")}
        <label class="shadow-filter shadow-filter-source">
          <span>Source</span>
          <select data-shadow-source>
            <option value="auto">auto</option>
            <option value="gateway">gateway</option>
            <option value="jsonl">jsonl</option>
          </select>
        </label>
        <button class="ghost" type="button" data-shadow-clear>Clear</button>
      </div>
    `;
    const source = container.querySelector("[data-shadow-source]") as HTMLSelectElement | null;
    if (source) source.value = String(dispatch.filters.source || "auto");
    _wireFilters(container, dispatch);
  },
  renderDetail(item: ShadowAuditRecord | null, container: HTMLElement): void {
    if (!item) {
      container.innerHTML = `
        <div class="detail-empty">
          <div class="detail-empty-title">Shadow Audit</div>
          <div class="detail-empty-text">Inspect mirrored Go/Python message events before any production cutover.</div>
        </div>
      `;
      return;
    }
    container.innerHTML = `
      <div class="detail-wrap">
        <div class="detail-toolbar">
          <div>
            <div class="detail-title">Shadow Event</div>
            <div class="detail-subtext">${escapeHtml(item.event_id || "")}</div>
          </div>
        </div>
        <div class="detail-grid">
          <div class="detail-row">
            <div class="detail-row-label">route</div>
            <div class="detail-row-val"><code>${escapeHtml(_route(item))}</code></div>
          </div>
          <div class="detail-row">
            <div class="detail-row-label">sender</div>
            <div class="detail-row-val"><code>${escapeHtml(item.sender_id || "")}</code></div>
          </div>
          <div class="detail-row">
            <div class="detail-row-label">decision</div>
            <div class="detail-row-val"><span class="shadow-source-pill ${_sourceClass(item)}">${escapeHtml(_decision(item))}</span></div>
          </div>
          <div class="detail-row">
            <div class="detail-row-label">source</div>
            <div class="detail-row-val"><code>${escapeHtml(item.source || "")}</code></div>
          </div>
        </div>
        <div class="detail-block">
          <div class="detail-label">Content</div>
          <div class="detail-content">${escapeHtml(item.content || "")}</div>
        </div>
        <div class="detail-block">
          <div class="detail-label">Attachments</div>
          ${_renderAttachments(item)}
        </div>
        <div class="detail-block">
          <div class="detail-label">Metadata</div>
          ${jvPlaceholder(item.metadata || {})}
        </div>
      </div>
    `;
    attachJsonViewers(container);
  },
  formatters: {
    "shadow-route": (_value: unknown, row: Record<string, unknown>) => _route(row as unknown as ShadowAuditRecord),
  },
});
