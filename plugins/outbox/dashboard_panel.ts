/// <reference path="../../types/akashic-dashboard.d.ts" />

interface OutboxChannel {
  kind?: string;
  account_id?: string;
  conversation_id?: string;
  conversation_type?: string;
}

interface OutboxDelivery {
  event_id: string;
  channel?: OutboxChannel;
  channel_kind?: string;
  account_id?: string;
  conversation_id?: string;
  conversation_type?: string;
  content: string;
  attachments: Record<string, unknown>[];
  status: string;
  attempts: number;
  max_attempts: number;
  lease_owner: string;
  lease_expires_at: string;
  error_message: string;
  created_at: string;
  updated_at: string;
  metadata: Record<string, string>;
}

interface OutboxListResponse {
  items: OutboxDelivery[];
  total: number;
}

function _route(item: OutboxDelivery): string {
  const channel = item.channel || {};
  const kind = channel.kind || item.channel_kind || "-";
  const account = channel.account_id || item.account_id || "-";
  const type = channel.conversation_type || item.conversation_type || "-";
  const conversation = channel.conversation_id || item.conversation_id || "-";
  return `${kind}:${account}:${type}:${conversation}`;
}

function _short(value: unknown, limit: number): string {
  const text = String(value ?? "").replace(/\s+/g, " ").trim();
  return text.length > limit ? `${text.slice(0, limit)}...` : text;
}

function _formatJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value ?? "");
  }
}

function _statusTag(status: string): string {
  const text = status || "unknown";
  return `<span class="outbox-status-${escapeHtml(text)}">${escapeHtml(text)}</span>`;
}

function _renderFilters(container: HTMLElement, dispatch: PluginDispatch): void {
  container.innerHTML = `
    <div class="outbox-filter-row">
      <label class="outbox-filter">
        <span>搜索</span>
        <input data-outbox-filter="q" value="${escapeHtml(dispatch.filters["q"] || "")}" placeholder="event / route / content / error" />
      </label>
      <label class="outbox-filter">
        <span>状态</span>
        <input data-outbox-filter="status" value="${escapeHtml(dispatch.filters["status"] || "")}" placeholder="queued / failed" />
      </label>
      <label class="outbox-filter">
        <span>账号</span>
        <input data-outbox-filter="account_id" value="${escapeHtml(dispatch.filters["account_id"] || "")}" placeholder="1049511700" />
      </label>
      <label class="outbox-filter">
        <span>会话</span>
        <input data-outbox-filter="conversation_id" value="${escapeHtml(dispatch.filters["conversation_id"] || "")}" placeholder="2365524513 / 群号" />
      </label>
      <button class="ghost" type="button" data-outbox-clear>清空</button>
    </div>
  `;
  container.querySelectorAll("[data-outbox-filter]").forEach((input) => {
    input.addEventListener("change", () => {
      const element = input as HTMLInputElement;
      const key = element.getAttribute("data-outbox-filter");
      if (!key) return;
      dispatch.setFilter(key, element.value.trim());
    });
  });
  container.querySelector("[data-outbox-clear]")?.addEventListener("click", () => {
    dispatch.setFilters({
      q: "",
      status: "",
      account_id: "",
      conversation_id: "",
      channel_kind: "",
      conversation_type: "",
    });
  });
}

async function _runAction(eventId: string, action: string, errorMessage = ""): Promise<OutboxDelivery> {
  const body = action === "failed" ? { error_message: errorMessage || "manual failure from dashboard" } : {};
  return api<OutboxDelivery>(`/api/dashboard/outbox/${encodePath(eventId)}/${action}`, {
    method: "POST",
    body: JSON.stringify(body),
    headers: { "Content-Type": "application/json" },
  });
}

function _scheduleRefresh(): void {
  window.dispatchEvent(new CustomEvent("akashic-dashboard-refresh"));
}

async function _retryBatch(ids: string[]): Promise<void> {
  for (const eventId of ids) {
    await _runAction(eventId, "retry");
  }
  _scheduleRefresh();
}

