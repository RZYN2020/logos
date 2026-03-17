import { useState, useEffect } from "react";
import { apiClient } from "../api/client";

interface LogAnalyzerProps {
  service: string;
}

interface LogEntry {
  timestamp: string;
  level: string;
  service: string;
  message: string;
  trace_id: string;
  user_id: string;
  error_code?: string;
}

export default function LogAnalyzer({ service }: LogAnalyzerProps) {
  const [query, setQuery] = useState<string>("SELECT * FROM logs WHERE service = 'api-gateway' LIMIT 100");
  const [results, setResults] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 当服务改变时，更新查询
  useEffect(() => {
    setQuery(`SELECT * FROM logs WHERE service = '${service}' ORDER BY timestamp DESC LIMIT 100`);
  }, [service]);

  const handleRunQuery = async () => {
    setLoading(true);
    setError(null);
    try {
      // 调用真实 API 查询日志
      const response = await apiClient.queryLogs({
        service: service,
        limit: 100,
      });

      const apiResults = (response.logs || []).map((log: unknown) => {
        const entry = log as Record<string, unknown>;
        return {
          timestamp: (entry.timestamp as string) || new Date().toISOString(),
          level: (entry.level as string) || "INFO",
          service: (entry.service as string) || service,
          message: (entry.message as string) || "",
          trace_id: (entry.trace_id as string) || "",
          user_id: (entry.user_id as string) || "",
          error_code: (entry.error_code as string) || "",
        };
      });

      setResults(apiResults);
    } catch (err) {
      setError("查询失败");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const setExampleQueries = () => {
    const queries: Record<string, string> = {
      recentErrors: `SELECT * FROM logs
WHERE level = 'ERROR'
  AND timestamp > NOW() - INTERVAL 1 HOUR
ORDER BY timestamp DESC
LIMIT 100;`,

      errorByService: `SELECT
  service,
  COUNT(*) as error_count
FROM logs
WHERE level = 'ERROR'
  AND timestamp > NOW() - INTERVAL 24 HOUR
GROUP BY service
ORDER BY error_count DESC;`,

      traceLogs: `SELECT * FROM logs
WHERE trace_id = '7a3c9f8d5e2b1a4'
ORDER BY timestamp;`,

      slowRequests: `SELECT * FROM logs
WHERE duration_ms > 1000
  AND timestamp > NOW() - INTERVAL 1 HOUR
ORDER BY duration_ms DESC
LIMIT 50;`,
    };

    const keys = Object.keys(queries);
    const randomKey = keys[Math.floor(Math.random() * keys.length)];
    setQuery(queries[randomKey]);
  };

  return (
    <div className="px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            日志分析
          </h1>
          <p className="text-sm text-gray-500 mt-1">
            分析 {service} 的日志数据
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 查询编辑器 */}
        <div className="lg:col-span-1">
          <div className="bg-white border border-gray-200 shadow-sm rounded-md p-6">
            <div className="mb-4 flex justify-between items-center">
              <h2 className="text-lg font-bold text-gray-900">
                SQL 查询编辑器
              </h2>
              <button
                onClick={setExampleQueries}
                className="text-sm text-blue-600 hover:text-blue-800 font-medium"
              >
                填入示例查询
              </button>
            </div>

            <textarea
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full h-64 px-4 py-3 border border-gray-300 rounded-md font-mono text-sm text-gray-900 outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-gray-50"
              placeholder="输入 SQL 查询语句..."
            />

            <button
              onClick={handleRunQuery}
              disabled={loading}
              className="mt-4 w-full inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:bg-gray-400"
            >
              {loading ? "查询中..." : "执行查询"}
            </button>

            {error && (
              <div className="mt-4 bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded text-sm">
                {error}
              </div>
            )}
          </div>

          {/* 快捷查询 */}
          <div className="bg-white border border-gray-200 shadow-sm rounded-md p-6 mt-6">
            <h2 className="text-lg font-bold text-gray-900 mb-4">
              快捷查询
            </h2>
            <div className="space-y-2">
              <button
                onClick={() => setQuery(`SELECT * FROM logs WHERE level = 'ERROR' ORDER BY timestamp DESC LIMIT 100`)}
                className="w-full text-left px-4 py-2 bg-red-50 text-red-700 rounded-md border border-red-200 hover:bg-red-100 transition"
              >
                最近错误日志
              </button>
              <button
                onClick={() => setQuery(`SELECT service, COUNT(*) as count FROM logs WHERE timestamp > NOW() - INTERVAL 1 HOUR GROUP BY service ORDER BY count DESC`)}
                className="w-full text-left px-4 py-2 bg-yellow-50 text-yellow-700 rounded-md border border-yellow-200 hover:bg-yellow-100 transition"
              >
                按服务统计
              </button>
              <button
                onClick={() => setQuery(`SELECT level, COUNT(*) as count FROM logs GROUP BY level ORDER BY count DESC`)}
                className="w-full text-left px-4 py-2 bg-blue-50 text-blue-700 rounded-md border border-blue-200 hover:bg-blue-100 transition"
              >
                按级别统计
              </button>
              <button
                onClick={() => setQuery(`SELECT user_id, COUNT(*) as request_count FROM logs WHERE event_type = 'request' GROUP BY user_id ORDER BY request_count DESC LIMIT 10`)}
                className="w-full text-left px-4 py-2 bg-green-50 text-green-700 rounded-md border border-green-200 hover:bg-green-100 transition"
              >
                活跃用户排行
              </button>
            </div>
          </div>
        </div>

        {/* 查询结果 */}
        <div className="lg:col-span-2">
          <div className="bg-white border border-gray-200 shadow-sm rounded-md p-6">
            <div className="flex justify-between items-center mb-4 border-b border-gray-200 pb-2">
              <h2 className="text-lg font-bold text-gray-900">
                查询结果 ({results.length} 条)
              </h2>
              {results.length > 0 && (
                <button className="text-sm text-blue-600 hover:text-blue-800 font-medium">
                  导出结果
                </button>
              )}
            </div>

            {results.length === 0 && !loading && (
              <div className="text-center py-12 text-gray-500">
                执行查询后在此查看结果
              </div>
            )}

            {results.length > 0 && (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        时间戳
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        级别
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        服务
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        消息
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        Trace ID
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                        用户 ID
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {results.map((log, index) => (
                      <tr key={index} className="hover:bg-gray-50">
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                          {new Date(log.timestamp).toLocaleString("zh-CN")}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {log.level === "ERROR" ? (
                            <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-red-100 text-red-800">
                              ERROR
                            </span>
                          ) : log.level === "WARN" ? (
                            <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800">
                              WARN
                            </span>
                          ) : (
                            <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                              INFO
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                          {log.service}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-700">
                          {log.message}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-xs text-gray-500 font-mono">
                          {log.trace_id}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                          {log.user_id}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
