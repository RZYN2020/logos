import { useState, useEffect } from "react";
import { apiClient } from "../api/client";

interface Props {
  service: string;
}

export default function Alerts({ service }: Props) {
  const [activeTab, setActiveTab] = useState<"rules" | "history">("rules");
  const [rules, setRules] = useState<any[]>([]);
  const [history, setHistory] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const [showForm, setShowForm] = useState(false);
  const [editingRule, setEditingRule] = useState<any>(null);

  const [formData, setFormData] = useState({
    name: "",
    description: "",
    enabled: true,
    condition: { field: "level", operator: "eq", value: "ERROR" },
    threshold: 10,
    window: 60,
    channels: { webhook: "" }
  });

  const fetchRules = async () => {
    setLoading(true);
    try {
      const data = await apiClient.listAlertRules(service);
      setRules(data);
    } catch (e) {
      console.error("Failed to fetch alert rules", e);
    } finally {
      setLoading(false);
    }
  };

  const fetchHistory = async () => {
    setLoading(true);
    try {
      const data = await apiClient.listAlertHistory(service);
      setHistory(data);
    } catch (e) {
      console.error("Failed to fetch alert history", e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (activeTab === "rules") {
      fetchRules();
    } else {
      fetchHistory();
    }
  }, [activeTab, service]);

  const handleResolve = async (id: string) => {
    try {
      await apiClient.resolveAlert(id);
      fetchHistory();
    } catch (e) {
      console.error("Failed to resolve alert", e);
    }
  };

  const handleCreateRule = () => {
    setEditingRule(null);
    setFormData({
      name: "",
      description: "",
      enabled: true,
      condition: { field: "level", operator: "eq", value: "ERROR" },
      threshold: 10,
      window: 60,
      channels: { webhook: "" }
    });
    setShowForm(true);
  };

  const handleCreateVolumeRule = () => {
    setEditingRule(null);
    setFormData({
      name: "日志超量告警",
      description: "当所有级别的日志总量超过系统配额或异常飙升时触发",
      enabled: true,
      condition: { field: "level", operator: "neq", value: "" }, // 匹配所有日志
      threshold: 10000, // 默认一万条
      window: 60, // 默认一分钟
      channels: { webhook: "" }
    });
    setShowForm(true);
  };

  const handleEditRule = (rule: any) => {
    setEditingRule(rule);
    setFormData({
      name: rule.name,
      description: rule.description || "",
      enabled: rule.enabled,
      condition: rule.condition,
      threshold: rule.threshold,
      window: rule.window,
      channels: rule.channels || { webhook: "" }
    });
    setShowForm(true);
  };

  const handleDeleteRule = async (id: string) => {
    if (!confirm("确定要删除这条告警规则吗？")) return;
    try {
      await apiClient.deleteAlertRule(id);
      fetchRules();
    } catch (e) {
      console.error("Failed to delete alert rule", e);
    }
  };

  const handleSaveRule = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { ...formData, service };
      if (editingRule) {
        await apiClient.updateAlertRule(editingRule.id, payload);
      } else {
        await apiClient.createAlertRule(payload);
      }
      setShowForm(false);
      fetchRules();
    } catch (e) {
      console.error("Failed to save alert rule", e);
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">告警管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理 {service} 的告警规则和历史</p>
        </div>
        <div className="flex bg-gray-100 rounded-md p-1">
          <button
            onClick={() => setActiveTab("rules")}
            className={`px-4 py-2 text-sm font-medium rounded-md ${
              activeTab === "rules" ? "bg-white shadow-sm text-gray-900" : "text-gray-500 hover:text-gray-700"
            }`}
          >
            告警规则
          </button>
          <button
            onClick={() => setActiveTab("history")}
            className={`px-4 py-2 text-sm font-medium rounded-md ${
              activeTab === "history" ? "bg-white shadow-sm text-gray-900" : "text-gray-500 hover:text-gray-700"
            }`}
          >
            告警历史
          </button>
        </div>
      </div>

      {loading && !showForm ? (
        <div className="text-center py-12 text-gray-500">加载中...</div>
      ) : showForm ? (
        <div className="bg-white border border-gray-200 rounded-md shadow-sm p-6">
          <div className="flex justify-between items-center mb-6 border-b border-gray-200 pb-4">
            <h2 className="text-lg font-bold text-gray-900">{editingRule ? "编辑告警规则" : "新建告警规则"}</h2>
            <button onClick={() => setShowForm(false)} className="text-sm text-gray-500 hover:text-gray-700">返回列表</button>
          </div>
          <form onSubmit={handleSaveRule} className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">规则名称</label>
                <input
                  required
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({...formData, name: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div className="flex items-end pb-2">
                <label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => setFormData({...formData, enabled: e.target.checked})}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <span className="text-sm font-medium text-gray-700">启用规则</span>
                </label>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">规则描述</label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({...formData, description: e.target.value})}
                className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                rows={2}
              />
            </div>
            
            <div className="pt-4 border-t border-gray-200">
              <h3 className="text-md font-medium text-gray-900 mb-4">触发条件</h3>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">字段</label>
                  <input
                    type="text"
                    value={formData.condition.field}
                    onChange={(e) => setFormData({...formData, condition: {...formData.condition, field: e.target.value}})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">操作符</label>
                  <select
                    value={formData.condition.operator}
                    onChange={(e) => setFormData({...formData, condition: {...formData.condition, operator: e.target.value}})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="eq">等于 (eq)</option>
                    <option value="neq">不等于 (neq)</option>
                    <option value="contains">包含 (contains)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">值</label>
                  <input
                    type="text"
                    value={formData.condition.value}
                    onChange={(e) => setFormData({...formData, condition: {...formData.condition, value: e.target.value}})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              </div>
            </div>

            <div className="pt-4 border-t border-gray-200">
              <h3 className="text-md font-medium text-gray-900 mb-4">阈值设置</h3>
              <div className="grid grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">时间窗口 (秒)</label>
                  <input
                    type="number"
                    min="1"
                    value={formData.window}
                    onChange={(e) => setFormData({...formData, window: parseInt(e.target.value) || 60})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">触发阈值 (次)</label>
                  <input
                    type="number"
                    min="1"
                    value={formData.threshold}
                    onChange={(e) => setFormData({...formData, threshold: parseInt(e.target.value) || 10})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              </div>
            </div>

            <div className="pt-4 border-t border-gray-200">
              <h3 className="text-md font-medium text-gray-900 mb-4">通知渠道</h3>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Webhook URL</label>
                <input
                  type="url"
                  placeholder="https://hook.example.com/..."
                  value={formData.channels.webhook || ""}
                  onChange={(e) => setFormData({...formData, channels: {...formData.channels, webhook: e.target.value}})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
            </div>

            <div className="pt-6 border-t border-gray-200 flex justify-end space-x-3">
              <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50">
                取消
              </button>
              <button type="submit" className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700">
                保存
              </button>
            </div>
          </form>
        </div>
      ) : activeTab === "rules" ? (
        <div className="bg-white border border-gray-200 rounded-md shadow-sm">
          <div className="p-4 border-b border-gray-200 flex justify-between items-center">
            <h2 className="text-lg font-bold text-gray-900">告警规则列表</h2>
            <div className="space-x-3">
              <button
                onClick={handleCreateVolumeRule}
                className="px-4 py-2 bg-white border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
              >
                新建日志超量告警
              </button>
              <button
                onClick={handleCreateRule}
                className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
              >
                新建告警规则
              </button>
            </div>
          </div>
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">条件</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">阈值/窗口</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {rules.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-4 text-center text-gray-500">暂无告警规则</td>
                </tr>
              ) : (
                rules.map((rule) => (
                  <tr key={rule.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{rule.name}</td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {rule.enabled ? (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">启用</span>
                      ) : (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800">禁用</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {rule.condition?.field} {rule.condition?.operator} {rule.condition?.value}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      &gt;= {rule.threshold} 次 / {rule.window} 秒
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                      <button
                        onClick={() => handleEditRule(rule)}
                        className="text-blue-600 hover:text-blue-900 mr-3"
                      >
                        编辑
                      </button>
                      <button
                        onClick={() => handleDeleteRule(rule.id)}
                        className="text-red-600 hover:text-red-900"
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
      ) : (
        <div className="bg-white border border-gray-200 rounded-md shadow-sm">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">触发时间</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">级别</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">内容</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {history.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-4 text-center text-gray-500">暂无告警历史</td>
                </tr>
              ) : (
                history.map((alert) => (
                  <tr key={alert.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(alert.trigger_time).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        alert.level === 'ERROR' ? 'bg-red-100 text-red-800' : 'bg-yellow-100 text-yellow-800'
                      }`}>
                        {alert.level}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">{alert.message}</td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {alert.status === 'resolved' ? (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">已处理</span>
                      ) : (
                        <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-red-100 text-red-800">未处理</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                      {alert.status !== 'resolved' && (
                        <button
                          onClick={() => handleResolve(alert.id)}
                          className="text-blue-600 hover:text-blue-900"
                        >
                          标记为已处理
                        </button>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}