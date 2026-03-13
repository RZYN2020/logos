// Package rule provides a unified rule engine for log processing across Log SDK, Log Processor, and Log Analyzer.
// It implements a Condition + Action model with support for nested conditions, multiple operators,
// and extensible actions.
package rule

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Rule represents a single rule with conditions and actions.
type Rule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Enabled     bool        `json:"enabled"`
	Condition   Condition   `json:"condition"`
	Actions     []ActionDef `json:"actions"`
	CreatedAt   time.Time   `json:"created_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
}

// Condition represents a matching condition.
// It can be either a single condition or a composite condition (all/any/not).
type Condition struct {
	// Single condition fields
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`

	// Composite condition fields
	All []Condition `json:"all,omitempty"`
	Any []Condition `json:"any,omitempty"`
	Not *Condition  `json:"not,omitempty"`
}

// IsSingle returns true if this is a single condition (not composite).
func (c Condition) IsSingle() bool {
	return c.Field != "" && c.Operator != ""
}

// IsComposite returns true if this is a composite condition.
func (c Condition) IsComposite() bool {
	return len(c.All) > 0 || len(c.Any) > 0 || c.Not != nil
}

// Validate checks if the condition is valid.
func (c Condition) Validate() error {
	if c.IsSingle() && c.IsComposite() {
		return errors.New("condition cannot be both single and composite")
	}
	if !c.IsSingle() && !c.IsComposite() {
		return errors.New("condition must be either single or composite")
	}

	if c.IsSingle() {
		if c.Field == "" {
			return errors.New("field is required for single condition")
		}
		if c.Operator == "" {
			return errors.New("operator is required for single condition")
		}
		// Validate operator
		validOps := map[string]bool{
			"eq": true, "ne": true, "gt": true, "lt": true, "ge": true, "le": true,
			"contains": true, "starts_with": true, "ends_with": true, "matches": true,
			"in": true, "not_in": true, "exists": true, "not_exists": true,
		}
		if !validOps[c.Operator] {
			return fmt.Errorf("unknown operator: %s", c.Operator)
		}
		// Value is required except for exists/not_exists
		if c.Operator != "exists" && c.Operator != "not_exists" && c.Value == nil {
			return errors.New("value is required for this operator")
		}
	}

	// Validate composite children
	for _, child := range c.All {
		if err := child.Validate(); err != nil {
			return fmt.Errorf("all condition: %w", err)
		}
	}
	for _, child := range c.Any {
		if err := child.Validate(); err != nil {
			return fmt.Errorf("any condition: %w", err)
		}
	}
	if c.Not != nil {
		if err := c.Not.Validate(); err != nil {
			return fmt.Errorf("not condition: %w", err)
		}
	}

	return nil
}

// ActionDef defines an action to be executed.
type ActionDef struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// Validate checks if the action definition is valid.
func (a ActionDef) Validate() error {
	if a.Type == "" {
		return errors.New("action type is required")
	}
	return nil
}

// LogEntry represents a log entry that can be evaluated by the rule engine.
// This is a generic interface that can be adapted to different log entry types.
type LogEntry interface {
	// GetField returns the value of a field, supporting dot notation for nested fields.
	GetField(field string) (interface{}, bool)
	// SetField sets the value of a field, supporting dot notation for nested fields.
	SetField(field string, value interface{}) error
	// DeleteField deletes a field, supporting dot notation for nested fields.
	DeleteField(field string) error
	// Clone creates a copy of the log entry.
	Clone() LogEntry
	// Raw returns the raw map representation of the log entry.
	Raw() map[string]interface{}
}

// MapLogEntry is a LogEntry implementation backed by a map[string]interface{}.
type MapLogEntry struct {
	data map[string]interface{}
}

// NewMapLogEntry creates a new MapLogEntry.
func NewMapLogEntry(data map[string]interface{}) *MapLogEntry {
	if data == nil {
		data = make(map[string]interface{})
	}
	return &MapLogEntry{data: data}
}

