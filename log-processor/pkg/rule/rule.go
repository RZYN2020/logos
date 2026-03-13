// Package rule provides a unified rule engine for the log processor.
// It uses the shared pkg/rule package for condition matching and action execution.
package rule

import (
	"fmt"
	"sync"
	"time"

	unifiedRule "github.com/log-system/logos/pkg/rule"
	"github.com/log-system/logos/pkg/rule/storage"
)

// Engine wraps the unified rule engine for use in the log processor
type Engine struct {
	engine     *unifiedRule.RuleEngine
	storage    *storage.ETCDStorage
	clientID   string
	mu         sync.RWMutex
	closed     bool
	serviceName string
	environment string
}

// Config holds configuration for the rule engine
type Config struct {
	// ServiceName is the name of the service
	ServiceName string
	// Environment is the deployment environment (e.g., prod, staging)
	Environment string
	// EtcdEndpoints are the etcd server endpoints
	EtcdEndpoints []string
	// DialTimeout is the timeout for etcd connections
	DialTimeout time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		DialTimeout: 5 * time.Second,
	}
}

// NewEngine creates a new rule engine for the log processor
func NewEngine(cfg Config) (*Engine, error) {
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = DefaultConfig().DialTimeout
	}

	// Create rule engine with audit and stats enabled
	ruleEngine := unifiedRule.NewRuleEngine(unifiedRule.RuleEngineConfig{
		EnableAudit: true,
		EnableStats: true,
	})

	// Create client ID as {service_name}.{environment}
	clientID := fmt.Sprintf("%s.%s", cfg.ServiceName, cfg.Environment)

	engine := &Engine{
		engine:      ruleEngine,
		clientID:    clientID,
		serviceName: cfg.ServiceName,
		environment: cfg.Environment,
	}

	// If etcd endpoints are provided, set up storage and load rules
	if len(cfg.EtcdEndpoints) > 0 {
		// Load processor-specific rules
		processorNamespace := "/rules/clients/" + clientID + "/processor"

		storage, err := storage.NewETCDStorage(storage.ETCDStorageConfig{
			Endpoints:       cfg.EtcdEndpoints,
			Namespace:       processorNamespace,
			DialTimeout:     cfg.DialTimeout,
			RefreshDuration: 30 * time.Second,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create etcd storage: %w", err)
		}

		engine.storage = storage

		// Load initial rules from processor namespace
		if err := ruleEngine.LoadRules(storage); err != nil {
			storage.Close()
			return nil, fmt.Errorf("failed to load processor rules: %w", err)
		}
	}

	return engine, nil
}

// Evaluate evaluates a parsed log entry against rules
// Returns whether the log should be kept and any results/errors
func (e *Engine) Evaluate(entry unifiedRule.LogEntry) (shouldKeep bool, results []unifiedRule.RuleResult, errors []error) {
	return e.engine.Evaluate(entry)
}

// Close closes the engine and releases resources
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil
	}

	e.closed = true

	if e.storage != nil {
		return e.storage.Close()
	}

	return nil
}

// GetStats returns engine statistics
func (e *Engine) GetStats() unifiedRule.EngineStats {
	return e.engine.GetStats()
}

// AddRule adds a rule dynamically
func (e *Engine) AddRule(r *unifiedRule.Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine.AddRule(r)
}

// RemoveRule removes a rule
func (e *Engine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine.RemoveRule(ruleID)
}
