// Package rule provides a unified rule engine for log processing across the Logos platform.
//
// The rule engine implements a Condition + Action model:
//   - Conditions: Support 14+ operators (eq, ne, gt, lt, ge, le, contains, starts_with, ends_with, matches, in, not_in, exists, not_exists)
//   - Composite conditions: Support all/any/not with arbitrary nesting
//   - Actions: Support 10+ action types (keep, drop, sample, mask, truncate, extract, rename, remove, set, mark)
//   - Storage: Support in-memory and ETCD storage with hot reload
//
// Basic usage:
//
//	engine := rule.NewRuleEngine(rule.RuleEngineConfig{})
//	engine.SetRules([]*rule.Rule{...})
//	shouldKeep, results, errors := engine.Evaluate(entry)
//
// For ETCD-backed storage with hot reload:
//
//	storage, _ := storage.NewETCDStorage(storage.ETCDStorageConfig{Endpoints: []string{"localhost:2379"}})
//	engine.LoadRules(storage)
package rule
