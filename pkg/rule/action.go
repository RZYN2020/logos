package rule

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ActionHandler defines the interface for executing actions.
type ActionHandler interface {
	// Execute executes the action on the log entry.
	// Returns true if the log should be kept, false otherwise.
	// Returns metadata for audit/monitoring purposes.
	Execute(entry LogEntry, config map[string]interface{}) (keep bool, metadata map[string]interface{}, err error)

	// Name returns the action type name.
	Name() string
}

// ActionExecutor executes actions on log entries.
type ActionExecutor struct {
	handlers map[string]ActionHandler
	mu       sync.RWMutex
	rand     *rand.Rand
}

// NewActionExecutor creates a new action executor.
func NewActionExecutor() *ActionExecutor {
	e := &ActionExecutor{
		handlers: make(map[string]ActionHandler),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	e.registerBuiltinActions()
	return e
}

// RegisterAction registers a custom action handler.
func (e *ActionExecutor) RegisterAction(handler ActionHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[handler.Name()] = handler
}

// Execute executes an action on a log entry.
func (e *ActionExecutor) Execute(entry LogEntry, action ActionDef) (keep bool, metadata map[string]interface{}, err error) {
	e.mu.RLock()
	handler, ok := e.handlers[action.Type]
	e.mu.RUnlock()

	if !ok {
		return false, nil, fmt.Errorf("unknown action type: %s", action.Type)
	}

	return handler.Execute(entry, action.Config)
}

// registerBuiltinActions registers all built-in actions.
func (e *ActionExecutor) registerBuiltinActions() {
	// Flow control actions
	e.RegisterAction(&keepAction{})
	e.RegisterAction(&dropAction{})
	e.RegisterAction(&sampleAction{})

	// Transformation actions
	e.RegisterAction(&maskAction{})
	e.RegisterAction(&truncateAction{})
	e.RegisterAction(&extractAction{})
	e.RegisterAction(&renameAction{})
	e.RegisterAction(&removeAction{})
	e.RegisterAction(&setAction{})

	// Metadata actions
	e.RegisterAction(&markAction{})
}

// keepAction keeps the log entry and terminates rule evaluation.
type keepAction struct{}

func (a *keepAction) Name() string { return ActionKeep }

func (a *keepAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	return true, map[string]interface{}{"action": "keep"}, nil
}

// dropAction drops the log entry and terminates rule evaluation.
type dropAction struct{}

func (a *dropAction) Name() string { return ActionDrop }

func (a *dropAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	return false, map[string]interface{}{"action": "drop"}, nil
}

// sampleAction samples logs at a given rate.
type sampleAction struct{}

func (a *sampleAction) Name() string { return ActionSample }

func (a *sampleAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	rate := GetFloat64Config(config, "rate", 1.0)
	keep := GetRand().Float64() < rate
	return keep, map[string]interface{}{"action": "sample", "rate": rate, "kept": keep}, nil
}

// maskAction masks sensitive data in a field.
type maskAction struct{}

func (a *maskAction) Name() string { return ActionMask }

func (a *maskAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	field := GetStringConfig(config, "field", "")
	pattern := GetStringConfig(config, "pattern", "")
	character := GetStringConfig(config, "character", "*")

	if field == "" {
		return true, nil, fmt.Errorf("mask action requires 'field' config")
	}

	value, exists := entry.GetField(field)
	if !exists {
		return true, map[string]interface{}{"skipped": true, "reason": "field not found"}, nil
	}

	valueStr, ok := value.(string)
	if !ok {
		return true, nil, fmt.Errorf("field %s is not a string", field)
	}

	var masked string
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return true, nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		masked = re.ReplaceAllStringFunc(valueStr, func(s string) string {
			return strings.Repeat(character, len(s))
		})
	} else {
		masked = strings.Repeat(character, len(valueStr))
	}

	if err := entry.SetField(field, masked); err != nil {
		return true, nil, fmt.Errorf("failed to set masked value: %w", err)
	}

	return true, map[string]interface{}{"action": "mask", "field": field}, nil
}

