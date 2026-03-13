package rule

import (
	"sort"
	"sync"
	"time"
)

// RuleEngine evaluates rules against log entries.
type RuleEngine struct {
	mu             sync.RWMutex
	rules          map[string]*Rule
	ruleOrder      []string // Sorted order of rule IDs
	evaluator      *ConditionEvaluator
	executor       *ActionExecutor
	auditLogger    AuditLogger
	errorHandler   ErrorHandler
	config         RuleEngineConfig
	stats          EngineStats
	statsMu        sync.Mutex
}

// RuleEngineConfig holds configuration for the rule engine.
type RuleEngineConfig struct {
	// EnableAudit enables audit logging
	EnableAudit bool
	// EnableStats enables statistics collection
	EnableStats bool
}

// EngineStats holds statistics about rule engine execution.
type EngineStats struct {
	TotalEvaluations    int64
	MatchedEvaluations  int64
	TotalActionsExecuted int64
	FailedActions       int64
}

// AuditLogger defines the interface for audit logging.
type AuditLogger interface {
	LogMatch(rule *Rule, entry LogEntry, result *RuleResult)
	LogError(rule *Rule, entry LogEntry, err error)
}

// ErrorHandler defines the interface for error handling.
type ErrorHandler interface {
	HandleError(err error, rule *Rule, entry LogEntry)
}

// DefaultAuditLogger is a default audit logger that logs to stdout.
type DefaultAuditLogger struct{}

func (l *DefaultAuditLogger) LogMatch(rule *Rule, entry LogEntry, result *RuleResult) {
	// In a real implementation, this would log to a structured audit log
	_ = rule
	_ = entry
	_ = result
}

func (l *DefaultAuditLogger) LogError(rule *Rule, entry LogEntry, err error) {
	// In a real implementation, this would log the error
	_ = rule
	_ = entry
	_ = err
}

// DefaultErrorHandler is a default error handler.
type DefaultErrorHandler struct{}

func (h *DefaultErrorHandler) HandleError(err error, rule *Rule, entry LogEntry) {
	// Best-effort: just record the error, don't fail the evaluation
	_ = err
	_ = rule
	_ = entry
}

// NewRuleEngine creates a new rule engine.
func NewRuleEngine(config RuleEngineConfig) *RuleEngine {
	engine := &RuleEngine{
		rules:        make(map[string]*Rule),
		ruleOrder:    make([]string, 0),
		evaluator:    NewConditionEvaluator(),
		executor:     NewActionExecutor(),
		config:       config,
		stats:        EngineStats{},
	}

	// Set up default audit logger and error handler
	if config.EnableAudit {
		engine.auditLogger = &DefaultAuditLogger{}
	}
	engine.errorHandler = &DefaultErrorHandler{}

	return engine
}

// SetAuditLogger sets the audit logger.
func (e *RuleEngine) SetAuditLogger(logger AuditLogger) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.auditLogger = logger
}

// SetErrorHandler sets the error handler.
func (e *RuleEngine) SetErrorHandler(handler ErrorHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errorHandler = handler
}

// LoadRules loads rules from a RuleStorage.
func (e *RuleEngine) LoadRules(storage RuleStorage) error {
	rules, err := storage.LoadRules()
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear existing rules
	e.rules = make(map[string]*Rule)
	e.ruleOrder = make([]string, 0)

	// Add new rules
	for _, rule := range rules {
		e.rules[rule.ID] = rule
		e.ruleOrder = append(e.ruleOrder, rule.ID)
	}

	// Sort rules by key (dictionary order)
	sort.Strings(e.ruleOrder)

	return nil
}

// SetRules sets the rules directly (for use without storage).
func (e *RuleEngine) SetRules(rules []*Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear existing rules
	e.rules = make(map[string]*Rule)
	e.ruleOrder = make([]string, 0)

	// Add new rules
	for _, rule := range rules {
		e.rules[rule.ID] = rule
		e.ruleOrder = append(e.ruleOrder, rule.ID)
	}

	// Sort rules by key (dictionary order)
	sort.Strings(e.ruleOrder)
}

// AddRule adds a single rule.
func (e *RuleEngine) AddRule(rule *Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules[rule.ID] = rule

	// Add to order if not already present
	found := false
	for _, id := range e.ruleOrder {
		if id == rule.ID {
			found = true
			break
		}
	}
	if !found {
		e.ruleOrder = append(e.ruleOrder, rule.ID)
		sort.Strings(e.ruleOrder)
	}
}

// RemoveRule removes a rule.
func (e *RuleEngine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.rules, ruleID)

	// Remove from order
	newOrder := make([]string, 0, len(e.ruleOrder))
	for _, id := range e.ruleOrder {
		if id != ruleID {
			newOrder = append(newOrder, id)
		}
	}
	e.ruleOrder = newOrder
}

