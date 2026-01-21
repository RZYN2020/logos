// API 类型定义

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

export interface StrategyVersion {
  version: string;
  created_at: string;
  author: string;
  comment?: string;
}

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
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
  etcd: string;
}
