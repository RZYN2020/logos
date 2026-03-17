const API_BASE = process.env.API_BASE || "http://localhost:8080/api/v1";
const USERNAME = process.env.ADMIN_USER || "admin";
const PASSWORD = process.env.ADMIN_PASS || "admin123";

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const request = async (path, { method = "GET", token, body } = {}) => {
  const headers = { "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = null;
  }
  if (!res.ok) {
    const msg = (json && (json.error || json.message)) || `${res.status} ${res.statusText}`;
    throw new Error(`${method} ${path}: ${msg}`);
  }
  return json;
};

const login = async () => {
  const data = await request("/auth/login", {
    method: "POST",
    body: { username: USERNAME, password: PASSWORD },
  });
  return data.token;
};

const seedRules = async (token) => {
  const rules = [
    {
      name: "drop-debug-logs",
      description: "丢弃 DEBUG 噪声日志",
      enabled: true,
      priority: 10,
      service: "api-gateway",
      component: "sdk",
      condition: { field: "level", operator: "eq", value: "DEBUG" },
      actions: [{ type: "drop" }],
    },
    {
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
    },
    {
      name: "mask-password-fields",
      description: "掩码用户密码字段",
      enabled: true,
      priority: 30,
      service: "user-service",
      component: "processor",
      condition: { field: "message", operator: "contains", value: "password=" },
      actions: [{ type: "mask", config: { field: "password" } }],
    },
    {
      name: "mark-payment-risk",
      description: "标记支付风险场景，辅助排查",
      enabled: true,
      priority: 80,
      service: "payment-service",
      component: "sdk",
      condition: {
        any: [
          { field: "message", operator: "contains", value: "risk" },
          { field: "message", operator: "contains", value: "fraud" },
        ],
      },
      actions: [{ type: "mark", config: { reason: "risk-signal" } }],
    },
  ];

  const created = [];
  for (const rule of rules) {
    const res = await request("/rules", { method: "POST", token, body: rule });
    created.push(res.id);
    await sleep(60);
  }
  return created;
};

const seedLogs = async (token) => {
  const now = Date.now();
  const ts = (offsetMs) => new Date(now - offsetMs).toISOString();

  const mk = (service, component, level, message, meta = {}) => ({
    service,
    component,
    timestamp: ts(Math.floor(Math.random() * 6 * 60 * 60 * 1000)),
    level,
    message,
    path: meta.path || "internal/http/handler.go",
    function: meta.function || "HandleRequest",
    line_number: meta.line || 142,
    trace_id: meta.trace_id || `tr_${Math.random().toString(16).slice(2, 10)}`,
    user_id: meta.user_id || `u_${Math.floor(Math.random() * 90 + 10)}`,
    fields: {
      env: "local",
      region: "cn",
      duration_ms: meta.duration_ms ?? Math.floor(Math.random() * 1200),
      ...meta.fields,
    },
  });

  const logs = [];

  for (let i = 0; i < 120; i++) {
    const ok = mk(
      "api-gateway",
      "sdk",
      "INFO",
      `request ok method=GET path=/api/v1/orders latency=${80 + (i % 40)}ms status=200`,
      { path: "gateway/router.go", function: "ServeHTTP", line: 87, duration_ms: 80 + (i % 40) }
    );
    logs.push(ok);
  }

  for (let i = 0; i < 48; i++) {
    const err = mk(
      "api-gateway",
      "sdk",
      "ERROR",
      `upstream timeout service=user-service timeout=5000ms attempt=${1 + (i % 3)}`,
      { path: "gateway/upstream.go", function: "CallUpstream", line: 212, duration_ms: 5000 }
    );
    logs.push(err);
  }

  for (let i = 0; i < 80; i++) {
    const lv = i % 9 === 0 ? "WARN" : "INFO";
    const msg =
      lv === "WARN"
        ? `slow query table=users duration=${900 + (i % 200)}ms rows=${20 + (i % 10)}`
        : `login ok user_id=u_${100 + (i % 20)} method=password`;
    logs.push(
      mk("user-service", "processor", lv, msg, {
        path: "user/auth.go",
        function: lv === "WARN" ? "QueryUser" : "Login",
        line: lv === "WARN" ? 310 : 154,
        duration_ms: lv === "WARN" ? 900 + (i % 200) : 30 + (i % 60),
      })
    );
  }

  for (let i = 0; i < 60; i++) {
    const lv = i % 10 === 0 ? "ERROR" : "INFO";
    const msg =
      lv === "ERROR"
        ? `payment failed risk=high code=RISK_${1000 + (i % 6)} user_id=u_${50 + (i % 10)}`
        : `payment ok amount=${(i % 7) + 1}9.90 currency=CNY channel=card`;
    logs.push(
      mk("payment-service", "sdk", lv, msg, {
        path: "payment/charge.go",
        function: lv === "ERROR" ? "ChargeWithRiskCheck" : "Charge",
        line: lv === "ERROR" ? 488 : 221,
        duration_ms: lv === "ERROR" ? 420 + (i % 220) : 180 + (i % 90),
      })
    );
  }

  const res = await request("/logs/batch", { method: "POST", token, body: { logs } });
  return { ingested: res.ingested || logs.length };
};

const main = async () => {
  console.log(`[seed-demo] API_BASE=${API_BASE}`);
  const token = await login();
  console.log(`[seed-demo] login ok; token=${token.slice(0, 12)}...`);

  const ruleIds = await seedRules(token);
  console.log(`[seed-demo] rules created: ${ruleIds.length}`);

  const { ingested } = await seedLogs(token);
  console.log(`[seed-demo] logs ingested: ${ingested}`);

  console.log("[seed-demo] done");
};

main().catch((e) => {
  console.error(e);
  process.exit(1);
});

