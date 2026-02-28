// API 客户端
import type {
  ApiResponse,
  Strategy,
  Rule,
  RuleVersion,
  SystemInfo,
  HealthCheck,
} from "./types";

const API_BASE = "http://localhost:8080/api/v1";

export const apiClient = {
  // ============ 策略 API (兼容旧版) ============

  // 获取所有策略
  async listStrategies(): Promise<Strategy[]> {
    const res = await fetch(`${API_BASE}/strategies`);
    const data: ApiResponse<Strategy[]> = await res.json();
    return (data.data || []) as Strategy[];
  },

  // 创建策略
  async createStrategy(strategy: Omit<Strategy, "id" | "version" | "created_at" | "updated_at">): Promise<{ id: string; version: string }> {
    const res = await fetch(`${API_BASE}/strategies`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(strategy),
    });
    const data: ApiResponse<{ id: string; version: string }> = await res.json();
    return data.data || { id: "", version: "" };
  },

  // 获取单个策略
  async getStrategy(id: string): Promise<Strategy | null> {
    const res = await fetch(`${API_BASE}/strategies/${id}`);
    if (res.status === 404) return null;
    const data: ApiResponse<Strategy> = await res.json();
    return data.data || null;
  },

  // 更新策略
  async updateStrategy(id: string, strategy: Partial<Strategy>): Promise<{ id: string; version: string }> {
    const res = await fetch(`${API_BASE}/strategies/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(strategy),
    });
    const data: ApiResponse<{ id: string; version: string }> = await res.json();
    return data.data || { id, version: "" };
  },

  // 删除策略
  async deleteStrategy(id: string): Promise<void> {
    await fetch(`${API_BASE}/strategies/${id}`, {
      method: "DELETE",
    });
  },

  // 获取策略历史
  async getStrategyHistory(id: string): Promise<StrategyVersion[]> {
    const res = await fetch(`${API_BASE}/strategies/${id}/history`);
    const data: ApiResponse<StrategyVersion[]> = await res.json();
    return data.data || [];
  },

  // ============ 规则 API (新版) ============

  // 获取所有规则
  async listRules(): Promise<Rule[]> {
    const res = await fetch(`${API_BASE}/rules`);
    const data: ApiResponse<Rule[]> = await res.json();
    return data.data || [];
  },

  // 获取单个规则
  async getRule(id: string): Promise<Rule | null> {
    const res = await fetch(`${API_BASE}/rules/${id}`);
    if (res.status === 404) return null;
    const data: ApiResponse<Rule> = await res.json();
    return data.data || null;
  },

  // 创建规则
  async createRule(rule: Omit<Rule, "id" | "version" | "created_at" | "updated_at">): Promise<{ id: string; version: number }> {
    const res = await fetch(`${API_BASE}/rules`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    });
    const data: ApiResponse<{ id: string; version: number }> = await res.json();
    return data.data || { id: "", version: 0 };
  },

  // 更新规则
  async updateRule(id: string, rule: Partial<Rule>): Promise<{ id: string; version: number }> {
    const res = await fetch(`${API_BASE}/rules/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    });
    const data: ApiResponse<{ id: string; version: number }> = await res.json();
    return data.data || { id, version: 0 };
  },

  // 删除规则
  async deleteRule(id: string): Promise<void> {
    await fetch(`${API_BASE}/rules/${id}`, {
      method: "DELETE",
    });
  },

  // 获取规则历史
  async getRuleHistory(id: string): Promise<RuleVersion[]> {
    const res = await fetch(`${API_BASE}/rules/${id}/history`);
    const data: ApiResponse<RuleVersion[]> = await res.json();
    return data.data || [];
  },

  // 回滚规则版本
  async rollbackRule(id: string, version: number): Promise<{ id: string; version: number }> {
    const res = await fetch(`${API_BASE}/rules/${id}/rollback/${version}`, {
      method: "POST",
    });
    const data: ApiResponse<{ id: string; version: number }> = await res.json();
    return data.data || { id: "", version: 0 };
  },

  // 验证规则
  async validateRule(id: string): Promise<{ valid: boolean; errors?: string[] }> {
    const res = await fetch(`${API_BASE}/rules/${id}/validate`, {
      method: "POST",
    });
    const data = await res.json();
    return data;
  },

  // 测试规则
  async testRule(id: string): Promise<{ matched: boolean; test_data?: Record<string, unknown> }> {
    const res = await fetch(`${API_BASE}/rules/${id}/test`, {
      method: "POST",
    });
    const data = await res.json();
    return data;
  },

  // 导出规则
  async exportRules(): Promise<Rule[]> {
    const res = await fetch(`${API_BASE}/rules/export`);
    const data: ApiResponse<{ rules: Rule[] }> = await res.json();
    return data.data?.rules || [];
  },

  // 导入规则
  async importRules(rules: Rule[]): Promise<{ imported: number }> {
    const res = await fetch(`${API_BASE}/rules/import`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ rules }),
    });
    const data: ApiResponse<{ imported: number }> = await res.json();
    return data.data || { imported: 0 };
  },

  // ============ 系统 API ============

  // 获取系统信息
  async getSystemInfo(): Promise<SystemInfo | null> {
    const res = await fetch(`${API_BASE}/info`);
    const data: ApiResponse<SystemInfo> = await res.json();
    return data.data || null;
  },

  // 健康检查
  async healthCheck(): Promise<HealthCheck | null> {
    try {
      const res = await fetch(`${API_BASE}/health`);
      if (!res.ok) return null;
      const data: ApiResponse<HealthCheck> = await res.json();
      return data.data || null;
    } catch {
      return null;
    }
  },
};
