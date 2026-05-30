/// <reference path="../../types/akashic-dashboard.d.ts" />

interface SendLedgerRecord {
  from_bot_id: string;
  conversation_id: string;
  content_hash: string;
  short_hash: string;
  timestamp: string;
}

interface SendLedgerListResponse {
  items: SendLedgerRecord[];
  total: number;
}

interface SendLedgerRecentResponse {
  recent: boolean;
  from_bot_id: string;
  conversation_id: string;
  content_hash: string;
  window_seconds: number;
  runtime_url?: string;
}

function _short(value: unknown, limit: number): string {
  const text = String(value ?? "").replace(/\s+/g, " ").trim();
  return text.length > limit ? `${text.slice(0, limit)}...` : text;
}

function _renderFilters(container: HTMLElement, dispatch: PluginDispatch): void {
  container.innerHTML = `
    <div class="send-ledger-filter-row">
      <label class="send-ledger-filter">
        <span>搜索</span>
        <input data-send-ledger-filter="q" value="${escapeHtml(dispatch.filters["q"] || "")}" placeholder="账号 / 会话 / hash" />
      </label>
      <label class="send-ledger-filter">
        <span>发送账号</span>
        <input data-send-ledger-filter="from_bot_id" value="${escapeHtml(dispatch.filters["from_bot_id"] || "")}" placeholder="1049511700" />
      </label>
      <label class="send-ledger-filter">
        <span>会话</span>
        <input data-send-ledger-filter="conversation_id" value="${escapeHtml(dispatch.filters["conversation_id"] || "")}" placeholder="2365524513 / 群号" />
      </label>
      <label class="send-ledger-filter">
        <span>Content Hash</span>
        <input data-send-ledger-filter="content_hash" value="${escapeHtml(dispatch.filters["content_hash"] || "")}" placeholder="sha256" />
      </label>
      <button class="ghost" type="button" data-send-ledger-clear>清空</button>
    </div>
  `;
  container.querySelectorAll("[data-send-ledger-filter]").forEach((input) => {
    input.addEventListener("change", () => {
      const element = input as HTMLInputElement;
      const key = element.getAttribute("data-send-ledger-filter");
      if (!key) return;
      dispatch.setFilter(key, element.value.trim());
    });
  });
  container.querySelector("[data-send-ledger-clear]")?.addEventListener("click", () => {
    dispatch.setFilters({
      q: "",
      from_bot_id: "",
      conversation_id: "",
      content_hash: "",
    });
  });
}

async function _checkRecent(params: {
  fromBotId: string;
  conversationId: string;
  content: string;
  contentHash: string;
  windowSeconds: string;
}): Promise<SendLedgerRecentResponse> {
  const query = new URLSearchParams();
  query.set("from_bot_id", params.fromBotId);
  query.set("conversation_id", params.conversationId);
  query.set("window_seconds", params.windowSeconds || "60");
  if (params.contentHash) query.set("content_hash", params.contentHash);
  if (params.content) query.set("content", params.content);
  return api<SendLedgerRecentResponse>(`/api/dashboard/send-ledger/recent?${query.toString()}`);
}

function _renderRecentResult(container: HTMLElement, result: SendLedgerRecentResponse): void {
  const cls = result.recent ? "send-ledger-hit" : "send-ledger-miss";
  const label = result.recent ? "命中 recent 防循环" : "未命中 recent 防循环";
  container.innerHTML = `
    <div class="${cls}">${escapeHtml(label)}</div>
    <div class="send-ledger-grid">
      <div>
        <div class="send-ledger-label">From Bot</div>
        <div class="send-ledger-value mono">${escapeHtml(result.from_bot_id || "-")}</div>
      </div>
      <div>
        <div class="send-ledger-label">Conversation</div>
        <div class="send-ledger-value mono">${escapeHtml(result.conversation_id || "-")}</div>
      </div>
      <div>
        <div class="send-ledger-label">Content Hash</div>
        <div class="send-ledger-value mono">${escapeHtml(result.content_hash || "-")}</div>
      </div>
      <div>
        <div class="send-ledger-label">Window</div>
        <div class="send-ledger-value">${escapeHtml(String(result.window_seconds || 0))}s</div>
      </div>
    </div>
  `;
}

