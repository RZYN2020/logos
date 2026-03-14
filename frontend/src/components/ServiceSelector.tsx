import { useState } from "react";

interface ServiceSelectorProps {
  onSelect: (service: string, component: 'sdk' | 'processor') => void;
}

// 预定义的服务列表
const SERVICES = [
  { name: 'api-gateway', description: 'API 网关服务' },
  { name: 'user-service', description: '用户服务' },
  { name: 'order-service', description: '订单服务' },
  { name: 'payment-service', description: '支付服务' },
  { name: 'notification-service', description: '通知服务' },
  { name: 'analytics-service', description: '分析服务' },
];

export default function ServiceSelector({ onSelect }: ServiceSelectorProps) {
  const [selectedService, setSelectedService] = useState<string>('');
  const [selectedComponent, setSelectedComponent] = useState<'sdk' | 'processor'>('sdk');

  const handleSubmit = () => {
    if (selectedService) {
      onSelect(selectedService, selectedComponent);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-lg p-8">
      <h1 className="text-2xl font-bold text-gray-900 mb-2">
        选择服务
      </h1>
      <p className="text-gray-500 mb-8">
        请选择要配置日志规则的服务和组件
      </p>

      {/* 服务选择 */}
      <div className="mb-6">
        <label className="block text-sm font-medium text-gray-700 mb-3">
          选择服务
        </label>
        <div className="grid grid-cols-2 gap-3">
          {SERVICES.map(service => (
            <button
              key={service.name}
              type="button"
              onClick={() => setSelectedService(service.name)}
              className={`p-4 rounded-lg border-2 text-left transition-all ${
                selectedService === service.name
                  ? 'border-blue-600 bg-blue-50'
                  : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
              }`}
            >
              <div className="font-medium text-gray-900">{service.name}</div>
              <div className="text-sm text-gray-500 mt-1">{service.description}</div>
            </button>
          ))}
        </div>

        {/* 自定义服务输入 */}
        <div className="mt-4">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex-1 h-px bg-gray-200"></div>
            <span className="text-sm text-gray-500">或输入自定义服务名</span>
            <div className="flex-1 h-px bg-gray-200"></div>
          </div>
          <input
            type="text"
            value={selectedService || ''}
            onChange={(e) => setSelectedService(e.target.value)}
            placeholder="输入服务名称..."
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      {/* 组件类型选择 */}
      <div className="mb-8">
        <label className="block text-sm font-medium text-gray-700 mb-3">
          组件类型
        </label>
        <div className="grid grid-cols-2 gap-3">
          <button
            type="button"
            onClick={() => setSelectedComponent('sdk')}
            className={`p-4 rounded-lg border-2 transition-all ${
              selectedComponent === 'sdk'
                ? 'border-blue-600 bg-blue-50'
                : 'border-gray-200 hover:border-gray-300'
            }`}
          >
            <div className="flex items-center">
              <div className="w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center mr-3">
                <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                </svg>
              </div>
              <div>
                <div className="font-medium text-gray-900">Log SDK</div>
                <div className="text-sm text-gray-500">应用内嵌 SDK</div>
              </div>
            </div>
          </button>

          <button
            type="button"
            onClick={() => setSelectedComponent('processor')}
            className={`p-4 rounded-lg border-2 transition-all ${
              selectedComponent === 'processor'
                ? 'border-blue-600 bg-blue-50'
                : 'border-gray-200 hover:border-gray-300'
            }`}
          >
            <div className="flex items-center">
              <div className="w-10 h-10 rounded-lg bg-green-100 flex items-center justify-center mr-3">
                <svg className="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                </svg>
              </div>
              <div>
                <div className="font-medium text-gray-900">Log Processor</div>
                <div className="text-sm text-gray-500">独立处理器</div>
              </div>
            </div>
          </button>
        </div>
      </div>

      {/* 确认按钮 */}
      <button
        type="button"
        onClick={handleSubmit}
        disabled={!selectedService}
        className="w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
      >
        开始配置
      </button>
    </div>
  );
}
