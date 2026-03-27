package guard

import (
	"sync"
	"time"
)

// TokenBucketGuard 实现了一个简单的令牌桶限流器
type TokenBucketGuard struct {
	capacity  int64
	tokens    int64
	fillRate  int64 // 每次添加的令牌数
	fillTime  time.Duration // 多久添加一次
	lastFill  time.Time
	mu        sync.Mutex
}

// NewTokenBucketGuard 创建一个令牌桶，指定容量、填充速率和填充周期
func NewTokenBucketGuard(capacity int64, fillRate int64, fillTime time.Duration) *TokenBucketGuard {
	return &TokenBucketGuard{
		capacity: capacity,
		tokens:   capacity,
		fillRate: fillRate,
		fillTime: fillTime,
		lastFill: time.Now(),
	}
}

// Allow 尝试获取一个令牌，如果成功返回 true，否则返回 false
func (g *TokenBucketGuard) Allow() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(g.lastFill)
	
	// 如果过了填充时间，增加令牌
	if elapsed >= g.fillTime {
		// 计算可以添加几次 fillRate
		ticks := int64(elapsed / g.fillTime)
		g.tokens += ticks * g.fillRate
		if g.tokens > g.capacity {
			g.tokens = g.capacity
		}
		// 更新最后填充时间，注意保留多余的时间，使得限流更平滑
		g.lastFill = g.lastFill.Add(time.Duration(ticks) * g.fillTime)
	}

	if g.tokens > 0 {
		g.tokens--
		return true
	}
	return false
}
