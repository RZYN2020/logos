import type { Rule, LogLineStat, LogPatternStat, LogReport } from "./types";

export const isMockEnabled = () => String(import.meta.env.VITE_USE_MOCK || "") === "1";

type MockLog = {
  id: number;
  service: string;
  component?: string;
  timestamp: string;
  level: string;
  message: string;
  path?: string;
  function?: string;
  line_number?: number;
  trace_id?: string;
  user_id?: string;
  fields?: Record<string, unknown>;
};

const makeId = () => {
  const c = globalThis.crypto as Crypto | undefined;
  if (c?.randomUUID) return c.randomUUID();
  return `r_${Math.random().toString(16).slice(2)}_${Date.now()}`;
};

const state: {
  seeded: boolean;
  rules: Rule[];
  logs: MockLog[];
  nextLogId: number;
} = {
  seeded: false,
  rules: [],
  logs: [],
  nextLogId: 1,
};

const iso = (offsetMs = 0) => new Date(Date.now() - offsetMs).toISOString();

const seed = () => {
  if (state.seeded) return;
  state.seeded = true;

  state.rules.push(
    {
      id: makeId(),
      name: "drop-debug-logs",
      description: "丢弃 DEBUG 噪声日志",
      enabled: true,
      priority: 10,
      service: "api-gateway",
      component: "sdk",
      condition: { field: "level", operator: "eq", value: "DEBUG" },
      actions: [{ type: "drop" }],
      created_at: iso(4 * 60 * 60 * 1000),
      updated_at: iso(2 * 60 * 60 * 1000),
    },
    {
      id: makeId(),
      name: "sample-timeout-errors",
      description: "对 timeout 错误进行 20% 采样",
      enabled: true,
      priority: 50,
      service: "api-gateway",
      component: "sdk",
      condition: {
        all: [
          { field: "level", operator: "eq", value: "ERROR" },
          { field: "message", operator: "contains", value: "timeout" },
        ],
      },
      actions: [{ type: "sample", config: { rate: 0.2 } }],
      created_at: iso(6 * 60 * 60 * 1000),
      updated_at: iso(1 * 60 * 60 * 1000),
    },
    {
      id: makeId(),
      name: "mask-password-fields",
      description: "掩码用户密码字段",
      enabled: true,
      priority: 30,
      service: "user-service",
      component: "processor",
      condition: { field: "message", operator: "contains", value: "password=" },
      actions: [{ type: "mask", config: { field: "password" } }],
      created_at: iso(10 * 60 * 60 * 1000),
      updated_at: iso(3 * 60 * 60 * 1000),
    }
  );

  const push = (log: Omit<MockLog, "id">) => {
    state.logs.push({ id: state.nextLogId++, ...log });
  };

  for (let i = 0; i < 140; i++) {
    push({
      service: "api-gateway",
      component: "sdk",
      timestamp: iso(Math.floor(Math.random() * 6 * 60 * 60 * 1000)),
      level: "INFO",
      message: `request ok method=GET path=/api/v1/orders latency=${80 + (i % 40)}ms status=200`,
      path: "gateway/router.go",
      function: "ServeHTTP",
      line_number: 87,
      trace_id: `tr_${Math.random().toString(16).slice(2, 10)}`,
      user_id: `u_${Math.floor(Math.random() * 90 + 10)}`,
      fields: { env: "local", duration_ms: 80 + (i % 40) },
    });
  }

  for (let i = 0; i < 60; i++) {
    push({
      service: "api-gateway",
      component: "sdk",
      timestamp: iso(Math.floor(Math.random() * 6 * 60 * 60 * 1000)),
      level: "ERROR",
      message: `upstream timeout service=user-service timeout=5000ms attempt=${1 + (i % 3)}`,
      path: "gateway/upstream.go",
      function: "CallUpstream",
      line_number: 212,
      trace_id: `tr_${Math.random().toString(16).slice(2, 10)}`,
      user_id: `u_${Math.floor(Math.random() * 90 + 10)}`,
      fields: { env: "local", duration_ms: 5000 },
    });
  }

  for (let i = 0; i < 90; i++) {
    const level = i % 10 === 0 ? "WARN" : "INFO";
    push({
      service: "user-service",
      component: "processor",
      timestamp: iso(Math.floor(Math.random() * 6 * 60 * 60 * 1000)),
      level,
      message:
        level === "WARN"
          ? `slow query table=users duration=${900 + (i % 200)}ms rows=${20 + (i % 10)}`
          : `login ok user_id=u_${100 + (i % 20)} method=password`,
      path: "user/auth.go",
      function: level === "WARN" ? "QueryUser" : "Login",
      line_number: level === "WARN" ? 310 : 154,
      trace_id: `tr_${Math.random().toString(16).slice(2, 10)}`,
      user_id: `u_${100 + (i % 20)}`,
      fields: { env: "local", duration_ms: level === "WARN" ? 900 + (i % 200) : 30 + (i % 60) },
    });
  }
};

