# Log Analyzer 性能测试脚本
# 使用 k6 进行负载测试

import http from 'k/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');

// 测试配置
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // 热身阶段：10 个用户
    { duration: '1m', target: 50 },    // 负载阶段：50 个用户
    { duration: '2m', target: 100 },   // 压力阶段：100 个用户
    { duration: '1m', target: 200 },   // 峰值阶段：200 个用户
    { duration: '30s', target: 0 },    // 冷却阶段
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% 的请求应该小于 500ms
    http_req_failed: ['rate<0.01'],    // 错误率应该小于 1%
    errors: ['rate<0.1'],              // 自定义错误率小于 10%
  },
};

// 测试数据
const BASE_URL = 'http://localhost:8080/api/v1';
let authToken = '';

// 初始化 - 登录获取 token
export function setup() {
  const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    username: 'admin',
    password: 'admin123',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  const success = check(loginRes, {
    'login successful': (r) => r.status === 200,
  });

  if (success) {
    authToken = loginRes.json('token');
  }

  return { token: authToken };
}

// 主要测试场景
export default function (data) {
  const token = data.token;
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  // 场景 1: 获取规则列表
  getRules(headers);

  sleep(1);

  // 场景 2: 创建规则
  const ruleId = createRule(headers);

  sleep(1);

  // 场景 3: 获取规则详情
  if (ruleId) {
    getRule(ruleId, headers);
  }

  sleep(1);

  // 场景 4: 验证规则
  if (ruleId) {
    validateRule(ruleId, headers);
  }

  sleep(1);

  // 场景 5: 测试规则
  if (ruleId) {
    testRule(ruleId, headers);
  }

  sleep(1);

  // 场景 6: 日志分析
  minePatterns(headers);
}

function getRules(headers) {
  const res = http.get(`${BASE_URL}/rules`, { headers });
  const success = check(res, {
    'get rules status 200': (r) => r.status === 200,
    'get rules duration OK': (r) => r.timings.duration < 200,
  });
  errorRate.add(!success);
}

function createRule(headers) {
  const rule = {
    name: `perf-test-rule-${Date.now()}`,
    description: 'Performance test rule',
    enabled: true,
    priority: 1,
    conditions: [
      { field: 'level', operator: '=', value: 'ERROR' },
    ],
    actions: [
      { type: 'filter', config: { sampling: 1.0 } },
    ],
  };

  const res = http.post(`${BASE_URL}/rules`, JSON.stringify(rule), { headers });
  const success = check(res, {
    'create rule status 201': (r) => r.status === 201,
  });
  errorRate.add(!success);

  if (res.status === 201) {
    return res.json('id');
  }
  return null;
}

function getRule(ruleId, headers) {
  const res = http.get(`${BASE_URL}/rules/${ruleId}`, { headers });
  const success = check(res, {
    'get rule status 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

function validateRule(ruleId, headers) {
  const res = http.post(`${BASE_URL}/rules/${ruleId}/validate`, null, { headers });
  const success = check(res, {
    'validate rule status 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

function testRule(ruleId, headers) {
  const res = http.post(`${BASE_URL}/rules/${ruleId}/test`, null, { headers });
  const success = check(res, {
    'test rule status 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

function minePatterns(headers) {
  const payload = {
    logs: [
      { timestamp: new Date().toISOString(), level: 'ERROR', service: 'api', message: 'Test error' },
      { timestamp: new Date().toISOString(), level: 'INFO', service: 'api', message: 'Test info' },
    ],
  };

  const res = http.post(`${BASE_URL}/analysis/mine`, JSON.stringify(payload), { headers });
  const success = check(res, {
    'mine patterns status 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

// 清理函数
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    './perf-results.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  return `
=====================================
性能测试结果
=====================================
总请求数：${data.metrics.http_reqs.values.count}
平均响应时间：${data.metrics.http_req_duration.values.avg.toFixed(2)}ms
P95 响应时间：${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms
P99 响应时间：${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms
错误率：${((data.metrics.http_req_failed.values.rate || 0) * 100).toFixed(2)}%
=====================================
`;
}
