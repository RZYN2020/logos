// Package integration ETCD 配置分发测试
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/log-system/log-analyzer/internal/etcd"
	"github.com/log-system/log-analyzer/internal/models"
	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestETCD_ConfigDistribution ETCD 配置分发测试
func TestETCD_ConfigDistribution(t *testing.T) {
	// 创建 ETCD 客户端
	cfg := etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 2 * time.Second,
	}

	cli, err := etcd.NewClient(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 ETCD (%v)", err)
		return
	}
	defer cli.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = cli.Get(ctx, "/analyzer/health")
	if err != nil {
		t.Skipf("跳过测试：ETCD 不可用 (%v)", err)
		return
	}

	ctx = context.Background()

	// 测试 1: 存储规则配置
	t.Run("PutRuleConfig", func(t *testing.T) {
		rule := &models.Rule{
			ID:          "test-rule-001",
			Name:        "test-rule",
			Description: "Test rule for ETCD distribution",
			Enabled:     true,
			Priority:    1,
			Version:     1,
		}

		key := "/analyzer/config/rules/test-rule-001"
		err := cli.Put(ctx, key, rule)
		assert.NoError(t, err)

		// 验证存储
		value, err := cli.Get(ctx, key)
		assert.NoError(t, err)
		assert.NotNil(t, value)

		var storedRule models.Rule
		err = json.Unmarshal(value, &storedRule)
		assert.NoError(t, err)
		assert.Equal(t, rule.ID, storedRule.ID)
		assert.Equal(t, rule.Name, storedRule.Name)
	})

	// 测试 2: 列出所有规则配置
	t.Run("ListRuleConfigs", func(t *testing.T) {
		prefix := "/analyzer/config/rules/"
		configs, err := cli.List(ctx, prefix)
		assert.NoError(t, err)
		assert.Greater(t, len(configs), 0)

		for key, value := range configs {
			assert.NotEmpty(t, key)
			assert.NotNil(t, value)
		}
	})

	// 测试 3: 更新规则配置
	t.Run("UpdateRuleConfig", func(t *testing.T) {
		key := "/analyzer/config/rules/test-rule-001"

		rule := &models.Rule{
			ID:          "test-rule-001",
			Name:        "updated-rule",
			Description: "Updated test rule",
			Enabled:     false,
			Priority:    2,
			Version:     2,
		}

		err := cli.Put(ctx, key, rule)
		assert.NoError(t, err)

		// 验证更新
		value, err := cli.Get(ctx, key)
		assert.NoError(t, err)

		var storedRule models.Rule
		err = json.Unmarshal(value, &storedRule)
		assert.NoError(t, err)
		assert.Equal(t, "updated-rule", storedRule.Name)
		assert.Equal(t, false, storedRule.Enabled)
		assert.Equal(t, 2, storedRule.Version)
	})

	// 测试 4: 删除规则配置
	t.Run("DeleteRuleConfig", func(t *testing.T) {
		key := "/analyzer/config/rules/test-rule-001"
		err := cli.Delete(ctx, key)
		assert.NoError(t, err)

		// 验证删除
		_, err = cli.Get(ctx, key)
		assert.Error(t, err)
	})
}

// TestETCD_WatchConfig ETCD 配置监听测试
func TestETCD_WatchConfig(t *testing.T) {
	cfg := etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 2 * time.Second,
	}

	cli, err := etcd.NewClient(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 ETCD (%v)", err)
		return
	}
	defer cli.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = cli.Get(ctx, "/analyzer/health")
	if err != nil {
		t.Skipf("跳过测试：ETCD 不可用 (%v)", err)
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建原生 ETCD 客户端用于 Watch
	nativeCli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		t.Skipf("跳过测试：无法创建原生 ETCD 客户端 (%v)", err)
		return
	}
	defer nativeCli.Close()

	key := "/analyzer/config/watch-test"
	value := "initial-value"

	// 存储初始值
	_, err = nativeCli.Put(ctx, key, value)
	if err != nil {
		t.Skipf("跳过测试：无法存储初始值 (%v)", err)
		return
	}

	// 开始监听
	watchChan := nativeCli.Watch(ctx, key)

	// 在 goroutine 中修改值
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, _ = nativeCli.Put(ctx, key, "updated-value")
		time.Sleep(100 * time.Millisecond)
		_, _ = nativeCli.Put(ctx, key, "final-value")
	}()

	// 接收变更
	var changeCount int
	for changeCount < 2 {
		select {
		case watchResp := <-watchChan:
			for _, event := range watchResp.Events {
				if event.Type == clientv3.EventTypePut {
					changeCount++
					t.Logf("检测到配置变更：%s", string(event.Kv.Value))
				}
			}
		case <-ctx.Done():
			break
		}
	}

	assert.Equal(t, 2, changeCount)

	// 清理
	_, _ = nativeCli.Delete(ctx, key)
}

// TestETCD_DistributionLatency ETCD 配置分发延迟测试
func TestETCD_DistributionLatency(t *testing.T) {
	cfg := etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 2 * time.Second,
	}

	cli, err := etcd.NewClient(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 ETCD (%v)", err)
		return
	}
	defer cli.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = cli.Get(ctx, "/analyzer/health")
	if err != nil {
		t.Skipf("跳过测试：ETCD 不可用 (%v)", err)
		return
	}

	ctx = context.Background()

	// 测试 100 次写入延迟
	var latencies []time.Duration
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("/analyzer/config/latency-test-%d", i)
		rule := &models.Rule{
			ID:      fmt.Sprintf("rule-%d", i),
			Name:    fmt.Sprintf("rule-%d", i),
			Version: 1,
		}

		start := time.Now()
		err := cli.Put(ctx, key, rule)
		latency := time.Since(start)

		assert.NoError(t, err)
		latencies = append(latencies, latency)
	}

	// 计算统计
	var total time.Duration
	var max time.Duration
	for _, l := range latencies {
		total += l
		if l > max {
			max = l
		}
	}

	avgLatency := total / time.Duration(len(latencies))

	t.Logf("ETCD 写入延迟测试:")
	t.Logf("  平均延迟：%v", avgLatency)
	t.Logf("  最大延迟：%v", max)

	// 断言平均延迟小于 50ms
	assert.Less(t, avgLatency, 50*time.Millisecond, "平均写入延迟应该小于 50ms")

	// 清理
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("/analyzer/config/latency-test-%d", i)
		_ = cli.Delete(ctx, key)
	}
}

// TestETCD_ConcurrentAccess ETCD 并发访问测试
func TestETCD_ConcurrentAccess(t *testing.T) {
	cfg := etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 2 * time.Second,
	}

	cli, err := etcd.NewClient(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 ETCD (%v)", err)
		return
	}
	defer cli.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = cli.Get(ctx, "/analyzer/health")
	if err != nil {
		t.Skipf("跳过测试：ETCD 不可用 (%v)", err)
		return
	}

	ctx = context.Background()
	key := "/analyzer/config/concurrent-test"

	// 并发写入
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			rule := &models.Rule{
				ID:      fmt.Sprintf("concurrent-rule-%d", id),
				Name:    fmt.Sprintf("concurrent-rule-%d", id),
				Version: 1,
			}
			_ = cli.Put(ctx, key, rule)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证最后一次写入
	value, err := cli.Get(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, value)

	// 清理
	_ = cli.Delete(ctx, key)
}