const normalizePattern = (s: string) => {
  return s
    .replace(/\b0x[0-9a-fA-F]+\b/g, "{hex}")
    .replace(/\b\d{2,}\b/g, "{n}")
    .replace(/u_\d+/g, "u_{n}")
    .replace(/tr_[0-9a-f]+/g, "tr_{id}");
};

const topLines = (service: string): { total_logs: number; top_lines: LogLineStat[] } => {
  seed();
  const logs = state.logs.filter((l) => l.service === service);
  const total = logs.length;
  const map = new Map<string, { key: string; file?: string; func?: string; line: number; count: number }>();
  for (const l of logs) {
    const file = l.path || "";
    const func = l.function || "";
    const line = l.line_number || 0;
    const key = `${file}::${func}::${line}`;
    const cur = map.get(key) || { key, file, func, line, count: 0 };
    cur.count++;
    map.set(key, cur);
  }
  const arr = Array.from(map.values()).sort((a, b) => b.count - a.count).slice(0, 10);
  return {
    total_logs: total,
    top_lines: arr.map((r) => ({
      file: r.file,
      function: r.func,
      line_number: r.line,
      count: r.count,
      percentage: total ? Number(((r.count / total) * 100).toFixed(1)) : 0,
    })),
  };
};

const topPatterns = (service: string): { total_logs: number; top_patterns: LogPatternStat[] } => {
  seed();
  const logs = state.logs.filter((l) => l.service === service);
  const total = logs.length;
  const map = new Map<string, { pattern: string; count: number; samples: string[] }>();
  for (const l of logs) {
    const p = normalizePattern(l.message || "");
    const cur = map.get(p) || { pattern: p, count: 0, samples: [] };
    cur.count++;
    if (cur.samples.length < 3) cur.samples.push(l.message);
    map.set(p, cur);
  }
  const arr = Array.from(map.values()).sort((a, b) => b.count - a.count).slice(0, 10);
  return {
    total_logs: total,
    top_patterns: arr.map((p) => ({
      pattern: p.pattern,
      count: p.count,
      percentage: total ? Number(((p.count / total) * 100).toFixed(1)) : 0,
      sample_logs: p.samples,
    })),
  };
};

export const mockApi = {
  async login(username: string, password: string): Promise<{ token: string }> {
    seed();
    if (username === "admin" && password === "admin123") return { token: "mock-token" };
    throw new Error("invalid username or password");
  },

  async getCurrentUser(): Promise<{ user_id: string; username: string; roles: string[] } | null> {
    seed();
    return { user_id: "admin-001", username: "admin", roles: ["admin", "user"] };
  },

  async listRules(service?: string, component?: "sdk" | "processor"): Promise<Rule[]> {
    seed();
    return state.rules.filter((r) => (!service || r.service === service) && (!component || r.component === component));
  },

  async getRule(id: string): Promise<Rule | null> {
    seed();
    return state.rules.find((r) => r.id === id) || null;
  },

  async createRule(rule: Omit<Rule, "id" | "version" | "created_at" | "updated_at">): Promise<{ id: string }> {
    seed();
    const id = makeId();
    state.rules.unshift({
      ...rule,
      id,
      created_at: iso(),
      updated_at: iso(),
    });
    return { id };
  },

  async updateRule(id: string, patch: Partial<Rule>): Promise<void> {
    seed();
    const idx = state.rules.findIndex((r) => r.id === id);
    if (idx < 0) throw new Error("rule not found");
    state.rules[idx] = { ...state.rules[idx], ...patch, updated_at: iso() };
  },

  async deleteRule(id: string): Promise<void> {
    seed();
    state.rules = state.rules.filter((r) => r.id !== id);
  },

  async getTopLines(
    service: string,
    _options?: { component?: string; from?: string; to?: string; limit?: number }
  ): Promise<{ total_logs: number; top_lines: LogLineStat[] }> {
    return topLines(service);
  },

  async getTopPatterns(
    service: string,
    _options?: { component?: string; limit?: number }
  ): Promise<{ total_logs: number; top_patterns: LogPatternStat[] }> {
    return topPatterns(service);
  },

  async getReport(service: string, _options?: { component?: string }): Promise<LogReport> {
    const lines = topLines(service);
    const patterns = topPatterns(service);
    return {
      service,
      total_logs: lines.total_logs,
      time_range: { from: iso(24 * 60 * 60 * 1000), to: iso() },
      top_lines: lines.top_lines,
      top_patterns: patterns.top_patterns,
    };
  },

  async ingestLog(_log: unknown): Promise<{ id: number }> {
    seed();
    return { id: state.nextLogId++ };
  },

  async ingestLogs(_logs: unknown): Promise<{ ingested: number }> {
    seed();
    return { ingested: 0 };
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
    seed();
    let logs = state.logs.slice();
    if (query.service) logs = logs.filter((l) => l.service === query.service);
    if (query.component) logs = logs.filter((l) => l.component === query.component);
    if (query.level) logs = logs.filter((l) => l.level === query.level);
    logs.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1));
    const total = logs.length;
    const offset = query.offset || 0;
    const limit = query.limit || 100;
    return { total, logs: logs.slice(offset, offset + limit) };
  },
};
