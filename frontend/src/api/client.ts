// API 客户端
import type {
  ApiResponse,
  Rule,
  LogReport,
  LogLineStat,
  LogPatternStat,
} from "./types";

const API_BASE = "http://localhost:8080/api/v1";

// 认证 token（用于需要认证的请求）
let authToken: string | null = null;

export const setAuthToken = (token: string | null) => {
  authToken = token;
};

const getHeaders = () => {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (authToken) {
    headers["Authorization"] = `Bearer ${authToken}`;
  }
  return headers;
};

export const apiClient = {
  // ============ 认证 API ============

  async login(username: string, password: string): Promise<{ token: string }> {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.message || "Login failed");
    }
    const token = data.token || data.data?.token || "";
    if (token) {
      setAuthToken(token);
    }
    return { token };
  },

  // ============ 规则 API ============

  // 获取所有规则（可根据服务和组件过滤）
  async listRules(service?: string, component?: 'sdk' | 'processor'): Promise<Rule[]> {
    let url = `${API_BASE}/rules`;
    const params = new URLSearchParams();
    if (service) params.append('service', service);
    if (component) params.append('component', component);
    const queryString = params.toString();
    if (queryString) url += `?${queryString}`;

    const res = await fetch(url, { headers: getHeaders() });
    const data: ApiResponse<Rule[] | { rules: Rule[] }> = await res.json();
    if (Array.isArray(data.data)) {
      return data.data;
    }
    return data.data?.rules || [];
  },

  // 获取单个规则
  async getRule(id: string): Promise<Rule | null> {
    const res = await fetch(`${API_BASE}/rules/${id}`, { headers: getHeaders() });
    if (res.status === 404) return null;
    const data: ApiResponse<Rule> = await res.json();
    return data.data || null;
  },

  // 创建规则
  async createRule(rule: Omit<Rule, "id" | "version" | "created_at" | "updated_at">): Promise<{ id: string }> {
    const res = await fetch(`${API_BASE}/rules`, {
      method: "POST",
      headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    });
    const data: ApiResponse<{ id: string }> = await res.json();
    return data.data || { id: "" };
  },

  // 更新规则
  async updateRule(id: string, rule: Partial<Rule>): Promise<void> {
    const res = await fetch(`${API_BASE}/rules/${id}`, {
      method: "PUT",
      headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    });
    if (!res.ok) throw new Error("Failed to update rule");
  },

  // 删除规则
  async deleteRule(id: string): Promise<void> {
    await fetch(`${API_BASE}/rules/${id}`, {
      method: "DELETE",
      headers: getHeaders(),
    });
  },

  // ============ 日志报告 API ============

  // 获取 TOP 行号统计
  async getTopLines(service: string, options?: { component?: string; from?: string; to?: string; limit?: number }): Promise<{ total_logs: number; top_lines: LogLineStat[] }> {
    let url = `${API_BASE}/report/${encodeURIComponent(service)}/top-lines`;
    const params = new URLSearchParams();
    if (options?.component) params.append('component', options.component);
    if (options?.from) params.append('from', options.from);
    if (options?.to) params.append('to', options.to);
    if (options?.limit) params.append('limit', String(options.limit));
    const queryString = params.toString();
    if (queryString) url += `?${queryString}`;

    const res = await fetch(url, { headers: getHeaders() });
    const data = await res.json();
    return {
      total_logs: data.total_logs || 0,
      top_lines: data.top_lines || [],
    };
  },

  // 获取 TOP 模式统计
  async getTopPatterns(service: string, options?: { component?: string; limit?: number }): Promise<{ total_logs: number; top_patterns: LogPatternStat[] }> {
    let url = `${API_BASE}/report/${encodeURIComponent(service)}/top-patterns`;
    const params = new URLSearchParams();
    if (options?.component) params.append('component', options.component);
    if (options?.limit) params.append('limit', String(options.limit));
    const queryString = params.toString();
    if (queryString) url += `?${queryString}`;

    const res = await fetch(url, { headers: getHeaders() });
    const data = await res.json();
    return {
      total_logs: data.total_logs || 0,
      top_patterns: data.top_patterns || [],
    };
  },

  // 获取完整日志报告
  async getReport(service: string, options?: { component?: string }): Promise<LogReport> {
    let url = `${API_BASE}/report/${encodeURIComponent(service)}`;
    const params = new URLSearchParams();
    if (options?.component) params.append('component', options.component);
    const queryString = params.toString();
    if (queryString) url += `?${queryString}`;

    const res = await fetch(url, { headers: getHeaders() });
    const data = await res.json();
    return {
      service: data.service || service,
      total_logs: data.total_logs || 0,
      time_range: { from: "", to: "" },
      top_lines: data.top_lines || [],
      top_patterns: data.top_patterns || [],
    };
  },

  // ============ 日志摄入 API ============

  async ingestLog(log: {
    service: string;
    component?: string;
    timestamp?: string;
    level: string;
    message: string;
    path?: string;
    function?: string;
    line_number?: number;
    trace_id?: string;
    user_id?: string;
    fields?: Record<string, unknown>;
  }): Promise<{ id: number }> {
    const res = await fetch(`${API_BASE}/logs`, {
      method: "POST",
      headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify(log),
    });
    const data = await res.json();
    return { id: data.id || 0 };
  },

  async ingestLogs(logs: Array<{
    service: string;
    component?: string;
    timestamp?: string;
    level: string;
    message: string;
    path?: string;
    function?: string;
    line_number?: number;
    trace_id?: string;
    user_id?: string;
    fields?: Record<string, unknown>;
  }>): Promise<{ ingested: number }> {
    const res = await fetch(`${API_BASE}/logs/batch`, {
      method: "POST",
      headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify({ logs }),
    });
    const data = await res.json();
    return { ingested: data.ingested || 0 };
  },

  async queryLogs(query: {
    service?: string;
    component?: string;
    level?: string;
    from?: string;
    to?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ total: number; logs: unknown[] }> {
    const res = await fetch(`${API_BASE}/logs/query`, {
      method: "POST",
      headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify(query),
    });
    const data = await res.json();
    return {
      total: data.total || 0,
      logs: data.logs || [],
    };
  },
};
