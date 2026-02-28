import { useState } from "react";
import { apiClient, type Rule, type Condition, type Action } from "./api/client";

interface Props {
  rule?: Rule;
  onSave: () => void;
  onCancel: () => void;
}

export default function RuleForm({ rule, onSave, onCancel }: Props) {
  const [name, setName] = useState(rule?.name || "");
  const [description, setDescription] = useState(rule?.description || "");
  const [enabled, setEnabled] = useState(rule?.enabled ?? true);
  const [priority, setPriority] = useState(rule?.priority ?? 0);
  const [conditionsJson, setConditionsJson] = useState<string>(
    rule?.conditions ? JSON.stringify(rule.conditions, null, 2) : "[{\"field\": \"level\", \"operator\": \"=\", \"value\": \"ERROR\"}]"
  );
  const [actionsJson, setActionsJson] = useState<string>(
    rule?.actions ? JSON.stringify(rule.actions, null, 2) : "[{\"type\": \"filter\", \"config\": {\"sampling\": 1.0}}]"
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

      let parsedConditions: Condition[];
      try {
        parsedConditions = JSON.parse(conditionsJson);
      } catch {
        setError("条件 JSON 格式错误");
        setSaving(false);
        return;
      }

      let parsedActions: Action[];
      try {
        parsedActions = JSON.parse(actionsJson);
      } catch {
        setError("动作 JSON 格式错误");
        setSaving(false);
        return;
      }

      if (rule) {
        await apiClient.updateRule(rule.id, {
          name,
          description,
          enabled,
          priority,
          conditions: parsedConditions,
          actions: parsedActions,
        });
      } else {
        await apiClient.createRule({
          name,
          description,
          enabled,
          priority,
          conditions: parsedConditions,
          actions: parsedActions,
        } as any);
      }

      onSave();
    } catch (err) {
      setError("保存失败");
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  const validateRule = async () => {
    if (!rule) {
      alert("请先保存规则后再验证");
      return;
    }
    try {
      const result = await apiClient.validateRule(rule.id);
      if (result.valid) {
        alert("规则验证通过");
      } else {
        alert("规则验证失败：" + (result.errors?.join(", ") || "未知错误"));
      }
    } catch (err) {
      alert("验证失败");
    }
  };

  const testRule = async () => {
    if (!rule) {
      alert("请先保存规则后再测试");
      return;
    }
    try {
      const result = await apiClient.testRule(rule.id);
      alert(`规则测试结果：${result.matched ? "匹配" : "不匹配"}\n测试数据：${JSON.stringify(result.test_data)}`);
    } catch (err) {
      alert("测试失败");
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
              placeholder="例如：error-log-filter"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              优先级
            </label>
            <input
              type="number"
              value={priority}
              onChange={(e) => setPriority(parseInt(e.target.value))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              placeholder="0"
            />
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

        <div className="flex items-center space-x-4">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="mr-2"
            />
            <span className="text-sm font-medium text-gray-700">启用规则</span>
          </label>
        </div>

        <div>
          <div className="flex justify-between items-center mb-2">
            <label className="block text-sm font-medium text-gray-700">
              条件配置 (JSON)
            </label>
          </div>
          <textarea
            value={conditionsJson}
            onChange={(e) => setConditionsJson(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
            rows={8}
          />
          <p className="mt-2 text-xs text-gray-500">
            条件示例：
          </p>
          <div className="bg-gray-50 p-3 rounded text-xs font-mono text-gray-700">
            {[
              '[',
              '  {"field": "level", "operator": "=", "value": "ERROR"},',
              '  {"field": "service", "operator": "in", "value": ["api", "web"]}',
              ']'
            ].join("\n")}
          </div>
        </div>

        <div>
          <div className="flex justify-between items-center mb-2">
            <label className="block text-sm font-medium text-gray-700">
              动作配置 (JSON)
            </label>
          </div>
          <textarea
            value={actionsJson}
            onChange={(e) => setActionsJson(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
            rows={8}
          />
          <p className="mt-2 text-xs text-gray-500">
            动作示例：
          </p>
          <div className="bg-gray-50 p-3 rounded text-xs font-mono text-gray-700">
            {[
              '[',
              '  {"type": "filter", "config": {"sampling": 1.0}},',
              '  {"type": "drop", "config": {}}',
              ']'
            ].join("\n")}
          </div>
        </div>

        {rule && (
          <div className="flex justify-start space-x-3">
            <button
              type="button"
              onClick={validateRule}
              className="px-4 py-2 border border-blue-600 text-blue-600 rounded-md hover:bg-blue-50"
            >
              验证规则
            </button>
            <button
              type="button"
              onClick={testRule}
              className="px-4 py-2 border border-green-600 text-green-600 rounded-md hover:bg-green-50"
            >
              测试规则
            </button>
          </div>
        )}

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
