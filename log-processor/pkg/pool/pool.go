// Package pool 提供对象池功能，用于内存优化
package pool

import (
	"sync"
	"time"

	"github.com/log-system/log-processor/pkg/parser"
)

// ParsedLogPool 解析结果对象池
type ParsedLogPool struct {
	pool sync.Pool
}

// NewParsedLogPool 创建解析结果对象池
func NewParsedLogPool() *ParsedLogPool {
	return &ParsedLogPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &parser.ParsedLog{
					Fields: make(map[string]interface{}, 16),
				}
			},
		},
	}
}

// Get 从池中获取对象
func (p *ParsedLogPool) Get() *parser.ParsedLog {
	log := p.pool.Get().(*parser.ParsedLog)
	// 重置字段
	log.Timestamp = time.Time{}
	log.Level = ""
	log.Message = ""
	log.Service = ""
	log.TraceID = ""
	log.SpanID = ""
	log.Format = ""
	// 清空 Fields 但不重新分配
	for k := range log.Fields {
		delete(log.Fields, k)
	}
	return log
}

// Put 将对象返回池中
func (p *ParsedLogPool) Put(log *parser.ParsedLog) {
	p.pool.Put(log)
}

// ByteSlicePool 字节切片池
type ByteSlicePool struct {
	pool sync.Pool
}

// NewByteSlicePool 创建字节切片池
func NewByteSlicePool() *ByteSlicePool {
	return &ByteSlicePool{
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 0, 4096)
				return &buf
			},
		},
	}
}

// Get 获取字节切片
func (p *ByteSlicePool) Get() []byte {
	buf := p.pool.Get().(*[]byte)
	return (*buf)[:0] // 重用底层数组
}

// Put 返回字节切片
func (p *ByteSlicePool) Put(buf []byte) {
	// 只在容量合理时回收
	if cap(buf) <= 65536 {
		p.pool.Put(&buf)
	}
}

// WorkerPool 工作池
type WorkerPool struct {
	workers   int
	queue     chan func()
	wg        sync.WaitGroup
	shutdown  chan struct{}
	closeOnce sync.Once
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		workers:  workers,
		queue:    make(chan func(), queueSize),
		shutdown: make(chan struct{}),
	}

	// 启动工作协程
	for i := 0; i < workers; i++ {
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	for {
		select {
		case task, ok := <-wp.queue:
			if !ok {
				return
			}
			task()
			wp.wg.Done()
		case <-wp.shutdown:
			return
		}
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task func()) {
	wp.wg.Add(1)
	select {
	case wp.queue <- task:
		// 成功提交
	case <-wp.shutdown:
		wp.wg.Done()
	default:
		// 队列满，同步执行
		task()
		wp.wg.Done()
	}
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// Close 关闭工作池
func (wp *WorkerPool) Close() {
	wp.closeOnce.Do(func() {
		close(wp.shutdown)
	})
}

// BatchProcessor 批处理器
type BatchProcessor struct {
	workers    int
	batchSize  int
	queue      chan []byte
	results    chan interface{}
	shutdown   chan struct{}
	closeOnce  sync.Once
}

// NewBatchProcessor 创建批处理器
func NewBatchProcessor(workers, batchSize int) *BatchProcessor {
	bp := &BatchProcessor{
		workers:   workers,
		batchSize: batchSize,
		queue:     make(chan []byte, batchSize*10),
		results:   make(chan interface{}, batchSize),
		shutdown:  make(chan struct{}),
	}

	// 启动处理协程
	for i := 0; i < workers; i++ {
		go bp.processor()
	}

	return bp
}

func (bp *BatchProcessor) processor() {
	batch := make([][]byte, 0, bp.batchSize)

	for {
		select {
		case data, ok := <-bp.queue:
			if !ok {
				return
			}
			batch = append(batch, data)

			// 批次满了就处理
			if len(batch) >= bp.batchSize {
				bp.processBatch(batch)
				batch = batch[:0]
			}

		case <-bp.shutdown:
			// 处理剩余数据
			if len(batch) > 0 {
				bp.processBatch(batch)
			}
			return
		}
	}
}

func (bp *BatchProcessor) processBatch(batch [][]byte) {
	// 这里应该调用实际的处理器
	// 为简化实现，直接返回
	for range batch {
		select {
		case bp.results <- nil:
		case <-bp.shutdown:
			return
		}
	}
}

// Submit 提交数据
func (bp *BatchProcessor) Submit(data []byte) {
	select {
	case bp.queue <- data:
	case <-bp.shutdown:
	}
}

// Results 获取结果通道
func (bp *BatchProcessor) Results() <-chan interface{} {
	return bp.results
}

// Close 关闭处理器
func (bp *BatchProcessor) Close() {
	bp.closeOnce.Do(func() {
		close(bp.shutdown)
		close(bp.queue)
	})
}
