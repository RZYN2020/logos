import { useState, useEffect } from "react";
import type { Condition, Action, ActionType, ConditionOperator } from "../api/types";
import { apiClient } from "../api/client";

interface Props {
  ruleId?: string;
  service: string;
  component: 'sdk' | 'processor';
  onSave: () => void;
  onCancel: () => void;
  initialLine?: number;
  initialFile?: string;
  initialFunction?: string;
  initialPattern?: string;
}

// 操作符选项
const OPERATOR_OPTIONS: { value: ConditionOperator; label: string }[] = [
  { value: 'eq', label: '等于' },
  { value: 'ne', label: '不等于' },
  { value: 'gt', label: '大于' },
  { value: 'lt', label: '小于' },
  { value: 'ge', label: '大于等于' },
  { value: 'le', label: '小于等于' },
  { value: 'contains', label: '包含' },
  { value: 'starts_with', label: '开始于' },
  { value: 'ends_with', label: '结束于' },
  { value: 'matches', label: '正则匹配' },
  { value: 'in', label: '在集合中' },
  { value: 'not_in', label: '不在集合中' },
  { value: 'exists', label: '字段存在' },
  { value: 'not_exists', label: '字段不存在' },
];

// 动作类型选项
const ACTION_TYPE_OPTIONS: { value: ActionType; label: string; description: string }[] = [
  { value: 'keep', label: '保留并终止', description: '保留日志并停止处理' },
  { value: 'drop', label: '丢弃并终止', description: '丢弃日志并停止处理' },
  { value: 'sample', label: '采样', description: '按比例采样日志' },
  { value: 'mask', label: '掩码', description: '掩码敏感数据' },
  { value: 'truncate', label: '截断', description: '截断字段值' },
  { value: 'extract', label: '提取', description: '提取子串到新字段' },
  { value: 'rename', label: '重命名', description: '重命名字段' },
  { value: 'remove', label: '删除', description: '删除字段' },
  { value: 'set', label: '设置', description: '设置字段值' },
  { value: 'mark', label: '标记', description: '添加标记' },
];

// 字段选项
const FIELD_OPTIONS = [
  { value: 'level', label: '日志级别 (level)' },
  { value: 'service', label: '服务名 (service)' },
  { value: 'environment', label: '环境 (environment)' },
  { value: 'cluster', label: '集群 (cluster)' },
  { value: 'pod', label: 'Pod (pod)' },
  { value: 'path', label: '路径 (path)' },
  { value: 'message', label: '消息 (message)' },
  { value: 'trace_id', label: '追踪 ID (trace_id)' },
];

// 单条件组件
interface SingleConditionProps {
  condition: Condition;
  onChange: (condition: Condition) => void;
  onRemove?: () => void;
  canRemove?: boolean;
}

