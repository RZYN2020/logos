package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Engine 策略引擎
type Engine struct {
	client     *clientv3.Client
	strategies map[string]Strategy
	mu         sync.RWMutex
	watchChan  clientv3.WatchChan
}

// Strategy 策略定义
type Strategy struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
	Rules    []Rule `json:"rules"`
	Version  string `json:"version"`
}

// Rule 策略规则
type Rule struct {
	Name      string    `json:"name"`
	Condition Condition `json:"condition"`
	Action    Action    `json:"action"`
}

// Condition 匹配条件
type Condition struct {
	Level       string `json:"level,omitempty"`
	Service     string `json:"service,omitempty"`
	Environment string `json:"environment,omitempty"`
	PathPattern string `json:"path_pattern,omitempty"`
}

// Action 执行动作
type Action struct {
	Enabled   bool    `json:"enabled"`
	Sampling  float64 `json:"sampling"`            // 采样率 0.0-1.0
	Priority  string  `json:"priority"`            // high/normal/low
	Transform string  `json:"transform,omitempty"` // mask/hash/none
}

// Decision 策略决策结果
type Decision struct {
	ShouldLog bool
	Sampling  float64
	Priority  string
	Transform string
}

// NewEngine 创建策略引擎
func NewEngine(etcdEndpoints []string) (*Engine, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	engine := &Engine{
		client:     client,
		strategies: make(map[string]Strategy),
	}

	// 加载初始策略
	if err := engine.loadStrategies(); err != nil {
		return nil, err
	}

	// 启动监听
	go engine.watchStrategies()

	return engine, nil
}

// Evaluate 评估日志是否应记录
func (e *Engine) Evaluate(level, service, environment string, fields map[string]interface{}) Decision {
	e.mu.RLock()
	defer e.mu.RUnlock()

	decision := Decision{
		ShouldLog: true,
		Sampling:  1.0,
		Priority:  "normal",
		Transform: "none",
	}

	for _, strategy := range e.strategies {
		if !strategy.Enabled {
			continue
		}

		for _, rule := range strategy.Rules {
			if e.matchCondition(rule.Condition, level, service, environment, fields) {
				// 应用动作
				if !rule.Action.Enabled {
					decision.ShouldLog = false
					return decision
				}

				if rule.Action.Sampling > 0 && rule.Action.Sampling < 1.0 {
					decision.Sampling = rule.Action.Sampling
				}

				if rule.Action.Priority != "" {
					decision.Priority = rule.Action.Priority
				}

				if rule.Action.Transform != "" {
					decision.Transform = rule.Action.Transform
				}
			}
		}
	}

	// 应用采样
	if decision.Sampling < 1.0 {
		if rand.Float64() > decision.Sampling {
			decision.ShouldLog = false
		}
	}

	return decision
}

// matchCondition 检查是否匹配条件
func (e *Engine) matchCondition(cond Condition, level, service, environment string, fields map[string]interface{}) bool {
	if cond.Level != "" && cond.Level != level {
		return false
	}
	if cond.Service != "" && cond.Service != service {
		return false
	}
	if cond.Environment != "" && cond.Environment != environment {
		return false
	}
	// 路径模式匹配
	if cond.PathPattern != "" {
		path, ok := fields["path"].(string)
		if !ok {
			return false // 没有 path 字段，不匹配
		}
		matched, err := filepath.Match(cond.PathPattern, path)
		if err != nil || !matched {
			return false
		}
	}
	return true
}

// loadStrategies 从etcd加载策略
func (e *Engine) loadStrategies() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, "/strategies/", clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to load strategies: %w", err)
	}

	for _, kv := range resp.Kvs {
		var strategy Strategy
		if err := json.Unmarshal(kv.Value, &strategy); err != nil {
			continue
		}
		e.strategies[strategy.ID] = strategy
	}

	return nil
}

// watchStrategies 监听策略变更
func (e *Engine) watchStrategies() {
	e.watchChan = e.client.Watch(context.Background(), "/strategies/", clientv3.WithPrefix())

	for watchResp := range e.watchChan {
		for _, event := range watchResp.Events {
			switch event.Type {
			case clientv3.EventTypePut:
				var strategy Strategy
				if err := json.Unmarshal(event.Kv.Value, &strategy); err == nil {
					e.mu.Lock()
					e.strategies[strategy.ID] = strategy
					e.mu.Unlock()
				}
			case clientv3.EventTypeDelete:
				key := string(event.Kv.Key)
				id := extractIDFromKey(key)
				e.mu.Lock()
				delete(e.strategies, id)
				e.mu.Unlock()
			}
		}
	}
}

// Close 关闭引擎
func (e *Engine) Close() error {
	return e.client.Close()
}

func extractIDFromKey(key string) string {
	// 从 "/strategies/{id}" 提取 id
	prefix := "/strategies/"
	if len(key) <= len(prefix) {
		return ""
	}
	return key[len(prefix):]
}
