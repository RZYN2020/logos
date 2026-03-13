// Package rule provides a unified rule engine that replaces the strategy package.
// It uses the shared pkg/rule package for condition matching and action execution.
package rule

import (
	"fmt"
	"sync"
	"time"

	"github.com/log-system/logos/pkg/rule"
	rulestorage "github.com/log-system/logos/pkg/rule/storage"
)

// Engine wraps the unified rule engine for use in the SDK
type Engine struct {
	engine     *rule.RuleEngine
	storage    *rulestorage.ETCDStorage
	clientID   string
	mu         sync.RWMutex
	closed     bool
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

// NewEngine creates a new rule engine
func NewEngine(cfg Config) (*Engine, error) {
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = DefaultConfig().DialTimeout
	}

	// Create rule engine with audit and stats enabled
	ruleEngine := rule.NewRuleEngine(rule.RuleEngineConfig{
		EnableAudit: true,
		EnableStats: true,
	})

	// Create client ID as {service_name}.{environment}
	clientID := fmt.Sprintf("%s.%s", cfg.ServiceName, cfg.Environment)

	engine := &Engine{
		engine:   ruleEngine,
		clientID: clientID,
	}

	// If etcd endpoints are provided, set up storage and load rules
	if len(cfg.EtcdEndpoints) > 0 {
		storage, err := rulestorage.NewETCDStorage(rulestorage.ETCDStorageConfig{
			Endpoints:       cfg.EtcdEndpoints,
			Namespace:       "/rules/clients/" + clientID + "/sdk",
			DialTimeout:     cfg.DialTimeout,
			RefreshDuration: 30 * time.Second,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create etcd storage: %w", err)
		}

		engine.storage = storage

		// Load initial rules
		if err := ruleEngine.LoadRules(storage); err != nil {
			storage.Close()
			return nil, fmt.Errorf("failed to load rules: %w", err)
		}
	}

	return engine, nil
}

// Evaluate evaluates a log entry against rules
// Returns true if the log should be logged, false if it should be dropped
func (e *Engine) Evaluate(level, service, environment string, fields map[string]interface{}) Decision {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.closed {
		return Decision{ShouldLog: true, Sampling: 1.0}
	}

	// Build log entry from the provided fields
	entryData := make(map[string]interface{})
	entryData["level"] = level
	entryData["service"] = service
	entryData["environment"] = environment
	for k, v := range fields {
		entryData[k] = v
	}

	entry := rule.NewMapLogEntry(entryData)

	// Evaluate against rules
	shouldKeep, _, _ := e.engine.Evaluate(entry)

	// Convert to old Decision format for backwards compatibility
	decision := Decision{
		ShouldLog: shouldKeep,
		Sampling:  1.0,
		Priority:  "normal",
		Transform: "none",
	}

	// Extract sampling rate from marks if set
	if marks, ok := entry.GetField("_marks"); ok {
		if m, ok := marks.(map[string]interface{}); ok {
			if rate, ok := m["sampling_rate"].(float64); ok {
				decision.Sampling = rate
			}
		}
	}

	return decision
}

// Decision represents the evaluation result
type Decision struct {
	ShouldLog bool
	Sampling  float64
	Priority  string
	Transform string
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
func (e *Engine) GetStats() rule.EngineStats {
	return e.engine.GetStats()
}

// AddRule adds a rule dynamically
func (e *Engine) AddRule(r *rule.Rule) {
	e.engine.AddRule(r)
}

// RemoveRule removes a rule
func (e *Engine) RemoveRule(ruleID string) {
	e.engine.RemoveRule(ruleID)
}
