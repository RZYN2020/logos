// Package config 提供配置管理功能
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdConfig  holds etcd client configuration
type EtcdConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
	Username    string
	Password    string
}

// DefaultEtcdConfig returns default etcd configuration
func DefaultEtcdConfig() EtcdConfig {
	return EtcdConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
}

// EtcdClient wraps etcd client with retry logic
type EtcdClient struct {
	client  *clientv3.Client
	config  EtcdConfig
	closed  bool
}

// NewEtcdClient creates a new etcd client
func NewEtcdClient(cfg EtcdConfig) (*EtcdClient, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no etcd endpoints provided")
	}

	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = DefaultEtcdConfig().DialTimeout
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	return &EtcdClient{
		client: client,
		config: cfg,
	}, nil
}

// Get 获取单个键值
func (c *EtcdClient) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return resp.Kvs[0].Value, nil
}

// GetWithPrefix 获取指定前缀的所有键值
func (c *EtcdClient) GetWithPrefix(prefix string) (map[string][]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	resp, err := c.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get keys with prefix: %w", err)
	}

	result := make(map[string][]byte)
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = kv.Value
	}

	return result, nil
}

// Put 保存键值对
func (c *EtcdClient) Put(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	_, err := c.client.Put(ctx, key, string(value))
	if err != nil {
		return fmt.Errorf("failed to put key: %w", err)
	}

	return nil
}

// Delete 删除键
func (c *EtcdClient) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	_, err := c.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// Watch 监听键值变化
func (c *EtcdClient) Watch(key string) (<-chan WatchEvent, <-chan error) {
	eventCh := make(chan WatchEvent, 10)
	errCh := make(chan error, 1)

	go c.watch(key, eventCh, errCh)

	return eventCh, errCh
}

// WatchWithPrefix 监听指定前缀的所有键值变化
func (c *EtcdClient) WatchWithPrefix(prefix string) (<-chan WatchEvent, <-chan error) {
	eventCh := make(chan WatchEvent, 10)
	errCh := make(chan error, 1)

	go c.watchWithPrefix(prefix, eventCh, errCh)

	return eventCh, errCh
}

// WatchEvent 表示一个监听事件
type WatchEvent struct {
	Type  EventType
	Key   string
	Value []byte
}

// EventType 表示事件类型
type EventType int

const (
	// EventPut 表示键被创建或更新
	EventPut EventType = iota
	// EventDelete 表示键被删除
	EventDelete
)

// String 返回事件类型的字符串表示
func (t EventType) String() string {
	switch t {
	case EventPut:
		return "PUT"
	case EventDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

func (c *EtcdClient) watch(key string, eventCh chan<- WatchEvent, errCh chan<- error) {
	defer close(eventCh)
	defer close(errCh)

	watchChan := c.client.Watch(context.Background(), key)

	for watchResp := range watchChan {
		if watchResp.Err() != nil {
			select {
			case errCh <- watchResp.Err():
			default:
			}
			continue
		}

		for _, event := range watchResp.Events {
			switch event.Type {
			case clientv3.EventTypePut:
				select {
				case eventCh <- WatchEvent{
					Type:  EventPut,
					Key:   string(event.Kv.Key),
					Value: event.Kv.Value,
				}:
				case <-time.After(time.Second):
					fmt.Println("Event channel full, dropping event")
				}

			case clientv3.EventTypeDelete:
				select {
				case eventCh <- WatchEvent{
					Type:  EventDelete,
					Key:   string(event.Kv.Key),
					Value: nil,
				}:
				case <-time.After(time.Second):
					fmt.Println("Event channel full, dropping event")
				}
			}
		}
	}
}

func (c *EtcdClient) watchWithPrefix(prefix string, eventCh chan<- WatchEvent, errCh chan<- error) {
	defer close(eventCh)
	defer close(errCh)

	watchChan := c.client.Watch(context.Background(), prefix, clientv3.WithPrefix())

	for watchResp := range watchChan {
		if watchResp.Err() != nil {
			select {
			case errCh <- watchResp.Err():
			default:
			}
			continue
		}

		for _, event := range watchResp.Events {
			switch event.Type {
			case clientv3.EventTypePut:
				select {
				case eventCh <- WatchEvent{
					Type:  EventPut,
					Key:   string(event.Kv.Key),
					Value: event.Kv.Value,
				}:
				case <-time.After(time.Second):
					fmt.Println("Event channel full, dropping event")
				}

			case clientv3.EventTypeDelete:
				select {
				case eventCh <- WatchEvent{
					Type:  EventDelete,
					Key:   string(event.Kv.Key),
					Value: nil,
				}:
				case <-time.After(time.Second):
					fmt.Println("Event channel full, dropping event")
				}
			}
		}
	}
}

// Close 关闭 etcd 客户端
func (c *EtcdClient) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return c.client.Close()
}

// IsConnected 检查是否已连接到 etcd
func (c *EtcdClient) IsConnected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	_, err := c.client.Get(ctx, "/health")
	return err == nil
}

// UnmarshalJSON 辅助函数，用于反序列化 JSON 数据
func UnmarshalJSON[T any](data []byte) (*T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &result, nil
}
