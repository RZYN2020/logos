package rule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ConditionEvaluator evaluates conditions against log entries.
type ConditionEvaluator struct {
	operators map[string]Operator
}

// NewConditionEvaluator creates a new condition evaluator.
func NewConditionEvaluator() *ConditionEvaluator {
	e := &ConditionEvaluator{
		operators: make(map[string]Operator),
	}
	e.registerBuiltinOperators()
	return e
}

// RegisterOperator registers a custom operator.
func (e *ConditionEvaluator) RegisterOperator(name string, op Operator) {
	e.operators[name] = op
}

// Evaluate evaluates a condition against a log entry.
func (e *ConditionEvaluator) Evaluate(cond Condition, entry LogEntry) (bool, error) {
	// Handle composite conditions
	if cond.IsComposite() {
		return e.evaluateComposite(cond, entry)
	}

	// Handle single condition
	return e.evaluateSingle(cond, entry)
}

// evaluateSingle evaluates a single condition.
func (e *ConditionEvaluator) evaluateSingle(cond Condition, entry LogEntry) (bool, error) {
	// Handle exists/not_exists operators specially
	if cond.Operator == OpExists {
		_, exists := entry.GetField(cond.Field)
		return exists, nil
	}
	if cond.Operator == OpNotExists {
		_, exists := entry.GetField(cond.Field)
		return !exists, nil
	}

	// Get the field value
	value, exists := entry.GetField(cond.Field)
	if !exists {
		return false, nil
	}

	// Get the operator
	op, ok := e.operators[cond.Operator]
	if !ok {
		return false, fmt.Errorf("unknown operator: %s", cond.Operator)
	}

	// Evaluate
	return op.Evaluate(value, cond.Value)
}

