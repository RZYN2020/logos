import { useState, useEffect } from "react";
import { apiClient } from "../api/client";
import type { LogReport as LogReportType, LogPatternStat } from "../api/types";

interface LogReportProps {
  service: string;
  onConfigureFromLine: (lineNumber: number, file?: string, func?: string) => void;
  onConfigureFromPattern: (pattern: string) => void;
}

export default function LogReport({ service, onConfigureFromLine, onConfigureFromPattern }: LogReportProps) {
  const [report, setReport] = useState<LogReportType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPattern, setSelectedPattern] = useState<LogPatternStat | null>(null);

  useEffect(() => {
    const loadReport = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await apiClient.getReport(service);
        setReport(data);
      } catch (err) {
        console.error("Failed to load report:", err);
        setError(err instanceof Error ? err.message : "Failed to load report");
      } finally {
        setLoading(false);
      }
    };

    loadReport();
  }, [service]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center text-gray-500">
          <p>加载中...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="font-medium text-red-600">加载失败：{error}</p>
        <p className="text-sm text-gray-500 mt-2">请确保后端服务正在运行</p>
      </div>
    );
  }

  if (!report) {
    return (
      <div className="text-center py-12 text-gray-500">
        暂无报告数据
      </div>
    );
  }

  // 计算时间范围（如果没有服务端返回的时间范围，则使用当前日期）
  const timeRange = report.time_range || { from: new Date().toISOString(), to: new Date().toISOString() };

  return (
    <div className="space-y-6 p-6">
      {/* 报告概览 */}
      <div className="bg-white border border-gray-200 rounded-md shadow-sm p-6">
        <h2 className="text-lg font-bold text-gray-900 mb-4">报告概览</h2>
        <div className="grid grid-cols-3 gap-6">
          <div className="text-center p-4 bg-blue-50 border border-blue-100 rounded-md">
            <div className="text-3xl font-bold text-blue-700">
              {report.total_logs.toLocaleString()}
            </div>
            <div className="text-sm text-blue-600 mt-1">总日志数</div>
          </div>
          <div className="text-center p-4 bg-green-50 border border-green-100 rounded-md">
            <div className="text-lg font-bold text-green-700">
              {timeRange.from.slice(0, 10)}
            </div>
            <div className="text-sm text-green-600 mt-1">开始时间</div>
          </div>
          <div className="text-center p-4 bg-purple-50 border border-purple-100 rounded-md">
            <div className="text-lg font-bold text-purple-700">
              {timeRange.to.slice(0, 10)}
            </div>
            <div className="text-sm text-purple-600 mt-1">结束时间</div>
          </div>
        </div>
      </div>

      {/* TOP 行号 */}
      <div className="bg-white border border-gray-200 rounded-md shadow-sm p-6">
        <div className="flex items-center justify-between mb-4 border-b border-gray-200 pb-2">
          <h2 className="text-lg font-bold text-gray-900">TOP 日志行号</h2>
          <span className="text-sm text-gray-500">点击可配置规则</span>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">排名</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">文件</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">函数</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">行号</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">次数</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">占比</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {report.top_lines.map((line, index) => (
                <tr key={index} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium ${
                      index === 0 ? 'bg-yellow-100 text-yellow-800' :
                      index === 1 ? 'bg-gray-200 text-gray-800' :
                      index === 2 ? 'bg-orange-100 text-orange-800' :
                      'bg-gray-100 text-gray-600'
                    }`}>
                      {index + 1}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm font-medium text-gray-900">{line.file}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">{line.function}</td>
                  <td className="px-4 py-3">
                    <code className="px-2 py-1 bg-gray-100 border border-gray-200 rounded text-sm text-gray-800">
                      L{line.line_number}
                    </code>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-900">{line.count.toLocaleString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center">
                      <div className="w-24 h-2 bg-gray-200 rounded-full mr-2 overflow-hidden">
                        <div
                          className="h-full bg-blue-500 rounded-full"
                          style={{ width: `${line.percentage}%` }}
                        />
                      </div>
                      <span className="text-sm text-gray-600">{line.percentage}%</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => onConfigureFromLine(line.line_number, line.file, line.function)}
                      className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                    >
                      配置规则
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* TOP 模式 */}
      <div className="bg-white border border-gray-200 rounded-md shadow-sm p-6">
        <div className="flex items-center justify-between mb-4 border-b border-gray-200 pb-2">
          <h2 className="text-lg font-bold text-gray-900">TOP 日志模式</h2>
          <span className="text-sm text-gray-500">基于模式识别算法</span>
        </div>
        <div className="space-y-4">
          {report.top_patterns.map((pattern, index) => (
            <div
              key={index}
              className={`border rounded-md p-4 transition-all cursor-pointer ${
                selectedPattern === pattern
                  ? 'border-blue-500 bg-blue-50'
                  : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
              }`}
              onClick={() => setSelectedPattern(pattern)}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-3">
                  <span className={`inline-flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium ${
                    index === 0 ? 'bg-yellow-100 text-yellow-800' :
                    index === 1 ? 'bg-gray-200 text-gray-800' :
                    index === 2 ? 'bg-orange-100 text-orange-800' :
                    'bg-gray-100 text-gray-600'
                  }`}>
                    {index + 1}
                  </span>
                  <code className="text-sm font-medium text-gray-800 bg-gray-100 px-2 py-1 rounded border border-gray-200">
                    {pattern.pattern}
                  </code>
                </div>
                <div className="flex items-center gap-4">
                  <span className="text-sm text-gray-600">{pattern.count.toLocaleString()} 次</span>
                  <span className="text-sm font-semibold text-blue-600">{pattern.percentage}%</span>
                </div>
              </div>

              {/* 展开显示示例日志 */}
              {selectedPattern === pattern && (
                <div className="mt-3 pt-3 border-t border-gray-200">
                  <div className="text-xs text-gray-500 mb-2">示例日志：</div>
                  <div className="space-y-1">
                    {pattern.sample_logs.map((log, i) => (
                      <div key={i} className="text-sm text-gray-700 bg-gray-100 border border-gray-200 px-3 py-2 rounded">
                        {log}
                      </div>
                    ))}
                  </div>
                  <div className="mt-3 flex gap-2">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onConfigureFromPattern(pattern.pattern);
                      }}
                      className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                    >
                      基于此模式创建规则
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