// truncateAction truncates field values to a maximum length.
type truncateAction struct{}

func (a *truncateAction) Name() string { return ActionTruncate }

func (a *truncateAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	field := GetStringConfig(config, "field", "")
	maxLength := GetIntConfig(config, "max_length", 1000)
	suffix := GetStringConfig(config, "suffix", "...")

	if field == "" {
		return true, nil, fmt.Errorf("truncate action requires 'field' config")
	}

	value, exists := entry.GetField(field)
	if !exists {
		return true, map[string]interface{}{"skipped": true, "reason": "field not found"}, nil
	}

	valueStr, ok := value.(string)
	if !ok {
		return true, nil, fmt.Errorf("field %s is not a string", field)
	}

	if len(valueStr) <= maxLength {
		return true, map[string]interface{}{"skipped": true, "reason": "value within limit"}, nil
	}

	truncated := valueStr[:maxLength] + suffix
	if err := entry.SetField(field, truncated); err != nil {
		return true, nil, fmt.Errorf("failed to set truncated value: %w", err)
	}

	return true, map[string]interface{}{"action": "truncate", "field": field, "original": len(valueStr), "truncated": len(truncated)}, nil
}

// extractAction extracts a substring using regex and stores it in a new field.
type extractAction struct{}

func (a *extractAction) Name() string { return ActionExtract }

