// API 客户端
import type {
  ApiResponse,
  Rule,
} from "./types";

const API_BASE = "http://localhost:8080/api/v1";

export const apiClient = {
  // ============ 规则 API ============

  // 获取所有规则（可根据服务和组件过滤）
  async listRules(service?: string, component?: 'sdk' | 'processor'): Promise<Rule[]> {
    let url = `${API_BASE}/rules`;
    const params = new URLSearchParams();
    if (service) params.append('service', service);
    if (component) params.append('component', component);
    const queryString = params.toString();
    if (queryString) url += `?${queryString}`;

    const res = await fetch(url);
    const data: ApiResponse<Rule[] | { rules: Rule[] }> = await res.json();
    // 处理两种可能的响应格式：直接返回数组或 { rules: Rule[] }
    if (Array.isArray(data.data)) {
      return data.data;
    }
    return data.data?.rules || [];
  },

  // 获取单个规则
  async getRule(id: string): Promise<Rule | null> {
    const res = await fetch(`${API_BASE}/rules/${id}`);
    if (res.status === 404) return null;
    const data: ApiResponse<Rule> = await res.json();
    return data.data || null;
  },

  // 创建规则
  async createRule(rule: Omit<Rule, "id" | "version" | "created_at" | "updated_at">): Promise<{ id: string }> {
    const res = await fetch(`${API_BASE}/rules`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    });
    const data: ApiResponse<{ id: string }> = await res.json();
    return data.data || { id: "" };
  },

  // 更新规则
  async updateRule(id: string, rule: Partial<Rule>): Promise<void> {
    const res = await fetch(`${API_BASE}/rules/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(rule),
    });
    if (!res.ok) throw new Error("Failed to update rule");
  },

  // 删除规则
  async deleteRule(id: string): Promise<void> {
    await fetch(`${API_BASE}/rules/${id}`, {
      method: "DELETE",
    });
  },
};