// Evaluate evaluates a log entry against all rules.
// Returns whether the log should be kept, any results, and any errors.
func (e *RuleEngine) Evaluate(entry LogEntry) (shouldKeep bool, results []RuleResult, errors []error) {
	e.mu.RLock()
	rules := make([]*Rule, len(e.ruleOrder))
	for i, id := range e.ruleOrder {
		rules[i] = e.rules[id]
	}
	evaluator := e.evaluator
	executor := e.executor
	auditLogger := e.auditLogger
	errorHandler := e.errorHandler
	enableStats := e.config.EnableStats
	e.mu.RUnlock()

	shouldKeep = true
	results = make([]RuleResult, 0)
	errors = make([]error, 0)

	// Evaluate each rule in order
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Evaluate condition
		matched, err := evaluator.Evaluate(rule.Condition, entry)
		if err != nil {
			errors = append(errors, err)
			if errorHandler != nil {
				errorHandler.HandleError(err, rule, entry)
			}
			continue
		}

		if !matched {
			continue
		}

		// Rule matched
		result := RuleResult{
			Matched:     true,
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			ShouldKeep:  true,
			EvaluatedAt: time.Now(),
		}

		// Execute actions
		if len(rule.Actions) > 0 {
			keep, terminate, actionResults, actionErrors := executor.ExecuteActions(entry, rule.Actions)
			result.Actions = actionResults
			result.Errors = actionErrors
			result.ShouldKeep = keep

			errors = append(errors, actionErrors...)

			// Audit log the match
			if auditLogger != nil {
				auditLogger.LogMatch(rule, entry, &result)
			}

			// Update stats
			if enableStats {
				e.updateStats(true, len(rule.Actions), len(actionErrors))
			}

			// Check for terminating actions
			if terminate {
				result.Terminate = true
				results = append(results, result)
				return shouldKeep == keep, results, errors
			}

			shouldKeep = keep
		} else {
			// No actions, just a match - keep by default
			if auditLogger != nil {
				auditLogger.LogMatch(rule, entry, &result)
			}
			if enableStats {
				e.updateStats(true, 0, 0)
			}
		}

		results = append(results, result)
	}

	if enableStats {
		e.updateStats(false, 0, 0)
	}

	return shouldKeep, results, errors
}

// EvaluateSingle evaluates a log entry against a single rule.
func (e *RuleEngine) EvaluateSingle(ruleID string, entry LogEntry) (*RuleResult, error) {
	e.mu.RLock()
	rule, ok := e.rules[ruleID]
	evaluator := e.evaluator
	executor := e.executor
	e.mu.RUnlock()

	if !ok {
		return nil, &RuleError{RuleID: ruleID, Err: ErrRuleNotFound}
	}

	if !rule.Enabled {
		return &RuleResult{
			Matched:     false,
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			EvaluatedAt: time.Now(),
		}, nil
	}

	// Evaluate condition
	matched, err := evaluator.Evaluate(rule.Condition, entry)
	if err != nil {
		return nil, err
	}

	result := &RuleResult{
		Matched:     matched,
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		ShouldKeep:  true,
		EvaluatedAt: time.Now(),
	}

	if matched && len(rule.Actions) > 0 {
		keep, _, actionResults, actionErrors := executor.ExecuteActions(entry, rule.Actions)
		result.Actions = actionResults
		result.Errors = actionErrors
		result.ShouldKeep = keep
		result.Terminate = false // Single rule doesn't terminate
	}

	return result, nil
}

// GetStats returns the engine statistics.
func (e *RuleEngine) GetStats() EngineStats {
	e.statsMu.Lock()
	defer e.statsMu.Unlock()
	return e.stats
}

func (e *RuleEngine) updateStats(matched bool, actionsExecuted int, failedActions int) {
	e.statsMu.Lock()
	defer e.statsMu.Unlock()

	e.stats.TotalEvaluations++
	if matched {
		e.stats.MatchedEvaluations++
	}
	e.stats.TotalActionsExecuted += int64(actionsExecuted)
	e.stats.FailedActions += int64(failedActions)
}

// RuleError represents an error related to a specific rule.
type RuleError struct {
	RuleID string
	Err    error
}

func (e *RuleError) Error() string {
	if e.RuleID != "" {
		return "rule " + e.RuleID + ": " + e.Err.Error()
	}
	return e.Err.Error()
}

func (e *RuleError) Unwrap() error {
	return e.Err
}

// Common errors
var (
	ErrRuleNotFound = &RuleError{Err: &ruleNotFoundError{}}
)

type ruleNotFoundError struct{}

func (e *ruleNotFoundError) Error() string {
	return "rule not found"
}
