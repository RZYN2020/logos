package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer 异步生产者
type Producer struct {
	writer        *kafka.Writer
	buffer        chan LogMessage
	batchSize     int
	flushInterval time.Duration
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// LogMessage 日志消息
type LogMessage struct {
	Topic   string
	Key     string
	Value   []byte
	Headers map[string]string
}

// NewProducer 创建异步生产者
func NewProducer(brokers []string, batchSize int, flushInterval time.Duration) *Producer {
	ctx, cancel := context.WithCancel(context.Background())

	p := &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			BatchSize:    batchSize,
			BatchTimeout: flushInterval,
			Async:        true,
			RequiredAcks: kafka.RequireOne,
		},
		buffer:        make(chan LogMessage, batchSize*10),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		ctx:           ctx,
		cancel:        cancel,
	}

	// 启动后台刷新协程
	p.wg.Add(1)
	go p.flushLoop()

	return p
}

// Send 异步发送日志
func (p *Producer) Send(msg LogMessage) error {
	select {
	case p.buffer <- msg:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("producer closed")
	default:
		// 缓冲区满，直接丢弃（背压）
		return fmt.Errorf("buffer full, message dropped")
	}
}

// flushLoop 后台刷新循环
func (p *Producer) flushLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	batch := make([]kafka.Message, 0, p.batchSize)

	for {
		select {
		case <-p.ctx.Done():
			// 刷新剩余消息
			p.flush(batch)
			return

		case msg := <-p.buffer:
			kafkaMsg := kafka.Message{
				Topic: msg.Topic,
				Key:   []byte(msg.Key),
				Value: msg.Value,
			}

			// 添加headers
			for k, v := range msg.Headers {
				kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
					Key:   k,
					Value: []byte(v),
				})
			}

			batch = append(batch, kafkaMsg)

			if len(batch) >= p.batchSize {
				p.flush(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				p.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

// flush 批量发送
func (p *Producer) flush(batch []kafka.Message) {
	if len(batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := p.writer.WriteMessages(ctx, batch...); err != nil {
		// 记录错误，但不中断处理
		fmt.Printf("Failed to write messages: %v\n", err)
	}
}

// Close 关闭生产者
func (p *Producer) Close() error {
	p.cancel()
	p.wg.Wait()
	return p.writer.Close()
}
