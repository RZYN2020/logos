// Package etcd ETCD 客户端封装
package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client ETCD 客户端
type Client struct {
	cli *clientv3.Client
}

// Config ETCD 配置
type Config struct {
	Endpoints   []string
	DialTimeout time.Duration
}

// NewClient 创建 ETCD 客户端
func NewClient(cfg Config) (*Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.cli.Close()
}

// Put 存储键值对
func (c *Client) Put(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	_, err = c.cli.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to put key: %w", err)
	}

	return nil
}

// Get 获取键值
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := c.cli.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return resp.Kvs[0].Value, nil
}

// Delete 删除键
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.cli.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// List 列出指定前缀的所有键
func (c *Client) List(ctx context.Context, prefix string) (map[string][]byte, error) {
	resp, err := c.cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	result := make(map[string][]byte)
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = kv.Value
	}

	return result, nil
}

// Watch 监听键变化
func (c *Client) Watch(ctx context.Context, key string) clientv3.WatchChan {
	return c.cli.Watch(ctx, key)
}

// WatchPrefix 监听前缀变化
func (c *Client) WatchPrefix(ctx context.Context, prefix string) clientv3.WatchChan {
	return c.cli.Watch(ctx, prefix, clientv3.WithPrefix())
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.cli.Get(ctx, "health")
	return err
}
