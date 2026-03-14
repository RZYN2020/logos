import { useState } from "react";
import RuleList from "./components/RuleList";
import RuleForm from "./components/RuleForm";
import ServiceSelector from "./components/ServiceSelector";
import LogAnalyzer from "./components/LogAnalyzer";
import LogReport from "./components/LogReport";

type Tab = "rules" | "analyzer" | "report";
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

  const handleServiceSelect = (service: string, component: ComponentType) => {
    setSelection({ service, component });
  };

  // 如果未选择服务，显示服务选择器
  if (!selection) {
    return (
      <div className="min-h-screen bg-gray-50">
        <nav className="bg-white shadow-sm border-b border-gray-200">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between h-16">
              <div className="flex items-center">
                <div className="h-8 w-8 rounded-lg bg-blue-600 flex items-center justify-center">
                  <span className="text-white font-bold text-lg">L</span>
                </div>
                <span className="ml-3 text-xl font-bold text-gray-900">
                  Logos 日志系统
                </span>
              </div>
            </div>
          </div>
        </nav>
        <main className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <ServiceSelector onSelect={handleServiceSelect} />
        </main>
      </div>
    );
  }

  // 已选择服务，显示主界面
  return (
    <div className="min-h-screen bg-gray-50">
      {/* 顶部导航栏 */}
      <nav className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <div className="h-8 w-8 rounded-lg bg-blue-600 flex items-center justify-center">
                <span className="text-white font-bold text-lg">L</span>
              </div>
              <span className="ml-3 text-xl font-bold text-gray-900">
                Logos 日志系统
              </span>
              <div className="ml-6 flex items-center space-x-2">
                <span className="px-3 py-1 bg-blue-100 text-blue-700 rounded-full text-sm font-medium">
                  {selection.service}
                </span>
                <span className="px-3 py-1 bg-gray-100 text-gray-700 rounded-full text-sm font-medium">
                  {selection.component === 'sdk' ? 'SDK' : 'Processor'}
                </span>
                <button
                  onClick={() => setSelection(null)}
                  className="ml-2 text-xs text-gray-500 hover:text-gray-700"
                >
                  切换服务
                </button>
              </div>
            </div>

            <div className="flex items-center space-x-1">
              <button
                onClick={() => setActiveTab("rules")}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  activeTab === "rules"
                    ? "bg-blue-600 text-white"
                    : "text-gray-700 hover:bg-gray-100"
                }`}
              >
                规则配置
              </button>
              <button
                onClick={() => setActiveTab("analyzer")}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  activeTab === "analyzer"
                    ? "bg-blue-600 text-white"
                    : "text-gray-700 hover:bg-gray-100"
                }`}
              >
                日志分析
              </button>
              <button
                onClick={() => setActiveTab("report")}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  activeTab === "report"
                    ? "bg-blue-600 text-white"
                    : "text-gray-700 hover:bg-gray-100"
                }`}
              >
                日志报告
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* 主内容区域 */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === "rules" && (
          <>
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
          </>
        )}

        {activeTab === "analyzer" && (
          <LogAnalyzer service={selection.service} />
        )}

        {activeTab === "report" && (
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
        )}
      </main>

      {/* 页脚 */}
      <footer className="bg-white border-t border-gray-200 mt-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="text-center text-gray-500 text-sm">
            Logos 语义化日志系统 - 云原生日志规则配置平台
          </div>
        </div>
      </footer>
    </div>
  );
}