window.AkashicDashboard.registerPlugin({
  id: "send_ledger",
  label: "Send Ledger",
  viewLabel: "send ledger",
  pageSize: 25,
  rowKey: "content_hash",
  defaultSortBy: "timestamp",
  defaultSortOrder: "desc",

  countTitle(total: number): string {
    return `${total} 条发送记录`;
  },

  columns: [
    { key: "from_bot_id", label: "From Bot", width: 130, cellClass: "mono cell-id", rawTitle: true },
    { key: "conversation_id", label: "Conversation", width: 150, cellClass: "mono cell-id", rawTitle: true },
    {
      key: "content_hash",
      label: "Content Hash",
      flex: true,
      cellClass: "mono cell-id",
      rawTitle: true,
      renderCell: (value) => escapeHtml(_short(value, 28)),
    },
    { key: "timestamp", label: "Timestamp", width: 170, fmt: "mono-time", cellClass: "mono cell-time", sortable: true },
  ],

  async getCount(): Promise<number | null> {
    const data = await api<SendLedgerListResponse>("/api/dashboard/send-ledger?page_size=1");
    return data.total || 0;
  },

  async fetchPage({ page, pageSize, filters, sortBy, sortOrder }: FetchPageOpts): Promise<FetchPageResult> {
    const params = new URLSearchParams();
    params.set("page", String(page));
    params.set("page_size", String(pageSize));
    params.set("sort_by", sortBy || "timestamp");
    params.set("sort_order", sortOrder || "desc");
    if (filters?.q) params.set("q", filters.q);
    if (filters?.from_bot_id) params.set("from_bot_id", filters.from_bot_id);
    if (filters?.conversation_id) params.set("conversation_id", filters.conversation_id);
    if (filters?.content_hash) params.set("content_hash", filters.content_hash);

    const payload = await api<SendLedgerListResponse>(`/api/dashboard/send-ledger?${params.toString()}`);
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
        <div class="send-ledger-empty">
          <div class="detail-title">Send Ledger</div>
          <div class="detail-subtext">查看 Go agent-runtime 记录的发送审计，并验证防循环 recent 判定。</div>
        </div>
      `;
      return;
    }
    const record = item as unknown as SendLedgerRecord;
    container.innerHTML = `
      <div class="send-ledger-detail">
        <div class="send-ledger-detail-toolbar">
          <div>
            <div class="detail-title">${escapeHtml(record.short_hash || record.content_hash || "-")}</div>
            <div class="detail-subtext">${escapeHtml(record.from_bot_id || "-")} → ${escapeHtml(record.conversation_id || "-")}</div>
          </div>
        </div>

        <div class="send-ledger-section">
          <div class="send-ledger-grid">
            <div>
              <div class="send-ledger-label">From Bot</div>
              <div class="send-ledger-value mono">${escapeHtml(record.from_bot_id || "-")}</div>
            </div>
            <div>
              <div class="send-ledger-label">Conversation</div>
              <div class="send-ledger-value mono">${escapeHtml(record.conversation_id || "-")}</div>
            </div>
            <div>
              <div class="send-ledger-label">Timestamp</div>
              <div class="send-ledger-value mono">${escapeHtml(record.timestamp || "-")}</div>
            </div>
            <div>
              <div class="send-ledger-label">Content Hash</div>
              <div class="send-ledger-value mono">${escapeHtml(record.content_hash || "-")}</div>
            </div>
          </div>
        </div>

        <div class="send-ledger-section">
          <div class="detail-label">Recent Check</div>
          <div class="send-ledger-recent-form">
            <label class="send-ledger-filter">
              <span>发送账号</span>
              <input data-send-ledger-recent="from_bot_id" value="${escapeHtml(record.from_bot_id || "")}" />
            </label>
            <label class="send-ledger-filter">
              <span>会话</span>
              <input data-send-ledger-recent="conversation_id" value="${escapeHtml(record.conversation_id || "")}" />
            </label>
            <label class="send-ledger-filter">
              <span>Content Hash</span>
              <input data-send-ledger-recent="content_hash" value="${escapeHtml(record.content_hash || "")}" />
            </label>
            <label class="send-ledger-filter">
              <span>内容</span>
              <textarea data-send-ledger-recent="content" placeholder="也可以输入原始内容，后端会按同一规则算 hash"></textarea>
            </label>
            <label class="send-ledger-filter">
              <span>窗口秒数</span>
              <input data-send-ledger-recent="window_seconds" value="60" />
            </label>
            <div class="send-ledger-actions">
              <button class="primary" type="button" data-send-ledger-recent-run>检查 recent</button>
              <span class="muted-text" data-send-ledger-recent-status></span>
            </div>
            <div class="send-ledger-result" data-send-ledger-recent-result></div>
          </div>
        </div>
      </div>
    `;
    const button = container.querySelector<HTMLButtonElement>("[data-send-ledger-recent-run]");
    const status = container.querySelector<HTMLElement>("[data-send-ledger-recent-status]");
    const resultTarget = container.querySelector<HTMLElement>("[data-send-ledger-recent-result]");
    button?.addEventListener("click", async () => {
      const value = (key: string): string => {
        const element = container.querySelector<HTMLInputElement | HTMLTextAreaElement>(`[data-send-ledger-recent="${key}"]`);
        return element?.value.trim() || "";
      };
      if (!button || !resultTarget) return;
      button.disabled = true;
      if (status) status.textContent = "检查中";
      try {
        const result = await _checkRecent({
          fromBotId: value("from_bot_id"),
          conversationId: value("conversation_id"),
          contentHash: value("content_hash"),
          content: value("content"),
          windowSeconds: value("window_seconds"),
        });
        _renderRecentResult(resultTarget, result);
        if (status) status.textContent = "";
      } catch (error) {
        if (status) status.textContent = error instanceof Error ? error.message : String(error);
      } finally {
        button.disabled = false;
      }
    });
  },

  formatters: {
    "mono-time": (value: unknown) => escapeHtml(String(value || "-")),
  },
});

export {};
