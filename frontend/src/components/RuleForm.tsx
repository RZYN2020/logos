import { useState } from "react";
import type { Rule, Condition, Action, ConditionOperator } from "../api/types";
import { apiClient } from "../api/client";

interface Props {
  rule?: Rule;
  onSave: () => void;
  onCancel: () => void;
}

// 操作符选项
const OPERATOR_OPTIONS: { value: ConditionOperator; label: string }[] = [
  { value: 'eq', label: '等于 (eq)' },
  { value: 'ne', label: '不等于 (ne)' },
  { value: 'gt', label: '大于 (gt)' },
  { value: 'lt', label: '小于 (lt)' },
  { value: 'ge', label: '大于等于 (ge)' },
  { value: 'le', label: '小于等于 (le)' },
  { value: 'contains', label: '包含 (contains)' },
  { value: 'starts_with', label: '开始于 (starts_with)' },
  { value: 'ends_with', label: '结束于 (ends_with)' },
  { value: 'matches', label: '正则匹配 (matches)' },
  { value: 'in', label: '在集合中 (in)' },
  { value: 'not_in', label: '不在集合中 (not_in)' },
  { value: 'exists', label: '字段存在 (exists)' },
  { value: 'not_exists', label: '字段不存在 (not_exists)' },
];

export default function RuleForm({ rule, onSave, onCancel }: Props) {
  const [name, setName] = useState(rule?.name || "");
  const [description, setDescription] = useState(rule?.description || "");
  const [enabled, setEnabled] = useState(rule?.enabled ?? true);
  const [conditionJson, setConditionJson] = useState<string>(
    rule?.condition ? JSON.stringify(rule.condition, null, 2) : JSON.stringify({
      field: "level",
      operator: "eq",
      value: "ERROR"
    }, null, 2)
  );
  const [actionsJson, setActionsJson] = useState<string>(
    rule?.actions ? JSON.stringify(rule.actions, null, 2) : JSON.stringify([{
      type: "drop"
    }], null, 2)
  );
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!name.trim()) {
      setError("规则名称不能为空");
      return;
    }

    try {
      setSaving(true);

      let parsedCondition: Condition;
      try {
        parsedCondition = JSON.parse(conditionJson);
      } catch (err) {
        setError("条件 JSON 格式错误：" + (err as Error).message);
        setSaving(false);
        return;
      }

      let parsedActions: Action[];
      try {
        parsedActions = JSON.parse(actionsJson);
      } catch (err) {
        setError("动作 JSON 格式错误：" + (err as Error).message);
        setSaving(false);
        return;
      }

      if (rule) {
        await apiClient.updateRule(rule.id, {
          name,
          description,
          enabled,
          condition: parsedCondition,
          actions: parsedActions,
        });
      } else {
        await apiClient.createRule({
          name,
          description,
          enabled,
          condition: parsedCondition,
          actions: parsedActions,
        });
      }

      onSave();
    } catch (err) {
      setError("保存失败：" + (err as Error).message);
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  const loadTemplate = (type: 'condition' | 'action', template: string) => {
    if (type === 'condition') {
      setConditionJson(template);
    } else {
      setActionsJson(template);
    }
  };

  return (
    <div className="px-4 py-6">
      <div className="mb-4">
        <button onClick={onCancel} className="text-gray-600 hover:text-gray-900 mb-4">
          ← 返回列表
        </button>
        <h2 className="text-2xl font-bold text-gray-900">
          {rule ? "编辑规则" : "新建规则"}
        </h2>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="bg-white shadow rounded-lg p-6 space-y-6">
        {/* 基本信息 */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              规则名称
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              placeholder="例如：drop-debug-logs"
              required
            />
          </div>

          <div className="flex items-end">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
                className="mr-2 h-4 w-4"
              />
              <span className="text-sm font-medium text-gray-700">启用规则</span>
            </label>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            规则描述
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            rows={2}
            placeholder="描述规则的作用和场景"
          />
        </div>

        {/* 条件配置 */}
        <div>
          <div className="flex justify-between items-center mb-2">
            <label className="block text-sm font-medium text-gray-700">
              条件配置 (JSON)
            </label>
            <div className="space-x-2">
              <button
                type="button"
                onClick={() => loadTemplate('condition', JSON.stringify({
                  field: "level",
                  operator: "eq",
                  value: "ERROR"
                }, null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                单条件模板
              </button>
              <button
                type="button"
                onClick={() => loadTemplate('condition', JSON.stringify({
                  all: [
                    { field: "level", operator: "eq", value: "ERROR" },
                    { field: "service", operator: "eq", value: "api" }
                  ]
                }, null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                AND 模板
              </button>
              <button
                type="button"
                onClick={() => loadTemplate('condition', JSON.stringify({
                  any: [
                    { field: "level", operator: "eq", value: "ERROR" },
                    { field: "level", operator: "eq", value: "PANIC" }
                  ]
                }, null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                OR 模板
              </button>
              <button
                type="button"
                onClick={() => loadTemplate('condition', JSON.stringify({
                  not: {
                    field: "environment",
                    operator: "eq",
                    value: "dev"
                  }
                }, null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                NOT 模板
              </button>
            </div>
          </div>
          <textarea
            value={conditionJson}
            onChange={(e) => setConditionJson(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
            rows={10}
          />
          <div className="mt-2 grid grid-cols-2 gap-4 text-xs text-gray-500">
            <div>
              <p className="font-medium mb-1">操作符列表：</p>
              <div className="grid grid-cols-2 gap-1">
                {OPERATOR_OPTIONS.map(op => (
                  <div key={op.value}>{op.label}</div>
                ))}
              </div>
            </div>
            <div>
              <p className="font-medium mb-1">复合条件：</p>
              <ul className="space-y-1">
                <li><code>all</code>: 所有条件都满足 (AND)</li>
                <li><code>any</code>: 任一条件满足 (OR)</li>
                <li><code>not</code>: 条件不满足 (NOT)</li>
                <li>支持任意嵌套组合</li>
              </ul>
            </div>
          </div>
        </div>

        {/* 动作配置 */}
        <div>
          <div className="flex justify-between items-center mb-2">
            <label className="block text-sm font-medium text-gray-700">
              动作配置 (JSON)
            </label>
            <div className="space-x-2">
              <button
                type="button"
                onClick={() => loadTemplate('action', JSON.stringify([{ type: "drop" }], null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                Drop
              </button>
              <button
                type="button"
                onClick={() => loadTemplate('action', JSON.stringify([{ type: "keep" }], null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                Keep
              </button>
              <button
                type="button"
                onClick={() => loadTemplate('action', JSON.stringify([{ type: "sample", config: { rate: 0.1 } }], null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                Sample
              </button>
              <button
                type="button"
                onClick={() => loadTemplate('action', JSON.stringify([{ type: "mask", config: { field: "password" } }], null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                Mask
              </button>
              <button
                type="button"
                onClick={() => loadTemplate('action', JSON.stringify([{ type: "set", config: { field: "processed", value: true } }], null, 2))}
                className="text-xs px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded"
              >
                Set
              </button>
            </div>
          </div>
          <textarea
            value={actionsJson}
            onChange={(e) => setActionsJson(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
            rows={10}
          />
          <div className="mt-2 text-xs text-gray-500">
            <p className="font-medium mb-1">动作类型及配置：</p>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <span className="font-medium">流控制:</span>
                <ul className="ml-4 space-y-1">
                  <li><code>keep</code> - 保留并终止</li>
                  <li><code>drop</code> - 丢弃并终止</li>
                  <li><code>sample</code> - 采样 (config.rate: 0.0-1.0)</li>
                </ul>
              </div>
              <div>
                <span className="font-medium">转换:</span>
                <ul className="ml-4 space-y-1">
                  <li><code>mask</code> - 掩码 (config.field, config.pattern)</li>
                  <li><code>truncate</code> - 截断 (config.field, config.max_length)</li>
                  <li><code>extract</code> - 提取 (config.source_field, config.target_field)</li>
                  <li><code>rename</code> - 重命名 (config.from, config.to)</li>
                  <li><code>remove</code> - 删除 (config.fields: string[])</li>
                  <li><code>set</code> - 设置 (config.field, config.value)</li>
                </ul>
              </div>
              <div>
                <span className="font-medium">元数据:</span>
                <ul className="ml-4 space-y-1">
                  <li><code>mark</code> - 标记 (config.reason)</li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        {/* 提交按钮 */}
        <div className="flex justify-end space-x-3">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
          >
            取消
          </button>
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? "保存中..." : "保存"}
          </button>
        </div>
      </form>
    </div>
  );
}
