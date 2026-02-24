package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdConfig holds etcd client configuration
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

// LoadStrategies loads all strategies from etcd
func (c *EtcdClient) LoadStrategies() (map[string]Strategy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	resp, err := c.client.Get(ctx, "/strategies/", clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to load strategies from etcd: %w", err)
	}

	strategies := make(map[string]Strategy)
	for _, kv := range resp.Kvs {
		var strategy Strategy
		if err := json.Unmarshal(kv.Value, &strategy); err != nil {
			// Log error but continue loading other strategies
			fmt.Printf("Failed to unmarshal strategy: %v\n", err)
			continue
		}
		strategies[strategy.ID] = strategy
	}

	return strategies, nil
}

// LoadStrategy loads a single strategy by ID
func (c *EtcdClient) LoadStrategy(id string) (*Strategy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	key := fmt.Sprintf("/strategies/%s", id)
	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to load strategy: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("strategy not found: %s", id)
	}

	var strategy Strategy
	if err := json.Unmarshal(resp.Kvs[0].Value, &strategy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal strategy: %w", err)
	}

	return &strategy, nil
}

// WatchStrategies watches for strategy changes
// Returns a channel that receives strategy updates and an error channel
func (c *EtcdClient) WatchStrategies() (<-chan StrategyEvent, <-chan error) {
	eventCh := make(chan StrategyEvent, 10)
	errCh := make(chan error, 1)

	go c.watch(eventCh, errCh)

	return eventCh, errCh
}

// StrategyEvent represents a strategy change event
type StrategyEvent struct {
	Type     EventType
	Strategy Strategy
}

// EventType indicates the type of change
type EventType int

const (
	// EventPut indicates a strategy was created or updated
	EventPut EventType = iota
	// EventDelete indicates a strategy was deleted
	EventDelete
)

// String returns string representation of event type
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

func (c *EtcdClient) watch(eventCh chan<- StrategyEvent, errCh chan<- error) {
	defer close(eventCh)
	defer close(errCh)

	watchChan := c.client.Watch(context.Background(), "/strategies/", clientv3.WithPrefix())

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
				var strategy Strategy
				if err := json.Unmarshal(event.Kv.Value, &strategy); err != nil {
					fmt.Printf("Failed to unmarshal strategy update: %v\n", err)
					continue
				}
				select {
				case eventCh <- StrategyEvent{Type: EventPut, Strategy: strategy}:
				case <-time.After(time.Second):
					fmt.Println("Event channel full, dropping event")
				}

			case clientv3.EventTypeDelete:
				key := string(event.Kv.Key)
				id := extractIDFromKey(key)
				select {
				case eventCh <- StrategyEvent{
					Type:     EventDelete,
					Strategy: Strategy{ID: id},
				}:
				case <-time.After(time.Second):
					fmt.Println("Event channel full, dropping event")
				}
			}
		}
	}
}

// Close closes the etcd client
func (c *EtcdClient) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return c.client.Close()
}

// IsConnected checks if the client is connected to etcd
func (c *EtcdClient) IsConnected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	_, err := c.client.Get(ctx, "/health")
	return err == nil
}

// PutStrategy saves a strategy to etcd
func (c *EtcdClient) PutStrategy(strategy Strategy) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	data, err := json.Marshal(strategy)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy: %w", err)
	}

	key := fmt.Sprintf("/strategies/%s", strategy.ID)
	_, err = c.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save strategy: %w", err)
	}

	return nil
}

// DeleteStrategy deletes a strategy from etcd
func (c *EtcdClient) DeleteStrategy(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	key := fmt.Sprintf("/strategies/%s", id)
	_, err := c.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete strategy: %w", err)
	}

	return nil
}

