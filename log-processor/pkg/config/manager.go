// Package config 提供配置管理功能
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	LoadFilters() ([]*FilterConfig, error)
	WatchFilters() (<-chan FilterEvent, <-chan error)
	GetActiveFilters() []*FilterConfig
	GetFilterByID(id string) *FilterConfig
	Close() error
}

// EtcdConfigManager ETCD 配置管理器实现
type EtcdConfigManager struct {
	client       *EtcdClient
	mu           sync.RWMutex
	filters      map[string]*FilterConfig
	configs      []*FilterConfig // 按优先级排序
	refreshDone  chan struct{}
	stopCh       chan struct{}
	refreshDoneM sync.Mutex
}

// ConfigManagerConfig 配置管理器配置
type ConfigManagerConfig struct {
	EtcdEndpoints   []string
	EtcdUsername    string
	EtcdPassword    string
	RefreshInterval time.Duration
}

// DefaultConfigManagerConfig 返回默认配置
func DefaultConfigManagerConfig() ConfigManagerConfig {
	return ConfigManagerConfig{
		EtcdEndpoints:   []string{"localhost:2379"},
		RefreshInterval: 30 * time.Second,
	}
}

// NewEtcdConfigManager 创建 ETCD 配置管理器
func NewEtcdConfigManager(cfg ConfigManagerConfig) (*EtcdConfigManager, error) {
	etcdCfg := EtcdConfig{
		Endpoints:   cfg.EtcdEndpoints,
		DialTimeout: 5 * time.Second,
		Username:    cfg.EtcdUsername,
		Password:    cfg.EtcdPassword,
	}

	client, err := NewEtcdClient(etcdCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	manager := &EtcdConfigManager{
		client:      client,
		filters:     make(map[string]*FilterConfig),
		configs:     make([]*FilterConfig, 0),
		refreshDone: make(chan struct{}),
		stopCh:      make(chan struct{}),
	}

	// 初始加载
	if err := manager.loadAllFilters(); err != nil {
		log.Printf("Failed to load initial filters: %v", err)
	}

	// 启动周期性刷新
	if cfg.RefreshInterval > 0 {
		go manager.refreshLoop(cfg.RefreshInterval)
	}

	// 启动监听
	go manager.watchLoop()

	return manager, nil
}

// loadAllFilters 从 ETCD 加载所有过滤配置
func (m *EtcdConfigManager) loadAllFilters() error {
	data, err := m.client.GetWithPrefix(FilterConfigPrefix)
	if err != nil {
		return fmt.Errorf("failed to get filters from etcd: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空现有配置
	m.filters = make(map[string]*FilterConfig)

	for key, value := range data {
		var config FilterConfig
		if err := json.Unmarshal(value, &config); err != nil {
			log.Printf("Failed to unmarshal filter config %s: %v", key, err)
			continue
		}
		m.filters[config.ID] = &config
	}

	// 按优先级排序
	m.sortConfigs()

	return nil
}

// sortConfigs 按优先级排序配置
func (m *EtcdConfigManager) sortConfigs() {
	m.configs = make([]*FilterConfig, 0, len(m.filters))
	for _, config := range m.filters {
		m.configs = append(m.configs, config)
	}

	// 按优先级降序排序（优先级高的在前）
	for i := 0; i < len(m.configs)-1; i++ {
		for j := i + 1; j < len(m.configs); j++ {
			if m.configs[i].Priority < m.configs[j].Priority {
				m.configs[i], m.configs[j] = m.configs[j], m.configs[i]
			}
		}
	}
}

// refreshLoop 周期性刷新配置
func (m *EtcdConfigManager) refreshLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.loadAllFilters(); err != nil {
				log.Printf("Failed to refresh filters: %v", err)
			}
			m.refreshDoneM.Lock()
			select {
			case m.refreshDone <- struct{}{}:
			default:
			}
			m.refreshDoneM.Unlock()

		case <-m.stopCh:
			return
		}
	}
}

// watchLoop 监听配置变更
func (m *EtcdConfigManager) watchLoop() {
	eventCh, errCh := m.client.WatchWithPrefix(FilterConfigPrefix)

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			m.handleWatchEvent(event)

		case err, ok := <-errCh:
			if !ok {
				return
			}
			log.Printf("ETCD watch error: %v", err)

		case <-m.stopCh:
			return
		}
	}
}

// handleWatchEvent 处理监听事件
func (m *EtcdConfigManager) handleWatchEvent(event WatchEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch event.Type {
	case EventPut:
		var config FilterConfig
		if err := json.Unmarshal(event.Value, &config); err != nil {
			log.Printf("Failed to unmarshal filter config update: %v", err)
			return
		}
		m.filters[config.ID] = &config
		log.Printf("Filter config updated: %s", config.ID)

	case EventDelete:
		// 从 key 中提取 ID
		id := extractIDFromKey(event.Key)
		if id != "" {
			delete(m.filters, id)
			log.Printf("Filter config deleted: %s", id)
		}
	}

	// 重新排序
	m.sortConfigs()
}

// extractIDFromKey 从 ETCD key 中提取 ID
func extractIDFromKey(key string) string {
	// 期望格式：/log-processor/filters/{id}
	prefix := FilterConfigPrefix
	if len(key) <= len(prefix) {
		return ""
	}
	return key[len(prefix):]
}

// LoadFilters 加载所有过滤配置
func (m *EtcdConfigManager) LoadFilters() ([]*FilterConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*FilterConfig, len(m.configs))
	copy(result, m.configs)
	return result, nil
}

// WatchFilters 监听过滤配置变更
func (m *EtcdConfigManager) WatchFilters() (<-chan FilterEvent, <-chan error) {
	eventCh := make(chan FilterEvent, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		etcdEventCh, etcdErrCh := m.client.WatchWithPrefix(FilterConfigPrefix)

		for {
			select {
			case event, ok := <-etcdEventCh:
				if !ok {
					return
				}

				var filterEvent FilterEvent
				filterEvent.Type = event.Type

				if event.Type == EventPut {
					var config FilterConfig
					if err := json.Unmarshal(event.Value, &config); err != nil {
						log.Printf("Failed to unmarshal filter event: %v", err)
						continue
					}
					filterEvent.Config = &config
				}

				select {
				case eventCh <- filterEvent:
				case <-time.After(time.Second):
					log.Println("Filter event channel full, dropping event")
				}

			case err, ok := <-etcdErrCh:
				if !ok {
					return
				}
				select {
				case errCh <- err:
				default:
				}

			case <-m.stopCh:
				return
			}
		}
	}()

	return eventCh, errCh
}

// GetActiveFilters 获取所有激活的过滤配置
func (m *EtcdConfigManager) GetActiveFilters() []*FilterConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*FilterConfig, 0)
	for _, config := range m.configs {
		if config.Enabled {
			result = append(result, config)
		}
	}
	return result
}

// GetFilterByID 根据 ID 获取过滤配置
func (m *EtcdConfigManager) GetFilterByID(id string) *FilterConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if config, ok := m.filters[id]; ok {
		return config
	}
	return nil
}

// Close 关闭配置管理器
func (m *EtcdConfigManager) Close() error {
	close(m.stopCh)
	return m.client.Close()
}

// RefreshDone 返回刷新完成信号
func (m *EtcdConfigManager) RefreshDone() <-chan struct{} {
	return m.refreshDone
}
