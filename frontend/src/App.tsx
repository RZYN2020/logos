import { useState, useEffect } from "react";
import { apiClient, type SystemInfo, type HealthCheck, type Rule } from "./api/client";
import RuleList from "./components/RuleList";
import RuleForm from "./components/RuleForm";
import LogAnalyzer from "./components/LogAnalyzer";

type Tab = "rules" | "analyzer" | "system";

export default function App() {
  const [activeTab, setActiveTab] = useState<Tab>("rules");
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [healthStatus, setHealthStatus] = useState<HealthCheck | null>(null);

  const loadSystemInfo = async () => {
    const info = await apiClient.getSystemInfo();
    setSystemInfo(info);
  };

  const checkHealth = async () => {
    const health = await apiClient.healthCheck();
    setHealthStatus(health);
  };

  useEffect(() => {
    loadSystemInfo();
    checkHealth();
    const interval = setInterval(checkHealth, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleSaveRule = () => {
    setEditingRuleId(null);
    setActiveTab("rules");
  };

  return (
    <div className="min-h-screen bg-gray-100">
      {/* 顶部导航栏 */}
      <nav className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <div className="flex-shrink-0 flex items-center">
                <div className="h-8 w-8 rounded-full bg-blue-600 flex items-center justify-center text-white font-bold">
                  L
                </div>
                <span className="ml-3 text-xl font-bold text-gray-900">
                  语义化日志系统
                </span>
              </div>
            </div>

            <div className="flex items-center space-x-4">
              <button
                onClick={() => setActiveTab("rules")}
                className={`px-3 py-2 rounded-md text-sm font-medium ${
                  activeTab === "rules"
                    ? "bg-blue-600 text-white"
                    : "text-gray-700 hover:bg-gray-100"
                }`}
              >
                规则配置
              </button>
              <button
                onClick={() => setActiveTab("analyzer")}
                className={`px-3 py-2 rounded-md text-sm font-medium ${
                  activeTab === "analyzer"
                    ? "bg-blue-600 text-white"
                    : "text-gray-700 hover:bg-gray-100"
                }`}
              >
                日志分析
              </button>
              <button
                onClick={() => setActiveTab("system")}
                className={`px-3 py-2 rounded-md text-sm font-medium ${
                  activeTab === "system"
                    ? "bg-blue-600 text-white"
                    : "text-gray-700 hover:bg-gray-100"
                }`}
              >
                系统信息
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* 系统状态 */}
      <div className="bg-blue-50 border-b border-blue-100">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-2">
          <div className="flex items-center space-x-6 text-sm">
            <div className="flex items-center">
              <span className="text-gray-500">系统状态：</span>
              <span className="flex items-center ml-2">
                {healthStatus?.status === "healthy" ? (
                  <>
                    <span className="h-2 w-2 rounded-full bg-green-500"></span>
                    <span className="ml-1 text-green-700 font-medium">健康</span>
                  </>
                ) : (
                  <>
                    <span className="h-2 w-2 rounded-full bg-red-500"></span>
                    <span className="ml-1 text-red-700 font-medium">异常</span>
                  </>
                )}
              </span>
            </div>
            {systemInfo && (
              <span className="text-gray-500 ml-4">
                版本：{systemInfo.version} | Etcd: {systemInfo.etcd_version} | 运行时间：{systemInfo.uptime}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* 主内容区域 */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === "rules" && (
          <div>
            {editingRuleId !== null ? (
              <RuleForm
                rule={editingRuleId ? null : undefined}
                onSave={handleSaveRule}
                onCancel={() => setEditingRuleId(null)}
              />
            ) : (
              <RuleList onEdit={setEditingRuleId} />
            )}
          </div>
        )}

        {activeTab === "analyzer" && <LogAnalyzer />}

        {activeTab === "system" && (
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-8">系统信息</h1>
            <div className="bg-white shadow rounded-lg p-6">
              {systemInfo ? (
                <div className="space-y-4">
                  <div>
                    <dt className="text-sm font-medium text-gray-500">系统名称</dt>
                    <dd className="mt-1 text-lg text-gray-900">{systemInfo.system}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-gray-500">版本</dt>
                    <dd className="mt-1 text-lg text-gray-900">{systemInfo.version}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-gray-500">Etcd 版本</dt>
                    <dd className="mt-1 text-lg text-gray-900">{systemInfo.etcd_version}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-gray-500">运行时间</dt>
                    <dd className="mt-1 text-lg text-gray-900">{systemInfo.uptime}</dd>
                  </div>
                </div>
              ) : (
                <p className="text-gray-500">加载中...</p>
              )}
            </div>

            <div className="mt-8 bg-white shadow rounded-lg p-6">
              <h2 className="text-xl font-bold text-gray-900 mb-4">健康检查</h2>
              {healthStatus ? (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-500">整体状态</span>
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        healthStatus.status === "healthy"
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {healthStatus.status === "healthy" ? "健康" : "异常"}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-500">Etcd 连接</span>
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        healthStatus.etcd === "connected"
                          ? "bg-green-100 text-green-800"
                          : "bg-gray-100 text-gray-800"
                      }`}
                    >
                      {healthStatus.etcd === "connected" ? "已连接" : "未检测"}
                    </span>
                  </div>
                </div>
              ) : (
                <p className="text-gray-500">检查中...</p>
              )}
            </div>
          </div>
        )}
      </main>

      {/* 页脚 */}
      <footer className="bg-white border-t border-gray-200 mt-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="text-center text-gray-500 text-sm">
            语义化日志系统 - 支持动态规则配置的高性能日志系统
          </div>
          <div className="text-center text-gray-400 text-xs mt-2">
            API: http://localhost:8080/api/v1
          </div>
        </div>
      </footer>
    </div>
  );
}
