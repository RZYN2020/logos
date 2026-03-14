import { useState, useEffect } from "react";
import type { Rule, Condition, Action } from "../api/types";
import { apiClient } from "../api/client";

interface RuleListProps {
  service: string;
  component: 'sdk' | 'processor';
  onEdit: (id: string) => void;
  onCreate: () => void;
}

export default function RuleList({ service, component, onEdit, onCreate }: RuleListProps) {
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadRules = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiClient.listRules(service, component);
      setRules(data);
    } catch (err) {
      setError("加载规则失败");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("确定删除此规则？")) return;
    try {
      await apiClient.deleteRule(id);
      await loadRules();
    } catch (err) {
      setError("删除失败");
    }
  };

  useEffect(() => {
    loadRules();
  }, [service, component]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-500">加载规则中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* 页面标题 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">规则配置</h1>
          <p className="text-sm text-gray-500 mt-1">
            管理 {service} ({component}) 的日志处理规则
          </p>
        </div>
        <button
          onClick={onCreate}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium"
        >
          ➕ 新建规则
        </button>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg">
          {error}
        </div>
      )}

      {/* 规则列表 */}
      <div className="bg-white shadow-sm border border-gray-200 rounded-xl overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                名称
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                描述
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                状态
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                条件
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                动作
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                操作
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {rules.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-6 py-12 text-center">
                  <div className="text-gray-400 mb-4">
                    <svg className="mx-auto h-12 w-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                  </div>
                  <p className="text-gray-500 mb-4">暂无规则</p>
                  <button
                    onClick={onCreate}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium"
                  >
                    创建第一条规则
                  </button>
                </td>
              </tr>
            ) : (
              rules.map((rule) => (
                <tr key={rule.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">
                      {rule.name}
                    </div>
                    <div className="text-xs text-gray-500">
                      {rule.id}
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="text-sm text-gray-900 max-w-xs truncate">
                      {rule.description || '-'}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    {rule.enabled ? (
                      <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                        启用
                      </span>
                    ) : (
                      <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800">
                        禁用
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4">
                    <div className="text-xs text-gray-500 max-w-xs truncate">
                      {getConditionSummary(rule.condition)}
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="text-xs text-gray-500">
                      {rule.actions.map((a: Action) => a.type).join(', ')}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button
                      onClick={() => onEdit(rule.id)}
                      className="text-blue-600 hover:text-blue-800 mr-3"
                    >
                      编辑
                    </button>
                    <button
                      onClick={() => handleDelete(rule.id)}
                      className="text-red-600 hover:text-red-800"
                    >
                      删除
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// 获取条件摘要
function getConditionSummary(condition: Condition): string {
  if (condition.all) {
    return `AND (${condition.all.length} 条件)`;
  }
  if (condition.any) {
    return `OR (${condition.any.length} 条件)`;
  }
  if (condition.not) {
    return `NOT (${condition.not.field || 'complex'})`;
  }
  if (condition.field && condition.operator) {
    return `${condition.field} ${condition.operator} ${JSON.stringify(condition.value)}`;
  }
  return '复杂条件';
}
