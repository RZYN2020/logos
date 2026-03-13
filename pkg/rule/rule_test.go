package rule

import (
	"testing"
)

func TestConditionEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		entry     LogEntry
		want      bool
		wantErr   bool
	}{
		{
			name: "eq operator - match",
			condition: Condition{
				Field:    "level",
				Operator: OpEq,
				Value:    "ERROR",
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "ERROR"}),
			want:    true,
			wantErr: false,
		},
		{
			name: "eq operator - no match",
			condition: Condition{
				Field:    "level",
				Operator: OpEq,
				Value:    "ERROR",
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "INFO"}),
			want:    false,
			wantErr: false,
		},
		{
			name: "contains operator - match",
			condition: Condition{
				Field:    "message",
				Operator: OpContains,
				Value:    "error",
			},
			entry:   NewMapLogEntry(map[string]interface{}{"message": "connection error occurred"}),
			want:    true,
			wantErr: false,
		},
		{
			name: "contains operator - no match",
			condition: Condition{
				Field:    "message",
				Operator: OpContains,
				Value:    "error",
			},
			entry:   NewMapLogEntry(map[string]interface{}{"message": "success"}),
			want:    false,
			wantErr: false,
		},
		{
			name: "gt operator - numbers",
			condition: Condition{
				Field:    "count",
				Operator: OpGt,
				Value:    10,
			},
			entry:   NewMapLogEntry(map[string]interface{}{"count": 15}),
			want:    true,
			wantErr: false,
		},
		{
			name: "exists operator - field exists",
			condition: Condition{
				Field:    "trace_id",
				Operator: OpExists,
			},
			entry:   NewMapLogEntry(map[string]interface{}{"trace_id": "abc123"}),
			want:    true,
			wantErr: false,
		},
		{
			name: "exists operator - field not exists",
			condition: Condition{
				Field:    "trace_id",
				Operator: OpExists,
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "INFO"}),
			want:    false,
			wantErr: false,
		},
		{
			name: "not_exists operator",
			condition: Condition{
				Field:    "trace_id",
				Operator: OpNotExists,
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "INFO"}),
			want:    true,
			wantErr: false,
		},
	}

	evaluator := NewConditionEvaluator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.condition, tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("Condition.Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Condition.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositeCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		entry     LogEntry
		want      bool
		wantErr   bool
	}{
		{
			name: "all conditions - all match",
			condition: Condition{
				All: []Condition{
					{Field: "level", Operator: OpEq, Value: "ERROR"},
					{Field: "service", Operator: OpEq, Value: "api"},
				},
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "ERROR", "service": "api"}),
			want:    true,
			wantErr: false,
		},
		{
			name: "all conditions - one fails",
			condition: Condition{
				All: []Condition{
					{Field: "level", Operator: OpEq, Value: "ERROR"},
					{Field: "service", Operator: OpEq, Value: "api"},
				},
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "ERROR", "service": "web"}),
			want:    false,
			wantErr: false,
		},
		{
			name: "any conditions - one matches",
			condition: Condition{
				Any: []Condition{
					{Field: "level", Operator: OpEq, Value: "ERROR"},
					{Field: "level", Operator: OpEq, Value: "PANIC"},
				},
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "ERROR"}),
			want:    true,
			wantErr: false,
		},
		{
			name: "any conditions - none match",
			condition: Condition{
				Any: []Condition{
					{Field: "level", Operator: OpEq, Value: "ERROR"},
					{Field: "level", Operator: OpEq, Value: "PANIC"},
				},
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "INFO"}),
			want:    false,
			wantErr: false,
		},
		{
			name: "not condition",
			condition: Condition{
				Not: &Condition{Field: "level", Operator: OpEq, Value: "DEBUG"},
			},
			entry:   NewMapLogEntry(map[string]interface{}{"level": "INFO"}),
			want:    true,
			wantErr: false,
		},
		{
			name: "nested conditions",
			condition: Condition{
				All: []Condition{
					{Field: "service", Operator: OpEq, Value: "api"},
					{
						Any: []Condition{
							{Field: "level", Operator: OpEq, Value: "ERROR"},
							{Field: "level", Operator: OpEq, Value: "PANIC"},
						},
					},
				},
			},
			entry:   NewMapLogEntry(map[string]interface{}{"service": "api", "level": "ERROR"}),
			want:    true,
			wantErr: false,
		},
	}

	evaluator := NewConditionEvaluator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.condition, tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("Condition.Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Condition.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuleEngine(t *testing.T) {
	tests := []struct {
		name       string
		rules      []*Rule
		entry      LogEntry
		wantKeep   bool
		wantMatch  bool
		wantTerminate bool
	}{
		{
			name: "single rule - matches and keeps",
			rules: []*Rule{
				{
					ID:      "rule1",
					Name:    "Error Filter",
					Enabled: true,
					Condition: Condition{
						Field:    "level",
						Operator: OpEq,
						Value:    "ERROR",
					},
					Actions: []ActionDef{
						{Type: ActionKeep},
					},
				},
			},
			entry:         NewMapLogEntry(map[string]interface{}{"level": "ERROR"}),
			wantKeep:      true,
			wantMatch:     true,
			wantTerminate: true,
		},
		{
			name: "single rule - matches and drops",
			rules: []*Rule{
				{
					ID:      "rule1",
					Name:    "Debug Filter",
					Enabled: true,
					Condition: Condition{
						Field:    "level",
						Operator: OpEq,
						Value:    "DEBUG",
					},
					Actions: []ActionDef{
						{Type: ActionDrop},
					},
				},
			},
			entry:         NewMapLogEntry(map[string]interface{}{"level": "DEBUG"}),
			wantKeep:      false,
			wantMatch:     true,
			wantTerminate: true,
		},
		{
			name: "no rules match - default keep",
			rules: []*Rule{
				{
					ID:      "rule1",
					Name:    "Error Filter",
					Enabled: true,
					Condition: Condition{
						Field:    "level",
						Operator: OpEq,
						Value:    "ERROR",
					},
					Actions: []ActionDef{
						{Type: ActionDrop},
					},
				},
			},
			entry:         NewMapLogEntry(map[string]interface{}{"level": "INFO"}),
			wantKeep:      true,
			wantMatch:     false,
			wantTerminate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRuleEngine(RuleEngineConfig{})
			engine.SetRules(tt.rules)

			shouldKeep, results, _ := engine.Evaluate(tt.entry)

			if shouldKeep != tt.wantKeep {
				t.Errorf("RuleEngine.Evaluate() shouldKeep = %v, want %v", shouldKeep, tt.wantKeep)
			}

			hasMatch := len(results) > 0
			if hasMatch != tt.wantMatch {
				t.Errorf("RuleEngine.Evaluate() hasMatch = %v, want %v", hasMatch, tt.wantMatch)
			}

			if hasMatch && tt.wantTerminate {
				if !results[0].Terminate {
					t.Errorf("RuleEngine.Evaluate() terminate = false, want true")
				}
			}
		})
	}
}

