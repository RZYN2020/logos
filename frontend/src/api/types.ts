// API 类型定义

export interface Rule {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  priority: number;
  conditions: Condition[];
  actions: Action[];
  version: number;
  created_at: string;
  updated_at: string;
}

export interface Condition {
  id: string;
  rule_id: string;
  field: string;
  operator: string;
  value: unknown;
}

export interface Action {
  id: string;
  rule_id: string;
  type: string; // filter/drop/transform
  config?: Record<string, unknown>;
}

export interface RuleVersion {
  id: string;
  rule_id: string;
  version: number;
  content: Record<string, unknown>;
  author: string;
  comment?: string;
  created_at: string;
}

// 兼容旧版 Strategy 类型
export interface Strategy {
  id: string;
  name: string;
  description: string;
  rules: StrategyRule[];
  version: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  author: string;
  metadata?: Record<string, unknown>;
}

export interface StrategyRule {
  condition: Record<string, unknown>;
  action: Record<string, unknown>;
}

export interface ApiResponse<T = unknown> {
  code?: number;
  message?: string;
  data?: T;
}

export interface SystemInfo {
  system: string;
  version: string;
  etcd_version: string;
  uptime: string;
}

export interface HealthCheck {
  status: string;
  etcd?: string;
}
