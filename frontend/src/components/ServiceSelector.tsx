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
    <div className="bg-white border border-gray-200 shadow-sm p-8 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 mb-2">
        选择服务
      </h1>
      <p className="text-gray-500 mb-8">
        请选择要配置日志规则的服务和组件
      </p>

      {/* 服务选择 */}
      <div className="mb-6">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          服务名称
        </label>
        <select
          value={selectedService}
          onChange={(e) => setSelectedService(e.target.value)}
          className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md border bg-gray-50"
        >
          <option value="" disabled>请选择服务...</option>
          {SERVICES.map(service => (
            <option key={service.name} value={service.name}>
              {service.name} - {service.description}
            </option>
          ))}
        </select>

        {/* 自定义服务输入 */}
        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            或输入自定义服务名
          </label>
          <input
            type="text"
            value={selectedService || ''}
            onChange={(e) => setSelectedService(e.target.value)}
            placeholder="输入服务名称..."
            className="mt-1 block w-full px-3 py-2 border border-gray-300 shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md bg-gray-50"
          />
        </div>
      </div>

      {/* 组件类型选择 */}
      <div className="mb-8">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          组件类型
        </label>
        <div className="flex space-x-4">
          <label className="flex items-center">
            <input
              type="radio"
              name="componentType"
              value="sdk"
              checked={selectedComponent === 'sdk'}
              onChange={() => setSelectedComponent('sdk')}
              className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300"
            />
            <span className="ml-2 text-sm text-gray-900">Log SDK</span>
          </label>
          <label className="flex items-center">
            <input
              type="radio"
              name="componentType"
              value="processor"
              checked={selectedComponent === 'processor'}
              onChange={() => setSelectedComponent('processor')}
              className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300"
            />
            <span className="ml-2 text-sm text-gray-900">Log Processor</span>
          </label>
        </div>
      </div>

      {/* 确认按钮 */}
      <div className="flex justify-end">
        <button
          type="button"
          onClick={handleSubmit}
          disabled={!selectedService}
          className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:bg-gray-300 disabled:cursor-not-allowed"
        >
          开始配置
        </button>
      </div>
    </div>
  );
}