window.AkashicDashboard.registerPlugin({
  id: "outbox",
  label: "Outbox",
  viewLabel: "outbox",
  pageSize: 25,
  rowKey: "event_id",
  defaultSortBy: "updated_at",
  defaultSortOrder: "desc",

  countTitle(total: number): string {
    return `${total} 条投递`;
  },

  columns: [
    { key: "event_id", label: "Event ID", width: 200, cellClass: "mono cell-id", rawTitle: true },
    { key: "status", label: "Status", width: 120, cellClass: "cell-status", renderCell: (value) => _statusTag(String(value || "")) },
    { key: "route", label: "Route", flex: true, renderCell: (_value, row) => escapeHtml(_route(row as unknown as OutboxDelivery)) },
    { key: "content", label: "Content", width: 180, renderCell: (value) => escapeHtml(_short(value, 64)), cellClass: "content-preview" },
    { key: "attempts", label: "Attempts", width: 92, fmt: "outbox-attempts", cellClass: "mono cell-metric", align: "right", sortable: true },
    { key: "lease_owner", label: "Lease", width: 120, renderCell: (value) => escapeHtml(_short(value || "-", 32)), cellClass: "mono cell-id" },
    { key: "error_message", label: "Error", width: 170, renderCell: (value) => escapeHtml(_short(value, 64)), cellClass: "content-preview" },
    { key: "updated_at", label: "Updated", width: 150, fmt: "mono-time", cellClass: "mono cell-time", sortable: true },
  ],

  batchActions: [
    {
      label: "批量重试",
      className: "ghost",
      async run(ids: string[]): Promise<void> {
        await _retryBatch(ids);
      },
    },
  ],

  async getCount(): Promise<number | null> {
    const data = await api<OutboxListResponse>("/api/dashboard/outbox?page_size=1");
    return data.total || 0;
  },

  async fetchPage({ page, pageSize, filters, sortBy, sortOrder }: FetchPageOpts): Promise<FetchPageResult> {
    const params = new URLSearchParams();
    params.set("page", String(page));
    params.set("page_size", String(pageSize));
    params.set("sort_by", sortBy || "updated_at");
    params.set("sort_order", sortOrder || "desc");
    if (filters?.q) params.set("q", filters.q);
    if (filters?.status) params.set("status", filters.status);
    if (filters?.account_id) params.set("account_id", filters.account_id);
    if (filters?.conversation_id) params.set("conversation_id", filters.conversation_id);
    if (filters?.channel_kind) params.set("channel_kind", filters.channel_kind);
    if (filters?.conversation_type) params.set("conversation_type", filters.conversation_type);

    const payload = await api<OutboxListResponse>(`/api/dashboard/outbox?${params.toString()}`);
    return {
      items: (payload.items || []) as unknown as Record<string, unknown>[],
      total: payload.total || 0,
    };
  },

  async fetchDetail(item: Record<string, unknown>): Promise<Record<string, unknown>> {
    const eventId = String(item.event_id || "");
    if (!eventId) return item;
    return api<Record<string, unknown>>(`/api/dashboard/outbox/${encodePath(eventId)}`);
  },

  renderFilters: _renderFilters,

  renderDetail(item: Record<string, unknown> | null, container: HTMLElement): void {
    if (!item) {
      container.innerHTML = `
        <div class="outbox-empty">
          <div class="detail-title">Outbox</div>
          <div class="detail-subtext">查看 Go agent-runtime 管理的投递状态、失败和重试记录。</div>
        </div>
      `;
      return;
    }
    const delivery = item as unknown as OutboxDelivery;
    container.innerHTML = `
      <div class="outbox-detail">
        <div class="outbox-detail-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(delivery.event_id || "-")}</div>
            <div class="detail-subtext">${escapeHtml(_route(delivery))} · ${escapeHtml(delivery.status || "-")}</div>
          </div>
          <div class="outbox-detail-actions">
            <button class="ghost" type="button" data-outbox-action="dispatching">Dispatching</button>
            <button class="ghost" type="button" data-outbox-action="lease-next">Lease Next</button>
            <button class="primary" type="button" data-outbox-action="retry">Retry</button>
            <button class="ghost" type="button" data-outbox-action="succeeded">Succeeded</button>
          </div>
        </div>

        <div class="outbox-section">
          <div class="outbox-grid">
            <div>
              <div class="outbox-label">Status</div>
              <div class="outbox-value">${_statusTag(delivery.status || "-")}</div>
            </div>
            <div>
              <div class="outbox-label">Attempts</div>
              <div class="outbox-value">${escapeHtml(String(delivery.attempts || 0))}/${escapeHtml(String(delivery.max_attempts || 0))}</div>
            </div>
            <div>
              <div class="outbox-label">Lease Owner</div>
              <div class="outbox-value mono">${escapeHtml(delivery.lease_owner || "-")}</div>
            </div>
            <div>
              <div class="outbox-label">Lease Expires</div>
              <div class="outbox-value mono">${escapeHtml(delivery.lease_expires_at || "-")}</div>
            </div>
            <div>
              <div class="outbox-label">Route</div>
              <div class="outbox-value mono">${escapeHtml(_route(delivery))}</div>
            </div>
            <div>
              <div class="outbox-label">Updated</div>
              <div class="outbox-value mono">${escapeHtml(delivery.updated_at || "-")}</div>
            </div>
            <div>
              <div class="outbox-label">Created</div>
              <div class="outbox-value mono">${escapeHtml(delivery.created_at || "-")}</div>
            </div>
            <div>
              <div class="outbox-label">Error</div>
              <div class="outbox-value outbox-error">${escapeHtml(delivery.error_message || "-")}</div>
            </div>
          </div>
        </div>

        <div class="outbox-section">
          <div class="detail-label">Content</div>
          <div class="outbox-value">${escapeHtml(delivery.content || "-")}</div>
        </div>

        <div class="outbox-section outbox-action-form">
          <div class="detail-label">Mark Failed</div>
          <label class="outbox-filter">
            <span>Error Message</span>
            <input data-outbox-failed-message value="${escapeHtml(delivery.error_message || "manual failure from dashboard")}" />
          </label>
          <div class="outbox-action-row">
            <button class="danger-ghost" type="button" data-outbox-action="failed">Mark Failed</button>
            <span class="muted-text" data-outbox-action-result></span>
          </div>
        </div>

        <div class="outbox-section">
          <div class="detail-label">Attachments</div>
          <pre class="outbox-json">${escapeHtml(_formatJson(delivery.attachments || []))}</pre>
        </div>

        <div class="outbox-section">
          <div class="detail-label">Metadata</div>
          <pre class="outbox-json">${escapeHtml(_formatJson(delivery.metadata || {}))}</pre>
        </div>
      </div>
    `;
    const eventId = String(delivery.event_id || "");
    const result = container.querySelector<HTMLElement>("[data-outbox-action-result]");
    container.querySelectorAll<HTMLButtonElement>("[data-outbox-action]").forEach((button) => {
      button.addEventListener("click", async () => {
        const action = button.getAttribute("data-outbox-action") || "";
        if (!eventId || !action) return;
        const errorInput = container.querySelector<HTMLInputElement>("[data-outbox-failed-message]");
        button.disabled = true;
        if (result) result.textContent = "提交中";
        try {
          await _runAction(eventId, action, errorInput?.value.trim() || "");
          _scheduleRefresh();
          if (result) result.textContent = `已提交 ${action}`;
        } catch (error) {
          if (result) result.textContent = error instanceof Error ? error.message : String(error);
        } finally {
          button.disabled = false;
        }
      });
    });
  },

  formatters: {
    "outbox-attempts": (_value: unknown, row?: Record<string, unknown>) => {
      const item = row as unknown as OutboxDelivery | undefined;
      return escapeHtml(`${item?.attempts ?? 0}/${item?.max_attempts ?? 0}`);
    },
    "mono-time": (value: unknown) => escapeHtml(String(value || "-")),
  },
});

export {};
