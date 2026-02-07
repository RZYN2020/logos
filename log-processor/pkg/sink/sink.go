// Package sink 提供日志输出功能
package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
)

// Sink 日志输出接口
type Sink interface {
	Write(ctx context.Context, logs []LogEntry) error
	Close() error
}

// LogEntry 输出日志条目
type LogEntry struct {
	Index     string
	ID        string
	Document  map[string]interface{}
	Timestamp time.Time
}

// ElasticsearchSink ES 输出
type ElasticsearchSink struct {
	client   *elastic.Client
	bulkSize int
}

// NewElasticsearchSink 创建 ES Sink
func NewElasticsearchSink(addresses []string, bulkSize int) (*ElasticsearchSink, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(addresses...),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ES client: %w", err)
	}

	return &ElasticsearchSink{
		client:   client,
		bulkSize: bulkSize,
	}, nil
}

// Write 批量写入 ES
func (s *ElasticsearchSink) Write(ctx context.Context, logs []LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	bulk := s.client.Bulk()

	for _, log := range logs {
		doc := elastic.NewBulkIndexRequest().
			Index(log.Index).
			Id(log.ID).
			Doc(log.Document)

		bulk.Add(doc)
	}

	_, err := bulk.Do(ctx)
	if err != nil {
		return fmt.Errorf("bulk insert failed: %w", err)
	}

	return nil
}

// Close 关闭连接
func (s *ElasticsearchSink) Close() error {
	// ES client 不需要显式关闭
	return nil
}

// ConsoleSink 控制台输出（用于调试）
type ConsoleSink struct {
	encoder *json.Encoder
}

// NewConsoleSink 创建控制台 Sink
func NewConsoleSink() *ConsoleSink {
	return &ConsoleSink{
		encoder: json.NewEncoder(bytes.NewBuffer(nil)),
	}
}

// Write 输出到控制台
func (s *ConsoleSink) Write(ctx context.Context, logs []LogEntry) error {
	for _, log := range logs {
		data, err := json.Marshal(log.Document)
		if err != nil {
			continue
		}
		fmt.Printf("[%s] %s\n", log.Index, string(data))
	}
	return nil
}

// Close 关闭
func (s *ConsoleSink) Close() error {
	return nil
}

// MultiSink 多目标输出
type MultiSink struct {
	sinks []Sink
}

// NewMultiSink 创建多目标 Sink
func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

// Write 写入所有目标
func (s *MultiSink) Write(ctx context.Context, logs []LogEntry) error {
	var lastErr error

	for _, sink := range s.sinks {
		if err := sink.Write(ctx, logs); err != nil {
			lastErr = err
			// 继续写入其他 sink，不中断
		}
	}

	return lastErr
}

// Close 关闭所有目标
func (s *MultiSink) Close() error {
	var lastErr error

	for _, sink := range s.sinks {
		if err := sink.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// WebhookSink Webhook 输出（用于告警）
type WebhookSink struct {
	url     string
	client  *http.Client
	headers map[string]string
}

// NewWebhookSink 创建 Webhook Sink
func NewWebhookSink(url string, headers map[string]string) *WebhookSink {
	return &WebhookSink{
		url:     url,
		client:  &http.Client{Timeout: 10 * time.Second},
		headers: headers,
	}
}

// Write 发送到 Webhook
func (s *WebhookSink) Write(ctx context.Context, logs []LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	// 只发送错误级别的日志作为告警
	var alerts []map[string]interface{}
	for _, log := range logs {
		if level, ok := log.Document["level"]; ok {
			if level == "ERROR" || level == "FATAL" {
				alerts = append(alerts, log.Document)
			}
		}
	}

	if len(alerts) == 0 {
		return nil
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("failed to marshal alerts: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

// Close 关闭
func (s *WebhookSink) Close() error {
	return nil
}