// GetField returns the value of a field with dot notation support.
func (m *MapLogEntry) GetField(field string) (interface{}, bool) {
	parts := strings.Split(field, ".")
	current := m.data

	for i, part := range parts {
		if i == len(parts)-1 {
			val, ok := current[part]
			return val, ok
		}

		next, ok := current[part]
		if !ok {
			return nil, false
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = nextMap
	}

	return nil, false
}

// SetField sets the value of a field with dot notation support.
func (m *MapLogEntry) SetField(field string, value interface{}) error {
	parts := strings.Split(field, ".")
	current := m.data

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}

		next, ok := current[part]
		if !ok {
			// Create nested map if it doesn't exist
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
			continue
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot set nested field on non-map value at %s", strings.Join(parts[:i+1], "."))
		}
		current = nextMap
	}

	return nil
}

// DeleteField deletes a field with dot notation support.
func (m *MapLogEntry) DeleteField(field string) error {
	parts := strings.Split(field, ".")
	current := m.data

	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
			return nil
		}

		next, ok := current[part]
		if !ok {
			return nil // Field doesn't exist, nothing to do
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot delete nested field from non-map value at %s", strings.Join(parts[:i+1], "."))
		}
		current = nextMap
	}

	return nil
}

// Clone creates a deep copy of the log entry.
func (m *MapLogEntry) Clone() LogEntry {
	return &MapLogEntry{
		data: deepCopyMap(m.data),
	}
}

// Raw returns the raw map data.
func (m *MapLogEntry) Raw() map[string]interface{} {
	return m.data
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopyMap(val)
		case []interface{}:
			result[k] = deepCopySlice(val)
		default:
			result[k] = val
		}
	}
	return result
}

func deepCopySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = deepCopyMap(val)
		case []interface{}:
			result[i] = deepCopySlice(val)
		default:
			result[i] = val
		}
	}
	return result
}

// RuleResult represents the result of evaluating a rule against a log entry.
type RuleResult struct {
	Matched      bool          `json:"matched"`
	RuleID       string        `json:"rule_id"`
	RuleName     string        `json:"rule_name"`
	Actions      []ActionResult `json:"actions,omitempty"`
	Terminate    bool          `json:"terminate"`     // If true, stop evaluating further rules
	ShouldKeep   bool          `json:"should_keep"`   // Whether the log should be kept (for drop/keep)
	EvaluatedAt  time.Time     `json:"evaluated_at"`
	Errors       []error       `json:"-"`             // Non-serialized errors
}

// ActionResult represents the result of executing an action.
type ActionResult struct {
	Type     string                 `json:"type"`
	Success  bool                   `json:"success"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates a rule.
func (r *Rule) Validate() error {
	if r.ID == "" {
		return errors.New("rule ID is required")
	}
	if r.Name == "" {
		return errors.New("rule name is required")
	}
	if err := r.Condition.Validate(); err != nil {
		return fmt.Errorf("invalid condition: %w", err)
	}
	for i, action := range r.Actions {
		if err := action.Validate(); err != nil {
			return fmt.Errorf("invalid action %d: %w", i, err)
		}
	}
	return nil
}

// UnmarshalJSON implements JSON unmarshaling with validation.
func (r *Rule) UnmarshalJSON(data []byte) error {
	type Alias Rule
	aux := &struct{ *Alias }{Alias: (*Alias)(r)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	return nil
}

// RuleStorage defines the interface for rule storage.
type RuleStorage interface {
	// LoadRules loads all rules from storage.
	LoadRules() ([]*Rule, error)
}

// Operator constants
const (
	OpEq         = "eq"
	OpNe         = "ne"
	OpGt         = "gt"
	OpLt         = "lt"
	OpGe         = "ge"
	OpLe         = "le"
	OpContains   = "contains"
	OpStartsWith = "starts_with"
	OpEndsWith   = "ends_with"
	OpMatches    = "matches"
	OpIn         = "in"
	OpNotIn      = "not_in"
	OpExists     = "exists"
	OpNotExists  = "not_exists"
)

// Action type constants
const (
	ActionKeep     = "keep"
	ActionDrop     = "drop"
	ActionSample   = "sample"
	ActionMask     = "mask"
	ActionTruncate = "truncate"
	ActionExtract  = "extract"
	ActionRename   = "rename"
	ActionRemove   = "remove"
	ActionSet      = "set"
	ActionMark     = "mark"
)

// regexCache caches compiled regex patterns.
var regexCache = make(map[string]*regexp.Regexp)
