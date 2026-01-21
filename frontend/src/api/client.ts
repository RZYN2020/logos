// API 客端
import type {
  ApiResponse,
  Strategy,
  StrategyVersion,
  SystemInfo,
  HealthCheck,
} from "./types";

const API_BASE = "http://localhost:8080/api/v1";

export const apiClient = {
  // 获取所有策略
  async listStrategies(): Promise<Strategy[]> {
    const res = await fetch(`${API_BASE}/strategies`);
    const data: ApiResponse<Strategy[]> = await res.json();
    return data.data || [];
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