func (a *extractAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	sourceField := GetStringConfig(config, "source_field", "")
	targetField := GetStringConfig(config, "target_field", "")
	pattern := GetStringConfig(config, "pattern", "")
	group := GetIntConfig(config, "group", 1)

	if sourceField == "" || targetField == "" || pattern == "" {
		return true, nil, fmt.Errorf("extract action requires 'source_field', 'target_field', and 'pattern' config")
	}

	value, exists := entry.GetField(sourceField)
	if !exists {
		return true, map[string]interface{}{"skipped": true, "reason": "source field not found"}, nil
	}

	valueStr, ok := value.(string)
	if !ok {
		return true, nil, fmt.Errorf("source field %s is not a string", sourceField)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return true, nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matches := re.FindStringSubmatch(valueStr)
	if len(matches) <= group {
		return true, map[string]interface{}{"skipped": true, "reason": "no match found"}, nil
	}

	if err := entry.SetField(targetField, matches[group]); err != nil {
		return true, nil, fmt.Errorf("failed to set extracted value: %w", err)
	}

	return true, map[string]interface{}{"action": "extract", "source_field": sourceField, "target_field": targetField, "extracted": matches[group]}, nil
}

// renameAction renames a field.
type renameAction struct{}

func (a *renameAction) Name() string { return ActionRename }

func (a *renameAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	fromField := GetStringConfig(config, "from", "")
	toField := GetStringConfig(config, "to", "")

	if fromField == "" || toField == "" {
		return true, nil, fmt.Errorf("rename action requires 'from' and 'to' config")
	}

	value, exists := entry.GetField(fromField)
	if !exists {
		return true, map[string]interface{}{"skipped": true, "reason": "source field not found"}, nil
	}

	if err := entry.SetField(toField, value); err != nil {
		return true, nil, fmt.Errorf("failed to set renamed field: %w", err)
	}

	if err := entry.DeleteField(fromField); err != nil {
		return true, nil, fmt.Errorf("failed to delete old field: %w", err)
	}

	return true, map[string]interface{}{"action": "rename", "from": fromField, "to": toField, "renamed": true}, nil
}

// removeAction removes one or more fields.
type removeAction struct{}

func (a *removeAction) Name() string { return ActionRemove }

func (a *removeAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	field := GetStringConfig(config, "field", "")
	fields := GetStringSliceConfig(config, "fields", nil)

	if field == "" && len(fields) == 0 {
		return true, nil, fmt.Errorf("remove action requires 'field' or 'fields' config")
	}

	removedFields := []string{}

	if field != "" {
		_, exists := entry.GetField(field)
		if exists {
			if err := entry.DeleteField(field); err != nil {
				return true, nil, fmt.Errorf("failed to remove field %s: %w", field, err)
			}
			removedFields = append(removedFields, field)
		}
	}

	for _, f := range fields {
		_, exists := entry.GetField(f)
		if exists {
			if err := entry.DeleteField(f); err != nil {
				return true, nil, fmt.Errorf("failed to remove field %s: %w", f, err)
			}
			removedFields = append(removedFields, f)
		}
	}

	return true, map[string]interface{}{"action": "remove", "removed": removedFields}, nil
}

// setAction sets a field to a specific value.
type setAction struct{}

func (a *setAction) Name() string { return ActionSet }

func (a *setAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	field := GetStringConfig(config, "field", "")
	value := config["value"]

	if field == "" {
		return true, nil, fmt.Errorf("set action requires 'field' config")
	}

	if err := entry.SetField(field, value); err != nil {
		return true, nil, fmt.Errorf("failed to set field %s: %w", field, err)
	}

	return true, map[string]interface{}{"action": "set", "field": field, "value": value}, nil
}

// markAction adds metadata marks to log entries.
type markAction struct{}

func (a *markAction) Name() string { return ActionMark }

func (a *markAction) Execute(entry LogEntry, config map[string]interface{}) (bool, map[string]interface{}, error) {
	field := GetStringConfig(config, "field", "_tags")
	value := config["value"]
	reason := GetStringConfig(config, "reason", "")

	if err := entry.SetField(field, value); err != nil {
		return true, nil, fmt.Errorf("failed to set mark field %s: %w", field, err)
	}

	return true, map[string]interface{}{"action": "mark", "field": field, "value": value, "reason": reason}, nil
}

// ExecuteActions executes a list of actions on a log entry.
// Returns whether the log should be kept, whether to terminate rule evaluation, and any errors.
func (e *ActionExecutor) ExecuteActions(entry LogEntry, actions []ActionDef) (shouldKeep bool, terminate bool, results []ActionResult, errs []error) {
	shouldKeep = true // Default is to keep
	terminate = false
	results = make([]ActionResult, 0, len(actions))

	for _, action := range actions {
		result := ActionResult{
			Type:     action.Type,
			Metadata: make(map[string]interface{}),
		}

		keep, metadata, err := e.Execute(entry, action)

		if err != nil {
			// Best-effort: log error and continue
			result.Success = false
			result.Error = err.Error()
			errs = append(errs, err)
		} else {
			result.Success = true
			result.Metadata = metadata
			shouldKeep = keep

			// Check for terminating actions
			if action.Type == ActionDrop || action.Type == ActionKeep {
				terminate = true
			}
		}

		results = append(results, result)

		// If terminating action, stop execution
		if terminate {
			break
		}
	}

	return
}

// Helper functions for actions - exported for use by actions subpackage

// GetStringConfig safely gets a string config value.
func GetStringConfig(config map[string]interface{}, key string, defaultValue string) string {
	if v, ok := config[key].(string); ok {
		return v
	}
	return defaultValue
}

// GetIntConfig safely gets an int config value.
func GetIntConfig(config map[string]interface{}, key string, defaultValue int) int {
	switch v := config[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// GetFloat64Config safely gets a float64 config value.
func GetFloat64Config(config map[string]interface{}, key string, defaultValue float64) float64 {
	switch v := config[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return defaultValue
	}
}

// GetStringSliceConfig safely gets a string slice config value.
func GetStringSliceConfig(config map[string]interface{}, key string, defaultValue []string) []string {
	if v, ok := config[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if v, ok := config[key].([]string); ok {
		return v
	}
	return defaultValue
}

// rand globals for use by actions
var (
	globalRand *rand.Rand
	randMu     sync.Mutex
)

// SetRand sets the global random source.
func SetRand(r *rand.Rand) {
	randMu.Lock()
	defer randMu.Unlock()
	globalRand = r
}

// GetRand returns the global random source.
func GetRand() *rand.Rand {
	randMu.Lock()
	defer randMu.Unlock()
	if globalRand == nil {
		globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return globalRand
}

// NewRand creates a new random source.
func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}