// evaluateComposite evaluates a composite condition.
func (e *ConditionEvaluator) evaluateComposite(cond Condition, entry LogEntry) (bool, error) {
	// Handle 'all' - all conditions must be true
	if len(cond.All) > 0 {
		for _, child := range cond.All {
			result, err := e.Evaluate(child, entry)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// Handle 'any' - at least one condition must be true
	if len(cond.Any) > 0 {
		for _, child := range cond.Any {
			result, err := e.Evaluate(child, entry)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// Handle 'not' - the condition must be false
	if cond.Not != nil {
		result, err := e.Evaluate(*cond.Not, entry)
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	return false, fmt.Errorf("invalid composite condition: no all/any/not")
}

// registerBuiltinOperators registers all built-in operators.
func (e *ConditionEvaluator) registerBuiltinOperators() {
	// Equality operators
	e.RegisterOperator(OpEq, &eqOperator{})
	e.RegisterOperator(OpNe, &neOperator{})

	// Comparison operators
	e.RegisterOperator(OpGt, &gtOperator{})
	e.RegisterOperator(OpLt, &ltOperator{})
	e.RegisterOperator(OpGe, &geOperator{})
	e.RegisterOperator(OpLe, &leOperator{})

	// String operators
	e.RegisterOperator(OpContains, &containsOperator{})
	e.RegisterOperator(OpStartsWith, &startsWithOperator{})
	e.RegisterOperator(OpEndsWith, &endsWithOperator{})
	e.RegisterOperator(OpMatches, &matchesOperator{})

	// Set operators
	e.RegisterOperator(OpIn, &inOperator{})
	e.RegisterOperator(OpNotIn, &notInOperator{})
}

// Operator defines the interface for condition operators.
type Operator interface {
	Evaluate(actual, expected interface{}) (bool, error)
}

// eqOperator implements equality check.
type eqOperator struct{}

func (o *eqOperator) Evaluate(actual, expected interface{}) (bool, error) {
	return compareValues(actual, expected) == 0, nil
}

// neOperator implements inequality check.
type neOperator struct{}

func (o *neOperator) Evaluate(actual, expected interface{}) (bool, error) {
	return compareValues(actual, expected) != 0, nil
}

// gtOperator implements greater than check.
type gtOperator struct{}

func (o *gtOperator) Evaluate(actual, expected interface{}) (bool, error) {
	return compareValues(actual, expected) > 0, nil
}

// ltOperator implements less than check.
type ltOperator struct{}

func (o *ltOperator) Evaluate(actual, expected interface{}) (bool, error) {
	return compareValues(actual, expected) < 0, nil
}

// geOperator implements greater than or equal check.
type geOperator struct{}

func (o *geOperator) Evaluate(actual, expected interface{}) (bool, error) {
	return compareValues(actual, expected) >= 0, nil
}

// leOperator implements less than or equal check.
type leOperator struct{}

func (o *leOperator) Evaluate(actual, expected interface{}) (bool, error) {
	return compareValues(actual, expected) <= 0, nil
}

// containsOperator implements string contains check.
type containsOperator struct{}

func (o *containsOperator) Evaluate(actual, expected interface{}) (bool, error) {
	actualStr, ok := toString(actual)
	if !ok {
		return false, nil
	}
	expectedStr, ok := expected.(string)
	if !ok {
		return false, nil
	}
	return strings.Contains(actualStr, expectedStr), nil
}

// startsWithOperator implements string starts with check.
type startsWithOperator struct{}

func (o *startsWithOperator) Evaluate(actual, expected interface{}) (bool, error) {
	actualStr, ok := toString(actual)
	if !ok {
		return false, nil
	}
	expectedStr, ok := expected.(string)
	if !ok {
		return false, nil
	}
	return strings.HasPrefix(actualStr, expectedStr), nil
}

// endsWithOperator implements string ends with check.
type endsWithOperator struct{}

func (o *endsWithOperator) Evaluate(actual, expected interface{}) (bool, error) {
	actualStr, ok := toString(actual)
	if !ok {
		return false, nil
	}
	expectedStr, ok := expected.(string)
	if !ok {
		return false, nil
	}
	return strings.HasSuffix(actualStr, expectedStr), nil
}

// matchesOperator implements regex match check.
type matchesOperator struct{}

func (o *matchesOperator) Evaluate(actual, expected interface{}) (bool, error) {
	actualStr, ok := toString(actual)
	if !ok {
		return false, nil
	}
	pattern, ok := expected.(string)
	if !ok {
		return false, fmt.Errorf("matches operator expects string pattern, got %T", expected)
	}

	// Check cache
	re, ok := regexCache[pattern]
	if !ok {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern: %w", err)
		}
		regexCache[pattern] = re
	}

	return re.MatchString(actualStr), nil
}

// inOperator implements set membership check.
type inOperator struct{}

func (o *inOperator) Evaluate(actual, expected interface{}) (bool, error) {
	list, ok := toSlice(expected)
	if !ok {
		return false, fmt.Errorf("in operator expects array/list value")
	}
	for _, item := range list {
		if compareValues(actual, item) == 0 {
			return true, nil
		}
	}
	return false, nil
}

// notInOperator implements set non-membership check.
type notInOperator struct{}

func (o *notInOperator) Evaluate(actual, expected interface{}) (bool, error) {
	result, err := (&inOperator{}).Evaluate(actual, expected)
	if err != nil {
		return false, err
	}
	return !result, nil
}

// compareValues compares two values and returns -1, 0, or 1.
func compareValues(a, b interface{}) int {
	// Handle nil
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Convert to comparable types
	aNum, aIsNum := toNumber(a)
	bNum, bIsNum := toNumber(b)

	if aIsNum && bIsNum {
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	// String comparison
	aStr, aIsStr := toString(a)
	bStr, bIsStr := toString(b)

	if aIsStr && bIsStr {
		if aStr < bStr {
			return -1
		}
		if aStr > bStr {
			return 1
		}
		return 0
	}

	// Fallback to fmt.Sprintf comparison
	aStrFinal := fmt.Sprintf("%v", a)
	bStrFinal := fmt.Sprintf("%v", b)

	if aStrFinal < bStrFinal {
		return -1
	}
	if aStrFinal > bStrFinal {
		return 1
	}
	return 0
}

// toNumber attempts to convert a value to float64.
func toNumber(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

// toString attempts to convert a value to string.
func toString(v interface{}) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		return fmt.Sprintf("%v", val), true
	}
}

// toSlice attempts to convert a value to []interface{}.
func toSlice(v interface{}) ([]interface{}, bool) {
	switch val := v.(type) {
	case []interface{}:
		return val, true
	case []string:
		result := make([]interface{}, len(val))
		for i, s := range val {
			result[i] = s
		}
		return result, true
	case []int:
		result := make([]interface{}, len(val))
		for i, i2 := range val {
			result[i] = i2
		}
		return result, true
	case []float64:
		result := make([]interface{}, len(val))
		for i, f := range val {
			result[i] = f
		}
		return result, true
	}
	return nil, false
}
