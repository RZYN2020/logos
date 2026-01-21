import { useState } from "react";
import { apiClient, type Strategy, type StrategyRule } from "./api/client";

interface Props {
  strategy?: Strategy;
  onSave: () => void;
  onCancel: () => void;
}

export default function StrategyForm({ strategy, onSave, onCancel }: Props) {
  const [name, setName] = useState(strategy?.name || "");
  const [description, setDescription] = useState(strategy?.description || "");
  const [rules, setRules] = useState<string>(strategy?.rules ? JSON.stringify(strategy.rules, null, 2) : "[]");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!name.trim()) {
      setError("策略名称不能为空");
      return;
    }

    try {
      setSaving(true);

      let parsedRules: StrategyRule[];
      try {
        parsedRules = JSON.parse(rules);
      } catch {
        setError("规则 JSON 格式错误");
        setSaving(false);
        return;
      }

      if (strategy) {
        await apiClient.updateStrategy(strategy.id, {
          name,
          description,
          rules: parsedRules,
        });
      } else {
        await apiClient.createStrategy({
          name,
          description,
          rules: parsedRules,
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

  const setExampleRules = () => {
    const example = {
      rules: [
        {
          condition: { level: "ERROR", environment: "production" },
          action: { enabled: true, priority: "high", sampling: 1.0 },
        },
      ],
    };
    setRules(JSON.stringify(example.rules, null, 2));
  };

  return (
    <div className="px-4 py-6">
      <div className="mb-4">
        <button onClick={onCancel} className="text-gray-600 hover:text-gray-900 mb-4">
          ← 返回列表
        </button>
        <h2 className="text-2xl font-bold text-gray-900">
          {strategy ? "编辑策略" : "新建策略"}
        </h2>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="bg-white shadow rounded-lg p-6 space-y-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            策略名称
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            placeholder="例如: production-error-filter"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            策略描述
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            rows={3}
            placeholder="描述策略的作用和场景"
          />
        </div>

        <div>
          <div className="flex justify-between items-center mb-2">
            <label className="block text-sm font-medium text-gray-700">
              策略规则 (JSON)
            </label>
            <button
              type="button"
              onClick={setExampleRules}
              className="text-sm text-blue-600 hover:text-blue-800"
            >
              填入示例
            </button>
          </div>
          <textarea
            value={rules}
            onChange={(e) => setRules(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
            rows={10}
            placeholder='[{"condition": {"level": "ERROR"}, "action": {"enabled": true}}]'
          />
          <p className="mt-2 text-xs text-gray-500">
            策略规则示例：
          </p>
          <div className="bg-gray-50 p-3 rounded text-xs font-mono text-gray-700">
            {`{`}
            "rules": [
              {
                "condition": {
                  "level": "ERROR",
                  "environment": "production"
                },
                "action": {
                  "enabled": true,
                  "priority": "high",
                  "sampling": 1.0
                }
              }
            ]
            {`}`}
          </div>
        </div>

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