func TestActionTransform(t *testing.T) {
	tests := []struct {
		name        string
		entry       LogEntry
		actionType  string
		config      map[string]interface{}
		wantSuccess bool
		wantField   string
		wantValue   interface{}
	}{
		{
			name:       "set action",
			entry:      NewMapLogEntry(map[string]interface{}{"level": "INFO"}),
			actionType: ActionSet,
			config: map[string]interface{}{
				"field": "environment",
				"value": "production",
			},
			wantSuccess: true,
			wantField:   "environment",
			wantValue:   "production",
		},
		{
			name:       "remove action",
			entry:      NewMapLogEntry(map[string]interface{}{"level": "INFO", "temp": "delete"}),
			actionType: ActionRemove,
			config: map[string]interface{}{
				"field": "temp",
			},
			wantSuccess: true,
			wantField:   "temp",
			wantValue:   nil, // Field should be removed
		},
	}

	executor := NewActionExecutor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := ActionDef{
				Type:   tt.actionType,
				Config: tt.config,
			}

			_, metadata, err := executor.Execute(tt.entry, action)
			gotSuccess := err == nil

			if gotSuccess != tt.wantSuccess {
				t.Errorf("Action.Execute() success = %v, want %v, err = %v", gotSuccess, tt.wantSuccess, err)
			}

			if tt.wantField != "" {
				val, exists := tt.entry.GetField(tt.wantField)
				if tt.wantValue == nil {
					if exists {
						t.Errorf("Action.Execute() field %s should be removed", tt.wantField)
					}
				} else {
					if !exists || val != tt.wantValue {
						t.Errorf("Action.Execute() field %s = %v, want %v", tt.wantField, val, tt.wantValue)
					}
				}
			}

			_ = metadata
		})
	}
}
