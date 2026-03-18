import { useEffect, useMemo, useState } from "react";
import RuleList from "./components/RuleList";
import RuleForm from "./components/RuleForm";
import ServiceSelector from "./components/ServiceSelector";
import LogAnalyzer from "./components/LogAnalyzer";
import LogReport from "./components/LogReport";
import Alerts from "./components/Alerts";
import { apiClient, setAuthToken } from "./api/client";

type Tab = "rules" | "analyzer" | "report" | "alerts";
type ComponentType = "sdk" | "processor";

interface ServiceSelection {
  service: string;
  component: ComponentType;
}

interface ConfiguringFromLine {
  lineNumber: number;
  file?: string;
  func?: string;
}

interface ConfiguringFromPattern {
  pattern: string;
}

export default function App() {
  const [activeTab, setActiveTab] = useState<Tab>("rules");
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null);
  const [selection, setSelection] = useState<ServiceSelection | null>(null);
  const [configuringFromLine, setConfiguringFromLine] = useState<ConfiguringFromLine | null>(null);
  const [configuringFromPattern, setConfiguringFromPattern] = useState<ConfiguringFromPattern | null>(null);
  const [authStatus, setAuthStatus] = useState<"checking" | "login" | "ready">("checking");
  const [authError, setAuthError] = useState<string | null>(null);
  const [loginUsername, setLoginUsername] = useState("admin");
  const [loginPassword, setLoginPassword] = useState("admin123");
  const [currentUser, setCurrentUser] = useState<{ username: string; roles: string[] } | null>(null);

  const autoLoginEnabled = useMemo(() => {
    return String(import.meta.env.VITE_AUTO_LOGIN || "") === "1";
  }, []);

  useEffect(() => {
    const init = async () => {
      const stored = localStorage.getItem("logos_token");
      if (stored) {
        setAuthToken(stored);
        const u = await apiClient.getCurrentUser();
        if (u) {
          setCurrentUser({ username: u.username, roles: u.roles });
          setAuthStatus("ready");
          return;
        }
        localStorage.removeItem("logos_token");
        setAuthToken(null);
      }

      if (autoLoginEnabled) {
        try {
          const { token } = await apiClient.login("admin", "admin123");
          localStorage.setItem("logos_token", token);
          const u = await apiClient.getCurrentUser();
          if (u) setCurrentUser({ username: u.username, roles: u.roles });
          setAuthStatus("ready");
          return;
        } catch (e) {
          setAuthError((e as Error).message);
        }
      }

      setAuthStatus("login");
    };

    init();
  }, [autoLoginEnabled]);

  const handleLogout = () => {
    localStorage.removeItem("logos_token");
    setAuthToken(null);
    setCurrentUser(null);
    setSelection(null);
    setEditingRuleId(null);
    setAuthStatus("login");
  };

  const handleLogin = async () => {
    setAuthError(null);
    try {
      const { token } = await apiClient.login(loginUsername, loginPassword);
      localStorage.setItem("logos_token", token);
      const u = await apiClient.getCurrentUser();
      if (u) setCurrentUser({ username: u.username, roles: u.roles });
      setAuthStatus("ready");
    } catch (e) {
      setAuthError((e as Error).message);
    }
  };

  const handleServiceSelect = (service: string, component: ComponentType) => {
    setSelection({ service, component });
  };

  if (authStatus === "checking") {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="text-sm text-gray-500">正在连接控制面...</div>
        </div>
      </div>
    );
  }

  if (authStatus === "login") {
    return (
      <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full space-y-8 bg-white p-8 border border-gray-200 shadow-sm">
          <div>
            <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
              Logos 控制台
            </h2>
            <p className="mt-2 text-center text-sm text-gray-600">
              语义化日志系统
            </p>
          </div>
          
          {authError && (
            <div className="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded text-sm">
              {authError}
            </div>
          )}

          <div className="mt-8 space-y-6">
            <div className="rounded-md shadow-sm space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">用户名</label>
                <input
                  value={loginUsername}
                  onChange={(e) => setLoginUsername(e.target.value)}
                  className="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm bg-gray-50"
                  placeholder="admin"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">密码</label>
                <input
                  value={loginPassword}
                  onChange={(e) => setLoginPassword(e.target.value)}
                  type="password"
                  className="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm bg-gray-50"
                  placeholder="admin123"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleLogin();
                  }}
                />
              </div>
            </div>

            <div>
              <button
                onClick={handleLogin}
                className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                登录
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 如果未选择服务，显示服务选择器
  if (!selection) {
    return (
      <div className="min-h-screen bg-gray-50">
        <header className="bg-white border-b border-gray-200">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between h-16">
              <div className="flex items-center">
                <span className="text-lg font-bold text-gray-900">Logos 控制台</span>
              </div>
              <div className="flex items-center space-x-4">
                {currentUser && (
                  <span className="text-sm text-gray-500">
                    {currentUser.username} ({currentUser.roles.includes("admin") ? "admin" : "user"})
                  </span>
                )}
                <button
                  onClick={handleLogout}
                  className="text-sm text-gray-500 hover:text-gray-700"
                >
                  退出
                </button>
              </div>
            </div>
          </div>
        </header>

        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <ServiceSelector onSelect={handleServiceSelect} />
        </main>
      </div>
    );
  }

  // 已选择服务，显示主界面
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center space-x-8">
              <span className="text-lg font-bold text-gray-900">Logos 控制台</span>
              
              <div className="flex space-x-4">
                <button
                  onClick={() => setActiveTab("rules")}
                  className={`px-3 py-2 text-sm font-medium ${
                    activeTab === "rules" ? "text-blue-600 border-b-2 border-blue-600" : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  规则配置
                </button>
                <button
                  onClick={() => setActiveTab("report")}
                  className={`px-3 py-2 text-sm font-medium ${
                    activeTab === "report" ? "text-blue-600 border-b-2 border-blue-600" : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  日志报告
                </button>
                <button
                  onClick={() => setActiveTab("analyzer")}
                  className={`px-3 py-2 text-sm font-medium ${
                    activeTab === "analyzer" ? "text-blue-600 border-b-2 border-blue-600" : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  日志查询
                </button>
                <button
                  onClick={() => setActiveTab("alerts")}
                  className={`px-3 py-2 text-sm font-medium ${
                    activeTab === "alerts" ? "text-blue-600 border-b-2 border-blue-600" : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  告警管理
                </button>
              </div>
            </div>

            <div className="flex items-center space-x-4">
              <div className="text-sm text-gray-500 bg-gray-100 px-3 py-1 rounded">
                当前服务: {selection.service} ({selection.component})
              </div>
              <button
                onClick={() => setSelection(null)}
                className="text-sm text-blue-600 hover:text-blue-800"
              >
                切换
              </button>
              <div className="h-4 w-px bg-gray-300"></div>
              {currentUser && (
                <span className="text-sm text-gray-500">
                  {currentUser.username}
                </span>
              )}
              <button
                onClick={handleLogout}
                className="text-sm text-gray-500 hover:text-gray-700"
              >
                退出
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === "rules" && (
          <div className="bg-white shadow-sm border border-gray-200">
            {editingRuleId !== null ? (
              <RuleForm
                ruleId={editingRuleId}
                service={selection.service}
                component={selection.component}
                onSave={() => {
                  setEditingRuleId(null);
                  setConfiguringFromLine(null);
                  setConfiguringFromPattern(null);
                }}
                onCancel={() => {
                  setEditingRuleId(null);
                  setConfiguringFromLine(null);
                  setConfiguringFromPattern(null);
                }}
                initialLine={configuringFromLine?.lineNumber}
                initialFile={configuringFromLine?.file}
                initialFunction={configuringFromLine?.func}
                initialPattern={configuringFromPattern?.pattern}
              />
            ) : (
              <RuleList
                service={selection.service}
                component={selection.component}
                onEdit={setEditingRuleId}
                onCreate={() => setEditingRuleId("")}
              />
            )}
          </div>
        )}

        {activeTab === "analyzer" && (
          <div className="bg-white shadow-sm border border-gray-200">
            <LogAnalyzer service={selection.service} />
          </div>
        )}

        {activeTab === "report" && (
          <div className="bg-white shadow-sm border border-gray-200">
            <LogReport
              service={selection.service}
              onConfigureFromLine={(lineNumber, file, func) => {
                setConfiguringFromLine({ lineNumber, file, func });
                setEditingRuleId("");
                setActiveTab("rules");
              }}
              onConfigureFromPattern={(pattern) => {
                setConfiguringFromPattern({ pattern });
                setEditingRuleId("");
                setActiveTab("rules");
              }}
            />
          </div>
        )}

        {activeTab === "alerts" && (
          <div className="bg-white shadow-sm border border-gray-200">
            <Alerts service={selection.service} />
          </div>
        )}
      </main>

      <footer className="border-t border-gray-200 mt-10 bg-white">
        <div className="max-w-7xl mx-auto px-6 py-6">
          <div className="text-center text-gray-500 text-sm">
            Logos · 语义化日志系统 · 本地演示
          </div>
        </div>
      </footer>
    </div>
  );
}
