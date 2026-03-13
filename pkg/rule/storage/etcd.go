package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/log-system/logos/pkg/rule"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// ETCDStorage is an ETCD-based rule storage with watch support.
type ETCDStorage struct {
	client       *clientv3.Client
	ctx          context.Context
	namespace    string
	mu           sync.RWMutex
	watchChan    clientv3.WatchChan
	watchCtx     context.CancelFunc
	rules        map[string]*rule.Rule
	refreshDur   time.Duration
	refreshDone  chan struct{}
}

// ETCDStorageConfig holds configuration for ETCD storage.
type ETCDStorageConfig struct {
	// Endpoints is the list of ETCD endpoints.
	Endpoints []string
	// Namespace is the prefix for all rule keys.
	Namespace string
	// DialTimeout is the timeout for dialing ETCD.
	DialTimeout time.Duration
	// RefreshDuration is the interval for periodic refresh.
	// If 0, no periodic refresh is done (only watch).
	RefreshDuration time.Duration
}

// NewETCDStorage creates a new ETCD storage.
func NewETCDStorage(config ETCDStorageConfig) (*ETCDStorage, error) {
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.Namespace == "" {
		config.Namespace = "/rules"
	}
	if config.RefreshDuration == 0 {
		config.RefreshDuration = 30 * time.Second
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &ETCDStorage{
		client:       client,
		ctx:          ctx,
		namespace:    config.Namespace,
		rules:        make(map[string]*rule.Rule),
		refreshDur:   config.RefreshDuration,
		refreshDone:  make(chan struct{}),
		watchCtx:     cancel,
	}

	// Load initial rules
	if err := s.loadRules(); err != nil {
		client.Close()
		cancel()
		return nil, err
	}

	// Start watch
	s.startWatch()

	// Start periodic refresh
	go s.periodicRefresh()

	return s, nil
}

// LoadRules loads all rules from ETCD.
func (s *ETCDStorage) LoadRules() ([]*rule.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*rule.Rule, 0, len(s.rules))
	for _, r := range s.rules {
		rules = append(rules, r)
	}
	return rules, nil
}

// loadRules loads rules from ETCD (internal, no lock).
func (s *ETCDStorage) loadRules() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.client.Get(ctx, s.namespace+"/", clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to load rules from etcd: %w", err)
	}

	newRules := make(map[string]*rule.Rule)
	for _, kv := range resp.Kvs {
		var r rule.Rule
		if err := json.Unmarshal(kv.Value, &r); err != nil {
			// Skip invalid rules
			continue
		}
		newRules[r.ID] = &r
	}

	s.rules = newRules
	return nil
}

// startWatch starts watching for rule changes.
func (s *ETCDStorage) startWatch() {
	s.watchChan = s.client.Watch(s.ctx, s.namespace+"/", clientv3.WithPrefix())

	go func() {
		for watchResp := range s.watchChan {
			for _, event := range watchResp.Events {
				switch event.Type {
				case clientv3.EventTypePut:
					var r rule.Rule
					if err := json.Unmarshal(event.Kv.Value, &r); err == nil {
						s.mu.Lock()
						s.rules[r.ID] = &r
						s.mu.Unlock()
					}

				case clientv3.EventTypeDelete:
					// Extract rule ID from key
					key := string(event.Kv.Key)
					ruleID := extractRuleIDFromKey(key, s.namespace)
					if ruleID != "" {
						s.mu.Lock()
						delete(s.rules, ruleID)
						s.mu.Unlock()
					}
				}
			}
		}
	}()
}

// periodicRefresh periodically reloads rules as a backup to watch.
func (s *ETCDStorage) periodicRefresh() {
	ticker := time.NewTicker(s.refreshDur)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			close(s.refreshDone)
			return
		case <-ticker.C:
			s.mu.Lock()
			_ = s.loadRules() // Ignore errors, watch is primary
			s.mu.Unlock()
		}
	}
}

// Close closes the storage.
func (s *ETCDStorage) Close() error {
	s.watchCtx()
	<-s.refreshDone
	return s.client.Close()
}

// GetClientNamespace returns the client-specific namespace.
func (s *ETCDStorage) GetClientNamespace(clientID string) string {
	return s.namespace + "/clients/" + clientID
}

// GetDefaultsNamespace returns the defaults namespace.
func (s *ETCDStorage) GetDefaultsNamespace() string {
	return s.namespace + "/defaults"
}

// PutRule stores a rule in ETCD.
func (s *ETCDStorage) PutRule(r *rule.Rule) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := s.namespace + "/" + r.ID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to put rule to etcd: %w", err)
	}

	return nil
}

// DeleteRule deletes a rule from ETCD.
func (s *ETCDStorage) DeleteRule(ruleID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := s.namespace + "/" + ruleID
	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete rule from etcd: %w", err)
	}

	return nil
}

// extractRuleIDFromKey extracts the rule ID from an ETCD key.
func extractRuleIDFromKey(key, namespace string) string {
	// Key format: /rules/{client}/{sdk|processor}/{ruleID}
	// or: /rules/{ruleID}
	prefix := namespace + "/"
	if len(key) <= len(prefix) {
		return ""
	}

	// Get the part after namespace
	remaining := key[len(prefix):]

	// Find the last slash to get the rule ID
	lastSlash := -1
	for i := len(remaining) - 1; i >= 0; i-- {
		if remaining[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return remaining
	}

	return remaining[lastSlash+1:]
}

// Ensure ETCDStorage implements rule.RuleStorage
var _ rule.RuleStorage = (*ETCDStorage)(nil)