function SingleCondition({ condition, onChange, onRemove, canRemove }: SingleConditionProps) {
  const field = condition.field || 'level';
  const operator = condition.operator || 'eq';
  const value = condition.value ?? '';

  return (
    <div className="flex items-end gap-2 p-3 bg-gray-50 border border-gray-200 rounded-md">
      <div className="flex-1">
        <label className="block text-xs text-gray-500 mb-1">字段</label>
        <select
          value={field}
          onChange={(e) => onChange({ ...condition, field: e.target.value })}
          className="block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md border bg-gray-50"
        >
          {FIELD_OPTIONS.map(f => (
            <option key={f.value} value={f.value}>{f.label}</option>
          ))}
          <option value="custom">自定义字段...</option>
        </select>
        {field === 'custom' && (
          <input
            type="text"
            placeholder="输入字段名"
            onChange={(e) => onChange({ ...condition, field: e.target.value })}
            className="mt-2 block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
          />
        )}
      </div>
      <div className="flex-1">
        <label className="block text-xs text-gray-500 mb-1">操作符</label>
        <select
          value={operator}
          onChange={(e) => onChange({ ...condition, operator: e.target.value as ConditionOperator })}
          className="block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md border bg-gray-50"
        >
          {OPERATOR_OPTIONS.map(opt => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
      </div>
      <div className="flex-1">
        <label className="block text-xs text-gray-500 mb-1">值</label>
        <input
          type="text"
          value={typeof value === 'string' ? value : JSON.stringify(value)}
          onChange={(e) => onChange({ ...condition, value: e.target.value })}
          placeholder="输入值"
          className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50 disabled:bg-gray-100 disabled:text-gray-500"
          disabled={operator === 'exists' || operator === 'not_exists'}
        />
      </div>
      {canRemove && onRemove && (
        <button
          type="button"
          onClick={onRemove}
          className="px-3 py-2 text-red-600 hover:text-red-800 hover:bg-red-50 rounded-md border border-transparent transition"
        >
          删除
        </button>
      )}
    </div>
  );
}

// 复合条件组件
interface CompositeConditionProps {
  type: 'all' | 'any' | 'not';
  condition: Condition;
  onChange: (condition: Condition) => void;
  onRemove?: () => void;
  canRemove?: boolean;
}

function CompositeCondition({ type, condition, onChange, onRemove, canRemove }: CompositeConditionProps) {
  const conditions = type === 'not'
    ? (condition.not ? [condition.not] : [])
    : (type === 'all' ? condition.all : condition.any) || [];

  const addCondition = () => {
    const newCondition = { field: 'level', operator: 'eq' as ConditionOperator, value: 'ERROR' };
    const updated = type === 'not'
      ? { not: newCondition }
      : { [type]: [...conditions, newCondition] };
    onChange(updated);
  };

  const updateCondition = (index: number, updated: Condition) => {
    if (type === 'not') {
      onChange({ not: updated });
    } else {
      const newConditions = [...conditions];
      newConditions[index] = updated;
      onChange({ [type]: newConditions });
    }
  };

  const removeCondition = (index: number) => {
    if (type === 'not') {
      onChange({});
    } else {
      const newConditions = conditions.filter((_, i) => i !== index);
      onChange({ [type]: newConditions });
    }
  };

  return (
    <div className="border border-gray-200 rounded-md p-4 bg-white shadow-sm">
      <div className="flex items-center justify-between mb-2">
        <span className={`text-xs font-semibold px-2 py-1 rounded border ${
          type === 'all' ? 'bg-blue-50 text-blue-700 border-blue-200' :
          type === 'any' ? 'bg-green-50 text-green-700 border-green-200' :
          'bg-red-50 text-red-700 border-red-200'
        }`}>
          {type === 'all' ? 'AND (且)' : type === 'any' ? 'OR (或)' : 'NOT (非)'}
        </span>
        {canRemove && onRemove && (
          <button
            type="button"
            onClick={onRemove}
            className="text-xs text-red-600 hover:text-red-800"
          >
            删除条件组
          </button>
        )}
      </div>
      <div className="space-y-2">
        {conditions.map((cond, index) => (
          <SingleCondition
            key={index}
            condition={cond}
            onChange={(updated) => updateCondition(index, updated)}
            onRemove={() => removeCondition(index)}
            canRemove
          />
        ))}
      </div>
      <button
        type="button"
        onClick={addCondition}
        className="mt-3 text-sm text-blue-600 hover:text-blue-800"
      >
        + 添加条件
      </button>
    </div>
  );
}

// 条件选择器组件
interface ConditionBuilderProps {
  condition: Condition;
  onChange: (condition: Condition) => void;
}

function ConditionBuilder({ condition, onChange }: ConditionBuilderProps) {
  const getConditionType = (): 'single' | 'all' | 'any' | 'not' => {
    if (condition.all && condition.all.length > 0) return 'all';
    if (condition.any && condition.any.length > 0) return 'any';
    if (condition.not) return 'not';
    return 'single';
  };

  const type = getConditionType();

  return (
    <div className="space-y-3">
      <div className="flex gap-2 mb-3">
        <button
          type="button"
          onClick={() => onChange({ field: 'level', operator: 'eq', value: 'ERROR' })}
          className={`px-3 py-2 text-xs font-medium rounded border transition ${
            type === 'single' ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
          }`}
        >
          单条件
        </button>
        <button
          type="button"
          onClick={() => onChange({ all: [{ field: 'level', operator: 'eq', value: 'ERROR' }] })}
          className={`px-3 py-2 text-xs font-medium rounded border transition ${
            type === 'all' ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
          }`}
        >
          AND 且
        </button>
        <button
          type="button"
          onClick={() => onChange({ any: [{ field: 'level', operator: 'eq', value: 'ERROR' }] })}
          className={`px-3 py-2 text-xs font-medium rounded border transition ${
            type === 'any' ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
          }`}
        >
          OR 或
        </button>
        <button
          type="button"
          onClick={() => onChange({ not: { field: 'level', operator: 'eq', value: 'ERROR' } })}
          className={`px-3 py-2 text-xs font-medium rounded border transition ${
            type === 'not' ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
          }`}
        >
          NOT 非
        </button>
      </div>

      {type === 'single' && (
        <SingleCondition condition={condition} onChange={onChange} />
      )}
      {type === 'all' && (
        <CompositeCondition type="all" condition={condition} onChange={onChange} />
      )}
      {type === 'any' && (
        <CompositeCondition type="any" condition={condition} onChange={onChange} />
      )}
      {type === 'not' && (
        <CompositeCondition type="not" condition={condition} onChange={onChange} />
      )}
    </div>
  );
}

// 动作配置组件
interface ActionEditorProps {
  actions: Action[];
  onChange: (actions: Action[]) => void;
}

function ActionEditor({ actions, onChange }: ActionEditorProps) {
  const addAction = () => {
    onChange([...actions, { type: 'drop' }]);
  };

  const updateAction = (index: number, action: Action) => {
    const newActions = [...actions];
    newActions[index] = action;
    onChange(newActions);
  };

  const removeAction = (index: number) => {
    onChange(actions.filter((_, i) => i !== index));
  };

  return (
    <div className="space-y-3">
      {actions.map((action, index) => (
        <div key={index} className="border border-gray-200 bg-white shadow-sm rounded-md p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-900">动作 #{index + 1}</span>
            {actions.length > 1 && (
              <button
                type="button"
                onClick={() => removeAction(index)}
                className="text-xs text-red-600 hover:text-red-800"
              >
                删除
              </button>
            )}
          </div>

          <div className="mb-3">
            <label className="block text-xs text-gray-500 mb-2">动作类型</label>
            <select
              value={action.type}
              onChange={(e) => {
                const newType = e.target.value as ActionType;
                let newConfig = undefined;
                if (newType === 'sample') {
                  newConfig = { rate: 0.1 };
                } else if (newType === 'mask') {
                  newConfig = { field: 'password' };
                } else if (newType === 'set') {
                  newConfig = { field: 'processed', value: true };
                }
                updateAction(index, { type: newType, config: newConfig });
              }}
              className="block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md border bg-gray-50"
            >
              {ACTION_TYPE_OPTIONS.map(opt => (
                <option key={opt.value} value={opt.value}>
                  {opt.label} - {opt.description}
                </option>
              ))}
            </select>
          </div>

          {/* 根据动作类型显示不同配置 */}
          {action.type === 'sample' && (
            <div className="mb-2">
              <label className="block text-xs text-gray-500 mb-2">采样率 (0.0-1.0)</label>
              <input
                type="number"
                min="0"
                max="1"
                step="0.01"
                value={action.config?.rate || 0.1}
                onChange={(e) => updateAction(index, {
                  ...action,
                  config: { ...action.config, rate: parseFloat(e.target.value) }
                })}
                className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
              />
            </div>
          )}

          {action.type === 'mask' && (
            <>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">要掩码的字段</label>
                <input
                  type="text"
                  value={action.config?.field || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, field: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                  placeholder="例如：password"
                />
              </div>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">掩码模式 (可选)</label>
                <input
                  type="text"
                  value={action.config?.pattern || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, pattern: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                  placeholder="例如：\d+"
                />
              </div>
            </>
          )}

          {action.type === 'truncate' && (
            <>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">要截断的字段</label>
                <input
                  type="text"
                  value={action.config?.field || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, field: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">最大长度</label>
                <input
                  type="number"
                  value={action.config?.max_length || 100}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, max_length: parseInt(e.target.value) }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
            </>
          )}

          {action.type === 'extract' && (
            <>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">源字段</label>
                <input
                  type="text"
                  value={action.config?.source_field || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, source_field: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">目标字段</label>
                <input
                  type="text"
                  value={action.config?.target_field || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, target_field: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
            </>
          )}

          {action.type === 'rename' && (
            <>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">原字段名</label>
                <input
                  type="text"
                  value={action.config?.from || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, from: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">新字段名</label>
                <input
                  type="text"
                  value={action.config?.to || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, to: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
            </>
          )}

          {action.type === 'remove' && (
            <div className="mb-2">
              <label className="block text-xs text-gray-500 mb-2">要删除的字段 (逗号分隔)</label>
              <input
                type="text"
                value={(action.config?.fields || []).join(', ')}
                onChange={(e) => updateAction(index, {
                  ...action,
                  config: { fields: e.target.value.split(',').map(s => s.trim()) }
                })}
                className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                placeholder="例如：field1, field2"
              />
            </div>
          )}

          {action.type === 'set' && (
            <>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">字段名</label>
                <input
                  type="text"
                  value={action.config?.field || ''}
                  onChange={(e) => updateAction(index, {
                    ...action,
                    config: { ...action.config, field: e.target.value }
                  })}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
              <div className="mb-2">
                <label className="block text-xs text-gray-500 mb-2">字段值</label>
                <input
                  type="text"
                  value={String(action.config?.value || '')}
                  onChange={(e) => {
                    const val = e.target.value;
                    let parsed: string | boolean | number = val;
                    if (val === 'true') parsed = true;
                    else if (val === 'false') parsed = false;
                    else if (!isNaN(Number(val))) parsed = Number(val);
                    updateAction(index, {
                      ...action,
                      config: { ...action.config, value: parsed }
                    });
                  }}
                  className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                />
              </div>
            </>
          )}

          {action.type === 'mark' && (
            <div className="mb-2">
              <label className="block text-xs text-gray-500 mb-2">标记原因</label>
              <input
                type="text"
                value={action.config?.reason || ''}
                onChange={(e) => updateAction(index, {
                  ...action,
                  config: { ...action.config, reason: e.target.value }
                })}
                className="block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
                placeholder="例如：匹配规则 #123"
              />
            </div>
          )}
        </div>
      ))}

      <button
        type="button"
        onClick={addAction}
        className="w-full py-2 border border-dashed border-gray-300 rounded-md text-gray-600 hover:border-blue-500 hover:text-blue-600 bg-gray-50 transition"
      >
        + 添加动作
      </button>
    </div>
  );
}

export default function RuleForm({
  ruleId,
  service,
  component,
  onSave,
  onCancel,
  initialLine,
  initialFile,
  initialFunction,
  initialPattern,
}: Props) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [condition, setCondition] = useState<Condition>(
    { field: "level", operator: "eq", value: "ERROR" }
  );
  const [actions, setActions] = useState<Action[]>([{ type: "drop" }]);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(!!ruleId || !!initialLine || !!initialPattern);

  // 加载现有规则数据或根据行号/模式生成建议规则
  useEffect(() => {
    if (!ruleId && !initialLine && !initialPattern) {
      setLoading(false);
      return;
    }

    const loadRuleData = async () => {
      setLoading(true);
      setError(null);

      try {
        if (ruleId) {
          // 编辑现有规则
          const rule = await apiClient.getRule(ruleId);
          if (rule) {
            setName(rule.name);
            setDescription(rule.description || "");
            setEnabled(rule.enabled ?? true);
            setCondition(rule.condition);
            setActions(rule.actions || [{ type: "drop" }]);
          }
        } else if (initialPattern) {
          // 根据模式生成建议规则
          const suggestedCondition: Condition = {
            field: "message",
            operator: "contains",
            value: initialPattern.split('{')[0]?.trim() || initialPattern,
          };

          setCondition(suggestedCondition);
          setName(`规则 - 模式匹配`);
          setDescription(`自动生成的规则：匹配模式 "${initialPattern}"`);

          // 默认设置为采样动作，因为模式匹配通常用于采样
          setActions([{ type: "sample", config: { rate: 0.1 } }]);
        } else if (initialLine) {
          // 根据行号生成建议规则
          const suggestedCondition: Condition = {
            all: [
              { field: "path", operator: "contains", value: initialFile || "" },
              { field: "line_number", operator: "eq", value: initialLine },
            ],
          };

          if (initialFunction) {
            suggestedCondition.all!.push({
              field: "function",
              operator: "eq",
              value: initialFunction,
            });
          }

          setCondition(suggestedCondition);
          setName(`规则 - ${initialFile || "行号"}:${initialLine}`);
          setDescription(`自动生成的规则：针对 ${initialFile || ""}:${initialLine} ${initialFunction ? `(${initialFunction})` : ""}`);
        }
      } catch (err) {
        setError("加载规则失败：" + (err as Error).message);
        console.error(err);
      } finally {
        setLoading(false);
      }
    };

    loadRuleData();
  }, [ruleId, initialLine, initialFile, initialFunction, initialPattern]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!name.trim()) {
      setError("规则名称不能为空");
      return;
    }

    try {
      setSaving(true);

      const ruleData = {
        name,
        description,
        enabled,
        condition,
        actions,
        service,
        component,
      };

      if (ruleId) {
        await apiClient.updateRule(ruleId, ruleData);
      } else {
        await apiClient.createRule(ruleData);
      }

      onSave();
    } catch (err) {
      setError("保存失败：" + (err as Error).message);
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <button onClick={onCancel} className="text-sm text-blue-600 hover:text-blue-800 mb-4">
          ← 返回列表
        </button>
        <div className="flex items-center justify-between border-b border-gray-200 pb-4">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">
              {ruleId ? "编辑规则" : "新建规则"}
            </h2>
            <p className="text-sm text-gray-500 mt-1">
              {service} · {component === 'sdk' ? 'SDK' : 'Processor'}
            </p>
          </div>
          <div className="flex gap-2">
            {initialLine && (
              <div className="px-3 py-1 bg-blue-50 text-blue-700 border border-blue-200 rounded text-sm">
                基于行号 L{initialLine} 生成
              </div>
            )}
            {initialPattern && (
              <div className="px-3 py-1 bg-green-50 text-green-700 border border-green-200 rounded text-sm">
                基于模式生成
              </div>
            )}
          </div>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded text-sm mb-6">
          {error}
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-12 text-gray-500">
          <p>加载中...</p>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-8 bg-white p-6 border border-gray-200 rounded-md">
        {/* 基本信息 */}
        <div>
          <h3 className="text-lg font-medium text-gray-900 mb-4">基本信息</h3>
          <div className="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
            <div className="sm:col-span-3">
              <label className="block text-sm font-medium text-gray-700">
                规则名称
              </label>
              <div className="mt-1">
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="shadow-sm focus:ring-blue-500 focus:border-blue-500 block w-full sm:text-sm border-gray-300 rounded-md py-2 px-3 border bg-gray-50"
                  placeholder="例如：drop-debug-logs"
                  required
                />
              </div>
            </div>

            <div className="sm:col-span-3 flex items-end pb-2">
              <div className="flex items-center">
                <input
                  type="checkbox"
                  checked={enabled}
                  onChange={(e) => setEnabled(e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label className="ml-2 block text-sm text-gray-900">
                  启用规则
                </label>
              </div>
            </div>

            <div className="sm:col-span-6">
              <label className="block text-sm font-medium text-gray-700">
                规则描述
              </label>
              <div className="mt-1">
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={2}
                  className="shadow-sm focus:ring-blue-500 focus:border-blue-500 block w-full sm:text-sm border-gray-300 rounded-md py-2 px-3 border bg-gray-50"
                  placeholder="描述规则的作用和场景"
                />
              </div>
            </div>
          </div>
        </div>

        {/* 条件配置 */}
        <div className="pt-6 border-t border-gray-200">
          <h3 className="text-lg font-medium text-gray-900 mb-4">触发条件</h3>
          <ConditionBuilder condition={condition} onChange={setCondition} />
        </div>

        {/* 动作配置 */}
        <div className="pt-6 border-t border-gray-200">
          <h3 className="text-lg font-medium text-gray-900 mb-4">执行动作</h3>
          <ActionEditor actions={actions} onChange={setActions} />
        </div>

        {/* 提交按钮 */}
        <div className="pt-5 border-t border-gray-200">
          <div className="flex justify-end">
            <button
              type="button"
              onClick={onCancel}
              className="bg-white py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={saving}
              className="ml-3 inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:bg-gray-400"
            >
              {saving ? "保存中..." : "保存"}
            </button>
          </div>
        </div>
      </form>
      )}
    </div>
  );
}
