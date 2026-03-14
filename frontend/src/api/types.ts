// API 类型定义 - 统一规则引擎格式

export interface Rule {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
  priority?: number;
  condition: Condition; // 单个复合条件，支持 all/any/not 嵌套
  actions: Action[];
  version?: number;
  created_at?: string;
  updated_at?: string;
}

// 条件类型 - 支持单条件和复合条件
export interface Condition {
  // 单条件字段
  field?: string;
  operator?: ConditionOperator;
  value?: unknown;

  // 复合条件字段
  all?: Condition[];  // AND 逻辑
  any?: Condition[];  // OR 逻辑
  not?: Condition;    // NOT 逻辑
}

// 条件操作符
export type ConditionOperator =
  | 'eq'          // 等于
  | 'ne'          // 不等于
  | 'gt'          // 大于
  | 'lt'          // 小于
  | 'ge'          // 大于等于
  | 'le'          // 小于等于
  | 'contains'    // 包含
  | 'starts_with' // 开始于
  | 'ends_with'   // 结束于
  | 'matches'     // 正则匹配
  | 'in'          // 在集合中
  | 'not_in'      // 不在集合中
  | 'exists'      // 字段存在
  | 'not_exists'; // 字段不存在

// 动作类型
export type ActionType =
  | 'keep'     // 保留并终止
  | 'drop'     // 丢弃并终止
  | 'sample'   // 采样
  | 'mask'     // 掩码敏感数据
  | 'truncate' // 截断字段
  | 'extract'  // 提取子串
  | 'rename'   // 重命名字段
  | 'remove'   // 删除字段
  | 'set'      // 设置字段值
  | 'mark';    // 添加标记

export interface Action {
  type: ActionType;
  config?: ActionConfig;
}

// 动作配置 - 根据类型不同而不同
export interface ActionConfig {
  // sample 动作
  rate?: number;  // 采样率 0.0-1.0

  // mask 动作
  field?: string;
  pattern?: string;  // 正则表达式，可选

  // truncate 动作
  max_length?: number;
  suffix?: string;

  // extract 动作
  source_field?: string;
  target_field?: string;

  // rename 动作
  from?: string;
  to?: string;

  // remove 动作
  fields?: string[];  // 字段列表

  // set 动作
  value?: unknown;

  // mark 动作
  reason?: string;
}

export interface RuleVersion {
  id: string;
  rule_id: string;
  version: number;
  content: Record<string, unknown>;
  author: string;
  comment?: string;
  created_at: string;
}

// 兼容旧版 Strategy 类型
export interface Strategy {
  id: string;
  name: string;
  description: string;
  rules: StrategyRule[];
  version: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  author: string;
  metadata?: Record<string, unknown>;
}

export interface StrategyRule {
  condition: Record<string, unknown>;
  action: Record<string, unknown>;
}

export interface ApiResponse<T = unknown> {
  code?: number;
  message?: string;
  data?: T;
}

// 服务配置相关类型
export interface ServiceConfig {
  name: string;
  components: ServiceComponent[];
}

export interface ServiceComponent {
  type: 'sdk' | 'processor';
  name: string;
  version?: string;
}

// 日志报告类型
export interface LogReport {
  service: string;
  total_logs: number;
  time_range: {
    from: string;
    to: string;
  };
  top_lines: LogLineStat[];
  top_patterns: LogPatternStat[];
}

export interface LogLineStat {
  line_number: number;
  file?: string;
  function?: string;
  count: number;
  percentage: number;
}

export interface LogPatternStat {
  pattern: string;
  count: number;
  percentage: number;
  sample_logs: string[];
}
